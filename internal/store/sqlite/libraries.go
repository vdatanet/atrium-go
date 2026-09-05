package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The store implements what 003's domain asked for. The assertion is here
// rather than in a test for the reason installation.go's and users.go's are: it
// is what makes the interface load-bearing while the subcommand that will call
// it is still unwritten.
//
// It is also the only place the absence spec §3.6 relies on is enforced from
// this side. ports.LibraryStore offers no way to write collection_type or
// case_sensitive after creation, and this type may not grow one and stay an
// implementation of it — a method added here that wrote either would compile,
// which is why the assertion the task asks for is over the interface and not
// over this file.
var _ ports.LibraryStore = (*Store)(nil)

// libraryColumns is the SELECT list every read of a library shares, in the
// order scanLibrary reads them.
//
// One constant rather than two spellings, for users.go's reason: the two reads
// differ only in their WHERE and ORDER BY clauses, and a column list that
// drifted between them would be a field filled from the wrong position — which
// compiles, and which nothing notices until a collection type turns up where a
// name belongs.
const libraryColumns = `id, name, name_folded, collection_type, case_sensitive, created_at`

// scanLibrary reads one row of libraryColumns. The roots are not in it: they
// are a table of their own and are read separately, so what this returns is a
// library with no roots yet rather than a library with none.
func scanLibrary(row interface{ Scan(...any) error }) (ports.Library, error) {
	var (
		library       ports.Library
		caseSensitive int64
		createdAt     int64
	)
	if err := row.Scan(
		&library.ID, &library.Name, &library.NameFolded,
		&library.CollectionType, &caseSensitive, &createdAt,
	); err != nil {
		return ports.Library{}, err
	}
	library.CaseSensitive = caseSensitive != 0
	library.CreatedAt = units.TimeFromTicks(units.Ticks(createdAt))
	return library, nil
}

// CreateLibrary writes a new library and its roots, in one transaction.
//
// One transaction rather than two calls because a library is its roots: a row
// in `libraries` with nothing in `library_roots` is a library every scan reads
// as empty, and 003 plan §6.5's second guard would then refuse the first scan
// of it as a root that lost its files. Half of this write is a worse state than
// none of it.
//
// There is no ON CONFLICT clause and deliberately none. A second library whose
// folded name is already taken is a mistake and not an update: the identifier
// here is allocated, so an upsert would move somebody else's identity onto a
// new library's name — and every item under the old one would keep pointing at
// a library that had quietly become a different one. The unique index refuses
// it instead.
func (s *Store) CreateLibrary(ctx context.Context, library ports.Library) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: creating the library %q: %w", s.path, library.Name, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	// Rollback after a commit is a no-op, so this is the whole of the "or none
	// of it" half: every return between here and the commit leaves the
	// database exactly as it was.
	defer transaction.Rollback()

	caseSensitive := 0
	if library.CaseSensitive {
		caseSensitive = 1
	}
	if _, err := transaction.ExecContext(ctx,
		`INSERT INTO libraries (id, name, name_folded, collection_type, case_sensitive, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		library.ID, library.Name, library.NameFolded, library.CollectionType,
		int64(caseSensitive), int64(library.CreatedAt.Ticks())); err != nil {
		return failed(err)
	}
	if err := insertRoots(ctx, transaction, library.ID, library.Roots); err != nil {
		return failed(err)
	}
	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// insertRoots writes roots for one library, the slice position being the
// ordinal.
//
// The ordinal comes from the position rather than from a counter the caller
// supplies, which is what makes "the order the operator gave them" a property
// of the argument instead of a number two call sites could disagree about.
func insertRoots(ctx context.Context, transaction *sql.Tx, libraryID string, roots []string) error {
	for ordinal, root := range roots {
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO library_roots (library_id, ordinal, path) VALUES (?, ?, ?)`,
			libraryID, int64(ordinal), root); err != nil {
			return fmt.Errorf("root %d (%s): %w", ordinal, root, err)
		}
	}
	return nil
}

// Libraries returns every configured library, in a stated order, with each
// library's roots in ordinal order.
//
// The order is name_folded and then id, stated here because architecture §2
// forbids one that derives from anything but stable input. The tie-break on id
// is not decoration: name_folded is unique today, so the pair can never tie —
// and if that uniqueness ever moved, the order would silently become an
// arbitrary one, which is exactly what the ordering exists to prevent.
//
// The roots are read in one query rather than one per library. That is not
// only about the number of round trips: a per-library read is filtered on
// library_id, which SQLite answers through the primary-key index and therefore
// in ordinal order whether or not anything asked it to — so the ORDER BY that
// makes the ordinal load-bearing would be a clause no test in this package
// could observe. Reading the whole table makes it observable, and
// TestLibraryRootsAreOrderedByTheirOrdinalAndNotByWhereTheRowsSit is what
// observes it.
func (s *Store) Libraries(ctx context.Context) ([]ports.Library, error) {
	libraries, err := s.readLibraries(ctx,
		`SELECT `+libraryColumns+` FROM libraries ORDER BY name_folded, id`)
	if err != nil {
		return nil, fmt.Errorf("%s: reading the libraries: %w", s.path, err)
	}

	roots, err := s.readAllRoots(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: reading the libraries: %w", s.path, err)
	}
	for i := range libraries {
		libraries[i].Roots = roots[libraries[i].ID]
	}
	return libraries, nil
}

func (s *Store) readLibraries(ctx context.Context, query string, arguments ...any) ([]ports.Library, error) {
	rows, err := s.reader.QueryContext(ctx, query, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var libraries []ports.Library
	for rows.Next() {
		library, err := scanLibrary(rows)
		if err != nil {
			return nil, err
		}
		libraries = append(libraries, library)
	}
	return libraries, rows.Err()
}

// readAllRoots returns every configured root, grouped by library and ordered
// by ordinal inside each group.
func (s *Store) readAllRoots(ctx context.Context) (map[string][]string, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT library_id, path FROM library_roots ORDER BY library_id, ordinal`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	roots := map[string][]string{}
	for rows.Next() {
		var libraryID, root string
		if err := rows.Scan(&libraryID, &root); err != nil {
			return nil, err
		}
		roots[libraryID] = append(roots[libraryID], root)
	}
	return roots, rows.Err()
}

// LibraryByFoldedName finds the library whose folded name is folded.
//
// An absence is reported as false and not as an error: an operator naming a
// library that is not there is an answer the subcommand prints, and `add`
// looking before it writes expects exactly this.
func (s *Store) LibraryByFoldedName(ctx context.Context, folded string) (ports.Library, bool, error) {
	library, err := scanLibrary(s.reader.QueryRowContext(ctx,
		`SELECT `+libraryColumns+` FROM libraries WHERE name_folded = ?`, folded))
	if errors.Is(err, sql.ErrNoRows) {
		return ports.Library{}, false, nil
	}
	if err != nil {
		return ports.Library{}, false, fmt.Errorf("%s: reading a library: %w", s.path, err)
	}

	library.Roots, err = s.rootsFor(ctx, library.ID)
	if err != nil {
		return ports.Library{}, false, fmt.Errorf("%s: reading a library's roots: %w", s.path, err)
	}
	return library, true, nil
}

// rootsFor returns one library's roots in ordinal order.
//
// The ORDER BY here is the one clause in this file no test can observe, and it
// is measured rather than assumed: removing it leaves the whole suite green.
// The filter is on library_id, which SQLite answers through the primary-key
// index on (library_id, ordinal), so the rows arrive in ordinal order whether
// or not anything asked. It stays because the ordering is a property of the
// contract and not of a query plan, and because the plan changes the day
// somebody adds an index — and it is written down here so the next person does
// not spend an afternoon finding out that a mutation of it survives.
// [measurement: 003 T10, 24 mutations, this the only survivor, 2026-09-05]
func (s *Store) rootsFor(ctx context.Context, libraryID string) ([]string, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT path FROM library_roots WHERE library_id = ? ORDER BY ordinal`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roots []string
	for rows.Next() {
		var root string
		if err := rows.Scan(&root); err != nil {
			return nil, err
		}
		roots = append(roots, root)
	}
	return roots, rows.Err()
}

// RenameLibrary replaces a library's name and the folded spelling uniqueness is
// enforced over.
//
// Both in one statement, because they are one fact spelled twice: a row whose
// name and name_folded disagree is unreachable by the name it displays.
func (s *Store) RenameLibrary(ctx context.Context, id, name, folded string) error {
	return s.updateOneLibrary(ctx, "renaming a library", id,
		`UPDATE libraries SET name = ?, name_folded = ? WHERE id = ?`, name, folded, id)
}

// ReplaceRoots replaces a library's roots with roots, in that order.
//
// The library is looked up inside the transaction rather than trusted, and the
// reason is the one case the foreign key cannot catch: replacing the roots of a
// library that does not exist with *no* roots is a DELETE that matches nothing
// followed by no INSERT at all, which succeeds and does nothing. An operator
// who mistyped a library's name would be told the roots were replaced.
func (s *Store) ReplaceRoots(ctx context.Context, id string, roots []string) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: replacing the roots of library %s: %w", s.path, id, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	defer transaction.Rollback()

	var exists int
	err = transaction.QueryRowContext(ctx, `SELECT 1 FROM libraries WHERE id = ?`, id).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return failed(errors.New("there is no such library"))
	}
	if err != nil {
		return failed(err)
	}

	if _, err := transaction.ExecContext(ctx,
		`DELETE FROM library_roots WHERE library_id = ?`, id); err != nil {
		return failed(err)
	}
	if err := insertRoots(ctx, transaction, id, roots); err != nil {
		return failed(err)
	}
	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// RemoveLibrary deletes a library and, through the cascade the schema declares,
// the roots configured under it.
//
// It does not touch the items scanned under it. They are in the derived half,
// which holds no reference into this one (architecture §6), so nothing here can
// reach them and the cascade cannot grow to.
func (s *Store) RemoveLibrary(ctx context.Context, id string) error {
	return s.updateOneLibrary(ctx, "removing a library", id,
		`DELETE FROM libraries WHERE id = ?`, id)
}

// updateOneLibrary runs a statement that must change exactly one library, and
// reports it when it did not.
//
// The guard is 001's, for the reason installation.go gives: an UPDATE or a
// DELETE that matched nothing succeeds, so without it every write against a
// library nobody has looks exactly like a write that worked.
func (s *Store) updateOneLibrary(ctx context.Context, what, id, statement string, arguments ...any) error {
	result, err := s.writer.ExecContext(ctx, statement, arguments...)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: %s changed %d rows for library %s, want 1", s.path, what, affected, id)
	}
	return nil
}
