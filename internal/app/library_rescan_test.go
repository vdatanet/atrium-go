package app

// AC-2, AC-3 and AC-10: the three criteria that are about scanning more than
// once.
//
// # Why they are one file and one set of helpers
//
// They are **one property measured three ways** — an item's identifier is a
// function of the item and of nothing else — and 003 tasks' T15 says in terms
// why writing them apart is the risk: *"writing them apart is how one of them
// ends up asserting a subset of another"*. So the comparison is written once,
// in [sameIdentifiers], and the three tests differ in exactly one thing each:
// what happens **between** the two scans.
//
//	AC-2   nothing happens                         the store keeps its memory
//	AC-3   the derived half is dropped and rebuilt  the store loses its memory
//	AC-10  the whole tree moves to another path     the root under it moves
//
// # Where they sit, and why it is not `internal/library`
//
// At the `app` level of 003 tasks' three: through the subcommand, over a real
// temporary data directory, a real SQLite store and a real tree. The criterion
// is about **what the store ends up holding** across two scans, and a function
// cannot be asked that — `library.Resolve` has no memory to keep or lose, so
// neither AC-2's *"the store believed the row it already had"* nor AC-3's
// *"there was no row to believe"* is a distinction it can express.
//
// # The separating mutations, which were run
//
// A criterion that no build fails is not a criterion, and each of these three
// is separated from the other two by a build that passes those two and fails
// only it (003 tasks, T15). Five were applied to a scratch copy of the tree,
// one at a time, and each was run against all three criteria, against 003
// T14's moved-root test and against the whole of `internal/library`
// `[measurement: 003 T15, 5 mutations, 2026-09-05]`:
//
//	                                              AC-2   AC-3  AC-10  T14  library
//	every examined item reported updated          RED    -     -      -    -
//	a previous row's identifier adopted           -      -     -      -    -
//	  the same, over an identifier allocated
//	  rather than derived                         -      RED   -      -    -
//	the root's path in every file-backed key      -      -     RED    RED  -
//	  the same, in the Series and MusicArtist
//	  key alone                                   -      -     RED    -    -
//
// Three things that table says and a sentence would not.
//
// **AC-2's half is the summary and nothing else.** AC-3's second scan reports
// every item as an addition — that is what removing the store's memory means —
// and AC-10's reports nothing because nothing moved, so a reconciliation that
// called every examined item updated is invisible to both and red only here.
//
// **Adopting a previous row's identifier is, on its own, a no-op that nothing
// in this project can see.** 003 tasks names AC-3's separating mutation as
// *"a derivation that reuses a previous row's identifier when it finds one"*,
// and measured, that build passes every test in the repository — because
// against a correct derivation the adopted string and the derived string are
// the same string. What AC-3 actually catches is the adoption **hiding an
// identifier that is not derived at all**: allocate one for an item with no
// previous row, adopt one for an item that has one, and the result is stable
// across a rescan, stable across a remount, correct in `internal/library`'s
// every table — and different on every installation, which is precisely what
// 003 §3.6 forbids in the words *"derived from the item's stable identity,
// never allocated"*. Removing the store's memory is the only thing that asks.
//
// **The root path in the key has a broad form and a narrow one, and only the
// narrow one is this criterion's own.** The broad form moves every `Movie`,
// `Episode` and `Audio` identifier and 003 T14's moved-root test over a tree
// of films is red on it too. The narrow form puts the root's path in the key
// of the two kinds whose §3.6 row literally says *"the library root plus the
// normalised name"* — and it is green on T14's test, green on
// `internal/library`'s own moved-root test, and red only here.
//
// # The corpus, and what it has to be able to see
//
// All three run over the **whole fixture** — four libraries, all eight kinds —
// and each asserts that before it asserts anything else ([theEightKinds]).
// That is not thoroughness for its own sake: §3.6's identity table has five
// rows, and only the `Movie`/`Episode`/`Audio` row keys on the path. A tree of
// films exercises one row of five, and the narrow form of AC-10's mutation —
// the root path in a **`Series`'** key and nowhere else — passes over such a
// tree. Both of the moved-root assertions that existed before this change are
// green under exactly that build: 003 T14's
// `TestRenameAndRootsLeaveEveryIdentifierUnchanged`, which moves three films,
// and `internal/library`'s own, which moves the roots of an **empty** library
// and therefore asks about one `CollectionFolder` row. **A corpus that cannot
// hold a series is a moved-root test that asserts one of §3.6's five rows.**
//
// # What none of this proves
//
// That the identifier a client receives is the one that was stored. 003 plan
// §8.3 row 1: this feature registers no route, so the string compared below is
// the **column**. It is the cheapest debt in the project to discharge — one
// `/Items` listing at 005 covers rows 1, 2, 3 and 4 together — and it is left
// visible here rather than papered over with a test at a level this feature
// cannot reach.

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/libraryfixture"
)

// --- The corpus ---------------------------------------------------------------

// theWholeFixture builds the fixture tree under a fresh directory and declares
// its four libraries through `atrium library add`, which is what allocates
// each library's identity (003 §3.6).
//
// It returns the data directory, the directory the tree was built into, and
// the libraries as `list` reports them, in the fixture's declaration order.
func theWholeFixture(t *testing.T, trees string) (string, []libraryReport) {
	t.Helper()

	if err := libraryfixture.Build(trees); err != nil {
		t.Fatalf("building the fixture tree in %s: %v", trees, err)
	}

	data := t.TempDir()
	declared := make([]libraryReport, 0, len(libraryfixture.Libraries()))
	for _, fixture := range libraryfixture.Libraries() {
		declared = append(declared, addLibrary(t, data,
			fixture.Name, fixture.CollectionType, filepath.Join(trees, fixture.Name)))
	}
	return data, declared
}

// theEightKinds fails the test unless the store holds an item of every
// `library.Kind`, over all the libraries given.
//
// **This is the corpus control, and it is the first assertion in all three
// tests.** 003 §3.6 derives an identifier from a different key per kind — a
// path for the three file-backed types, a parent's identity plus a
// distinguisher for a `Season` and a `MusicAlbum`, the library's identity for
// a `CollectionFolder`, and *"the library root plus the normalised name"* for
// a `Series` and a `MusicArtist`. A corpus that holds only films asserts one
// of those five rows and reports a green for the other four.
func theEightKinds(t *testing.T, data string, libraries []libraryReport) {
	t.Helper()

	held := map[string]bool{}
	total := 0
	for _, lib := range libraries {
		for _, item := range storedItems(t, data, lib.ID) {
			held[item.Type] = true
			total++
		}
	}

	var missing []string
	for _, kind := range library.AllKinds() {
		if !held[string(kind)] {
			missing = append(missing, string(kind))
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the corpus holds no %v, so it cannot see a build that keys those kinds "+
			"differently: 003 §3.6's identity table has five rows and this tree exercises fewer "+
			"(the store holds %d items across %d libraries)", missing, total, len(libraries))
	}
}

// --- What the store holds, in the shapes an identifier can move in ------------

// storedIdentity is one library's identifiers, in the two shapes that tell a
// stable identifier from one that merely stayed in the set.
//
// A sorted list alone is satisfied by a build that swapped two items'
// identifiers, and a path-keyed map alone says nothing about the containers —
// which are four of the eight kinds and the rows §3.6 keys on something other
// than a path. Both are compared.
type storedIdentity struct {
	// identifiers is every identifier the library holds, sorted.
	identifiers []string

	// names is the identifier of every item, against what that item is: its
	// type, and its root-relative path or its name where it has none.
	names map[string]string

	// atPath is the identifier of every item that has a path, against the
	// root ordinal and the path it was derived from.
	atPath map[string]string
}

// identityOf reads what the store holds for one library.
func identityOf(t *testing.T, data, libraryID string) storedIdentity {
	t.Helper()

	held := storedIdentity{names: map[string]string{}, atPath: map[string]string{}}
	for _, item := range storedItems(t, data, libraryID) {
		held.identifiers = append(held.identifiers, item.ID)

		where := fmt.Sprintf("named %q", item.Name)
		if item.Path != "" {
			where = fmt.Sprintf("at root %d %q", item.RootOrdinal, item.Path)
			held.atPath[fmt.Sprintf("%d/%s", item.RootOrdinal, item.Path)] = item.ID
		}
		held.names[item.ID] = fmt.Sprintf("%s %s", item.Type, where)
	}
	slices.Sort(held.identifiers)
	return held
}

// identityOfEvery reads what the store holds for every library, by name.
func identityOfEvery(t *testing.T, data string, libraries []libraryReport) map[string]storedIdentity {
	t.Helper()

	held := make(map[string]storedIdentity, len(libraries))
	for _, lib := range libraries {
		held[lib.Name] = identityOf(t, data, lib.ID)
	}
	return held
}

// sameIdentifiers is the assertion all three criteria share: every identifier
// the store held before is the identifier it holds now, for the same item.
//
// `what` names what happened between the two scans, so a failure reads as the
// criterion it belongs to rather than as a diff.
func sameIdentifiers(t *testing.T, what string, before, after map[string]storedIdentity) {
	t.Helper()

	for name, was := range before {
		is, ok := after[name]
		if !ok {
			t.Errorf("%s: the library %q is gone", what, name)
			continue
		}

		if !slices.Equal(was.identifiers, is.identifiers) {
			t.Errorf("%s: %q holds %d identifiers and held %d, and they are not the same set:\n"+
				" gone:    %v\n appeared: %v",
				what, name, len(is.identifiers), len(was.identifiers),
				missingFrom(was.identifiers, is.identifiers),
				missingFrom(is.identifiers, was.identifiers))
		}

		// An identifier that survived as a string but now names a different
		// item is the failure a set comparison cannot see.
		for id, wasNamed := range was.names {
			if isNamed, ok := is.names[id]; ok && isNamed != wasNamed {
				t.Errorf("%s: %q: the identifier %s named %s and now names %s",
					what, name, id, wasNamed, isNamed)
			}
		}

		// And the other direction: the item at a path must have kept its
		// identifier, which is what a client's favourites and resume
		// positions are keyed on (003 §3.6).
		for path, wasID := range was.atPath {
			isID, ok := is.atPath[path]
			if !ok {
				t.Errorf("%s: %q: nothing is stored at %s any more, and %s was",
					what, name, path, wasID)
				continue
			}
			if isID != wasID {
				t.Errorf("%s: %q: the item at %s is %s and was %s", what, name, path, isID, wasID)
			}
		}
	}
}

// missingFrom is everything in `was` that `is` does not hold — a failure
// message that names the identifiers rather than counting them.
func missingFrom(was, is []string) []string {
	var gone []string
	for _, id := range was {
		if !slices.Contains(is, id) {
			gone = append(gone, id)
		}
	}
	return gone
}

// --- The summary --------------------------------------------------------------

// scanSummaries runs one `atrium library scan --format json` and parses what
// it wrote, by library name.
//
// The **document** is what a test reads and the table is what an operator
// reads (003 plan §6.7). The three lists are lists and never `null` — a scan
// that changed nothing writes `[]` — so nothing here tolerates both spellings.
func scanSummaries(t *testing.T, data string, args ...string) map[string]scanReport {
	t.Helper()

	stdout := mustScan(t, data, append([]string{"--" + flagFormat, formatJSON}, args...)...)

	var document struct {
		Libraries []scanReport `json:"libraries"`
	}
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("parsing the summary %q: %v", stdout, err)
	}

	summaries := make(map[string]scanReport, len(document.Libraries))
	for _, report := range document.Libraries {
		summaries[report.Name] = report
	}
	return summaries
}

// --- AC-2 ---------------------------------------------------------------------

// TestScanningTwiceKeepsEveryIdentifierAndReportsNothingChanged is AC-2:
// *"Scanning twice produces byte-identical item identifiers, and the second
// scan reports no changes."*
//
// The half that is only this criterion's is the **summary**. AC-3's second
// scan reports every item as an addition — that is what removing the store's
// memory means — and AC-10's reports nothing because nothing moved, so a
// reconciliation that called every examined item updated is invisible to both
// of them and red only here.
func TestScanningTwiceKeepsEveryIdentifierAndReportsNothingChanged(t *testing.T) {
	t.Parallel()

	data, libraries := theWholeFixture(t, t.TempDir())

	first := scanSummaries(t, data)
	theEightKinds(t, data, libraries)
	before := identityOfEvery(t, data, libraries)

	// The corpus control, in the direction this criterion needs it: the first
	// scan has to have *added* something, or "the second scan added nothing"
	// is a sentence about an installation with nothing in it.
	added := 0
	for _, summary := range first {
		added += len(summary.Added)
	}
	if added < 40 {
		t.Fatalf("the first scan added %d items over the whole fixture, which is too few for "+
			"the second scan's three empty lists to mean anything", added)
	}

	// Nothing happens here. That absence is the whole of what separates this
	// criterion from the two below it.
	second := scanSummaries(t, data)

	if len(second) != len(first) {
		t.Fatalf("the second scan reported %d libraries and the first reported %d",
			len(second), len(first))
	}
	for name, summary := range second {
		if len(summary.Added) != 0 || len(summary.Updated) != 0 || len(summary.Removed) != 0 {
			t.Errorf("the second scan of %q reports %d added, %d updated and %d removed, "+
				"and the tree did not change:\n added:   %v\n updated: %v\n removed: %v",
				name, len(summary.Added), len(summary.Updated), len(summary.Removed),
				summary.Added, summary.Updated, summary.Removed)
		}

		// The control that keeps the three empty lists honest: the second scan
		// read the same tree the first one did. A scan that examined nothing
		// also changes nothing.
		if got, want := summary.Examined, first[name].Examined; got != want {
			t.Errorf("the second scan of %q examined %d files and the first examined %d",
				name, got, want)
		}
	}

	sameIdentifiers(t, "scanning the same tree again", before, identityOfEvery(t, data, libraries))
}

// --- AC-3 ---------------------------------------------------------------------

// TestScanningIntoARebuiltDerivedHalfDerivesTheSameIdentifiers is AC-3:
// *"Scanning into an empty database produces the same identifiers as the first
// scan did."*
//
// **This is AC-2's criterion with the store's memory removed**, and it is the
// one that catches a derivation that reads a stored row. Between the two scans
// the derived half is dropped and created again through `RebuildDerived`
// (003 plan §6.8) — which is the act ADR-0003 licenses and the state a
// generation bump puts an installation in, so this is not a contrivance but
// the ordinary consequence of shipping a new schema.
//
// The precious half is untouched by that, which is why the four libraries keep
// their allocated identities across it. **That is the criterion and not a
// convenience**: an item's identifier derives from its library's identity, so
// a rebuild that dropped the libraries too would give every item a new
// identifier and would be right to.
func TestScanningIntoARebuiltDerivedHalfDerivesTheSameIdentifiers(t *testing.T) {
	t.Parallel()

	data, libraries := theWholeFixture(t, t.TempDir())

	mustScan(t, data)
	theEightKinds(t, data, libraries)
	before := identityOfEvery(t, data, libraries)

	rebuildTheDerivedHalf(t, data)

	// The control, and it is the one that separates this test from AC-2: the
	// store's memory really is gone. A `RebuildDerived` that did nothing would
	// make every assertion below a second, weaker spelling of AC-2.
	for _, lib := range libraries {
		if items := storedItems(t, data, lib.ID); len(items) != 0 {
			t.Fatalf("%q still holds %d items after the derived half was rebuilt, so this test "+
				"is AC-2 again: %s", lib.Name, len(items), describe(items))
		}
	}
	// And the other half of the same control: the libraries survived, because
	// they are precious (ADR-0003) and an item's identifier derives from the
	// library's identity.
	survived := listedLibraries(t, data)
	if len(survived) != len(libraries) {
		t.Fatalf("%d libraries survived the rebuild and %d were declared: the derived half's "+
			"rebuild reached the precious one", len(survived), len(libraries))
	}
	for _, lib := range libraries {
		if !slices.ContainsFunc(survived, func(r libraryReport) bool { return r.ID == lib.ID }) {
			t.Fatalf("the library %q (%s) lost its allocated identity in the rebuild", lib.Name, lib.ID)
		}
	}

	second := scanSummaries(t, data)

	// Every item is an addition now, and that is the sentence this criterion
	// exists to make true of a **freshly derived** identifier rather than of a
	// remembered one.
	for _, lib := range libraries {
		summary, ok := second[lib.Name]
		if !ok {
			t.Fatalf("the scan after the rebuild did not report on %q", lib.Name)
		}
		held := identityOf(t, data, lib.ID)
		if !slices.Equal(slices.Sorted(slices.Values(summary.Added)), held.identifiers) {
			t.Errorf("the scan after the rebuild reports %d additions for %q and the store holds "+
				"%d items: with no previous row anywhere, every item is an addition",
				len(summary.Added), lib.Name, len(held.identifiers))
		}
		if len(summary.Updated) != 0 || len(summary.Removed) != 0 {
			t.Errorf("the scan after the rebuild reports %d updated and %d removed for %q, "+
				"and there was nothing to update or remove",
				len(summary.Updated), len(summary.Removed), lib.Name)
		}
	}

	sameIdentifiers(t, "scanning into a rebuilt derived half", before, identityOfEvery(t, data, libraries))
}

// rebuildTheDerivedHalf drops every object of the derived schema and creates
// it again — 003 plan §6.8's rebuild, performed the way a caller performs it.
func rebuildTheDerivedHalf(t *testing.T, data string) {
	t.Helper()

	store, err := openStoreAt(context.Background(), data)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	if err := store.RebuildDerived(context.Background()); err != nil {
		t.Fatalf("rebuilding the derived half: %v", err)
	}
}

// --- AC-10 --------------------------------------------------------------------

// TestMovingTheWholeTreeToAnotherPathLeavesEveryIdentifierUnchanged is AC-10:
// *"Moving the library root to a different path leaves every identifier
// unchanged."*
//
// It is the criterion 003 plan §6.3's *"the library identifier is in the key,
// and the root path is not"* exists for, and the reference does not have it:
// every one of 448 measured identifiers there is reproducible from the file's
// absolute path alone, containers included
// `[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`, so
// an operator who remounts a library there loses every favourite under it
// (behaviours §1.4).
//
// # Why the whole fixture and not a tree of films
//
// Because §3.6 keys a `Series` and a `MusicArtist` on *"the library root plus
// the normalised name"*, and reading those four words as the root's **path**
// is the misreading plan §6.3 refuses in a paragraph of its own. A build that
// made only that misreading — the root path in a `Series`' key, the relative
// path still the whole of a `Movie`'s — moves no identifier in a movies
// library, so 003 T14's `TestRenameAndRootsLeaveEveryIdentifierUnchanged`,
// which moves a tree of three films, is green under it. **A corpus that cannot
// hold a series is a moved-root test that asserts one of §3.6's five rows.**
//
// # Why the whole tree moves, and not one root
//
// The four libraries move together, which is the shape an operator's remount
// actually has: a container started with a different mount point, or a media
// disc mounted somewhere else after a reboot. Moving one and leaving three
// would leave three libraries proving nothing beside the one that did.
func TestMovingTheWholeTreeToAnotherPathLeavesEveryIdentifierUnchanged(t *testing.T) {
	t.Parallel()

	// Both directories are under one parent so that the move is a rename
	// within a filesystem and cannot silently become a copy.
	parent := t.TempDir()
	before := filepath.Join(parent, "before")
	after := filepath.Join(parent, "after")

	data, libraries := theWholeFixture(t, before)

	first := scanSummaries(t, data)
	theEightKinds(t, data, libraries)
	held := identityOfEvery(t, data, libraries)

	if err := os.Rename(before, after); err != nil {
		t.Fatalf("moving the whole tree from %s to %s: %v", before, after, err)
	}

	for _, lib := range libraries {
		mustRunLibrary(t, libraryRoots, "--"+flagDataDirectory, data,
			"--"+flagName, lib.Name, "--"+flagRoot, filepath.Join(after, lib.Name))
	}

	// The control: the roots really did move, and to a path that shares no
	// prefix with the old one below the parent. Without this the scan below
	// would be a second scan of the same tree, which is AC-2.
	for _, listed := range listedLibraries(t, data) {
		wanted := []string{filepath.Join(after, listed.Name)}
		if !slices.Equal(listed.Roots, wanted) {
			t.Fatalf("%q is rooted at %v and should be at %v", listed.Name, listed.Roots, wanted)
		}
	}

	second := scanSummaries(t, data)

	// And the control that the second scan read the moved tree rather than
	// nothing: a scan that examined no file changes no identifier either.
	for name, summary := range second {
		if got, want := summary.Examined, first[name].Examined; got != want {
			t.Errorf("after the move the scan of %q examined %d files and it examined %d before",
				name, got, want)
		}
	}

	sameIdentifiers(t, "moving the whole tree to another path", held, identityOfEvery(t, data, libraries))
}
