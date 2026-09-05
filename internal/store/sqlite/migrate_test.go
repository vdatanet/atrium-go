package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// newDatabase opens a bare writing handle with the schema-version table
// prepared and no migrations applied, for the tests that drive the runner with
// a lineage of their own.
//
// The runner is exercised with synthetic lineages rather than only with the one
// this feature ships, because what is under test is the rule — apply the
// pending steps of a half, in order, once — and a single-step lineage cannot
// tell "applied what was pending" from "applied everything".
func newDatabase(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open(driverName, writerDSN(filepath.Join(t.TempDir(), DatabaseFile)))
	if err != nil {
		t.Fatalf("opening the database: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if err := bootstrap(context.Background(), db); err != nil {
		t.Fatalf("bootstrap returned %v", err)
	}
	return db
}

// TestOpenAppliesThePreciousLineageAndSeedsTheInstallation is the first clause
// of T3's definition of done: a migration on an empty directory creates the row
// with server_name = 'atrium' (spec 3.1).
//
// Amended 2026-09-03 by 002 T1, which adds a second precious migration. The
// assertion read "want [1]" and was a literal that every later migration
// invalidates; it now states the rule it always meant — a first start applies
// the whole lineage, in order, from 1 — so that the next feature to file a
// precious migration does not have to decide whether a red test here is its own
// mistake.
func TestOpenAppliesThePreciousLineageAndSeedsTheInstallation(t *testing.T) {
	store := openForTest(t)

	lineage, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}
	whole := make([]int, 0, len(lineage))
	for _, m := range lineage {
		whole = append(whole, m.version)
	}
	if applied := store.AppliedMigrations(Precious); !slices.Equal(applied, whole) {
		t.Errorf("a first start applied %v precious migrations, want the whole lineage %v", applied, whole)
	}

	var (
		name        string
		completedAt sql.Null[int64]
	)
	if err := store.reader.QueryRow(
		`SELECT server_name, setup_completed_at FROM installation WHERE id = 1`,
	).Scan(&name, &completedAt); err != nil {
		t.Fatalf("reading the installation row: %v", err)
	}
	if name != "atrium" {
		t.Errorf("server_name is %q, want %q (spec 3.1)", name, "atrium")
	}
	if completedAt.Valid {
		t.Errorf("setup_completed_at is %d on a fresh installation, want NULL", completedAt.V)
	}
}

// TestASecondStartAppliesNothing is the second clause. It matters more than it
// looks: migrations run on every start (plan 4), so a runner that re-applied
// its lineage would reset an operator's server name on every restart, and
// 0001's INSERT would fail the start outright.
func TestASecondStartAppliesNothing(t *testing.T) {
	directory := t.TempDir()

	first, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("the first Open returned %v", err)
	}
	if err := first.SetServerName(context.Background(), "renamed by the operator"); err != nil {
		t.Fatalf("SetServerName returned %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}

	second := openIn(t, directory)
	for _, half := range halves {
		if applied := second.AppliedMigrations(half); len(applied) != 0 {
			t.Errorf("a second start applied %v to the %s half, want nothing", applied, half)
		}
	}

	installation, err := second.Installation(context.Background())
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if installation.Name != "renamed by the operator" {
		t.Errorf("the name is %q after a restart, want the operator's", installation.Name)
	}
}

// TestTheHalvesCarrySeparateSchemaVersions is the last clause, and it is the
// claim ADR-0003 rests the whole split on: a rescan rebuilds the derived half
// without touching the precious one, which is only possible if the two versions
// are two numbers.
//
// The two numbers count different things, and that is the point rather than an
// untidiness: the precious half is at the end of its lineage, and the derived
// half is at the generation this build declares its whole schema at. A single
// number could not hold both, and a single *lineage* could not hold the second
// one at all.
//
// Amended 2026-09-03 by 002 T1. The precious half was asserted as 1; it is now
// asserted as the length of the lineage this build ships, for the reason above
// TestOpenAppliesThePreciousLineageAndSeedsTheInstallation.
//
// Amended 2026-09-05 by 003 T11, which gives the derived half a schema and
// makes it stop being a lineage. The derived half was a literal 0 with the note
// "neither 001 nor 002 scans anything" — a claim about what those two features
// owned, which was the right claim to make and the wrong way to spell it. It is
// derivedGeneration now, through theDerivedHalfIsAtItsGeneration, which is the
// helper the four assertions that meant this share.
func TestTheHalvesCarrySeparateSchemaVersions(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	lineage, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}

	precious, err := store.SchemaVersion(ctx, Precious)
	if err != nil {
		t.Fatalf("SchemaVersion(precious) returned %v", err)
	}

	if precious != len(lineage) {
		t.Errorf("the precious half is at version %d, want %d — the whole lineage", precious, len(lineage))
	}
	theDerivedHalfIsAtItsGeneration(t, store, "a first start")

	// And they are separate rows rather than one number read twice.
	var rows int
	if err := store.reader.QueryRow(`SELECT count(*) FROM schema_version`).Scan(&rows); err != nil {
		t.Fatalf("counting the schema-version rows: %v", err)
	}
	if rows != len(halves) {
		t.Errorf("schema_version holds %d rows, want one per half (%d)", rows, len(halves))
	}
}

// TestTheRunnerAppliesOnlyWhatIsPending drives the runner over a lineage of its
// own, growing it between runs the way a release does.
//
// The runner still takes a half rather than assuming one, and the two-run shape
// is what proves it applies the *pending* steps rather than all of them: a
// single-step lineage cannot tell the two apart, and re-running the first step
// would fail on the table it already created — so "applied only 2" is asserted
// by the second run succeeding as much as by the versions it reports.
//
// Amended 2026-09-05 by 003 T11. This drove the runner over the **derived**
// half, because that was the half this repository shipped no migration for and
// so the one a synthetic lineage could not collide with. After T11 the derived
// half is not a lineage at all and migrate is never called with it, so the test
// was proving the runner did something nothing asks it to do. It runs against
// the precious half over a synthetic lineage, on a database no shipped
// migration has been applied to, and proves the same thing.
func TestTheRunnerAppliesOnlyWhatIsPending(t *testing.T) {
	db := newDatabase(t)
	ctx := context.Background()

	first := []migration{
		{version: 1, name: "0001_first.sql", sql: `CREATE TABLE first (id INTEGER PRIMARY KEY) STRICT;`},
	}
	applied, err := migrate(ctx, db, Precious, first)
	if err != nil {
		t.Fatalf("the first migrate returned %v", err)
	}
	if !slices.Equal(applied, []int{1}) {
		t.Errorf("the first migrate applied %v, want [1]", applied)
	}

	// The release that adds a second step.
	grown := append(slices.Clone(first), migration{
		version: 2, name: "0002_second.sql", sql: `CREATE TABLE second (id INTEGER PRIMARY KEY) STRICT;`,
	})
	applied, err = migrate(ctx, db, Precious, grown)
	if err != nil {
		t.Fatalf("the second migrate returned %v", err)
	}
	if !slices.Equal(applied, []int{2}) {
		t.Errorf("the second migrate applied %v, want [2]", applied)
	}

	if version, err := schemaVersion(ctx, db, Precious); err != nil || version != 2 {
		t.Errorf("the precious half is at %d (err %v), want 2", version, err)
	}
	// The other half is untouched by either run, which is the whole point of
	// the runner taking one. It is 0 here and not derivedGeneration, because
	// this database was bootstrapped by the runner's own test helper and never
	// went through Open — nothing has created a derived schema in it.
	if version, err := schemaVersion(ctx, db, Derived); err != nil || version != 0 {
		t.Errorf("the derived half is at %d (err %v), want 0 — nothing migrated or rebuilt it", version, err)
	}
}

// TestAFailingMigrationLeavesTheVersionUnchanged is the test that proves the
// runner's transaction does something.
//
// A step and the record of it must commit together: a step that ran without
// being recorded is applied again on the next start against a schema that has
// it, and a version recorded for a step that did not run is a database claiming
// a shape it does not have. Neither is recoverable without a person.
func TestAFailingMigrationLeavesTheVersionUnchanged(t *testing.T) {
	db := newDatabase(t)
	ctx := context.Background()

	// The first statement succeeds and the second does not, so a runner
	// without a transaction would leave the table behind.
	broken := []migration{{
		version: 1,
		name:    "0001_broken.sql",
		sql: `CREATE TABLE half_applied (id INTEGER PRIMARY KEY) STRICT;
		      INSERT INTO no_such_table (id) VALUES (1);`,
	}}

	applied, err := migrate(ctx, db, Precious, broken)
	if err == nil {
		t.Fatal("migrate returned nil for a migration that cannot run, want an error")
	}
	if len(applied) != 0 {
		t.Errorf("migrate reported %v applied, want nothing", applied)
	}
	if !strings.Contains(err.Error(), "0001_broken.sql") {
		t.Errorf("error %q does not name the migration that failed", err)
	}

	if version, err := schemaVersion(ctx, db, Precious); err != nil || version != 0 {
		t.Errorf("the precious half is at %d (err %v) after a failed migration, want 0", version, err)
	}
	var tables int
	if err := db.QueryRow(
		`SELECT count(*) FROM sqlite_schema WHERE type = 'table' AND name = 'half_applied'`).Scan(&tables); err != nil {
		t.Fatalf("looking for the table: %v", err)
	}
	if tables != 0 {
		t.Error("the failed migration left its first statement behind, want the transaction rolled back")
	}
}

// TestTheRunnerRefusesADatabaseWrittenByANewerBuild covers the downgrade. There
// is no down migration to run and no way to know what the newer build changed,
// so refusing to start is the only honest answer.
func TestTheRunnerRefusesADatabaseWrittenByANewerBuild(t *testing.T) {
	db := newDatabase(t)
	ctx := context.Background()

	if _, err := db.ExecContext(ctx,
		`UPDATE schema_version SET version = 7 WHERE half = 'precious'`); err != nil {
		t.Fatalf("setting the version: %v", err)
	}

	_, err := migrate(ctx, db, Precious, []migration{{version: 1, name: "0001_first.sql", sql: `SELECT 1;`}})
	if err == nil {
		t.Fatal("migrate accepted a database from the future, want an error")
	}
	if !strings.Contains(err.Error(), "newer Atrium") {
		t.Errorf("error %q does not say the database was written by a newer build", err)
	}
}

// TestLoadLineageRefusesAGap protects the number the runner counts with. The
// recorded version is a count, so "1 and 3 applied" and "1 and 2 applied" are
// the same number — and a 0002 written after 0003 shipped is skipped forever on
// every database that already recorded 3.
func TestLoadLineageRefusesAGap(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/precious/0001_first.sql": {Data: []byte(`SELECT 1;`)},
		"migrations/precious/0003_third.sql": {Data: []byte(`SELECT 1;`)},
	}

	if _, err := loadLineage(fsys, Precious); err == nil {
		t.Fatal("loadLineage accepted a lineage with a gap, want an error")
	} else if !strings.Contains(err.Error(), "0003_third.sql") {
		t.Errorf("error %q does not name the file that is out of place", err)
	}
}

// TestLoadLineageRefusesAnUnnumberedFile keeps the directory from quietly
// growing a file the runner ignores.
func TestLoadLineageRefusesAnUnnumberedFile(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/precious/0001_first.sql": {Data: []byte(`SELECT 1;`)},
		"migrations/precious/scratch.sql":    {Data: []byte(`SELECT 1;`)},
	}

	if _, err := loadLineage(fsys, Precious); err == nil {
		t.Fatal("loadLineage accepted an unnumbered file, want an error")
	}
}

// TestLoadLineageReadsAHalfWithNothingInIt is the branch a half with no
// directory takes: an empty lineage and not an error.
//
// Amended 2026-09-05 by 003 T11, and this one is the interesting amendment of
// the five that task made. The test read `loadLineage(migrationFiles, Derived)`
// and asserted it was empty, which was 001's way of saying *no derived table
// has been written yet*. After T11 the assertion **stays true for ever and
// stops meaning anything**: nothing loads the derived lineage, so a migration
// filed under migrations/derived would be applied by nothing and this test
// would still be green. A test that cannot fail is worse than one that fails
// for the wrong reason, because nobody is going to look at it again.
//
// The claim about the directory moved to TestNoDerivedMigrationsDirectoryExists
// and into filedUnderThePreciousLineage. What is left here is the loader's own
// behaviour, over a filesystem that is not this build's — which is the thing
// this test could always see and the only thing it can still prove.
func TestLoadLineageReadsAHalfWithNothingInIt(t *testing.T) {
	fsys := fstest.MapFS{
		"migrations/precious/0001_first.sql": {Data: []byte(`SELECT 1;`)},
	}

	lineage, err := loadLineage(fsys, Derived)
	if err != nil {
		t.Fatalf("loadLineage over a filesystem with no derived directory returned %v, "+
			"want an empty lineage", err)
	}
	if len(lineage) != 0 {
		t.Errorf("a half with no directory loaded %d migrations, want none", len(lineage))
	}
}

// TestTheShippedPreciousLineageIsWellFormed reads the files this feature
// actually ships through the same loader, so that a migration added later with
// a name the runner cannot parse fails here rather than at a customer's start.
func TestTheShippedPreciousLineageIsWellFormed(t *testing.T) {
	lineage, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loadLineage(precious) returned %v", err)
	}
	if len(lineage) == 0 {
		t.Fatal("the precious lineage is empty, want at least 0001_installation.sql")
	}
	for i, m := range lineage {
		if m.version != i+1 {
			t.Errorf("%s is numbered %d, want %d", m.name, m.version, i+1)
		}
		if strings.TrimSpace(m.sql) == "" {
			t.Errorf("%s is empty", m.name)
		}
	}
}

// filedUnderThePreciousLineage is the assertion 002's T1 wrote for its own
// migration and 003's T10 needed for a second one: a feature's migration is in
// the **precious** lineage, and applying it advances the precious version by
// exactly one while the derived version does not move.
//
// It is the one mistake here that no test of the SQL itself would catch. A file
// that created a feature's tables out of the derived directory would create
// exactly the same tables, so every other test over them would pass. What it
// would have changed is the policy: the derived half is the one a rescan is
// entitled to drop and rebuild (ADR-0003), so 002's accounts would be deleted
// by a library scan and 003's libraries would take every identifier in the
// installation with them.
//
// It is a helper rather than a third copy of the same body, and the reason is
// the mistake it fixes. 002's T1 rewrote 001's literals with the note that
// "the assertion read want [1] and was a literal that every later migration
// invalidates" — and then wrote `len(precious) != 2`, `[]int{2}` and `want 2`,
// which is the same literal one number along. 003's `0003_libraries.sql` turned
// all three red on the day it landed, in a task whose subject is a library and
// not a runner. The rule underneath them never mentioned a total: it is *this
// file, in this half, advancing this half by one*. So the helper is given the
// file's name, finds it, and asserts the rule about that file — which stays
// true however many migrations are filed after it.
func filedUnderThePreciousLineage(t *testing.T, filename string) {
	t.Helper()
	ctx := context.Background()
	db := newDatabase(t)

	precious, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}
	index := slices.IndexFunc(precious, func(m migration) bool { return m.name == filename })
	if index < 0 {
		var shipped []string
		for _, m := range precious {
			shipped = append(shipped, m.name)
		}
		t.Fatalf("%s is not in the precious lineage, which holds %v. Filed under the derived "+
			"directory it would create exactly the same tables, and a rescan would be entitled "+
			"to drop them", filename, shipped)
	}

	// And the other half of "filed in the right place": there is nowhere else
	// to file it.
	//
	// Amended 2026-09-05 by 003 T11. This read the derived lineage and required
	// it to be empty, which was the assertion that would have noticed a
	// migration filed there — right up until T11 made the derived half stop
	// being a lineage, at which point an empty answer became the only answer
	// and the check became a sentence that cannot fail. The directory not
	// existing is what carries the claim now: a file under migrations/derived
	// is applied by nothing at all, which is worse than a missing migration
	// rather than better.
	if entries, err := migrationFiles.ReadDir("migrations/" + string(Derived)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("migrations/%s exists and holds %d entries (err %v): the derived half is not a "+
			"lineage and nothing applies a file filed there", Derived, len(entries), err)
	}

	// The state the release before this migration left behind.
	if _, err := migrate(ctx, db, Precious, precious[:index]); err != nil {
		t.Fatalf("applying the lineage up to %s returned %v", filename, err)
	}
	before := versions(t, db)
	if before != (halfVersions{precious: index, derived: 0}) {
		t.Fatalf("before %s the versions are %+v, want precious %d and derived 0",
			filename, before, index)
	}

	// The lineage is truncated after this migration rather than applied whole,
	// so that "advances by exactly one" stays an assertion about this file and
	// not about how many features have filed a migration since.
	applied, err := migrate(ctx, db, Precious, precious[:index+1])
	if err != nil {
		t.Fatalf("applying %s returned %v", filename, err)
	}
	if !slices.Equal(applied, []int{index + 1}) {
		t.Errorf("applying the lineage through %s applied %v, want exactly [%d]",
			filename, applied, index+1)
	}

	after := versions(t, db)
	if after.precious != before.precious+1 {
		t.Errorf("the precious half moved from %d to %d, want an advance of exactly one",
			before.precious, after.precious)
	}
	if after.derived != before.derived {
		t.Errorf("the derived half moved from %d to %d: %s is filed under the wrong lineage, "+
			"and a rescan would drop what it creates", before.derived, after.derived, filename)
	}
}
