package sqlite

import (
	"context"
	"database/sql"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// The identifiers these tests share. They are the 32 lowercase hex 003 §3.6
// requires rather than convenient short strings, so a column too narrow for one
// would fail here and not in the feature that first declares a real library.
const (
	testFilmLibraryID  = "3c59dc048e8850243be8079a5c74d079"
	testShowLibraryID  = "b6d767d2f8ed5d21a44b0e5886680cb9"
	testMusicLibraryID = "37693cfc748049e45d87b8c7d8b9aacd"
)

// aLibrary is the record every test here starts from, with the two frozen
// columns at values that are not Go's zero: `movies` is not the empty string
// and `case_sensitive` is deliberately true in some tests and false in others,
// because a column read back as false proves nothing when false is also what an
// unread column returns.
func aLibrary(id, name, folded string) ports.Library {
	return ports.Library{
		ID:             id,
		Name:           name,
		NameFolded:     folded,
		CollectionType: string(library.Movies),
		CaseSensitive:  false,
		Roots:          []string{"/mnt/films"},
		CreatedAt:      units.TimeFromTicks(638_000_000_000_000_000),
	}
}

// TestTheLibrariesMigrationIsFiledUnderThePreciousLineage is T10's first
// clause, and it is the clause the whole of 003 §3.6 rests on.
//
// A library's identity is **allocated** and never derived (003 §3.6), so these
// two tables are not reconstructible from anything on disk — and the two frozen
// columns beside the identifier are worse than it if they are lost, because
// `case_sensitive` decides every identifier under the library and
// `collection_type` decides every item's type as well. Filed in the derived
// directory this migration would create exactly the same tables and every other
// test in this file would pass; what it would have changed is that a rescan
// would be entitled to drop them, and the first symptom would be an
// installation whose every item had a new identifier after a scan.
func TestTheLibrariesMigrationIsFiledUnderThePreciousLineage(t *testing.T) {
	filedUnderThePreciousLineage(t, "0003_libraries.sql")
}

// TestAFirstStartCreatesTheTwoLibraryTables is the same clause seen from a
// start rather than from the runner.
func TestAFirstStartCreatesTheTwoLibraryTables(t *testing.T) {
	store := openForTest(t)

	for _, table := range []string{"libraries", "library_roots"} {
		var name string
		if err := store.reader.QueryRow(
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table,
		).Scan(&name); err != nil {
			t.Errorf("the %s table is not there after a first start: %v", table, err)
		}
	}

	derived, err := store.SchemaVersion(context.Background(), Derived)
	if err != nil {
		t.Fatalf("SchemaVersion(derived) returned %v", err)
	}
	if derived != 0 {
		t.Errorf("the derived half is at version %d, want 0: this task files nothing there", derived)
	}
}

// TestALibraryRoundTripsWholeWithItsRootsAndItsFrozenColumns is the base every
// clause below narrows: what CreateLibrary wrote is what the two reads answer.
//
// It is read back through both reads rather than one, because they are two
// queries — Libraries reads the whole roots table and groups it,
// LibraryByFoldedName reads one library's — and a field filled from the wrong
// column in one of them is invisible to a test that only uses the other.
func TestALibraryRoundTripsWholeWithItsRootsAndItsFrozenColumns(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	want := aLibrary(testMusicLibraryID, "Albums", "albums")
	want.CollectionType = string(library.Music)
	want.CaseSensitive = true
	want.Roots = []string{"/mnt/music", "/mnt/more-music"}

	if err := store.CreateLibrary(ctx, want); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}

	byName, found, err := store.LibraryByFoldedName(ctx, "albums")
	if err != nil {
		t.Fatalf("LibraryByFoldedName returned %v", err)
	}
	if !found {
		t.Fatal("LibraryByFoldedName did not find the library that was just created")
	}
	assertSameLibrary(t, "LibraryByFoldedName", byName, want)

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("Libraries returned %d libraries, want 1", len(all))
	}
	assertSameLibrary(t, "Libraries", all[0], want)
}

// assertSameLibrary compares two records field by field, and names the field.
//
// Field by field rather than with one equality, because CreatedAt is a
// units.Time and a Time carries a monotonic reading and a location that make ==
// answer a question nobody asked; units.Time.Equal is the comparison this
// project uses (003 T9 measured why).
func assertSameLibrary(t *testing.T, through string, got, want ports.Library) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("%s: ID is %q, want %q", through, got.ID, want.ID)
	}
	if got.Name != want.Name {
		t.Errorf("%s: Name is %q, want %q", through, got.Name, want.Name)
	}
	if got.NameFolded != want.NameFolded {
		t.Errorf("%s: NameFolded is %q, want %q", through, got.NameFolded, want.NameFolded)
	}
	if got.CollectionType != want.CollectionType {
		t.Errorf("%s: CollectionType is %q, want %q", through, got.CollectionType, want.CollectionType)
	}
	if got.CaseSensitive != want.CaseSensitive {
		t.Errorf("%s: CaseSensitive is %v, want %v — it decides every identifier under the library",
			through, got.CaseSensitive, want.CaseSensitive)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("%s: CreatedAt is %s, want %s", through, got.CreatedAt, want.CreatedAt)
	}
	if !slices.Equal(got.Roots, want.Roots) {
		t.Errorf("%s: Roots are %v, want %v", through, got.Roots, want.Roots)
	}
}

// TestLibraryNamesDifferingOnlyInCaseAreRefused is T10's second clause.
//
// `name_folded` exists so that the subcommand's assumption is the database's
// rule and not a convention: 003 plan §6.7 addresses a library by name, because
// an operator has the name and never the allocated identifier, and two
// libraries that fold to one name leave that lookup choosing between two
// libraries with no defined answer — which is two different sets of resolution
// rules and two different identifier spaces for the same words.
func TestLibraryNamesDifferingOnlyInCaseAreRefused(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if err := store.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Films", "films")); err != nil {
		t.Fatalf("creating the first library returned %v", err)
	}

	second := aLibrary(testShowLibraryID, "FILMS", "films")
	err := store.CreateLibrary(ctx, second)
	if err == nil {
		t.Fatal("creating a second library whose folded name is taken succeeded, want a refusal")
	}
	if !strings.Contains(err.Error(), "FILMS") {
		t.Errorf("the refusal is %q, and does not name the library that was refused", err)
	}

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	if len(all) != 1 {
		t.Errorf("there are %d libraries after the refusal, want 1: the whole write is refused, "+
			"and a roots row for a library that does not exist is what a half-applied create "+
			"would leave", len(all))
	}
}

// TestARenameIntoATakenFoldIsRefused is the same uniqueness through the other
// verb.
//
// Worth its own test rather than left to the create: renaming is the free edit
// 003 §3.6 makes free precisely so that recreating stays the expensive one, and
// a rename that could collide would be the collision arriving through the door
// nobody guards.
func TestARenameIntoATakenFoldIsRefused(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if err := store.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Films", "films")); err != nil {
		t.Fatalf("creating the first library returned %v", err)
	}
	shows := aLibrary(testShowLibraryID, "Shows", "shows")
	shows.CollectionType = string(library.Shows)
	if err := store.CreateLibrary(ctx, shows); err != nil {
		t.Fatalf("creating the second library returned %v", err)
	}

	if err := store.RenameLibrary(ctx, testShowLibraryID, "films", "films"); err == nil {
		t.Fatal("renaming a library onto a taken folded name succeeded, want a refusal")
	}

	unchanged, found, err := store.LibraryByFoldedName(ctx, "shows")
	if err != nil || !found {
		t.Fatalf("LibraryByFoldedName(shows) returned (%v, %v) after the refused rename", found, err)
	}
	if unchanged.Name != "Shows" {
		t.Errorf("the library is named %q after a refused rename, want %q", unchanged.Name, "Shows")
	}
}

// TestWhatTheFoldedNameRefusesIsWhatTheCallerFolded is the honest edge of the
// clause above, and it is here because 003 T3 found the mechanism.
//
// SQLite compares TEXT by bytes, so the uniqueness on `name_folded` refuses
// exactly the pairs the caller folded to one string and no others. The fold is
// the domain's — `RenameLibrary` takes the folded spelling as a parameter for
// that reason — so *which* names are one name is a decision above this layer,
// and this test records where the line falls rather than asserting a fold.
//
// The pair that shows it is Unicode form and not case: `Amélie` written with a
// precomposed U+00E9 and written with `e` + U+0301 are the same name to a
// reader and two byte strings to this column. T3's note is the same mechanism
// seen from the other side — NFC has singleton mappings, so two byte-different
// spellings can be one key — and the consequence for a name is that a fold that
// lowercases without normalising leaves one library declarable twice.
//
// That is T14's decision to take when it writes the subcommand, and it is
// recorded here because this is the layer that cannot take it.
func TestWhatTheFoldedNameRefusesIsWhatTheCallerFolded(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	const (
		precomposed = "am\u00e9lie"  // am + U+00E9 + lie
		decomposed  = "ame\u0301lie" // am + e + U+0301 + lie
	)
	if precomposed == decomposed {
		t.Fatal("the two spellings are the same string, and this test proves nothing")
	}

	if err := store.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Amélie", precomposed)); err != nil {
		t.Fatalf("creating the first library returned %v", err)
	}

	// Folded to the same bytes: refused, which is the clause above.
	same := aLibrary(testShowLibraryID, "AMÉLIE", precomposed)
	if err := store.CreateLibrary(ctx, same); err == nil {
		t.Error("a second library folding to the same bytes was accepted, want a refusal")
	}

	// Folded to different bytes: accepted, whatever it looks like on a screen.
	// This is the store answering honestly rather than a behaviour to rely on:
	// a domain fold that normalised first would hand this column one string and
	// the row would be refused, and nothing here would have to change.
	other := aLibrary(testShowLibraryID, "Amélie", decomposed)
	if err := store.CreateLibrary(ctx, other); err != nil {
		t.Fatalf("a second library folding to different bytes was refused (%v). The uniqueness "+
			"is over the stored bytes, so this column cannot be what decides that two Unicode "+
			"spellings are one name", err)
	}
}

// TestAFourthCollectionTypeIsRefused is T10's third clause.
//
// The CHECK is a backstop and not the refusal an operator meets — 003 plan §7
// refuses a fourth value at flag parsing and lists the three — so what it is
// for is the value that got past that: a library whose type no resolver knows
// is a library `library.Resolve` errors on for the life of the installation,
// and 003 §3.6 offers no verb to correct it.
//
// The three accepted are read from `library.AllCollectionTypes` rather than
// typed here, which is the assertion that matters more than the fourth being
// refused: a fourth type added to the domain and forgotten in this migration
// would be a collection type the code resolves and the database will not store,
// and nothing else in the project compares the two lists.
func TestAFourthCollectionTypeIsRefused(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	ids := []string{testFilmLibraryID, testShowLibraryID, testMusicLibraryID}
	types := library.AllCollectionTypes()
	if len(types) > len(ids) {
		t.Fatalf("there are %d collection types and %d identifiers to give them", len(types), len(ids))
	}
	for i, collectionType := range types {
		accepted := aLibrary(ids[i], string(collectionType), string(collectionType))
		accepted.CollectionType = string(collectionType)
		if err := store.CreateLibrary(ctx, accepted); err != nil {
			t.Errorf("the collection type %q is refused by the schema but is one of "+
				"library.AllCollectionTypes: %v", collectionType, err)
		}
	}

	fourth := aLibrary("d3d9446802a44259755d38e6d163e820", "Photos", "photos")
	fourth.CollectionType = "photos"
	if err := store.CreateLibrary(ctx, fourth); err == nil {
		t.Error("a library whose collection type is \"photos\" was accepted, want a refusal: " +
			"no resolver knows it, and 003 §3.6 offers no verb to change it afterwards")
	}
}

// TestReplaceRootsReordersAndTheRootsReadBackInOrdinalOrder is T10's fourth
// clause.
//
// The order decides nothing about an item's identity — 003 §3.6 derives that
// from the path relative to the root — and that is exactly why it has to hold
// still: a list that moved between two reads would move every recorded
// `RootOrdinal` with it, and an item whose ordinal no longer names the root its
// path is relative to is an item nothing can open. A list that moves is a list
// nothing can be compared against.
//
// The roots are reordered rather than replaced with different ones, so that a
// read which sorted on the path instead of on the ordinal fails: sorted by
// path, the answer is the order they were created in.
func TestReplaceRootsReordersAndTheRootsReadBackInOrdinalOrder(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	created := aLibrary(testFilmLibraryID, "Films", "films")
	created.Roots = []string{"/mnt/a", "/mnt/b", "/mnt/c"}
	if err := store.CreateLibrary(ctx, created); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}

	reordered := []string{"/mnt/c", "/mnt/a", "/mnt/b"}
	if err := store.ReplaceRoots(ctx, testFilmLibraryID, reordered); err != nil {
		t.Fatalf("ReplaceRoots returned %v", err)
	}

	byName, _, err := store.LibraryByFoldedName(ctx, "films")
	if err != nil {
		t.Fatalf("LibraryByFoldedName returned %v", err)
	}
	if !slices.Equal(byName.Roots, reordered) {
		t.Errorf("LibraryByFoldedName answers roots %v, want %v", byName.Roots, reordered)
	}

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	if len(all) != 1 || !slices.Equal(all[0].Roots, reordered) {
		t.Errorf("Libraries answers roots %v, want %v", all[0].Roots, reordered)
	}

	// And the ordinal is what carries it, rather than the read having sorted
	// something. Three roots replaced by three, so a build that appended
	// instead of replacing would have six rows here.
	if got := storedRoots(t, store); !slices.Equal(got, []string{"0 /mnt/c", "1 /mnt/a", "2 /mnt/b"}) {
		t.Errorf("library_roots holds %v, want the ordinals 0, 1 and 2 over the new order", got)
	}
}

// TestReplacingTheRootsWithNoneLeavesALibraryWithNoRoots states the answer for
// the empty case rather than leaving it to be discovered.
//
// A library with no roots is a state — it is what an operator who is about to
// move a mount leaves behind — and the read answers no roots rather than the
// roots it used to have.
func TestReplacingTheRootsWithNoneLeavesALibraryWithNoRoots(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if err := store.CreateLibrary(ctx, aLibrary(testFilmLibraryID, "Films", "films")); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}
	if err := store.ReplaceRoots(ctx, testFilmLibraryID, nil); err != nil {
		t.Fatalf("ReplaceRoots with no roots returned %v", err)
	}

	byName, _, err := store.LibraryByFoldedName(ctx, "films")
	if err != nil {
		t.Fatalf("LibraryByFoldedName returned %v", err)
	}
	if len(byName.Roots) != 0 {
		t.Errorf("the library has roots %v after they were all replaced away, want none", byName.Roots)
	}
}

// TestLibraryRootsAreOrderedByTheirOrdinalAndNotByWhereTheRowsSit is the clause
// above with the coincidence removed, and it is the reason `Libraries` reads
// the whole roots table in one query.
//
// A read filtered on `library_id` is answered through the primary-key index on
// (library_id, ordinal), so SQLite returns it in ordinal order whether or not
// anything asked — which makes the ORDER BY that carries this feature's
// contract a clause no test could observe, and an unobservable clause is one a
// tidy-up deletes. The whole-table read is a scan in row order, so this test
// can rewrite the rows into an order that disagrees with the ordinal and see
// which of the two the answer follows.
//
// The hostile order is asserted before it is used. A test that scrambled the
// rows and did not check the scramble took would pass for the wrong reason on
// the day SQLite chose a different plan.
//
// That this read is where the clause is observable is measured and not argued:
// of the 24 mutations run over this task, `rootsFor` losing its ORDER BY is the
// only survivor, and `readAllRoots` losing the same clause fails here.
// [measurement: 003 T10, 2026-09-05]
func TestLibraryRootsAreOrderedByTheirOrdinalAndNotByWhereTheRowsSit(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	created := aLibrary(testFilmLibraryID, "Films", "films")
	created.Roots = []string{"/mnt/a", "/mnt/b", "/mnt/c"}
	if err := store.CreateLibrary(ctx, created); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}

	// Rewrite the three rows so that the order they sit in is the reverse of
	// the order their ordinals name.
	if _, err := store.writer.ExecContext(ctx, `DELETE FROM library_roots`); err != nil {
		t.Fatalf("emptying library_roots returned %v", err)
	}
	for ordinal := len(created.Roots) - 1; ordinal >= 0; ordinal-- {
		if _, err := store.writer.ExecContext(ctx,
			`INSERT INTO library_roots (library_id, ordinal, path) VALUES (?, ?, ?)`,
			testFilmLibraryID, ordinal, created.Roots[ordinal]); err != nil {
			t.Fatalf("rewriting library_roots returned %v", err)
		}
	}

	unordered := rootsInRowOrder(t, store.reader)
	if !slices.Equal(unordered, []string{"/mnt/c", "/mnt/b", "/mnt/a"}) {
		t.Fatalf("a read with no ORDER BY answers %v, want the reversed order this test just "+
			"wrote. The scramble did not take, so nothing below distinguishes an ordered read "+
			"from an unordered one", unordered)
	}

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("Libraries returned %d libraries, want 1", len(all))
	}
	if !slices.Equal(all[0].Roots, created.Roots) {
		t.Errorf("Libraries answers roots %v, want %v — the order is the ordinal column's, "+
			"not the order the rows happen to sit in", all[0].Roots, created.Roots)
	}
}

// rootsInRowOrder reads the roots table the way a query with no ORDER BY does.
func rootsInRowOrder(t *testing.T, db *sql.DB) []string {
	t.Helper()

	rows, err := db.Query(`SELECT path FROM library_roots`)
	if err != nil {
		t.Fatalf("reading library_roots returned %v", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			t.Fatalf("reading library_roots returned %v", err)
		}
		paths = append(paths, path)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading library_roots returned %v", err)
	}
	return paths
}

// storedRoots reads every root row as "ordinal path", in ordinal order.
func storedRoots(t *testing.T, store *Store) []string {
	t.Helper()

	rows, err := store.reader.Query(
		`SELECT ordinal, path FROM library_roots ORDER BY library_id, ordinal`)
	if err != nil {
		t.Fatalf("reading library_roots returned %v", err)
	}
	defer rows.Close()

	var stored []string
	for rows.Next() {
		var (
			ordinal int
			path    string
		)
		if err := rows.Scan(&ordinal, &path); err != nil {
			t.Fatalf("reading library_roots returned %v", err)
		}
		stored = append(stored, strconv.Itoa(ordinal)+" "+path)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("reading library_roots returned %v", err)
	}
	return stored
}

// TestLibrariesAreReturnedInAStatedOrder is architecture §2's rule at the read
// a scan of every library walks.
//
// The libraries are created in an order that disagrees with the answer, so a
// read with no ORDER BY — which is SQLite's storage order, stable until a row
// is rewritten and then not — answers the creation order and fails.
func TestLibrariesAreReturnedInAStatedOrder(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	for _, created := range []ports.Library{
		aLibrary(testMusicLibraryID, "Zed", "zed"),
		aLibrary(testFilmLibraryID, "Alpha", "alpha"),
		aLibrary(testShowLibraryID, "Middle", "middle"),
	} {
		if err := store.CreateLibrary(ctx, created); err != nil {
			t.Fatalf("CreateLibrary(%s) returned %v", created.Name, err)
		}
	}

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries returned %v", err)
	}
	var names []string
	for _, one := range all {
		names = append(names, one.Name)
	}
	if want := []string{"Alpha", "Middle", "Zed"}; !slices.Equal(names, want) {
		t.Errorf("Libraries answers %v, want %v — the order is name_folded and then id", names, want)
	}
}

// TestRemovingALibraryTakesItsRootsWithItAndLeavesAnotherLibraryAlone is the
// cascade this migration declares, asserted rather than assumed.
//
// Foreign keys are on (ADR-0003's writer DSN), so without the cascade the
// delete is refused outright and `atrium library remove` never works. With a
// cascade written too wide it is the over-broad DELETE 002's T4 already had to
// guard against once — which here costs another library's configuration rather
// than a token.
func TestRemovingALibraryTakesItsRootsWithItAndLeavesAnotherLibraryAlone(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	films := aLibrary(testFilmLibraryID, "Films", "films")
	films.Roots = []string{"/mnt/a", "/mnt/b"}
	if err := store.CreateLibrary(ctx, films); err != nil {
		t.Fatalf("creating the first library returned %v", err)
	}
	shows := aLibrary(testShowLibraryID, "Shows", "shows")
	shows.CollectionType = string(library.Shows)
	shows.Roots = []string{"/mnt/shows"}
	if err := store.CreateLibrary(ctx, shows); err != nil {
		t.Fatalf("creating the second library returned %v", err)
	}

	if err := store.RemoveLibrary(ctx, testFilmLibraryID); err != nil {
		t.Fatalf("RemoveLibrary returned %v", err)
	}

	if _, found, err := store.LibraryByFoldedName(ctx, "films"); err != nil || found {
		t.Errorf("the removed library is still there (found %v, err %v)", found, err)
	}
	survivor, found, err := store.LibraryByFoldedName(ctx, "shows")
	if err != nil || !found {
		t.Fatalf("LibraryByFoldedName(shows) returned (%v, %v) after the other was removed", found, err)
	}
	if !slices.Equal(survivor.Roots, []string{"/mnt/shows"}) {
		t.Errorf("the surviving library has roots %v, want [/mnt/shows]", survivor.Roots)
	}
	if got := storedRoots(t, store); !slices.Equal(got, []string{"0 /mnt/shows"}) {
		t.Errorf("library_roots holds %v after one library was removed, want only the "+
			"survivor's", got)
	}
}

// TestRenamingALibraryChangesNothingElse is 003 §3.6's sharpest consequence
// stated as a test: editing a library is free, and recreating one is not.
//
// The identity, the two frozen columns and the roots all have to survive a
// rename, because every identifier under the library is derived from the first
// three and none of them is stored anywhere else to recover from.
func TestRenamingALibraryChangesNothingElse(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	created := aLibrary(testFilmLibraryID, "Films", "films")
	created.CaseSensitive = true
	created.Roots = []string{"/mnt/a", "/mnt/b"}
	if err := store.CreateLibrary(ctx, created); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
	}

	if err := store.RenameLibrary(ctx, testFilmLibraryID, "Cinema", "cinema"); err != nil {
		t.Fatalf("RenameLibrary returned %v", err)
	}

	renamed, found, err := store.LibraryByFoldedName(ctx, "cinema")
	if err != nil || !found {
		t.Fatalf("LibraryByFoldedName(cinema) returned (%v, %v) after the rename", found, err)
	}
	want := created
	want.Name, want.NameFolded = "Cinema", "cinema"
	assertSameLibrary(t, "after a rename", renamed, want)

	if _, found, err := store.LibraryByFoldedName(ctx, "films"); err != nil || found {
		t.Errorf("the old folded name still finds the library (found %v, err %v)", found, err)
	}
}

// TestAWriteAgainstALibraryThatIsNotThereIsRefused covers the three verbs that
// address a library by its identifier.
//
// An UPDATE or a DELETE that matched nothing succeeds in SQL, so without a
// guard every one of these looks exactly like a write that worked — and
// `ReplaceRoots` is the one that needs the guard most, because replacing the
// roots of a library that does not exist with *no* roots is a DELETE matching
// nothing followed by no INSERT at all. Nothing fails, nothing is written, and
// an operator who mistyped a name is told the roots were replaced.
func TestAWriteAgainstALibraryThatIsNotThereIsRefused(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)
	const nobody = "00000000000000000000000000000000"

	if err := store.RenameLibrary(ctx, nobody, "Cinema", "cinema"); err == nil {
		t.Error("renaming a library that is not there succeeded, want a refusal")
	}
	if err := store.RemoveLibrary(ctx, nobody); err == nil {
		t.Error("removing a library that is not there succeeded, want a refusal")
	}
	if err := store.ReplaceRoots(ctx, nobody, []string{"/mnt/a"}); err == nil {
		t.Error("replacing the roots of a library that is not there succeeded, want a refusal")
	}
	if err := store.ReplaceRoots(ctx, nobody, nil); err == nil {
		t.Error("replacing the roots of a library that is not there with no roots succeeded. " +
			"That is the one case the foreign key cannot catch: no row is written, so nothing " +
			"refuses it but the lookup this method performs")
	}
}

// TestALibraryReadThatFindsNothingIsNotAnError is the shape 002's own lookups
// have, kept the same on purpose: an operator naming a library that is not
// there is an answer the subcommand prints.
func TestALibraryReadThatFindsNothingIsNotAnError(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	got, found, err := store.LibraryByFoldedName(ctx, "films")
	if err != nil {
		t.Fatalf("LibraryByFoldedName over an empty store returned %v, want no error", err)
	}
	if found {
		t.Errorf("LibraryByFoldedName found %+v in an empty store", got)
	}

	all, err := store.Libraries(ctx)
	if err != nil {
		t.Fatalf("Libraries over an empty store returned %v", err)
	}
	if len(all) != 0 {
		t.Errorf("Libraries answers %d libraries in an empty store", len(all))
	}
}

// TestALibraryOutlivesARestart is what "precious" means, at the only level this
// task can assert it.
//
// The stronger claim — that the derived half being dropped and rebuilt leaves
// these rows untouched — is T11's, because nothing here can drop anything yet.
func TestALibraryOutlivesARestart(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()

	first, err := Open(ctx, directory)
	if err != nil {
		t.Fatalf("the first Open returned %v", err)
	}
	created := aLibrary(testFilmLibraryID, "Films", "films")
	created.CaseSensitive = true
	if err := first.CreateLibrary(ctx, created); err != nil {
		t.Fatalf("CreateLibrary returned %v", err)
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
	back, found, err := second.LibraryByFoldedName(ctx, "films")
	if err != nil || !found {
		t.Fatalf("LibraryByFoldedName after a restart returned (%v, %v)", found, err)
	}
	assertSameLibrary(t, "after a restart", back, created)
}
