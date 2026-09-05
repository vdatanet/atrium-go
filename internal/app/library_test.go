package app

// `atrium library scan`'s tests sit at the `app` level of 003 tasks' three:
// through the subcommand, over a real temporary data directory, a real SQLite
// store and a real tree.
//
// # Why through the subcommand, and not through `scan.Scanner`
//
// 001's closing audit found twice that *a criterion written about a request is
// not met by a test about the mechanism that serves it, however good that test
// is*, and the way to tell was to break the wiring. 003 has no request, but it
// has the same class of wiring, and 003 plan §8.1 names the two seams: between
// `library.Resolve` and what `ApplyScanBatch` writes, and between `Reconcile`
// and which library a removal lands on. A build whose resolver is right and
// whose store call is wrong is green in `internal/library` and in
// `internal/scan`, and it is red here.
//
// So every assertion below reads the **store** afterwards, not the scanner's
// return value, except where the summary is the thing under test — and there it
// reads the JSON document, because 003 plan §6.7 says parsing a human table in
// a test is how a test starts constraining prose.
//
// # What none of this proves
//
// Anything a client would receive. 003 registers no route, produces no wire
// representation, and 003 plan §8.3 lists the six claims that only become
// observable at 005. Most sharply here: the `parent_id` asserted below is the
// column, and that a client asking `/Items?parentId=` is answered with these
// children is row 3 of that table and is 005's.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/libraryfixture"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/scan"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
)

// --- Running the subcommand ---------------------------------------------------

// runScan runs one `atrium library scan` the way the binary runs it, and
// returns what it wrote to standard output.
func runScan(t *testing.T, dataDirectory string, args ...string) (string, error) {
	t.Helper()
	stdout := &strings.Builder{}
	all := append([]string{"scan", "--" + flagDataDirectory, dataDirectory}, args...)
	err := RunLibrary(context.Background(), all, noEnvironment, stdout, io.Discard)
	return stdout.String(), err
}

// mustScan runs one scan and fails the test if the subcommand refused.
func mustScan(t *testing.T, dataDirectory string, args ...string) string {
	t.Helper()
	stdout, err := runScan(t, dataDirectory, args...)
	if err != nil {
		t.Fatalf("atrium library scan %s: %v", strings.Join(args, " "), err)
	}
	return stdout
}

// --- Declaring a library ------------------------------------------------------

// declareLibrary writes a library into the installation's store.
//
// It goes through `CreateLibrary` rather than through `atrium library add`
// because that verb is 003 T14's. What matters for every assertion below is
// that the library the scan reads is the library the store holds, and that is
// true either way.
func declareLibrary(t *testing.T, dataDirectory string, lib ports.Library) ports.Library {
	t.Helper()
	store, err := openStoreAt(context.Background(), dataDirectory)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	lib.NameFolded = library.FoldName(lib.Name)
	if err := store.CreateLibrary(context.Background(), lib); err != nil {
		t.Fatalf("declaring the library %q: %v", lib.Name, err)
	}
	return lib
}

// aMoviesLibrary is a library of films at the given roots. The identifier is
// fixed rather than generated because every item identifier is derived from it
// and a digest that moved between two runs would make a failure unreadable.
func aMoviesLibrary(name string, roots ...string) ports.Library {
	return ports.Library{
		ID:             fmt.Sprintf("%032x", []byte(name + "................")[:16]),
		Name:           name,
		CollectionType: string(library.Movies),
		Roots:          roots,
	}
}

// replaceRoots is 003 T14's `atrium library roots` performed through the store,
// for the cases that need a root to move or to disappear.
func replaceRoots(t *testing.T, dataDirectory, libraryID string, roots ...string) {
	t.Helper()
	store, err := openStoreAt(context.Background(), dataDirectory)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()
	if err := store.ReplaceRoots(context.Background(), libraryID, roots); err != nil {
		t.Fatalf("replacing the roots of %s: %v", libraryID, err)
	}
}

// --- Reading what the store ended up holding ----------------------------------

// storedItems is every item the store holds for a library.
func storedItems(t *testing.T, dataDirectory, libraryID string) []ports.ScannedItem {
	t.Helper()
	store, err := sqlite.Open(context.Background(), dataDirectory)
	if err != nil {
		t.Fatalf("opening the store: %v", err)
	}
	defer store.Close()

	items, err := store.ItemsForLibrary(context.Background(), libraryID)
	if err != nil {
		t.Fatalf("reading the items of %s: %v", libraryID, err)
	}
	return items
}

// storedIdentifiers is every identifier the store holds for a library, sorted.
func storedIdentifiers(t *testing.T, dataDirectory, libraryID string) []string {
	t.Helper()
	items := storedItems(t, dataDirectory, libraryID)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	slices.Sort(ids)
	return ids
}

// identifierAt is the identifier of the one stored item at a root-relative
// path, which is how a test names an item without writing a digest down.
func identifierAt(t *testing.T, items []ports.ScannedItem, path string) string {
	t.Helper()
	for _, item := range items {
		if item.Path == path {
			return item.ID
		}
	}
	t.Fatalf("no stored item at %q; the store holds %s", path, describe(items))
	return ""
}

func describe(items []ports.ScannedItem) string {
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "\n  %s %q at %q", item.Type, item.Name, item.Path)
	}
	return b.String()
}

func holds(ids []string, wanted string) bool { return slices.Contains(ids, wanted) }

// --- A tree of films ----------------------------------------------------------

// aTreeOfFilms writes the named films into a fresh directory under parent.
func aTreeOfFilms(t *testing.T, parent, name string, films ...string) string {
	t.Helper()
	root := filepath.Join(parent, name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("making %s: %v", root, err)
	}
	for i, film := range films {
		path := filepath.Join(root, film+".mkv")
		if err := os.WriteFile(path, []byte(strings.Repeat("x", i+1)), 0o644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}
	return root
}

// --- AC-12: a root that cannot be read fails the scan and removes nothing -----

// TestARootThatCannotBeReadFailsTheScanAndRemovesNothing is AC-12, in the two
// shapes an unreadable root arrives in.
//
// 003 plan §7 says in terms why the assertion is not on the error: *"a test that
// asserts only the error is met by a build that errors after removing"*. So it
// is on the item count and on three named identifiers, before and after — and
// the mutation that must fail it is a reconciliation moved ahead of the guard.
//
// The second shape **skips itself** when the directory turns out to be readable
// anyway. `os.Chmod` does not make a directory unreadable for `root`, and a test
// that silently passed under a root user would be a green proving nothing
// (003 plan §3).
func TestARootThatCannotBeReadFailsTheScanAndRemovesNothing(t *testing.T) {
	for _, testCase := range []struct {
		name string
		// breakRoot makes the root unreadable and returns false when the
		// case could not be constructed on this machine.
		breakRoot func(t *testing.T, root string) bool
	}{
		{
			name: "a path that does not exist",
			breakRoot: func(t *testing.T, root string) bool {
				if err := os.RemoveAll(root); err != nil {
					t.Fatalf("removing %s: %v", root, err)
				}
				return true
			},
		},
		{
			name: "a directory whose permissions were removed",
			breakRoot: func(t *testing.T, root string) bool {
				if err := os.Chmod(root, 0o000); err != nil {
					t.Fatalf("chmod %s: %v", root, err)
				}
				// So that t.TempDir's own cleanup can still remove it.
				t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

				// The skip, and the reason it is here rather than a comment:
				// chmod does not make a directory unreadable for root, and
				// under a root user this case does not exist. A test that
				// passed anyway would be a green proving nothing.
				if entries, err := os.ReadDir(root); err == nil {
					t.Skipf("this user can still read %s (%d entries) after chmod 000, "+
						"so the case does not exist here; the absent-root case above still ran",
						root, len(entries))
					return false
				}
				return true
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			data := t.TempDir()
			trees := t.TempDir()
			root := aTreeOfFilms(t, trees, "films",
				"The Matrix (1999)", "A Bridge Too Far (1977)", "Wall-E (2008)")
			lib := declareLibrary(t, data, aMoviesLibrary("Films", root))

			mustScan(t, data, "--"+flagName, "Films")

			before := storedItems(t, data, lib.ID)
			if len(before) != 4 {
				t.Fatalf("the first scan stored %d items, want 4: %s", len(before), describe(before))
			}
			// Three named identifiers, so that the assertion after the failed
			// scan is about *these* items and not about a count that could be
			// made up of different ones.
			named := []string{
				identifierAt(t, before, "The Matrix (1999).mkv"),
				identifierAt(t, before, "A Bridge Too Far (1977).mkv"),
				identifierAt(t, before, "Wall-E (2008).mkv"),
			}

			if !testCase.breakRoot(t, root) {
				return
			}

			_, err := runScan(t, data, "--"+flagName, "Films")
			if !errors.Is(err, scan.ErrUnavailableRoot) {
				t.Fatalf("scanning over an unreadable root: got %v, want an unavailable root", err)
			}
			if !strings.Contains(err.Error(), root) {
				t.Errorf("the refusal does not name the root %q: %s", root, err)
			}

			after := storedIdentifiers(t, data, lib.ID)
			if len(after) != len(before) {
				t.Errorf("the store holds %d items after the failed scan, want the %d it held before",
					len(after), len(before))
			}
			for _, id := range named {
				if !holds(after, id) {
					t.Errorf("the item %s was removed by a scan whose root could not be read", id)
				}
			}
		})
	}
}

// TestALibraryWhoseSecondRootFailsRemovesNothingAtAll is guard 3 of 003 plan
// §6.5, and it is the state a per-root reconciliation gets wrong.
//
// A reconciliation performed a root at a time would read the first root
// successfully, find every item of the *second* root missing from that reading,
// and remove them — and it would do it before it ever discovered that the
// second root could not be read. `Reconcile` takes whole sets, so the library
// whose second root failed never reaches it.
func TestALibraryWhoseSecondRootFailsRemovesNothingAtAll(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	first := aTreeOfFilms(t, trees, "first", "The Matrix (1999)", "Wall-E (2008)")
	second := aTreeOfFilms(t, trees, "second", "A Bridge Too Far (1977)", "Amélie (2001)")
	lib := declareLibrary(t, data, aMoviesLibrary("Films", first, second))

	mustScan(t, data, "--"+flagName, "Films")
	before := storedIdentifiers(t, data, lib.ID)
	if len(before) != 5 {
		t.Fatalf("the first scan stored %d items, want 5 (the library's own row and four films)", len(before))
	}

	// The control the case rests on: the *first* root still reads perfectly.
	// A test whose every root failed would pass on a build that removes
	// nothing because it read nothing.
	if err := os.RemoveAll(second); err != nil {
		t.Fatalf("removing %s: %v", second, err)
	}
	if _, err := os.ReadDir(first); err != nil {
		t.Fatalf("the first root is no longer readable, so this case is not the case it is named for: %v", err)
	}

	_, err := runScan(t, data, "--"+flagName, "Films")
	if !errors.Is(err, scan.ErrUnavailableRoot) {
		t.Fatalf("scanning a library whose second root is gone: got %v, want an unavailable root", err)
	}

	after := storedIdentifiers(t, data, lib.ID)
	if !slices.Equal(after, before) {
		t.Errorf("the store holds %d items after the failed scan, want the same %d it held before",
			len(after), len(before))
	}
}

// --- AC-16: a root that reads as holding nothing ------------------------------

// TestARootThatReadsAsHoldingNoCandidateFileRefusesAndSaysWhichRoot is AC-16 in
// both halves.
//
// A test asserting only the refusal passes on a build whose `--allow-empty-root`
// does nothing, which is what the criterion says in terms — so the second half
// asserts that the same scan, with the operator's explicit permission, proceeds
// and removes.
func TestARootThatReadsAsHoldingNoCandidateFileRefusesAndSaysWhichRoot(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films", "The Matrix (1999)", "Wall-E (2008)")
	lib := declareLibrary(t, data, aMoviesLibrary("Films", root))

	mustScan(t, data, "--"+flagName, "Films")
	before := storedItems(t, data, lib.ID)
	matrix := identifierAt(t, before, "The Matrix (1999).mkv")

	// The share that mounts as an empty directory: perfectly readable, and
	// holding nothing. This is the form 003 §3.8's *"cannot be read at all"*
	// rule does not catch, which is why the amendment exists.
	for _, film := range []string{"The Matrix (1999).mkv", "Wall-E (2008).mkv"} {
		if err := os.Remove(filepath.Join(root, film)); err != nil {
			t.Fatalf("emptying the root: %v", err)
		}
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatalf("the root is not a readable empty directory (%d entries, %v), "+
			"so this case is not the case it is named for", len(entries), err)
	}

	// Half one: it refuses, and it names the root.
	_, err := runScan(t, data, "--"+flagName, "Films")
	if !errors.Is(err, scan.ErrEmptyRoot) {
		t.Fatalf("scanning over a root that reads as empty: got %v, want ErrEmptyRoot", err)
	}
	if !strings.Contains(err.Error(), root) {
		t.Errorf("the refusal does not name the root %q: %s", root, err)
	}
	if refused := storedIdentifiers(t, data, lib.ID); !holds(refused, matrix) {
		t.Fatalf("the refused scan removed items: %d left, want the 3 the first scan stored", len(refused))
	}

	// Half two: with the operator's explicit permission it proceeds, and it
	// removes. Without this the test above is met by a build whose flag does
	// nothing at all.
	mustScan(t, data, "--"+flagName, "Films", "--"+flagAllowEmptyRoot)

	after := storedItems(t, data, lib.ID)
	if len(after) != 1 || after[0].Type != string(library.KindCollectionFolder) {
		t.Fatalf("after the permitted scan the store holds %s, want only the library's own row", describe(after))
	}

	// And the guard does not then refuse for ever. The library still holds its
	// own `CollectionFolder` row, which has no file behind it, so a guard that
	// counted *items* rather than *files* would read *"the last scan recorded
	// one"* and refuse every scan of this root from now on — with
	// `--allow-empty-root` as the only way to scan a library an operator
	// deliberately emptied. The count is of files for exactly this.
	if _, err := runScan(t, data, "--"+flagName, "Films"); err != nil {
		t.Errorf("scanning the emptied library again: %v", err)
	}
}

// TestOneLibraryWhoseRootFailsDoesNotStopTheNext is 003 plan §6.5's and §7's
// scoping made observable: a failed reading fails **that library**.
//
// A loop that gave up on the first failure would let an unmounted share on one
// library leave every library after it unscanned, with nothing in the summary
// saying so — which is the quiet half of the same destructive failure the
// guards exist for.
func TestOneLibraryWhoseRootFailsDoesNotStopTheNext(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	broken := aTreeOfFilms(t, trees, "broken", "The Matrix (1999)")
	healthy := aTreeOfFilms(t, trees, "healthy", "Wall-E (2008)", "Amélie (2001)")

	// "Broken" folds before "Healthy", and `Libraries` is ordered on the
	// folded name — so the failing library really is scanned first and this
	// case is the case it is named for.
	first := declareLibrary(t, data, aMoviesLibrary("Broken", broken))
	second := declareLibrary(t, data, aMoviesLibrary("Healthy", healthy))
	if err := os.RemoveAll(broken); err != nil {
		t.Fatalf("removing %s: %v", broken, err)
	}

	stdout, err := runScan(t, data, "--"+flagFormat, formatJSON)
	if !errors.Is(err, scan.ErrUnavailableRoot) {
		t.Fatalf("scanning both libraries: got %v, want the broken one's refusal", err)
	}
	if !strings.Contains(err.Error(), "Broken") {
		t.Errorf("the refusal does not name the library that failed: %s", err)
	}

	if items := storedIdentifiers(t, data, first.ID); len(items) != 0 {
		t.Errorf("the failed library stored %d items", len(items))
	}
	if items := storedIdentifiers(t, data, second.ID); len(items) != 3 {
		t.Errorf("the healthy library stored %d items, want 3 — it was not scanned", len(items))
	}

	// And it is in the summary, so an operator reading the output sees what
	// did happen beside the failure that did not.
	if !strings.Contains(stdout, `"name":"Healthy"`) || strings.Contains(stdout, `"name":"Broken"`) {
		t.Errorf("the summary is %s, want the healthy library and not the broken one", stdout)
	}
}

func TestPermissionToScanOverAnEmptyRootIsNotPermissionToScanOverAnUnreadableOne(t *testing.T) {
	// The two guards answer two different failures and only one of them has an
	// override. *"I emptied this directory"* is not *"I unmounted this share"*,
	// and a flag that disabled both would turn the criterion into a way of
	// deleting a library by mistyping a path.
	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films", "The Matrix (1999)")
	lib := declareLibrary(t, data, aMoviesLibrary("Films", root))

	mustScan(t, data, "--"+flagName, "Films")
	before := storedIdentifiers(t, data, lib.ID)

	if err := os.RemoveAll(root); err != nil {
		t.Fatalf("removing %s: %v", root, err)
	}

	_, err := runScan(t, data, "--"+flagName, "Films", "--"+flagAllowEmptyRoot)
	if !errors.Is(err, scan.ErrUnavailableRoot) {
		t.Fatalf("scanning an absent root with --%s: got %v, want an unavailable root",
			flagAllowEmptyRoot, err)
	}
	if after := storedIdentifiers(t, data, lib.ID); !slices.Equal(after, before) {
		t.Errorf("the store holds %d items, want the %d it held before", len(after), len(before))
	}
}

// --- The two seams 003 plan §8.1 asks for -------------------------------------

// TestTheParentTheStoreEndsUpHoldingIsTheParentTheResolverDerived is the first
// seam: a build whose `Resolve` is right and whose `ApplyScanBatch` writes the
// wrong `parent_id` goes red here and nowhere else.
//
// It is asserted over the fixture's declared parent-child structure rather than
// over one hand-picked chain, because the failure it is written against is a
// column filled from the wrong value — which is invisible wherever a parent
// happens to be the library's own row.
func TestTheParentTheStoreEndsUpHoldingIsTheParentTheResolverDerived(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	if err := libraryfixture.Build(trees); err != nil {
		t.Fatalf("building the fixture tree: %v", err)
	}

	declared := map[string]ports.Library{}
	for _, fixture := range libraryfixture.Libraries() {
		lib := declareLibrary(t, data, ports.Library{
			ID:             fmt.Sprintf("%032x", []byte(fixture.Name + "................")[:16]),
			Name:           fixture.Name,
			CollectionType: fixture.CollectionType,
			Roots:          []string{filepath.Join(trees, fixture.Name)},
		})
		declared[fixture.Name] = lib
	}

	mustScan(t, data)

	// The expected set names each item's parent by *path*, and the library's
	// own row by a sentinel. So the assertion is: for every declared item, the
	// `parent_id` the store holds is the `id` the store holds for the item the
	// declaration names as its parent.
	checked := 0
	for name, lib := range declared {
		items := storedItems(t, data, lib.ID)
		byPath := map[string]ports.ScannedItem{}
		var root ports.ScannedItem
		for _, item := range items {
			if item.Type == string(library.KindCollectionFolder) {
				root = item
				continue
			}
			byPath[item.Path] = item
		}
		if root.ID == "" {
			t.Fatalf("%s: the store holds no library row: %s", name, describe(items))
		}

		for _, expected := range libraryfixture.ExpectedItems() {
			if expected.Library != name || expected.Path == "" {
				continue
			}
			item, ok := byPath[expected.Path]
			if !ok {
				t.Errorf("%s: nothing stored at %q: %s", name, expected.Path, describe(items))
				continue
			}

			want := root.ID
			if expected.Parent != libraryfixture.LibraryRoot {
				parent, ok := byPath[expected.Parent]
				if !ok {
					t.Errorf("%s: nothing stored at %q, which %q names as its parent",
						name, expected.Parent, expected.Path)
					continue
				}
				want = parent.ID
			}
			if item.ParentID != want {
				t.Errorf("%s: the stored parent of %q is %s, want %s (the row at %q)",
					name, expected.Path, item.ParentID, want, expected.Parent)
			}
			checked++
		}
	}

	// The control: a loop that matched nothing would report no failures.
	if checked < 40 {
		t.Fatalf("only %d parent-child pairs were checked; the declaration holds more than that", checked)
	}
}

// TestARemovalLandsOnTheLibraryItWasComputedFor is the second seam: a build
// whose `Reconcile` is right and whose removal is applied to the wrong library
// goes red here.
//
// The failure it is written against is the ordinary one — a loop over libraries
// that reuses one library's reading for the next — and the reason it needs two
// libraries with items is that with one, every wrong answer is also the right
// one.
func TestARemovalLandsOnTheLibraryItWasComputedFor(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	firstRoot := aTreeOfFilms(t, trees, "first", "The Matrix (1999)", "Wall-E (2008)")
	secondRoot := aTreeOfFilms(t, trees, "second", "A Bridge Too Far (1977)", "Amélie (2001)")

	first := declareLibrary(t, data, aMoviesLibrary("First", firstRoot))
	second := declareLibrary(t, data, aMoviesLibrary("Second", secondRoot))

	mustScan(t, data)
	firstBefore := storedItems(t, data, first.ID)
	secondBefore := storedIdentifiers(t, data, second.ID)
	if len(firstBefore) != 3 || len(secondBefore) != 3 {
		t.Fatalf("the first scan stored %d and %d items, want 3 each", len(firstBefore), len(secondBefore))
	}
	matrix := identifierAt(t, firstBefore, "The Matrix (1999).mkv")

	// One file leaves the first library. Every scan below is over both.
	if err := os.Remove(filepath.Join(firstRoot, "The Matrix (1999).mkv")); err != nil {
		t.Fatalf("removing the film: %v", err)
	}
	mustScan(t, data)

	firstAfter := storedIdentifiers(t, data, first.ID)
	if holds(firstAfter, matrix) {
		t.Errorf("the deleted film is still stored under the library it was in")
	}
	if len(firstAfter) != 2 {
		t.Errorf("the first library holds %d items, want 2", len(firstAfter))
	}

	// And the whole point: the other library is untouched, identifier for
	// identifier. A count would pass on a build that removed one row and added
	// another.
	if secondAfter := storedIdentifiers(t, data, second.ID); !slices.Equal(secondAfter, secondBefore) {
		t.Errorf("the second library holds %v, want the %v it held before", secondAfter, secondBefore)
	}
}

// --- The summary --------------------------------------------------------------

// TestTheSummaryCountsSkippedFilesAndUnplaceableItemsApart is 003 §3.8's
// closing paragraph: *"the last two are counted apart"*.
//
// The two numbers are both non-zero over the fixture's `Shows` library and the
// examined count differs from both, so a build that added any pair of them
// together is red. A test in which one of the two happened to be zero would
// pass on exactly that build.
func TestTheSummaryCountsSkippedFilesAndUnplaceableItemsApart(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	if err := libraryfixture.Build(trees); err != nil {
		t.Fatalf("building the fixture tree: %v", err)
	}
	declareLibrary(t, data, ports.Library{
		ID:             "aaaaaaaabbbbbbbbccccccccdddddddd",
		Name:           "Shows",
		CollectionType: "tvshows",
		Roots:          []string{filepath.Join(trees, "Shows")},
	})

	stdout := mustScan(t, data, "--"+flagFormat, formatJSON)

	var document struct {
		Libraries []struct {
			Name        string   `json:"name"`
			Added       []string `json:"added"`
			Updated     []string `json:"updated"`
			Removed     []string `json:"removed"`
			Examined    int      `json:"examined"`
			Skipped     int      `json:"skipped"`
			Unplaceable int      `json:"unplaceable"`
		} `json:"libraries"`
	}
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("parsing the summary %q: %v", stdout, err)
	}
	if len(document.Libraries) != 1 {
		t.Fatalf("the summary names %d libraries, want 1: %s", len(document.Libraries), stdout)
	}
	summary := document.Libraries[0]

	// The fixture's `Shows` library: nine candidate files, one path refused
	// for its extension (`Not An Episode.mka`), and one file whose name says
	// too little to place it (`blob.mkv`). The unplaceable file **is** in the
	// library — it is examined, and it has an item — which is exactly why
	// adding it to the skip count is the mistake §3.8 is written to prevent.
	if summary.Examined != 9 {
		t.Errorf("examined %d, want 9", summary.Examined)
	}
	if summary.Skipped != 1 {
		t.Errorf("skipped %d, want 1", summary.Skipped)
	}
	if summary.Unplaceable != 1 {
		t.Errorf("unplaceable %d, want 1", summary.Unplaceable)
	}
	// The controls, stated rather than assumed: both are non-zero, so neither
	// assertion above is satisfied by a zero, and the sum is not the examined
	// count either.
	if summary.Skipped == 0 || summary.Unplaceable == 0 {
		t.Fatal("one of the two counts is zero over this fixture, so this test cannot see a build that adds them together")
	}
	if summary.Examined == summary.Skipped+summary.Unplaceable {
		t.Fatal("examined equals skipped plus unplaceable over this fixture, so the corpus cannot discriminate")
	}

	// And the item the unplaceable count is about really is stored, which is
	// the difference between the two numbers made concrete.
	items := storedItems(t, data, "aaaaaaaabbbbbbbbccccccccdddddddd")
	unplaceable := 0
	for _, item := range items {
		if item.Unplaceable {
			unplaceable++
		}
	}
	if unplaceable != summary.Unplaceable {
		t.Errorf("the summary reports %d unplaceable items and the store holds %d",
			summary.Unplaceable, unplaceable)
	}
}

func TestTheSummaryIsWrittenAsATableUnlessJSONWasAsked(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films", "The Matrix (1999)")
	declareLibrary(t, data, aMoviesLibrary("Films", root))

	table := mustScan(t, data)
	if !strings.Contains(table, "UNPLACEABLE") || !strings.Contains(table, "Films") {
		t.Errorf("the default summary is not the operator's table: %q", table)
	}
	if strings.HasPrefix(strings.TrimSpace(table), "{") {
		t.Errorf("the default summary is a document: %q", table)
	}

	if _, err := runScan(t, data, "--"+flagFormat, "yaml"); err == nil {
		t.Error("--format yaml was accepted")
	}
}

// --- A tree the domain does not object to and the store does ------------------

// TestATreeThatDerivesOneIdentifierForTwoFilesFailsTheLibrarysScanAndRemovesNothing
// is 003 T3's finding met by 003 T12's decision, at the only level either is
// visible.
//
// NFC has singleton mappings: U+212A KELVIN SIGN normalises to `K`, so a
// filesystem can hold two byte-different filenames that derive **one**
// identifier — and nothing in `internal/library` can notice, because the
// derivation is a pure function of the key. `ApplyScanBatch` refuses the batch,
// and what this asserts is the consequence for the scan: the library's scan
// fails, the message names the identifier, and **nothing is removed**, because
// the removal is after every batch.
//
// It skips itself where the filesystem folds the two names into one file, which
// is what a normalising volume does — the case does not exist there.
func TestATreeThatDerivesOneIdentifierForTwoFilesFailsTheLibrarysScanAndRemovesNothing(t *testing.T) {
	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films", "The Matrix (1999)", "Wall-E (2008)")
	lib := declareLibrary(t, data, aMoviesLibrary("Films", root))

	mustScan(t, data, "--"+flagName, "Films")
	before := storedIdentifiers(t, data, lib.ID)

	// A film whose name carries the Kelvin sign, and the same name spelled
	// with an ordinary K. Two files on disk, one derived identifier.
	const kelvin = "K"
	pair := []string{"Kelvin (2001)" + kelvin + ".mkv", "Kelvin (2001)K.mkv"}
	for i, name := range pair {
		if err := os.WriteFile(filepath.Join(root, name), []byte(strings.Repeat("y", i+2)), 0o644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading %s: %v", root, err)
	}
	if len(entries) != 4 {
		t.Skipf("this filesystem folded the two spellings into one file (%d entries, want 4), "+
			"so the case does not exist here", len(entries))
	}

	// One more file removed, so that the scan has a removal to *not* perform.
	// Without it the assertion below is an assertion about an empty list.
	if err := os.Remove(filepath.Join(root, "Wall-E (2008).mkv")); err != nil {
		t.Fatalf("removing the film: %v", err)
	}

	_, err = runScan(t, data, "--"+flagName, "Films")
	if !errors.Is(err, sqlite.ErrRepeatedIdentifier) {
		t.Fatalf("scanning a tree that derives one identifier twice: got %v, want ErrRepeatedIdentifier", err)
	}

	after := storedIdentifiers(t, data, lib.ID)
	for _, id := range before {
		if !holds(after, id) {
			t.Errorf("the item %s was removed by a scan whose batch failed", id)
		}
	}
}

// --- The subcommand itself ----------------------------------------------------

func TestLibraryRefusesAVerbItDoesNotHave(t *testing.T) {
	stderr := &strings.Builder{}
	if err := RunLibrary(context.Background(), []string{"delete"}, noEnvironment, io.Discard, stderr); err == nil {
		t.Fatal("`atrium library delete` was accepted")
	}
	if err := RunLibrary(context.Background(), nil, noEnvironment, io.Discard, stderr); err == nil {
		t.Fatal("`atrium library` with no verb was accepted")
	}
	if !strings.Contains(stderr.String(), "atrium "+LibraryCommand) {
		t.Errorf("no usage text was written: %q", stderr)
	}
}

func TestScanningANameNoLibraryHasIsARefusalAndNotAnEmptyRun(t *testing.T) {
	// An operator who mistyped a library's name and was told *"0 libraries
	// scanned"* would read it as *"nothing changed"*, which is the same
	// sentence a successful scan of an unchanged tree produces.
	data := t.TempDir()
	trees := t.TempDir()
	declareLibrary(t, data, aMoviesLibrary("Films", aTreeOfFilms(t, trees, "films", "The Matrix (1999)")))

	if _, err := runScan(t, data, "--"+flagName, "Flims"); err == nil {
		t.Fatal("a name no library has was accepted")
	}
	// And the name is matched with the domain's fold, not by bytes: two
	// library names differing only in case are one name (003 §3.6).
	if _, err := runScan(t, data, "--"+flagName, "FILMS"); err != nil {
		t.Errorf("scanning FILMS: %v", err)
	}
}
