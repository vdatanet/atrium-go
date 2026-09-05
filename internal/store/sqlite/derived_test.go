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

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// aTestInstant is the clock every test in this file reads. A fixed instant
// rather than time.Now, for architecture §2's reason: a store that read the
// wall clock would hold a value no test could hold still.
var aTestInstant = units.TimeFromTicks(638_500_000_000_000_000)

type fixedClock struct{ at units.Time }

func (c fixedClock) Now() units.Time { return c.at }

// openRawDatabase opens a writing handle on an existing database file without
// going through Open, for the tests that have to leave a database in a state
// Open would have corrected on the way past.
func openRawDatabase(t *testing.T, directory string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, writerDSN(filepath.Join(directory, DatabaseFile)))
	if err != nil {
		t.Fatalf("opening the database directly: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	return db
}

// recordDerivedVersion writes a derived generation into a closed database, which
// is how both directions of the rebuild are reached: a database written by a
// build that declared something else.
func recordDerivedVersion(t *testing.T, directory string, version int) {
	t.Helper()
	db := openRawDatabase(t, directory)
	if _, err := db.Exec(`UPDATE schema_version SET version = ? WHERE half = 'derived'`, version); err != nil {
		t.Fatalf("recording derived version %d: %v", version, err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("closing the direct handle: %v", err)
	}
}

// markScanState writes one scan_state row, which is the marker every test here
// uses to tell a rebuild from a start that did nothing.
//
// scan_state rather than items, because it is one column and because its
// absence afterwards is itself one of T11's clauses: a rebuilt library is in
// the same state as one that has never been scanned.
func markScanState(t *testing.T, db *sql.DB, libraryID string) {
	t.Helper()
	if _, err := db.Exec(
		`INSERT INTO scan_state (library_id, claimed_by, claimed_at) VALUES (?, 'a scanner', 1)`,
		libraryID); err != nil {
		t.Fatalf("writing a scan_state row: %v", err)
	}
}

func countScanState(t *testing.T, db *sql.DB) int {
	t.Helper()
	var rows int
	if err := db.QueryRow(`SELECT count(*) FROM scan_state`).Scan(&rows); err != nil {
		t.Fatalf("counting scan_state rows: %v", err)
	}
	return rows
}

// TestTheDerivedSchemaAndItsGenerationMoveTogether is what makes the bump
// deliberate rather than forgotten, and it is 002 T1's "the constraint is
// redundant today on purpose: it is the one thing in the schema that notices"
// applied to a constant instead of to a UNIQUE.
//
// A generation nobody bumps is a schema change that ships as a silent
// corruption: every installation keeps the tables of the last generation while
// every query written above them is compiled against this one, and nothing in
// this package or any other would fail. There is no way to detect that later,
// so the detection is here and it is a byte comparison of the file.
//
// It fails on a comment and on a whitespace change too, which is deliberate.
// 003 plan §6.8 rejected a *fingerprint* as the version precisely because a
// rebuild triggered by a comment is a rescan nobody can justify from the diff —
// but a failing test that says "decide whether this needs a rescan" costs one
// line and asks the question at the only moment somebody can answer it.
func TestTheDerivedSchemaAndItsGenerationMoveTogether(t *testing.T) {
	got := digestOf(derivedSchema)
	if got != derivedSchemaDigest {
		t.Errorf("derived/library.sql hashes to %s and derivedSchemaDigest is %s.\n"+
			"The schema changed. Every installation must drop and rebuild its derived half to get "+
			"the new shape, so bump derivedGeneration (it is %d) and record the new digest — or, if "+
			"the edit really cannot change what a query sees, record the digest alone and say why.",
			got, derivedSchemaDigest, derivedGeneration)
	}
}

// theDerivedHalfIsAtItsGeneration is what the assertions this task broke had in
// common, and it is one helper rather than a fifth literal.
//
// 001 wrote `derived != 0` with the note *"neither 001 nor 002 scans
// anything"*; 002 T1 and 003 T10 each copied it into a test of their own, and
// this task adds a fourth caller. All of them mean one rule — **a start leaves
// the derived half where this build's schema puts it, and no migration moves
// it** — and all of them spelled it as the number that rule happened to produce
// while nothing had a derived schema. One task moving that number turned three
// tests red across three files whose subjects are a runner, a user and a
// library.
//
// That is T10's finding one feature along: **a correction that restates a
// literal is not a correction.** Its filedUnderThePreciousLineage replaced
// 002's three literals with one helper for exactly this reason, and writing
// `derived != derivedGeneration` into four files instead of this would be the
// same mistake with a longer number in it.
//
// becauseOf names what the caller just did, so the failure says which start or
// which migration the assertion is about.
func theDerivedHalfIsAtItsGeneration(t *testing.T, store *Store, becauseOf string) {
	t.Helper()

	version, err := store.SchemaVersion(context.Background(), Derived)
	if err != nil {
		t.Fatalf("SchemaVersion(derived) returned %v", err)
	}
	if version != derivedGeneration {
		t.Errorf("the derived half is at version %d after %s, want derivedGeneration %d. "+
			"The derived half is created and moved by derived/library.sql alone (003 plan §6.8); "+
			"if a migration moved it, it is filed in the wrong lineage",
			version, becauseOf, derivedGeneration)
	}
}

// TestAFirstStartCreatesTheDerivedSchemaAtItsGeneration is the branch a first
// start takes, and it is the same branch as an upgrade: an empty database
// records generation 0, this build declares 1, and 0 is a difference.
//
// That the three tables are there is asserted through the schema this build
// declares rather than against a list typed here, for the reason
// TestARebuildDropsEveryObjectTheSchemaDeclares gives at length.
func TestAFirstStartCreatesTheDerivedSchemaAtItsGeneration(t *testing.T) {
	store := openForTest(t)

	if !store.DerivedRebuilt() {
		t.Error("a first start reports the derived half was not rebuilt, want it built from generation 0")
	}

	theDerivedHalfIsAtItsGeneration(t, store, "a first start")

	objects, err := derivedObjects(derivedSchema)
	if err != nil {
		t.Fatalf("derivedObjects returned %v", err)
	}
	for _, object := range objects {
		var name string
		if err := store.reader.QueryRow(
			`SELECT name FROM sqlite_schema WHERE lower(type) = lower(?) AND name = ?`,
			object.kind, object.name,
		).Scan(&name); err != nil {
			t.Errorf("the derived schema declares %s %s and a first start did not create it: %v",
				object.kind, object.name, err)
		}
	}
}

// TestAGenerationAheadAndAGenerationBehindAreBothRebuilt is T11's first clause,
// and both halves of it are named because a one-directional branch is 003 plan
// §6.8's *first rejected shape*.
//
// A runner that rebuilt only on "newer than known" — which is the shape the
// precious half has to take, because a precious downgrade has no answer at all
// — would answer a downgrade and leave an upgrade to be discovered as queries
// against columns that have moved. Nothing else in this suite could see the
// difference: an upgrade is the common case and it is also what a first start
// looks like, so a build with the wrong comparison passes every other test
// here.
//
// The third case is the control, and without it the two above are met by a
// build that rebuilds unconditionally: at the recorded generation, nothing
// happens and the row written before the restart is still there.
func TestAGenerationAheadAndAGenerationBehindAreBothRebuilt(t *testing.T) {
	for _, testCase := range []struct {
		name     string
		recorded int
		rebuilt  bool
	}{
		{name: "a database from a newer build", recorded: derivedGeneration + 1, rebuilt: true},
		{name: "a database from an older build", recorded: derivedGeneration - 1, rebuilt: true},
		{name: "a database at this generation", recorded: derivedGeneration, rebuilt: false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := context.Background()
			directory := t.TempDir()

			first := openIn(t, directory)
			if err := first.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Films", "films")); err != nil {
				t.Fatalf("CreateLibrary returned %v", err)
			}
			markScanState(t, first.writer, testFilmLibraryID)
			if rows := countScanState(t, first.writer); rows != 1 {
				t.Fatalf("the marker row was not written: scan_state holds %d rows", rows)
			}
			if err := first.Close(); err != nil {
				t.Fatalf("Close returned %v", err)
			}

			recordDerivedVersion(t, directory, testCase.recorded)

			second := openIn(t, directory)
			if rebuilt := second.DerivedRebuilt(); rebuilt != testCase.rebuilt {
				t.Errorf("a start over a database recorded at %d reports rebuilt=%v, want %v "+
					"(this build declares %d)", testCase.recorded, rebuilt, testCase.rebuilt, derivedGeneration)
			}

			version, err := second.SchemaVersion(ctx, Derived)
			if err != nil {
				t.Fatalf("SchemaVersion(derived) returned %v", err)
			}
			if version != derivedGeneration {
				t.Errorf("the derived half is at %d after the start, want derivedGeneration %d",
					version, derivedGeneration)
			}

			wantRows := 1
			if testCase.rebuilt {
				wantRows = 0
			}
			if rows := countScanState(t, second.writer); rows != wantRows {
				t.Errorf("scan_state holds %d rows after the start, want %d", rows, wantRows)
			}
		})
	}
}

// TestARebuildLeavesEveryLibraryWithNoScanState is the state a rebuild leaves a
// configured library in, and it is a claim about the *pair* of halves rather
// than about either.
//
// The library is still declared — it is precious — and it has no scan_state
// row, which is exactly the state a library that has never been scanned is in.
// That is why 003 plan §4.3 puts scan_state in the derived half against the
// tempting other answer: "when this library was last scanned" reads like
// operator-facing history, and history is precious, but a last-scan instant
// kept across a rebuild would sit over an empty items table and every reader of
// the pair would have to decide which half to believe.
func TestARebuildLeavesEveryLibraryWithNoScanState(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	for _, declared := range []ports.Library{
		aLibrary(testFilmLibraryID, "Films", "films"),
		aLibrary(testShowLibraryID, "Shows", "shows"),
	} {
		if err := store.CreateLibrary(ctx, declared); err != nil {
			t.Fatalf("CreateLibrary(%s) returned %v", declared.NameFolded, err)
		}
		markScanState(t, store.writer, declared.ID)
	}
	if rows := countScanState(t, store.writer); rows != 2 {
		t.Fatalf("the two marker rows were not written: scan_state holds %d rows", rows)
	}

	if err := store.RebuildDerived(ctx); err != nil {
		t.Fatalf("RebuildDerived returned %v", err)
	}

	if rows := countScanState(t, store.writer); rows != 0 {
		t.Errorf("scan_state holds %d rows after a rebuild, want none: a rebuilt library is in the "+
			"same state as one that has never been scanned", rows)
	}

	libraries, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	if len(libraries) != 2 {
		t.Errorf("%d libraries survived the rebuild, want 2: they are precious", len(libraries))
	}
}

// TestARebuildLeavesThePreciousHalfUntouched is ADR-0003's central claim, and
// it is the one thing a wrong drop destroys for good.
//
// An account is created before the rebuild and authenticates after it, through
// the real login path and a real Argon2id record rather than through a row
// comparison. A test that read the credential column back would pass over a
// rebuild that had emptied the sessions table, or reset the invalid-attempt
// count, or dropped the installation row the login path does not touch: what an
// operator would notice is that nobody can log in, and that is what this
// asserts. The library beside it is 003's own precious rows, compared field by
// field with the helper T10 wrote so that there is one such comparison.
//
// 002 T10's TestALibraryOutlivesARestart is the weak half of this claim and is
// deliberately weak — it proves a library survives a *restart*, which is a
// start that drops nothing.
func TestARebuildLeavesThePreciousHalfUntouched(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	const password = "a password the rebuild must not forget"

	policyDocument, err := users.DefaultPolicy().Document()
	if err != nil {
		t.Fatalf("building the policy document returned %v", err)
	}
	configurationDocument, err := users.DefaultConfiguration().Document()
	if err != nil {
		t.Fatalf("building the configuration document returned %v", err)
	}
	account := ports.User{
		ID:                    users.DeriveID("Ada"),
		Username:              "Ada",
		UsernameFolded:        users.Fold("Ada"),
		PolicyDocument:        policyDocument,
		ConfigurationDocument: configurationDocument,
	}
	if err := store.CreateUser(ctx, account); err != nil {
		t.Fatalf("CreateUser returned %v", err)
	}
	record, err := users.Derive(users.NewPlaintext(password))
	if err != nil {
		t.Fatalf("Derive returned %v", err)
	}
	if err := store.ReplaceCredential(ctx, account.ID, record, aTestInstant); err != nil {
		t.Fatalf("ReplaceCredential returned %v", err)
	}

	declared := aLibrary(testMusicLibraryID, "Albums", "albums")
	declared.CollectionType = string(library.Music)
	declared.CaseSensitive = true
	declared.Roots = []string{"/mnt/music", "/mnt/more-music"}
	if err := store.CreateLibrary(ctx, declared); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}

	// The control. Without it the assertion below is met by a build whose
	// rebuild does nothing at all, which is the same green a correct build
	// gives.
	markScanState(t, store.writer, declared.ID)
	if rows := countScanState(t, store.writer); rows != 1 {
		t.Fatalf("the marker row was not written: scan_state holds %d rows", rows)
	}

	if err := store.RebuildDerived(ctx); err != nil {
		t.Fatalf("RebuildDerived returned %v", err)
	}
	if rows := countScanState(t, store.writer); rows != 0 {
		t.Fatalf("the derived half was not rebuilt: scan_state still holds %d rows, so what "+
			"follows would pass over a rebuild that did nothing", rows)
	}

	login := users.NewLogin(store, fixedClock{at: aTestInstant})
	authenticated, err := login.Authenticate(ctx, "Ada", users.NewPlaintext(password))
	if err != nil {
		t.Fatalf("the account created before the rebuild cannot authenticate after it: %v.\n"+
			"This is ADR-0003's central claim and the one thing a wrong drop destroys for good", err)
	}
	if authenticated.ID != account.ID {
		t.Errorf("the login answered account %q, want %q", authenticated.ID, account.ID)
	}

	survived, found, err := store.LibraryByFoldedName(ctx, "albums")
	if err != nil {
		t.Fatalf("LibraryByFoldedName returned %v", err)
	}
	if !found {
		t.Fatal("the library declared before the rebuild is gone after it")
	}
	assertSameLibrary(t, "after a rebuild", survived, declared)
}

// derivedObjectsInTheDatabase reads what the derived schema put in a database,
// **without asking derivedObjects**.
//
// It is the difference between what a full start creates and what the precious
// lineage alone creates, so it is computed by the same route a reader with a
// shell would take. Asking derivedObjects instead would make the test agree
// with the code under test by construction: a schema whose parse missed a table
// would produce a drop that missed it and a check that never looked for it, and
// the pair would be green.
func derivedObjectsInTheDatabase(t *testing.T, db *sql.DB) map[string]string {
	t.Helper()

	preciousOnly := newDatabase(t)
	lineage, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}
	if _, err := migrate(context.Background(), preciousOnly, Precious, lineage); err != nil {
		t.Fatalf("applying the precious lineage returned %v", err)
	}

	everything := func(from *sql.DB) map[string]string {
		t.Helper()
		rows, err := from.Query(`SELECT type, name FROM sqlite_schema`)
		if err != nil {
			t.Fatalf("reading sqlite_schema: %v", err)
		}
		defer rows.Close()
		found := map[string]string{}
		for rows.Next() {
			var kind, name string
			if err := rows.Scan(&kind, &name); err != nil {
				t.Fatalf("reading sqlite_schema: %v", err)
			}
			found[name] = kind
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("reading sqlite_schema: %v", err)
		}
		return found
	}

	precious := everything(preciousOnly)
	derived := map[string]string{}
	for name, kind := range everything(db) {
		if _, isPrecious := precious[name]; !isPrecious {
			derived[name] = kind
		}
	}
	if len(derived) == 0 {
		t.Fatal("the database holds no object the precious lineage did not create, so this test " +
			"is comparing two empty sets")
	}
	return derived
}

// TestARebuildDropsEveryObjectTheSchemaDeclares is the clause that guards the
// mistake nothing else in the project could see.
//
// A table added to derived/library.sql and forgotten in a hand-written drop
// list survives a rebuild **carrying its old columns**: the database then claims
// this generation while holding the last one's shape, every query above it is
// compiled against columns that are not there, and no other test here looks. So
// the drop list is read out of the schema by the code, and this test refuses to
// read it the same way — it computes the derived objects as everything a full
// start creates minus everything the precious lineage creates, marks every one
// of them, and asserts the marks are gone.
//
// The mark is a column and not a row, because a row proves the weaker thing: a
// rebuild that emptied the tables instead of dropping them would delete every
// row and keep every stale column, which is precisely the state that cannot be
// detected later.
func TestARebuildDropsEveryObjectTheSchemaDeclares(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	const markerColumn = "a_column_from_the_previous_generation"

	before := derivedObjectsInTheDatabase(t, store.writer)
	marked := 0
	for name, kind := range before {
		if kind != "table" {
			continue
		}
		if _, err := store.writer.ExecContext(ctx,
			`ALTER TABLE `+name+` ADD COLUMN `+markerColumn+` TEXT`); err != nil {
			t.Fatalf("marking %s: %v", name, err)
		}
		marked++
	}
	if marked == 0 {
		t.Fatal("no derived table was marked, so the assertion below is over nothing")
	}

	if err := store.RebuildDerived(ctx); err != nil {
		t.Fatalf("RebuildDerived returned %v", err)
	}

	after := derivedObjectsInTheDatabase(t, store.writer)
	for name, kind := range before {
		if afterKind, present := after[name]; !present {
			t.Errorf("%s %s is gone after a rebuild: it was dropped and not recreated", kind, name)
		} else if afterKind != kind {
			t.Errorf("%s came back as a %s, want a %s", name, afterKind, kind)
		}
	}
	for name, kind := range after {
		if _, present := before[name]; !present {
			t.Errorf("a rebuild left a %s named %s that the schema before it did not have", kind, name)
		}
	}

	for name, kind := range after {
		if kind != "table" {
			continue
		}
		if columns := columnsOf(t, store.writer, name); slices.Contains(columns, markerColumn) {
			t.Errorf("%s survived the rebuild carrying %s: it is declared by the schema and is not "+
				"dropped by it, so this database claims generation %d while holding the shape "+
				"before it", name, markerColumn, derivedGeneration)
		}
	}
}

// columnsOf reads a table's columns, which is how a stale shape is seen at all.
func columnsOf(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?)`, table)
	if err != nil {
		t.Fatalf("reading the columns of %s: %v", table, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("reading the columns of %s: %v", table, err)
		}
		columns = append(columns, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading the columns of %s: %v", table, err)
	}
	return columns
}

// TestForeignKeysStayOnAcrossARebuildAndPointNowherePrecious is architecture
// §6's rule at the one place it actually bites, and the assertion had to be
// built rather than assumed — the measurement is below.
//
// The rule: no reference points between the two halves, and SQLite's
// foreign_keys(1) is on (ADR-0003's writer DSN). The failure it guards is a
// `REFERENCES libraries(id)` written into the derived schema, which ties the
// half that is rebuilt to the half that is kept.
//
// **Measured 2026-09-05, and the obvious assertion does not hold.** A derived
// table declaring `REFERENCES libraries(id)`, holding rows, in a database whose
// `libraries` holds rows, with `PRAGMA foreign_keys` reading 1, **drops
// without complaint**: DROP TABLE performs an implicit DELETE of the child
// rows, and deleting a child violates nothing. So "the rebuild refuses" is not
// a thing that happens, and a test that only rebuilt and expected an error
// would have been a green proving the rule was unbreakable when it is not.
//
// What does bite is reading the constraint. Every foreign key the derived
// schema declares is listed by the engine, and every target of one must itself
// be a derived object — which is the rule stated as an assertion instead of as
// a comment. The two clauses beside it are what make it worth reading: foreign
// keys are still *on* after the rebuild, and still *enforcing* over the freshly
// created tables, so a rebuild that reached for `PRAGMA foreign_keys = OFF`
// around its drop — the tempting shape, and the one that would make the check
// above meaningless — fails here.
func TestForeignKeysStayOnAcrossARebuildAndPointNowherePrecious(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if err := store.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Films", "films")); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}
	if on := foreignKeysOn(t, store.writer); !on {
		t.Fatal("foreign keys are off before the rebuild, so nothing below is about them")
	}

	if err := store.RebuildDerived(ctx); err != nil {
		t.Fatalf("RebuildDerived returned %v with libraries holding rows and foreign keys on", err)
	}

	if on := foreignKeysOn(t, store.writer); !on {
		t.Error("foreign keys are off after the rebuild: something turned them off to get the drop " +
			"through, and the rule below is then unenforced at run time")
	}

	// Still enforcing, and not merely still switched on. item_files references
	// items within the derived half, so a file naming an item that does not
	// exist is the cheapest thing the engine can refuse.
	_, err := store.writer.ExecContext(ctx,
		`INSERT INTO item_files (item_id, ordinal, path, size, modified_at)
		 VALUES ('no such item', 0, 'a/path', 1, 1)`)
	if err == nil {
		t.Error("a file row naming an item that does not exist was accepted after the rebuild: " +
			"the constraint is declared and not enforced")
	}

	// And the rule itself, read off the schema the rebuild just created.
	derived := derivedObjectsInTheDatabase(t, store.writer)
	for name, kind := range derived {
		if kind != "table" {
			continue
		}
		for _, target := range foreignKeyTargetsOf(t, store.writer, name) {
			if _, isDerived := derived[target]; !isDerived {
				t.Errorf("the derived table %s declares a foreign key into %s, which is not in the "+
					"derived half. Architecture §6 forbids a reference across the halves, and this "+
					"is the direction that ties the half a rescan rebuilds to the half that is the "+
					"only copy. Carry the identifier as a string, as items.library_id does",
					name, target)
			}
		}
	}
}

func foreignKeysOn(t *testing.T, db *sql.DB) bool {
	t.Helper()
	var on int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&on); err != nil {
		t.Fatalf("reading PRAGMA foreign_keys: %v", err)
	}
	return on == 1
}

func foreignKeyTargetsOf(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT "table" FROM pragma_foreign_key_list(?)`, table)
	if err != nil {
		t.Fatalf("reading the foreign keys of %s: %v", table, err)
	}
	defer rows.Close()
	var targets []string
	for rows.Next() {
		var target string
		if err := rows.Scan(&target); err != nil {
			t.Fatalf("reading the foreign keys of %s: %v", table, err)
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading the foreign keys of %s: %v", table, err)
	}
	return targets
}

// TestDerivedObjectsReadsEveryKindADropWouldHaveToName covers the parse the
// drop list is built from, over a schema that is not the one this build ships.
//
// An index is in it because derived/library.sql declares none today and the
// first one added must be dropped rather than left behind a table that is
// dropped out from under it. A view and a trigger are there for the same
// reason one step further out: the parse is what decides, so it is what is
// tested, and the schema this build ships is a single case of it.
func TestDerivedObjectsReadsEveryKindADropWouldHaveToName(t *testing.T) {
	objects, err := derivedObjects(`
-- A comment mentioning CREATE TABLE not_an_object, which is not a declaration.
CREATE TABLE items (id TEXT PRIMARY KEY) STRICT;
CREATE INDEX items_by_sort_key ON items (id);
CREATE VIEW a_view AS SELECT 1;
CREATE TRIGGER a_trigger AFTER INSERT ON items BEGIN SELECT 1; END;
`)
	if err != nil {
		t.Fatalf("derivedObjects returned %v", err)
	}
	want := []derivedObject{
		{kind: "TABLE", name: "items"},
		{kind: "INDEX", name: "items_by_sort_key"},
		{kind: "VIEW", name: "a_view"},
		{kind: "TRIGGER", name: "a_trigger"},
	}
	if !slices.Equal(objects, want) {
		t.Errorf("derivedObjects read %v, want %v — in declaration order, because the drop runs in "+
			"reverse of it", objects, want)
	}
}

// TestDerivedObjectsRefusesASchemaThatDeclaresNothing is the guard that keeps
// this file's other assertions from passing for the wrong reason.
//
// A schema that parsed to nothing would make the drop a no-op and the rebuild
// an append: every table would keep every stale column and every row, and
// "the derived half was rebuilt" would be a sentence with nothing behind it.
func TestDerivedObjectsRefusesASchemaThatDeclaresNothing(t *testing.T) {
	if _, err := derivedObjects("-- nothing but a comment\n"); err == nil {
		t.Fatal("derivedObjects accepted a schema declaring no object, want an error")
	}
}

// TestNoDerivedMigrationsDirectoryExists is the replacement for 001's
// TestLoadLineageReadsAHalfWithNothingInIt, which stayed true and stopped
// meaning anything.
//
// That test asserted the derived lineage was empty — and it still is, for ever,
// because nothing loads it. What changed is what an empty answer means: before
// this task it meant "no derived table has been written yet", and after it a
// migration filed under migrations/derived would be **applied by nothing** and
// the same assertion would still be green. A migration nobody runs is worse
// than a missing one: it is a schema change a reader can see in the tree and a
// database will never have.
//
// So the assertion moves to the directory. It is also in
// filedUnderThePreciousLineage, which is where both features' migration tests
// meet it; here it stands alone, because that helper is about where a *file*
// went and this is about the shape of the half.
func TestNoDerivedMigrationsDirectoryExists(t *testing.T) {
	entries, err := migrationFiles.ReadDir("migrations/" + string(Derived))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("migrations/%s exists and holds %d entries (err %v). The derived half is not a "+
			"lineage: its whole schema is derived/library.sql and nothing applies a file filed "+
			"here, so a migration in it is a schema change that no database will ever have",
			Derived, len(entries), err)
	}
	if strings.TrimSpace(derivedSchema) == "" {
		t.Error("derived/library.sql is empty, and it is the only thing that creates the derived half")
	}
}
