package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The store implements the derived contract as well as the precious one. The
// assertion is here rather than in a test for libraries.go's reason, and it is
// the line 003 T11's handover said would not compile until all six methods
// exist — [Store.RebuildDerived] was the only one of them written before this
// file.
//
// It matters more here than anywhere else so far, because **nothing outside a
// test calls any of these methods until 003 T13**. The interface is what makes
// them load-bearing in the meantime.
var _ ports.ItemStore = (*Store)(nil)

// itemColumns is the SELECT list every read of an item shares, in the order
// scanItem reads them.
//
// One constant rather than a spelling per read, for libraryColumns' reason: a
// list that drifted between two reads is a field filled from the wrong
// position, which compiles and which nothing notices until a sort key turns up
// where a name belongs.
const itemColumns = `id, library_id, parent_id, type, name, sort_key, path, root_ordinal, ` +
	`index_number, parent_index_number, index_number_end, production_year, premiere_date, unplaceable`

// scanItem reads one row of itemColumns. The files are not in it: they are a
// table of their own and are read separately.
func scanItem(row interface{ Scan(...any) error }) (ports.ScannedItem, error) {
	var (
		item        ports.ScannedItem
		parentID    sql.NullString
		path        sql.NullString
		rootOrdinal sql.NullInt64
		index       sql.NullInt64
		parentIndex sql.NullInt64
		indexEnd    sql.NullInt64
		year        sql.NullInt64
		premiere    sql.Null[int64]
		unplaceable int64
	)
	if err := row.Scan(
		&item.ID, &item.LibraryID, &parentID, &item.Type, &item.Name, &item.SortKey,
		&path, &rootOrdinal, &index, &parentIndex, &indexEnd, &year, &premiere, &unplaceable,
	); err != nil {
		return ports.ScannedItem{}, err
	}
	item.ParentID = parentID.String
	item.Path = path.String
	item.RootOrdinal = int(rootOrdinal.Int64)
	item.IndexNumber = optionalInt(index)
	item.ParentIndexNumber = optionalInt(parentIndex)
	item.IndexNumberEnd = optionalInt(indexEnd)
	item.ProductionYear = optionalInt(year)
	item.PremiereDate = nullableTime(premiere)
	item.Unplaceable = unplaceable != 0
	return item, nil
}

// optionalInt turns a nullable column into the pointer the record carries.
//
// The pointers are not decoration and this conversion is where they are earned:
// absent and zero are different answers, because season 0 is `Specials`
// (003 §3.4) and a season with no number at all is not it. A column read into a
// plain int would make the two one.
func optionalInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	n := int(value.Int64)
	return &n
}

// ItemsForLibrary returns every item recorded under a library, ordered by sort
// key and then by identifier, with each item's files in ordinal order.
//
// # The order is the contract and not a convenience
//
// A scan compares this set against the one it has just derived, so a store
// answering in storage order would make a scan's answer depend on the order
// rows happened to be inserted in — Principle VII, one layer below where this
// project usually enforces it. The tie-break on the identifier is what makes
// the order total: sort keys are not unique, and two items sharing one would
// otherwise arrive in whichever order the sorter felt like.
//
// The comparison is SQLite's default `BINARY` collation and the column carries
// no `COLLATE` clause, which is ADR-0003's decision rather than an omission:
// `NOCASE` is ASCII-only, so it would order two names by a rule this project
// cannot explain and would leave every non-ASCII title in a place nothing
// derived. `TestTheStoredSortKeyComparesAsBytesAndNotUnderNOCASE` is where that
// is observable.
//
// The files are read in one query rather than one per item, for readAllRoots'
// reason as well as for the round trips.
func (s *Store) ItemsForLibrary(ctx context.Context, libraryID string) ([]ports.ScannedItem, error) {
	failed := func(err error) error {
		return fmt.Errorf("%s: reading the items of library %s: %w", s.path, libraryID, err)
	}

	rows, err := s.reader.QueryContext(ctx,
		`SELECT `+itemColumns+` FROM items WHERE library_id = ? ORDER BY sort_key, id`, libraryID)
	if err != nil {
		return nil, failed(err)
	}
	defer rows.Close()

	var items []ports.ScannedItem
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, failed(err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, failed(err)
	}

	files, err := s.filesForLibrary(ctx, libraryID)
	if err != nil {
		return nil, failed(err)
	}
	for i := range items {
		items[i].Files = files[items[i].ID]
	}
	return items, nil
}

// filesForLibrary returns every file behind an item of one library, grouped by
// item and in ordinal order inside each group.
//
// The ORDER BY carries a contract 003 plan §6.4's change detection rests on:
// the reconciliation compares two items' files **index by index**, so a store
// that returned the two parts of a multi-part film in storage order would
// report the film updated on every scan, for ever, with nothing failing.
//
// It is nevertheless a clause no test in this package can observe, and that is
// measured rather than assumed. The primary key on (item_id, ordinal) answers
// this shape whichever way the rows sit, so removing the clause leaves the
// suite green — the same survivor 003 T10 found on the single-library roots
// read, one table along. It stays because the ordering is a property of the
// contract rather than of a query plan, and because the plan changes the day
// somebody adds an index.
// [measurement: 003 T12, 25 mutations, this the only ordering one that survives,
// 2026-09-05]
func (s *Store) filesForLibrary(ctx context.Context, libraryID string) (map[string][]ports.ScannedFile, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT item_id, ordinal, path, size, modified_at
		   FROM item_files
		  WHERE item_id IN (SELECT id FROM items WHERE library_id = ?)
		  ORDER BY item_id, ordinal`, libraryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := map[string][]ports.ScannedFile{}
	for rows.Next() {
		var (
			itemID     string
			ordinal    int64
			file       ports.ScannedFile
			modifiedAt int64
		)
		if err := rows.Scan(&itemID, &ordinal, &file.Path, &file.Size, &modifiedAt); err != nil {
			return nil, err
		}
		file.Ordinal = int(ordinal)
		file.ModifiedAt = units.TimeFromTicks(units.Ticks(modifiedAt))
		files[itemID] = append(files[itemID], file)
	}
	return files, rows.Err()
}

// ErrRepeatedIdentifier is what a batch naming one item twice is refused with.
//
// It is a decision rather than defensive plumbing, and 003 T3 is where it was
// named: NFC has singleton mappings, so U+212A KELVIN SIGN normalises to `K`
// and a filesystem can hold `K.mkv` written both ways side by side. Two files,
// one derived identifier, and nothing in `internal/library` can notice —
// the derivation is a pure function of the key.
//
// So the pair reaches a batch, and there are only two answers. A last write
// winning silently gives a library that holds one of the two files and reports
// nothing, differently on each scan depending on which the walk read second.
// Failing the library's scan says which library and leaves the operator with a
// tree to look at. The second is the answer here.
var ErrRepeatedIdentifier = errors.New("the batch names one item twice")

// ApplyScanBatch writes one batch of additions and updates and renews the
// scanner's claim, in one transaction (003 plan §6.9).
//
// # Why the claim travels with the items
//
// The claim is renewed on every committed batch rather than by a timer beside
// the scan, and that is what makes `staleAfter` a number somebody can argue
// for: it has to exceed the time between two batches and not the time a whole
// library takes. Committing the renewal separately would give a scanner that
// could report progress it did not make, or hold a claim over work it had
// abandoned.
//
// The renewal is the **first** statement in the transaction rather than the
// last, so that a failure among the items rolls back a renewal that had already
// succeeded rather than one that had not yet happened. That is a property of
// the *test* rather than of the behaviour, and it is declared as such: moving
// the renewal to the end survives every assertion in this package, because both
// orders are inside one transaction and no failure this store can reach tells
// them apart. It is written this way so that the rollback the transaction
// clause is named for is a rollback of work that had been done
// [measurement: 003 T12, 25 mutations, 2 declared survivors, 2026-09-05].
//
// It is conditional on [ports.ScanBatch.ClaimedBy], which is why that field is
// on the batch. A renewal that did not name the claimant would let a scanner
// whose claim had gone stale and been taken renew one it no longer holds — two
// scanners writing one library, each believing it is alone.
func (s *Store) ApplyScanBatch(ctx context.Context, batch ports.ScanBatch) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: writing a scan batch for library %s: %w", s.path, batch.LibraryID, err)
	}

	// Before the transaction, because it is a fault in what was handed over
	// and not a failure of the write. See ErrRepeatedIdentifier.
	seen := make(map[string]struct{}, len(batch.Items))
	for _, item := range batch.Items {
		if _, repeated := seen[item.ID]; repeated {
			return failed(fmt.Errorf("%w: %s, at %q", ErrRepeatedIdentifier, item.ID, item.Path))
		}
		seen[item.ID] = struct{}{}
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	// Rollback after a commit is a no-op, so this is the whole of the "or none
	// of it" half: every return between here and the commit leaves the
	// database exactly as it was.
	defer transaction.Rollback()

	if err := renewClaim(ctx, transaction, batch); err != nil {
		return failed(err)
	}
	for _, item := range batch.Items {
		if err := writeItem(ctx, transaction, item); err != nil {
			return failed(fmt.Errorf("item %s (%s): %w", item.ID, item.Name, err))
		}
	}
	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// renewClaim stamps the claim this batch is written under, and refuses when the
// batch does not hold it.
//
// The refusal is what an UPDATE matching nothing would otherwise hide — 001's
// rows-affected rule, at the one place in this feature where a write that
// changed nothing means the scanner has already been replaced.
func renewClaim(ctx context.Context, transaction *sql.Tx, batch ports.ScanBatch) error {
	result, err := transaction.ExecContext(ctx,
		`UPDATE scan_state SET claimed_at = ? WHERE library_id = ? AND claimed_by = ?`,
		int64(batch.At.Ticks()), batch.LibraryID, batch.ClaimedBy)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("the claim is no longer held by %q: renewing it changed %d rows, want 1",
			batch.ClaimedBy, affected)
	}
	return nil
}

// writeItem writes one item and the files behind it, replacing both.
//
// The item is an upsert because a batch carries additions and updates together
// (003 plan §5) and the store is not told which is which — the reconciliation
// already decided, and a store that asked again would be deciding it twice
// from less information.
//
// The files are deleted and rewritten rather than upserted, because the set can
// shrink: a two-part film whose second part was removed keeps its identifier
// and loses a row, and an upsert leaves the old part behind as a media source
// pointing at a file that is not there.
func writeItem(ctx context.Context, transaction *sql.Tx, item ports.ScannedItem) error {
	if _, err := transaction.ExecContext(ctx,
		`INSERT INTO items (`+itemColumns+`)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     library_id          = excluded.library_id,
		     parent_id           = excluded.parent_id,
		     type                = excluded.type,
		     name                = excluded.name,
		     sort_key            = excluded.sort_key,
		     path                = excluded.path,
		     root_ordinal        = excluded.root_ordinal,
		     index_number        = excluded.index_number,
		     parent_index_number = excluded.parent_index_number,
		     index_number_end    = excluded.index_number_end,
		     production_year     = excluded.production_year,
		     premiere_date       = excluded.premiere_date,
		     unplaceable         = excluded.unplaceable`,
		item.ID, item.LibraryID, nullableText(item.ParentID), item.Type, item.Name, item.SortKey,
		nullableText(item.Path), int64(item.RootOrdinal),
		nullableInt(item.IndexNumber), nullableInt(item.ParentIndexNumber),
		nullableInt(item.IndexNumberEnd), nullableInt(item.ProductionYear),
		nullableTicks(item.PremiereDate), boolAsInteger(item.Unplaceable),
	); err != nil {
		return err
	}

	if _, err := transaction.ExecContext(ctx,
		`DELETE FROM item_files WHERE item_id = ?`, item.ID); err != nil {
		return err
	}
	for _, file := range item.Files {
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO item_files (item_id, ordinal, path, size, modified_at) VALUES (?, ?, ?, ?, ?)`,
			item.ID, int64(file.Ordinal), file.Path, file.Size, int64(file.ModifiedAt.Ticks()),
		); err != nil {
			return fmt.Errorf("file %d (%s): %w", file.Ordinal, file.Path, err)
		}
	}
	return nil
}

// nullableText writes the empty string as NULL.
//
// `parent_id` and `path` are the two columns this applies to and both mean
// something by it: a library's own row hangs from nothing, and an inferred
// container has no directory of its own. Storing the empty string instead would
// make "no parent" a parent whose identifier is empty, and the two are not the
// same question.
//
// `root_ordinal` is deliberately **not** in this list even though its column is
// nullable. An inferred season has no path and still belongs to a root — the
// episode that implied it came from one — so a root ordinal dropped with the
// path would come back as 0 and make every scan report the season updated.
func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// nullableInt writes an absent number as NULL, which is optionalInt's inverse.
func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return int64(*value)
}

// boolAsInteger is the one place a Go bool becomes a column. SQLite has no
// boolean type and the table is STRICT, so the conversion is explicit rather
// than left to the driver.
func boolAsInteger(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

// RemoveItems deletes the items named by ids and, through the cascade the
// schema declares, the files beneath them.
//
// It takes identifiers rather than a library, because 003 plan §6.4 decides
// every removal in this project in one pure function and this is where that
// decision is applied. A method that removed *by library* would be the
// over-broad DELETE 002 T4 already had to guard against once — and here it
// costs a whole library rather than a token, because the guards spec §3.8 puts
// in front of a mass delete would be on the other side of the call.
//
// It refuses when it did not remove exactly what it was given. That is 001's
// rows-affected rule at the place it bites hardest: the identifiers come from
// a set this store returned, so one of them missing means the caller is holding
// a reading of some other library — and a DELETE that quietly matched fewer
// rows leaves the next scan computing the same removal again, for ever, with
// nothing reporting it.
func (s *Store) RemoveItems(ctx context.Context, ids []string) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: removing %d items: %w", s.path, len(ids), err)
	}
	if len(ids) == 0 {
		return nil
	}

	wanted := make([]string, 0, len(ids))
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if _, repeated := seen[id]; repeated {
			continue
		}
		seen[id] = struct{}{}
		wanted = append(wanted, id)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	defer transaction.Rollback()

	var removed int64
	for _, chunk := range chunks(wanted, maximumIdentifiersPerDelete) {
		result, err := transaction.ExecContext(ctx,
			`DELETE FROM items WHERE id IN (`+placeholders(len(chunk))+`)`, asArguments(chunk)...)
		if err != nil {
			return failed(err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return failed(err)
		}
		removed += affected
	}
	if removed != int64(len(wanted)) {
		return failed(fmt.Errorf("the DELETE removed %d rows, want %d", removed, len(wanted)))
	}
	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// maximumIdentifiersPerDelete keeps one statement's placeholder count well
// inside SQLite's limit on bound parameters, which defaults to 32,766 and is a
// compile-time setting of the engine rather than something this package can
// read. A removal of a whole library is one DELETE per five hundred items, and
// they are in one transaction, so the chunking costs round trips and never
// atomicity.
const maximumIdentifiersPerDelete = 500

// chunks yields successive slices of at most size elements.
func chunks(values []string, size int) [][]string {
	var out [][]string
	for start := 0; start < len(values); start += size {
		end := min(start+size, len(values))
		out = append(out, values[start:end])
	}
	return out
}

// placeholders is `?, ?, ?` for an IN list of n values. The identifiers are
// bound rather than interpolated; nothing here builds SQL out of data.
func placeholders(n int) string {
	return strings.TrimSuffix(strings.Repeat("?, ", n), ", ")
}

// asArguments widens a slice of identifiers into what ExecContext takes.
func asArguments(values []string) []any {
	out := make([]any, len(values))
	for i, value := range values {
		out[i] = value
	}
	return out
}

// ClaimScan takes the scanning claim on a library at at, breaking one older
// than staleAfter, and reports whether it won and which claimant it displaced
// or lost to.
//
// # The boolean, and the string beside it
//
// A library already being scanned is reported as false rather than as an error,
// because two scanners over one store is a state this feature creates on
// purpose — an operator may run `atrium library scan` against a data directory
// a server is serving from (003 plan §6.7) — and *"somebody else is scanning"*
// is an outcome the caller reports, not a fault.
//
// The claimant is returned beside it because 003 plan §7 requires both a
// refusal that says who holds the claim and a log line naming the process whose
// claim was broken, and neither is recoverable afterwards: the row now names
// the winner. It is empty when there was nothing to displace, which is the
// first scan of a library and every scan after a rebuild.
//
// # Why this is a transaction rather than one conditional upsert
//
// 003 plan §6.9 said *"one conditional statement"*, and the previous claimant
// is what that shape cannot produce: an upsert's `RETURNING` answers the row as
// it now stands, so the name the log line needs has already been overwritten by
// the time it could be read. The read and the write are therefore one
// transaction — which takes SQLite's write lock at BEGIN (`_txlock=immediate`,
// ADR-0003's writer DSN) rather than upgrading part way, and runs on the
// single-connection writing handle. The atomicity is the same and the reason it
// is spelled differently is this comment.
//
// A claim whose instant is **after** at — a clock that moved backwards — is
// treated as live rather than as infinitely stale. Breaking a claim on the
// strength of a clock adjustment is the one outcome this method must not
// produce: it is two scanners writing one library.
func (s *Store) ClaimScan(ctx context.Context, libraryID, by string, at units.Time, staleAfter units.Ticks) (bool, string, error) {
	failed := func(err error) error {
		return fmt.Errorf("%s: claiming the scan of library %s: %w", s.path, libraryID, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return false, "", failed(err)
	}
	defer transaction.Rollback()

	var (
		claimedAt sql.NullInt64
		claimedBy sql.NullString
	)
	err = transaction.QueryRowContext(ctx,
		`SELECT claimed_at, claimed_by FROM scan_state WHERE library_id = ?`, libraryID).
		Scan(&claimedAt, &claimedBy)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// A library with no row here has never been scanned, and that is the
		// state a rebuild leaves every library in — which is why this is an
		// insert and not an UPDATE. An UPDATE matching no row succeeds and
		// claims nothing.
		if _, err := transaction.ExecContext(ctx,
			`INSERT INTO scan_state (library_id, claimed_at, claimed_by) VALUES (?, ?, ?)`,
			libraryID, int64(at.Ticks()), by); err != nil {
			return false, "", failed(err)
		}
		if err := transaction.Commit(); err != nil {
			return false, "", failed(err)
		}
		return true, "", nil
	case err != nil:
		return false, "", failed(err)
	}

	previous := claimedBy.String
	if claimedAt.Valid && claimedBy.Valid {
		age := at.Ticks() - units.Ticks(claimedAt.Int64)
		if age < staleAfter {
			// Committing rather than rolling back: nothing was written, and a
			// rollback and a commit are the same here except that one of them
			// says the read finished.
			if err := transaction.Commit(); err != nil {
				return false, "", failed(err)
			}
			return false, previous, nil
		}
	} else {
		previous = ""
	}

	if _, err := transaction.ExecContext(ctx,
		`UPDATE scan_state SET claimed_at = ?, claimed_by = ? WHERE library_id = ?`,
		int64(at.Ticks()), by, libraryID); err != nil {
		return false, "", failed(err)
	}
	if err := transaction.Commit(); err != nil {
		return false, "", failed(err)
	}
	return true, previous, nil
}

// ReleaseScan drops the claim and records what the scan did.
//
// It is the last statement of the final transaction of a scan (003 plan §6.9),
// which is also the one that applies the removals — so a scan that died before
// it removed nothing and left a claim to go stale, which is the only partial
// state this feature can leave behind.
//
// The row must exist, because ClaimScan created it: releasing a claim on a
// library nothing claimed is a caller that lost track of which library it was
// scanning, and 001's rows-affected rule is what turns that from a silent
// success into a message.
func (s *Store) ReleaseScan(ctx context.Context, libraryID string, at units.Time, summary []byte, full bool) error {
	result, err := s.writer.ExecContext(ctx,
		`UPDATE scan_state
		    SET claimed_at = NULL, claimed_by = NULL,
		        last_scan_at = ?, last_scan_full = ?, summary_document = ?
		  WHERE library_id = ?`,
		int64(at.Ticks()), boolAsInteger(full), nullableDocument(summary), libraryID)
	if err != nil {
		return fmt.Errorf("%s: releasing the scan of library %s: %w", s.path, libraryID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: releasing the scan of library %s: %w", s.path, libraryID, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: releasing the scan of library %s changed %d rows, want 1",
			s.path, libraryID, affected)
	}
	return nil
}
