package sqlite

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Half names one of the two parts ADR-0003 splits this database into.
//
// The split is the record's central claim rather than a filing convention. The
// derived half is a cache of what a scan can compute again from the files on
// disk, so a schema change there is answered by dropping it and rescanning and
// no migration is ever written. The precious half is the only copy of what a
// user did, so every change to it is a forward-only migration that never
// destroys anything.
//
// A Half is therefore a migration lineage and a policy at once, and the two
// carry separate schema versions — which is what makes rebuilding the derived
// half possible without touching the precious one.
type Half string

const (
	// Precious holds users, credentials, tokens, favourites, played state,
	// resume positions, playlists and library configuration. Migrated.
	Precious Half = "precious"

	// Derived holds the scanned library: items, media streams, images and the
	// by-name aggregates. Dropped and rescanned.
	//
	// 001 owns nothing here — it scans nothing (plan 4) — so this lineage is
	// empty and its version is 0. It exists now because the runner takes a
	// half: a runner that hardcoded the precious lineage would have to be
	// rewritten by whoever adds the first derived table, and the version
	// recorded separately is what a rescan will compare against.
	Derived Half = "derived"
)

// halves is every half, in the order a start applies them. Precious first: a
// derived table may reference a precious identifier by value, never the other
// way round (architecture 6), so this order is the one that stays valid when
// the derived lineage stops being empty.
var halves = []Half{Precious, Derived}

// migrationDirectory is where a half's lineage lives inside migrationFiles.
const migrationDirectory = "migrations"

//go:embed migrations
var migrationFiles embed.FS

// migration is one numbered, forward-only step of one half's lineage.
type migration struct {
	version int
	name    string
	sql     string
}

// bootstrapSQL creates the table the runner keeps its own state in, and seeds a
// row for each half at version 0.
//
// It is not itself a numbered migration, because a numbered migration is
// recorded here and this is the table that does the recording. IF NOT EXISTS
// rather than a version check for the same reason.
const bootstrapSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
    half    TEXT    NOT NULL PRIMARY KEY CHECK (half IN ('precious', 'derived')),
    version INTEGER NOT NULL
) STRICT;

INSERT OR IGNORE INTO schema_version (half, version) VALUES ('precious', 0), ('derived', 0);
`

// bootstrap prepares the schema-version table. It is idempotent.
func bootstrap(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, bootstrapSQL); err != nil {
		return fmt.Errorf("preparing the schema-version table: %w", err)
	}
	return nil
}

// schemaVersion reports how many migrations of half have been applied. A half
// that has never been migrated is at 0, which is a state and not an absence.
func schemaVersion(ctx context.Context, db *sql.DB, half Half) (int, error) {
	var version int
	err := db.QueryRowContext(ctx, `SELECT version FROM schema_version WHERE half = ?`, string(half)).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("the %s half has no schema-version row: the database was not prepared by this runner", half)
	}
	if err != nil {
		return 0, fmt.Errorf("reading the %s schema version: %w", half, err)
	}
	return version, nil
}

// migrate applies every step of lineage that half has not had yet, in order,
// and returns the versions it applied. A second run over the same database
// applies nothing and returns nothing, which is what makes running this on
// every start the plan's "applied at start" rather than a risk.
//
// lineage is contiguous from 1 — loadLineage refuses anything else — so the
// pending steps are exactly the tail past the recorded version.
func migrate(ctx context.Context, db *sql.DB, half Half, lineage []migration) ([]int, error) {
	current, err := schemaVersion(ctx, db, half)
	if err != nil {
		return nil, err
	}
	if current > len(lineage) {
		// The database was written by a newer build. Refusing is the only
		// answer this feature can give: a rollback would need the down
		// migration this project does not write, and carrying on would run
		// queries against columns that may have moved.
		//
		// ADR-0003 says a *derived*-version mismatch is a rescan rather than
		// an error. That branch needs a scanner to rescan with, which is
		// 003's, and until then the derived half is empty and cannot reach
		// this line from anything but a downgrade. The refusal is owed a
		// replacement there, not here.
		return nil, fmt.Errorf("the %s half is at schema version %d and this build knows %d: it was written by a newer Atrium",
			half, current, len(lineage))
	}

	var applied []int
	for _, m := range lineage[current:] {
		if err := applyMigration(ctx, db, half, m); err != nil {
			return applied, err
		}
		applied = append(applied, m.version)
	}
	return applied, nil
}

// applyMigration runs one step and records it in the same transaction.
//
// The two must commit together or not at all. A step that ran and was not
// recorded is applied again on the next start — against a schema that already
// has it — and a version recorded for a step that did not run is a database
// claiming a shape it does not have. Both are unrecoverable without a person;
// one transaction is what makes them impossible.
func applyMigration(ctx context.Context, db *sql.DB, half Half, m migration) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s migration %s: %w", half, m.name, err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, m.sql); err != nil {
		return fmt.Errorf("%s migration %s: %w", half, m.name, err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE schema_version SET version = ? WHERE half = ?`, m.version, string(half)); err != nil {
		return fmt.Errorf("%s migration %s: recording it: %w", half, m.name, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s migration %s: %w", half, m.name, err)
	}
	return nil
}

// loadLineage reads one half's migrations out of fsys and returns them in the
// order they are applied.
//
// A half with no directory has an empty lineage rather than an error: 001 owns
// no derived table, and a runner that could not represent that would need the
// first derived feature to also change the runner.
func loadLineage(fsys fs.FS, half Half) ([]migration, error) {
	directory := path.Join(migrationDirectory, string(half))

	entries, err := fs.ReadDir(fsys, directory)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading the %s lineage: %w", half, err)
	}

	var lineage []migration
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return nil, fmt.Errorf("the %s lineage: %w", half, err)
		}
		statements, err := fs.ReadFile(fsys, path.Join(directory, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", entry.Name(), err)
		}
		lineage = append(lineage, migration{version: version, name: entry.Name(), sql: string(statements)})
	}

	sort.Slice(lineage, func(i, j int) bool { return lineage[i].version < lineage[j].version })

	// Contiguous from 1, which is not tidiness. The recorded version is a
	// count, so a lineage with a gap makes "applied 1 and 3" and "applied 1
	// and 2" the same number; and a 0002 added after 0003 shipped would be
	// skipped on every database that already recorded 3. Refusing at load
	// turns both into a failure to start rather than a silent wrong schema.
	for i, m := range lineage {
		if m.version != i+1 {
			return nil, fmt.Errorf("the %s lineage is not numbered 1..%d without gaps: %s is out of place",
				half, len(lineage), m.name)
		}
	}
	return lineage, nil
}

// migrationVersion reads the leading number of a file named NNNN_description.sql.
func migrationVersion(name string) (int, error) {
	digits, _, found := strings.Cut(strings.TrimSuffix(name, ".sql"), "_")
	if !found || digits == "" {
		return 0, fmt.Errorf("%s is not named NNNN_description.sql", name)
	}
	version, err := strconv.Atoi(digits)
	if err != nil || version < 1 {
		return 0, fmt.Errorf("%s does not begin with a migration number", name)
	}
	return version, nil
}
