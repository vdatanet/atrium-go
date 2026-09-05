package app

// `atrium library`'s five configuring verbs, asserted at the `app` level of 003
// tasks' three: through the subcommand, over a real temporary data directory,
// a real SQLite store and a real tree.
//
// # Why the assertions are about identifiers and not about rows changing
//
// 003 §3.6 makes a library's identity **allocated** and three of its fields
// frozen, and the observable consequence is a pair of inequalities rather than
// a pair of successes: `rename` and `roots` are free and leave every identifier
// under the library alone, while `remove` followed by `add` with the same name
// and the same roots is a **different** library whose every item derives a new
// identifier. A test that asserted only that each verb *worked* would pass on a
// build that had those two the wrong way round, which is the one mistake in this
// section an operator cannot undo.
//
// # What none of this proves
//
// Anything a client would receive. 003 registers no route, and the identifier
// compared below is the column (003 plan §8.3, row 1).

import (
	"context"
	"encoding/json"
	"flag"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/ports"
)

// --- Running the verbs --------------------------------------------------------

// runLibrary runs one `atrium library <verb>` the way the binary runs it.
func runLibrary(t *testing.T, args ...string) (string, error) {
	t.Helper()
	stdout := &strings.Builder{}
	err := RunLibrary(context.Background(), args, noEnvironment, stdout, io.Discard)
	return stdout.String(), err
}

// mustRunLibrary runs one verb and fails the test if the subcommand refused.
func mustRunLibrary(t *testing.T, args ...string) string {
	t.Helper()
	stdout, err := runLibrary(t, args...)
	if err != nil {
		t.Fatalf("atrium library %s: %v", strings.Join(args, " "), err)
	}
	return stdout
}

// addLibrary declares a library through the subcommand and answers what `list`
// then says about it, which is the round trip every test below starts with.
func addLibrary(t *testing.T, data, name, collectionType string, roots ...string) libraryReport {
	t.Helper()
	args := []string{libraryAdd, "--" + flagDataDirectory, data,
		"--" + flagName, name, "--" + flagCollectionType, collectionType}
	for _, root := range roots {
		args = append(args, "--"+flagRoot, root)
	}
	mustRunLibrary(t, args...)

	for _, report := range listedLibraries(t, data) {
		if report.Name == name {
			return report
		}
	}
	t.Fatalf("the library %q is not in the list after declaring it", name)
	return libraryReport{}
}

// listedLibraries is `atrium library list --format json`, parsed.
//
// **The document is what a test reads and the table is what an operator reads.**
// 003 plan §6.7 makes that split deliberate: parsing the human table in a test
// is how a test starts constraining prose, and the prose is the half that is
// meant to change when somebody finds it unreadable.
func listedLibraries(t *testing.T, data string) []libraryReport {
	t.Helper()
	stdout := mustRunLibrary(t, libraryList, "--"+flagDataDirectory, data, "--"+flagFormat, formatJSON)

	var document struct {
		Libraries []libraryReport `json:"libraries"`
	}
	if err := json.Unmarshal([]byte(stdout), &document); err != nil {
		t.Fatalf("`library list --format json` did not write a document: %v\n%s", err, stdout)
	}
	return document.Libraries
}

// --- add, and the round trip through list -------------------------------------

// The first half of 003 T14's definition of done: `add` then `list` reads the
// library back with its roots **in the order given**.
//
// The order is the assertion and not a detail. `ports.ScannedItem.RootOrdinal`
// indexes this list, so two roots that came back swapped would leave every
// item's recorded ordinal naming the root its path is *not* relative to — and
// the roots are therefore chosen so that the order given is neither
// alphabetical nor the order the filesystem would hand them back in.
func TestAddThenListReadsTheLibraryBackWithItsRootsInTheOrderGiven(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	trees := t.TempDir()
	zulu := aTreeOfFilms(t, trees, "zulu", "Zulu (1964)")
	alpha := aTreeOfFilms(t, trees, "alpha", "Alphaville (1965)")
	mike := aTreeOfFilms(t, trees, "mike", "Mikey and Nicky (1976)")

	report := addLibrary(t, data, "Films", string(library.Movies), zulu, alpha, mike)

	if want := []string{zulu, alpha, mike}; !slices.Equal(report.Roots, want) {
		t.Errorf("roots = %v, want %v — in the order given, which is what a root ordinal indexes",
			report.Roots, want)
	}
	if report.CollectionType != string(library.Movies) {
		t.Errorf("collectionType = %q, want %q", report.CollectionType, library.Movies)
	}
	if report.CaseSensitive {
		t.Error("caseSensitive = true, want false: it defaults to comparing without regard to case")
	}
	if len(report.ID) != 32 {
		t.Errorf("id = %q, want 32 characters (behaviours §1.4)", report.ID)
	}
}

// A root is made absolute and a root that is not a usable directory is refused
// where it was typed (003 plan §7's first row).
func TestAddRefusesARootThatIsNotAReadableDirectory(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films", "The Matrix (1999)")

	file := filepath.Join(trees, "not-a-directory")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("writing %s: %v", file, err)
	}

	for _, testCase := range []struct{ name, root string }{
		{"a path that does not exist", filepath.Join(trees, "absent")},
		{"a path that is a file", file},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
				"--"+flagName, "Films", "--"+flagCollectionType, string(library.Movies),
				"--"+flagRoot, testCase.root)
			if err == nil {
				t.Fatalf("--root %q was accepted", testCase.root)
			}
			if !strings.Contains(err.Error(), testCase.root) {
				t.Errorf("the refusal does not name the path: %v", err)
			}
		})
	}

	// The control: the same command with a real directory is accepted, so the
	// two refusals above are about the root and not about the command.
	mustRunLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
		"--"+flagName, "Films", "--"+flagCollectionType, string(library.Movies),
		"--"+flagRoot, root)

	// And a library with no root at all is refused rather than declared empty:
	// a library nothing can be scanned from is not a library.
	if _, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
		"--"+flagName, "Rootless", "--"+flagCollectionType, string(library.Movies)); err == nil {
		t.Error("a library with no --root was accepted")
	}
}

// `add` refuses a fourth collection type **listing the three**, because an
// operator with a shell is this feature's whole failure audience (003 plan §7).
//
// The three are read out of the domain rather than written here, so this fails
// if the message ever stops enumerating what `library.AllCollectionTypes`
// declares — and the second half declares each of the three actually usable,
// without which "it is one of these three" could be a message beside a command
// that accepts one of them.
func TestAddRefusesAFourthCollectionTypeListingTheThree(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films", "The Matrix (1999)")

	_, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
		"--"+flagName, "Books", "--"+flagCollectionType, "books", "--"+flagRoot, root)
	if err == nil {
		t.Fatal("--type books was accepted")
	}
	for _, name := range collectionTypeNames() {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the refusal does not list %q: %v", name, err)
		}
	}
	// The reference's own spelling of the second type is `tvshows`; a build
	// that listed the Go identifiers would satisfy the loop above only by
	// accident, so the one an operator is most likely to guess wrong is named.
	if !strings.Contains(err.Error(), "tvshows") {
		t.Errorf("the refusal does not name tvshows, which is the spelling an operator has to "+
			"get right: %v", err)
	}

	for _, collectionType := range library.AllCollectionTypes() {
		name := "Library of " + string(collectionType)
		if _, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
			"--"+flagName, name, "--"+flagCollectionType, string(collectionType),
			"--"+flagRoot, root); err != nil {
			t.Errorf("--type %s was refused, and it is one of the three the message lists: %v",
				collectionType, err)
		}
	}
}

// `add` refuses a folded name that exists, **naming the library that holds it**.
//
// The name in the message is the point: an operator told only that a name is
// taken, in an installation whose libraries they cannot see, has to go and look
// — and the library holding it may be spelled nothing like what they typed,
// because the rule is a fold and not an equality.
func TestAddRefusesAFoldedNameThatExistsNamingTheLibraryThatHoldsIt(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films", "The Matrix (1999)")
	addLibrary(t, data, "Films", string(library.Movies), root)

	_, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
		"--"+flagName, "FILMS", "--"+flagCollectionType, string(library.Movies),
		"--"+flagRoot, root)
	if err == nil {
		t.Fatal("FILMS was declared beside Films: two names differing only in case are one name")
	}
	if !strings.Contains(err.Error(), "Films") {
		t.Errorf("the refusal does not name the library that holds it: %v", err)
	}

	// And nothing was written: a refusal that had created the row and then
	// complained would leave two libraries and a message saying there is one.
	if libraries := listedLibraries(t, data); len(libraries) != 1 {
		t.Errorf("the installation holds %d libraries after the refusal, want 1", len(libraries))
	}
}

// The decision 003 T10 could not take and this task owns: **what folds a
// library's name**.
//
// `library.FoldName` normalises the Unicode form before it lowercases, which is
// `Normalise`'s own order for a path (003 §3.6: *"normalised means the same
// thing for a path and for a name"*). The consequence is here rather than in a
// comment: `Amélie` written precomposed and written as `Amelie` plus a combining
// acute are **one** library name, so the second is refused and `--name` finds
// the first whichever way it is typed.
//
// A fold that only lowercased would let both be declared, and the operator would
// then have two libraries they cannot tell apart and one of which they cannot
// address. The store is not what refuses it: `name_folded` is UNIQUE over the
// **stored bytes**, so the column refuses exactly the pairs this fold folds
// together and no others (003 T10).
func TestTwoUnicodeSpellingsOfOneLibraryNameAreOneName(t *testing.T) {
	t.Parallel()

	const precomposed = "Amélie"      // U+00E9
	const decomposed = "Ame\u0301lie" // e + U+0301 COMBINING ACUTE ACCENT
	if precomposed == decomposed {
		t.Fatal("the two spellings are the same string, so this test asserts nothing")
	}

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films", "The Matrix (1999)")
	addLibrary(t, data, precomposed, string(library.Movies), root)

	if _, err := runLibrary(t, libraryAdd, "--"+flagDataDirectory, data,
		"--"+flagName, decomposed, "--"+flagCollectionType, string(library.Movies),
		"--"+flagRoot, root); err == nil {
		t.Error("a second spelling of one name was declared as a second library")
	}

	// The half that makes the fold useful rather than merely strict: the
	// library is addressable by either spelling, and by either case.
	for _, spelling := range []string{precomposed, decomposed, strings.ToUpper(precomposed)} {
		if _, err := runLibrary(t, libraryRename, "--"+flagDataDirectory, data,
			"--"+flagName, spelling, "--"+flagTo, precomposed); err != nil {
			t.Errorf("--name %q found no library: %v", spelling, err)
		}
	}
}

// --- The refusal that is an absence -------------------------------------------

// **No verb but `add` offers a way to write a library's collection type or its
// case sensitivity**, asserted over the parsed flag sets rather than by reading
// this repository's source.
//
// # This is 003 §3.6's refusal, not a precaution beside it
//
// The section says an attempt to change either *"is refused, not accepted with a
// warning"*, and the way this design refuses it is that **there is nothing to
// type**. So the absence is the implementation, and the assertion has to be over
// what a flag set declares: a test that grepped this package for a string would
// pass on a build that offered the same setting under another name.
//
// # It does not duplicate the one in internal/ports, and neither implies the
// # other
//
// `TestNoMethodOfTheLibraryStoreCanCarryAFrozenColumn` asserts that no *store*
// method can carry one, which is what stops a `SetCaseSensitive` from being
// written at all. It says nothing about the command line: a `rename` verb that
// took `--type` and implemented it as a remove followed by an add would satisfy
// every assertion in that file, and would silently give every item in the
// library a new identifier. This is the operator-facing half; that one is the
// store-facing half.
//
// The sweep is deliberately annoying. A new flag whose name contains `type`,
// `collection`, `case` or `sensitive` fails here, and updating the list is the
// correct fix only *after* somebody has answered whether the flag writes a
// frozen column.
func TestNoVerbButAddCanWriteAFrozenColumn(t *testing.T) {
	t.Parallel()

	// Every spelling of the two the plan and the task list use between them —
	// the flag as it is declared (`--type`), the column it writes
	// (`collection_type`), and the name 003 T14's own definition of done gives
	// it (`--collection-type`).
	frozen := []string{
		"type", "collection-type", "collection_type", "collectiontype",
		"case-sensitive", "case_sensitive", "casesensitive",
	}
	// A flag that merely mentions one of these is refused too, because the
	// failure this guards against is a setting that arrived under another name.
	suspicious := []string{"type", "collection", "case", "sensitive"}

	verbs := libraryVerbSet(io.Discard)
	if len(verbs) != 6 {
		t.Fatalf("the flag table holds %d verbs, want the six of 003 plan §6.7", len(verbs))
	}

	for _, verb := range verbs {
		if verb.Verb == libraryAdd {
			continue
		}
		for _, name := range frozen {
			if verb.Flags.Lookup(name) != nil {
				t.Errorf("`library %s` declares --%s: 003 §3.6 freezes a library's collection "+
					"type and its case sensitivity at creation, and refuses a change by there "+
					"being nothing to type", verb.Verb, name)
			}
		}
		verb.Flags.VisitAll(func(f *flag.Flag) {
			for _, fragment := range suspicious {
				if strings.Contains(f.Name, fragment) {
					t.Errorf("`library %s` declares --%s, whose name mentions %q. If it writes a "+
						"frozen column it is 003 §3.6's refusal being undone; if it does not, say "+
						"so here", verb.Verb, f.Name, fragment)
				}
			}
		})
	}

	// The half without which the whole test is vacuous: `add` **does** offer
	// both, so the absences above are an absence from five verbs rather than
	// from the program.
	add, _ := newLibraryAddFlags(io.Discard)
	for _, name := range []string{flagCollectionType, flagCaseSensitive} {
		if add.Lookup(name) == nil {
			t.Errorf("`library %s` does not declare --%s: a library's %s could then be set by "+
				"nothing at all, and every assertion above would hold for the wrong reason",
				libraryAdd, name, name)
		}
	}
}

// The table the assertion above reads is the table the dispatch answers to.
//
// Without this, a seventh verb could be added to [RunLibrary] and left out of
// `libraryVerbSet`, and the frozen-column sweep would pass by never having
// looked at it.
func TestEveryVerbTheDispatchAcceptsHasARowInTheFlagTable(t *testing.T) {
	t.Parallel()

	for _, verb := range libraryVerbSet(io.Discard) {
		// `-h` reaches the verb's own flag set and stops there: a verb the
		// dispatch does not know answers "is not a subcommand" instead.
		err := RunLibrary(context.Background(), []string{verb.Verb, "-h"}, noEnvironment,
			io.Discard, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "help requested") {
			t.Errorf("`library %s -h` returned %v, want flag.ErrHelp: the table names a verb the "+
				"dispatch does not", verb.Verb, err)
		}
		if verb.Flags.Lookup(flagDataDirectory) == nil {
			t.Errorf("`library %s` does not declare --%s", verb.Verb, flagDataDirectory)
		}
	}

	err := RunLibrary(context.Background(), []string{"set", "-h"}, noEnvironment, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "is not a subcommand") {
		t.Errorf("`library set` returned %v, want a refusal: a `set` verb that quietly recreated "+
			"a library is exactly what two verbs exist to prevent (003 plan §6.7)", err)
	}
}

// --- The inequality, and the equality beside it -------------------------------

// **`remove` then `add` with the same name and the same roots is a different
// library, and every item under it has a different identifier.**
//
// This is 003 §3.6's sharpest consequence — *"editing a library is free;
// recreating one is not"* — and a test asserting merely that `remove` works
// cannot see it. It is asserted as the inequality it is, over a corpus in which
// the wrong answer is not also the right one: the tree is untouched between the
// two declarations, so the **paths** are identical and only the identifiers
// move. A build that derived a library's identity from its name would produce
// the same identifiers here and pass every other test in this file.
func TestRemoveThenAddIsADifferentLibraryAndEveryItemHasANewIdentifier(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films",
		"The Matrix (1999)", "A Bridge Too Far (1977)", "Wall-E (2008)")

	first := addLibrary(t, data, "Films", string(library.Movies), root)
	mustScan(t, data, "--"+flagName, "Films")
	before := storedItems(t, data, first.ID)
	beforeIdentifiers := storedIdentifiers(t, data, first.ID)
	if len(before) != 4 {
		t.Fatalf("the first scan stored %d items, want 4: %s", len(before), describe(before))
	}

	mustRunLibrary(t, libraryRemove, "--"+flagDataDirectory, data, "--"+flagName, "Films")

	// Removing a library removes what was scanned under it. Nothing in the
	// schema does this — `items.library_id` is a string and not a foreign key,
	// because the derived half may not reference the precious one — so a build
	// that only deleted the library row leaves every one of these rows behind
	// for ever, invisible and unreachable.
	if orphans := storedItems(t, data, first.ID); len(orphans) != 0 {
		t.Errorf("%d items outlived the library they were scanned under: %s",
			len(orphans), describe(orphans))
	}
	if libraries := listedLibraries(t, data); len(libraries) != 0 {
		t.Fatalf("`library remove` left %d libraries", len(libraries))
	}

	second := addLibrary(t, data, "Films", string(library.Movies), root)
	if second.ID == first.ID {
		t.Fatal("declaring the library again produced the same identity: it is allocated and " +
			"never derived from the name or the roots (003 §3.6), and a build that derived it " +
			"would make `remove` followed by `add` a no-op")
	}

	mustScan(t, data, "--"+flagName, "Films")
	after := storedItems(t, data, second.ID)
	afterIdentifiers := storedIdentifiers(t, data, second.ID)

	// The corpus control: the same tree, so the same items. Without this the
	// inequality below would be satisfied by a second scan that found nothing.
	if len(after) != len(before) {
		t.Fatalf("the second scan stored %d items and the first stored %d: %s",
			len(after), len(before), describe(after))
	}
	beforePaths, afterPaths := pathsOf(before), pathsOf(after)
	if !slices.Equal(beforePaths, afterPaths) {
		t.Fatalf("the two scans read different trees:\n %v\n %v", beforePaths, afterPaths)
	}

	for _, id := range beforeIdentifiers {
		if slices.Contains(afterIdentifiers, id) {
			t.Errorf("the item %s survived a remove and an add: every item under a recreated "+
				"library gets a new identifier, which is what makes recreating one expensive",
				id)
		}
	}
}

// **`rename` and `roots` leave every identifier unchanged**, which is the same
// criterion from the free side.
//
// A library's name is an input to no identifier and its roots are an input to
// none either — an item's key is its path *relative* to a root — so both verbs
// are free, and that is why they exist separately from `remove` and `add`. The
// failure this catches is a `rename` implemented as a remove followed by an add,
// which would look like it worked and would cost the operator every favourite
// and resume position in the library.
//
// The roots are replaced with the **same tree at a different absolute path**, so
// the assertion is over a change that a build keying on the root would notice.
func TestRenameAndRootsLeaveEveryIdentifierUnchanged(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	trees := t.TempDir()
	root := aTreeOfFilms(t, trees, "films",
		"The Matrix (1999)", "A Bridge Too Far (1977)", "Wall-E (2008)")

	declared := addLibrary(t, data, "Films", string(library.Movies), root)
	mustScan(t, data, "--"+flagName, "Films")
	before := storedIdentifiers(t, data, declared.ID)
	if len(before) != 4 {
		t.Fatalf("the first scan stored %d items, want 4", len(before))
	}

	mustRunLibrary(t, libraryRename, "--"+flagDataDirectory, data,
		"--"+flagName, "Films", "--"+flagTo, "Cinema")

	renamed := listedLibraries(t, data)
	if len(renamed) != 1 || renamed[0].Name != "Cinema" {
		t.Fatalf("after the rename the installation holds %v", renamed)
	}
	if renamed[0].ID != declared.ID {
		t.Errorf("the library's identity changed with its name: %s became %s — a rename that "+
			"recreated the library would give every item under it a new identifier",
			declared.ID, renamed[0].ID)
	}
	if got := storedIdentifiers(t, data, declared.ID); !slices.Equal(got, before) {
		t.Errorf("the rename changed what the store holds:\n before %v\n after  %v", before, got)
	}

	// The same tree, moved. An identifier derives from the path relative to the
	// root and never from the root, so nothing under the library may move.
	moved := filepath.Join(trees, "films-elsewhere")
	if err := os.Rename(root, moved); err != nil {
		t.Fatalf("moving %s to %s: %v", root, moved, err)
	}
	mustRunLibrary(t, libraryRoots, "--"+flagDataDirectory, data,
		"--"+flagName, "Cinema", "--"+flagRoot, moved)

	listed := listedLibraries(t, data)
	if len(listed) != 1 || !slices.Equal(listed[0].Roots, []string{moved}) {
		t.Fatalf("after replacing the roots the installation holds %v", listed)
	}
	if listed[0].ID != declared.ID {
		t.Errorf("the library's identity changed with its roots: %s became %s",
			declared.ID, listed[0].ID)
	}
	if got := storedIdentifiers(t, data, declared.ID); !slices.Equal(got, before) {
		t.Errorf("replacing the roots changed what the store holds:\n before %v\n after  %v",
			before, got)
	}

	// And the library still scans, from the new path, to the same items. This
	// is the half that a build keying an identifier on the root would fail —
	// 003 AC-10 proper is 003 T15's, which asserts it with the mutation that
	// separates it from the two criteria beside it.
	mustScan(t, data, "--"+flagName, "Cinema")
	if got := storedIdentifiers(t, data, declared.ID); !slices.Equal(got, before) {
		t.Errorf("scanning the moved tree changed every identifier:\n before %v\n after  %v",
			before, got)
	}
}

// pathsOf is the stored path of every item that has one, sorted — the corpus
// control above. A container carries none, which is why this is shorter than the
// item count.
func pathsOf(items []ports.ScannedItem) []string {
	paths := make([]string, 0, len(items))
	for _, item := range items {
		if item.Path != "" {
			paths = append(paths, item.Path)
		}
	}
	slices.Sort(paths)
	return paths
}

// --- Refusals the other verbs share -------------------------------------------

// A name no library has is a refusal for every verb, and never a run that
// changed nothing.
//
// 003 T13 asserted it for `scan`, where being told *"0 libraries scanned"* reads
// as *"nothing changed"*. The other three are worse: an operator who mistyped
// `remove` and saw nothing said would believe the library was gone.
func TestAVerbGivenANameNoLibraryHasRefuses(t *testing.T) {
	t.Parallel()

	root := aTreeOfFilms(t, t.TempDir(), "films", "The Matrix (1999)")

	for _, testCase := range []struct {
		verb  string
		extra []string
	}{
		{libraryRename, []string{"--" + flagTo, "Cinema"}},
		{libraryRoots, []string{"--" + flagRoot, root}},
		{libraryRemove, nil},
	} {
		t.Run(testCase.verb, func(t *testing.T) {
			// An installation of its own for each verb: the control below runs
			// the verb for real, and `rename` run against a shared one would
			// leave the next case looking for a library it had renamed.
			data := t.TempDir()
			addLibrary(t, data, "Films", string(library.Movies), root)

			args := append([]string{testCase.verb, "--" + flagDataDirectory, data,
				"--" + flagName, "Flims"}, testCase.extra...)
			_, err := runLibrary(t, args...)
			if err == nil {
				t.Fatalf("`library %s --name Flims` was accepted", testCase.verb)
			}
			if !strings.Contains(err.Error(), "Flims") {
				t.Errorf("the refusal does not name what was typed: %v", err)
			}

			// The control: the same command against the name that does exist,
			// folded differently, is accepted — so the refusal above is about
			// the name and not about the verb.
			args[4] = "FILMS"
			if _, err := runLibrary(t, args...); err != nil {
				t.Errorf("`library %s --name FILMS`: %v", testCase.verb, err)
			}
		})
	}
}

// `list` writes a table unless a document was asked for, and the two are
// different things rather than two spellings of one.
//
// The table is asserted to *contain* what an operator came for and is parsed
// nowhere: 003 plan §6.7 says a test that parses the human table starts
// constraining prose.
func TestTheLibraryListIsWrittenAsATableUnlessJSONWasAsked(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := aTreeOfFilms(t, t.TempDir(), "films", "The Matrix (1999)")
	declared := addLibrary(t, data, "Films", string(library.Movies), root)

	table := mustRunLibrary(t, libraryList, "--"+flagDataDirectory, data)
	if json.Valid([]byte(table)) {
		t.Errorf("the default output is a JSON document: --%s %s is what a test parses\n%s",
			flagFormat, formatJSON, table)
	}
	for _, wanted := range []string{declared.ID, "Films", string(library.Movies), root} {
		if !strings.Contains(table, wanted) {
			t.Errorf("the table does not carry %q:\n%s", wanted, table)
		}
	}

	if _, err := runLibrary(t, libraryList, "--"+flagDataDirectory, data,
		"--"+flagFormat, "yaml"); err == nil {
		t.Error("--format yaml was accepted")
	}

	// An installation with no libraries answers an empty list rather than
	// nothing, because `[]` and `null` are different answers to a parser.
	empty := mustRunLibrary(t, libraryList, "--"+flagDataDirectory, t.TempDir(),
		"--"+flagFormat, formatJSON)
	if !strings.Contains(empty, `"libraries":[]`) {
		t.Errorf("an installation with no libraries answered %q, want an empty list", empty)
	}
}
