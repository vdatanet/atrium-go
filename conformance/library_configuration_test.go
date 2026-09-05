package conformance_test

// What a package that speaks HTTP can prove about a feature that registers no
// route — and, stated in the same file, what it cannot.
//
// 003 plan §8.1 opens with the sentence this file is written under: *"Principle
// VIII wants a conformance check at the HTTP boundary asserting on bytes. 003
// has no route, so there is no boundary and there are no bytes."* What is left
// is a boundary of a different kind, the **process** boundary, where everything
// a test knows is something an operator could have known: the binary is run,
// its arguments are the ones a shell types, and its `--format json` documents
// are the ones a shell parses.
//
// # The four assertions, and why each one is here rather than in internal/app
//
//   - `library add`, `list` and `remove` end to end, and no verb but `add`
//     offering a way to write a frozen column. The operator's interface asserted
//     where an operator stands: `internal/app` calls `RunLibrary` and reads a
//     `*flag.FlagSet`, and neither of those is a command line.
//   - A scan of the built tree reporting the counts of spec §3.8. Every number
//     below is a literal in this file, not a value derived from the declaration
//     the tree was built from — plan §8.5's rule, which this package could not
//     break if it wanted to.
//   - A **second** scan of an unchanged tree reporting no changes. AC-2's second
//     half made a fact about the binary rather than about a function; the `app`
//     half is T15's and this is the one the handover left open.
//   - The L0 registration check staying green **with no new rows**, which is how
//     a feature proves it added no route (theRowsThisFeatureMayNotHaveAdded,
//     below).
//
// # What none of them can see, and the phrasing matters
//
// Not one assertion in this file can see an item's *shape*. There is no field
// name to be PascalCase, no `null` to be told apart from an absent key, no
// integer that could have been a string, no key order and no list order —
// because **003 produces no wire representation at all**. The instrument is not
// *weaker* than the HTTP boundary at catching those; it is **inapplicable**, and
// the difference is not pedantry: a weaker instrument would leave a residual
// risk somebody could argue about, where an inapplicable one has nothing yet to
// be applied to.
//
// What it *is* weaker at is everything plan §8.3 lists, and that list is the
// reason this comment exists at all. **A green run of this package is not
// evidence for any of the following**, and 005 must not read it as any:
//
//	1. The derived identifier's bytes on the wire. What is asserted here is the
//	   string the store holds, printed back by the same program that stored it.
//	2. The sort key's bytes, and therefore the order of every list. Nothing here
//	   orders anything a client can reach.
//	3. Parent-child structure as a client sees it. `?parentId=` exists nowhere.
//	4. IndexNumber, ParentIndexNumber, IndexNumberEnd and ProductionYear, and in
//	   particular their **type** on the wire — which is exactly the class
//	   behaviours §1.1 to §1.7 exist for.
//	5. A multi-part film being one item with two media sources (008).
//	6. A container with nothing under it not being offered (005 entirely).
//
// Each of the six needs an assertion at the HTTP boundary in 005's own tests,
// and 001's closing audit says how to check that such an assertion is real:
// break the wiring and watch it go red.

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The remaining verbs an operator types, beside the three
// library_subcommand_test.go already spells.
//
// Spelled rather than imported, for that file's reason and doc.go's: this
// package may import nothing of ours, so what is written here is the command
// line and not a constant that happens to agree with it.
const (
	libraryRemoveCommand = "remove"
	libraryRenameCommand = "rename"
	libraryRootsCommand  = "roots"
	libraryScanCommand   = "scan"
)

// The flags, as a shell writes them.
const (
	dataDirectoryFlag  = "--data-dir"
	nameFlag           = "--name"
	collectionTypeFlag = "--type"
	caseSensitiveFlag  = "--case-sensitive"
	rootFlag           = "--root"
	formatFlag         = "--format"
	jsonFormat         = "json"
)

// --- the fixture tree, reached as a process ------------------------------------

// fixtureBuilder is the program that writes the scanning fixture, named by the
// path `go run` takes.
//
// **This is the whole of plan §3's argument, and it is worth stating where it is
// used rather than only where it was decided.** The tree is declared once, in
// internal/libraryfixture, and this package may not import it. So it runs the
// builder as a subprocess — a subprocess is not an import, and
// tools/check_conformance_imports reads a dependency graph rather than a process
// tree. TestTheFixtureReachedThisPackageWithoutAnImport below is what turns that
// sentence from an intention into a check.
const fixtureBuilder = "./tools/build_library_fixture"

// repositoryRoot is where a `go` command has to run for a package path relative
// to the module to mean anything, and it is one directory up from this package.
const repositoryRoot = ".."

// buildFixtureTree writes the fixture into a fresh directory and hands back its
// path.
func buildFixtureTree(t *testing.T) string {
	t.Helper()

	into := filepath.Join(t.TempDir(), "trees")
	command := exec.Command("go", "run", fixtureBuilder, "-into", into)
	command.Dir = repositoryRoot
	command.Env = withoutAtriumEnvironment(os.Environ())

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go run %s -into %s: %v\n%s", fixtureBuilder, into, err, output)
	}
	return into
}

// fixtureLibrary is one of the libraries the built tree holds, as an operator
// would have to describe it to `library add`: a directory name and the
// collection type whose rules apply to it.
type fixtureLibrary struct {
	name           string
	collectionType string
}

// theFixtureLibraries is the four libraries the builder writes.
//
// A literal, because this package cannot read the declaration — and the literal
// is checked against the tree that was actually built (checkTheTreeHolds), so a
// fifth library appearing in the fixture fails this file rather than being
// quietly left unscanned by every test in it.
var theFixtureLibraries = []fixtureLibrary{
	{name: "Movies", collectionType: "movies"},
	{name: "Shows", collectionType: "tvshows"},
	{name: "Music", collectionType: "music"},
	{name: "Empty", collectionType: "movies"},
}

// checkTheTreeHolds is the control on that literal: the directories under the
// built tree are exactly the libraries named above, no more and no fewer.
func checkTheTreeHolds(t *testing.T, tree string) {
	t.Helper()

	entries, err := os.ReadDir(tree)
	if err != nil {
		t.Fatalf("reading the built tree at %s: %v", tree, err)
	}
	var built []string
	for _, entry := range entries {
		built = append(built, entry.Name())
	}
	slices.Sort(built)

	var declared []string
	for _, library := range theFixtureLibraries {
		declared = append(declared, library.name)
	}
	slices.Sort(declared)

	if !slices.Equal(built, declared) {
		t.Fatalf("the built tree holds %v and this file names %v: a library the fixture grew is a "+
			"library nothing here scans", built, declared)
	}
}

// declareTheFixture runs `atrium library add` once per library and hands back
// the data directory it declared them in.
func declareTheFixture(t *testing.T, tree string) string {
	t.Helper()

	checkTheTreeHolds(t, tree)
	data := t.TempDir()
	for _, library := range theFixtureLibraries {
		output, err := runBinary(t, "", libraryCommand, libraryAddCommand,
			dataDirectoryFlag, data,
			nameFlag, library.name,
			collectionTypeFlag, library.collectionType,
			rootFlag, filepath.Join(tree, library.name))
		if err != nil {
			t.Fatalf("atrium %s %s %s: %v\n%s", libraryCommand, libraryAddCommand, library.name, err, output)
		}
	}
	return data
}

// --- what `library list` and `library scan` print ------------------------------

// listedLibrary is one row of `library list --format json`.
type listedLibrary struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	CollectionType string   `json:"collectionType"`
	CaseSensitive  bool     `json:"caseSensitive"`
	Roots          []string `json:"roots"`
}

// listLibraries runs `library list --format json` and decodes it.
func listLibraries(t *testing.T, data string) []listedLibrary {
	t.Helper()

	output, err := runBinary(t, "", libraryCommand, libraryListCommand,
		dataDirectoryFlag, data, formatFlag, jsonFormat)
	if err != nil {
		t.Fatalf("atrium %s %s: %v\n%s", libraryCommand, libraryListCommand, err, output)
	}

	var document struct {
		Libraries []listedLibrary `json:"libraries"`
	}
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatalf("atrium %s %s printed something that is not the document it documents: %v\n%s",
			libraryCommand, libraryListCommand, err, output)
	}
	return document.Libraries
}

// scanSummary is one library's row of `library scan --format json`.
//
// The three change lists are held **raw**. A scan of an unchanged tree reports
// them as `[]`, and `[]` and `null` decode to the same nil slice — so a decoded
// summary cannot tell "nothing changed" from "the program printed nothing at
// all", which is the one distinction AC-2's second half is about. There is an
// assertion in internal/app pinning the empty list; this is the same pin one
// process boundary out.
type scanSummary struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Added       json.RawMessage `json:"added"`
	Updated     json.RawMessage `json:"updated"`
	Removed     json.RawMessage `json:"removed"`
	Examined    int             `json:"examined"`
	Skipped     int             `json:"skipped"`
	Unplaceable int             `json:"unplaceable"`
}

// identifiersIn decodes one of the three change lists.
func identifiersIn(t *testing.T, list json.RawMessage) []string {
	t.Helper()

	var identifiers []string
	if err := json.Unmarshal(list, &identifiers); err != nil {
		t.Fatalf("a change list is not a list of strings: %v\n%s", err, list)
	}
	return identifiers
}

// scanEverything runs `library scan --format json` over every library and
// returns the summaries by library name.
func scanEverything(t *testing.T, data string, extra ...string) map[string]scanSummary {
	t.Helper()

	arguments := append([]string{libraryCommand, libraryScanCommand,
		dataDirectoryFlag, data, formatFlag, jsonFormat}, extra...)
	output, err := runBinary(t, "", arguments...)
	if err != nil {
		t.Fatalf("atrium %s: %v\n%s", strings.Join(arguments, " "), err, output)
	}

	// The scan writes its log to standard error and its summary to standard
	// output, and runBinary hands back both together. The document is the last
	// line, and it is found by taking the last line rather than by matching a
	// log format — which would be this package reading a log it has no business
	// constraining.
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	document := lines[len(lines)-1]

	var summaries struct {
		Libraries []scanSummary `json:"libraries"`
	}
	if err := json.Unmarshal([]byte(document), &summaries); err != nil {
		t.Fatalf("atrium %s printed something that is not a summary document: %v\n%s",
			strings.Join(arguments, " "), err, output)
	}

	byName := map[string]scanSummary{}
	for _, summary := range summaries.Libraries {
		byName[summary.Name] = summary
	}
	if len(byName) != len(summaries.Libraries) {
		t.Fatalf("two libraries share a name in one summary document:\n%s", document)
	}
	return byName
}

// --- assertion 1: add, list and remove, and the columns with no verb -----------

// TestTheOperatorCanDeclareListAndRemoveALibraryThroughTheBinary is the first
// row of plan §8.1's table.
//
// It is one test and not three because the three verbs are only meaningful
// against one another: an `add` nothing lists is not an add, and a `remove`
// after which the library is still listed is not a remove. What each verb
// answers to is the **document** `--format json` prints, never the table — plan
// §6.7 keeps the table for an operator to read and says in terms that a test
// parsing it would start constraining prose.
//
// The last third of it is 003 §3.6's sharpest consequence, and it is observable
// here and nowhere else in this package: **`remove` followed by `add` is a
// different library.** The identity is allocated rather than derived (T14), so
// two libraries declared over the same name, the same type and the same root
// have two identifiers — which is what makes the frozen columns having no verb
// a decision an operator takes rather than an accident they discover.
func TestTheOperatorCanDeclareListAndRemoveALibraryThroughTheBinary(t *testing.T) {
	t.Parallel()

	tree := buildFixtureTree(t)
	data := declareTheFixture(t, tree)

	listed := listLibraries(t, data)
	if len(listed) != len(theFixtureLibraries) {
		t.Fatalf("declared %d libraries and the binary lists %d:\n%v",
			len(theFixtureLibraries), len(listed), listed)
	}

	byName := map[string]listedLibrary{}
	for _, library := range listed {
		byName[library.Name] = library
	}
	for _, declared := range theFixtureLibraries {
		got, found := byName[declared.name]
		if !found {
			t.Fatalf("the binary lists no library called %s:\n%v", declared.name, listed)
		}
		if got.CollectionType != declared.collectionType {
			t.Errorf("%s: collection type %q, want %q", declared.name, got.CollectionType, declared.collectionType)
		}
		if got.ID == "" {
			t.Errorf("%s carries no identifier", declared.name)
		}
		want := []string{filepath.Join(tree, declared.name)}
		if !slices.Equal(got.Roots, want) {
			t.Errorf("%s: roots %v, want %v", declared.name, got.Roots, want)
		}
	}

	// Remove one, and it is the *only* one that goes.
	removed := theFixtureLibraries[0]
	identityBefore := byName[removed.name].ID
	if output, err := runBinary(t, "", libraryCommand, libraryRemoveCommand,
		dataDirectoryFlag, data, nameFlag, removed.name); err != nil {
		t.Fatalf("atrium %s %s %s: %v\n%s", libraryCommand, libraryRemoveCommand, removed.name, err, output)
	}

	after := listLibraries(t, data)
	if len(after) != len(theFixtureLibraries)-1 {
		t.Errorf("after removing %s the binary lists %d libraries, want %d:\n%v",
			removed.name, len(after), len(theFixtureLibraries)-1, after)
	}
	for _, library := range after {
		if library.Name == removed.name {
			t.Errorf("%s was removed and is still listed", removed.name)
		}
	}

	// And declaring it again is a *new* library, not the old one back.
	if output, err := runBinary(t, "", libraryCommand, libraryAddCommand,
		dataDirectoryFlag, data,
		nameFlag, removed.name,
		collectionTypeFlag, removed.collectionType,
		rootFlag, filepath.Join(tree, removed.name)); err != nil {
		t.Fatalf("re-declaring %s: %v\n%s", removed.name, err, output)
	}
	for _, library := range listLibraries(t, data) {
		if library.Name != removed.name {
			continue
		}
		if library.ID == identityBefore {
			t.Errorf("%s was removed and declared again and kept the identifier %s: "+
				"003 §3.6 allocates a library's identity rather than deriving it, and every "+
				"item identifier under the library depends on that", removed.name, identityBefore)
		}
	}
}

// TestNoVerbButAddOffersTheOperatorAWayToChangeAFrozenColumn is 003 §3.6's
// refusal as an operator meets it.
//
// §3.6 says an attempt to change a library's collection type or its case
// sensitivity is *"refused, not accepted with a warning"*, and the way this
// design refuses it is that **there is nothing to type**. So the assertion is
// that typing it is a usage error, on every verb but the one that declares the
// library.
//
// # What this half proves and what the other half proves
//
// This one is over the command line: five verbs, two flags, ten refusals an
// operator would actually see. It does **not** prove that the five verbs below
// are all the verbs there are — this package cannot enumerate a dispatch. That
// completeness is `internal/app`'s `TestEveryVerbTheDispatchAcceptsHasARowInThe
// FlagTable` together with `TestNoVerbButAddCanWriteAFrozenColumn`, which run
// over the flag sets themselves. Neither half implies the other, and stating so
// is cheaper than a reader assuming one of them is redundant.
//
// The control against a verb quietly disappearing is built into the assertion
// rather than bolted beside it: a verb the dispatch no longer knows is refused
// with *"is not a subcommand"*, which is a different message from the one this
// test requires, so a renamed verb fails here instead of passing vacuously.
func TestNoVerbButAddOffersTheOperatorAWayToChangeAFrozenColumn(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	frozen := []struct {
		flag     string
		argument []string
	}{
		{flag: collectionTypeFlag, argument: []string{collectionTypeFlag, "movies"}},
		{flag: caseSensitiveFlag, argument: []string{caseSensitiveFlag}},
	}

	for _, verb := range []string{
		libraryListCommand,
		libraryRenameCommand,
		libraryRootsCommand,
		libraryRemoveCommand,
		libraryScanCommand,
	} {
		for _, column := range frozen {
			arguments := append([]string{libraryCommand, verb, dataDirectoryFlag, data}, column.argument...)
			output, err := runBinary(t, "", arguments...)
			if err == nil {
				t.Errorf("atrium %s accepted %s: a verb that takes a frozen column is a verb that can change one",
					strings.Join(arguments, " "), column.flag)
				continue
			}
			// Go's flag package spells a rejected flag with one hyphen
			// whatever the operator typed, which is the message an operator
			// reads and therefore the one this asserts.
			want := "flag provided but not defined: -" + strings.TrimPrefix(column.flag, "--")
			if !strings.Contains(output, want) {
				t.Errorf("atrium %s failed, and not for the reason this test is about.\nwant a line containing %q\ngot:\n%s",
					strings.Join(arguments, " "), want, output)
			}
		}
	}

	// The other half, without which this test passes on a build where nothing
	// anywhere can set either flag — an absence with the wrong cause, which is
	// the shape T14's own handover warns about.
	for _, column := range frozen {
		arguments := append([]string{libraryCommand, libraryAddCommand, dataDirectoryFlag, data,
			nameFlag, "Declared", collectionTypeFlag, "movies", rootFlag, data}, column.argument...)
		if output, err := runBinary(t, "", arguments...); err != nil {
			t.Fatalf("atrium %s refused %s, which is the verb that is supposed to declare it: %v\n%s",
				strings.Join(arguments, " "), column.flag, err, output)
		}
		// Declared once per column, so the second pass needs the first gone.
		if output, err := runBinary(t, "", libraryCommand, libraryRemoveCommand,
			dataDirectoryFlag, data, nameFlag, "Declared"); err != nil {
			t.Fatalf("removing the library this test declared: %v\n%s", err, output)
		}
	}
}

// --- assertions 2 and 3: what a scan reports, and what a second one reports ----

// countsOfTheFixture is spec §3.8's summary over the built tree, per library.
//
// **Every number is a literal read off a run of the binary and written down
// here** `[measurement: 003 T18, atrium library scan --format json over the
// built fixture, 2026-09-05]`. It is not derived from internal/libraryfixture's
// declaration, and this package could not derive it if a later reader wanted to:
// plan §8.5 forbids a count computed from the declaration it is checked against,
// and the import boundary makes that forbidding structural here rather than
// disciplinary.
//
// The four rows are also where the fixture's own shape is legible at a glance:
// `Movies` skips seven paths and places every file it examines, `Shows` has the
// one file whose name says too little to place the item it produced — counted
// apart from a skip, because §3.8 says an operator told that both were "skipped"
// would go looking for something that is not missing — and `Empty` examines
// nothing and still produces one item, its own row.
var countsOfTheFixture = map[string]struct {
	added       int
	examined    int
	skipped     int
	unplaceable int
}{
	"Movies": {added: 17, examined: 17, skipped: 7},
	"Shows":  {added: 18, examined: 9, skipped: 1, unplaceable: 1},
	"Music":  {added: 18, examined: 10},
	"Empty":  {added: 1},
}

// TestAScanOfTheBuiltTreeReportsTheCountsOfTheSpecification is the second row of
// plan §8.1's table, and its instrument is `scan --format json` through the
// binary.
//
// What it proves is stated as narrowly as it is true: **that the scan ran, and
// what it says it did.** It does not prove that any of the items it counted is
// the item the specification describes — that is `internal/library`'s and
// `internal/app`'s, over the same tree, at the layer where an item has a name, a
// type and a parent. A summary is six numbers.
func TestAScanOfTheBuiltTreeReportsTheCountsOfTheSpecification(t *testing.T) {
	t.Parallel()

	data := declareTheFixture(t, buildFixtureTree(t))
	summaries := scanEverything(t, data)

	if len(summaries) != len(countsOfTheFixture) {
		t.Fatalf("the scan summarised %d libraries and %d are declared: %v",
			len(summaries), len(countsOfTheFixture), summaries)
	}

	for name, want := range countsOfTheFixture {
		got, found := summaries[name]
		if !found {
			t.Errorf("the scan summarised no library called %s", name)
			continue
		}
		if added := len(identifiersIn(t, got.Added)); added != want.added {
			t.Errorf("%s: added %d items, want %d", name, added, want.added)
		}
		if updated := len(identifiersIn(t, got.Updated)); updated != 0 {
			t.Errorf("%s: a first scan of a library reported %d updated items, want 0", name, updated)
		}
		if removed := len(identifiersIn(t, got.Removed)); removed != 0 {
			t.Errorf("%s: a first scan of a library reported %d removed items, want 0", name, removed)
		}
		if got.Examined != want.examined {
			t.Errorf("%s: examined %d files, want %d", name, got.Examined, want.examined)
		}
		if got.Skipped != want.skipped {
			t.Errorf("%s: skipped %d paths, want %d", name, got.Skipped, want.skipped)
		}
		if got.Unplaceable != want.unplaceable {
			t.Errorf("%s: %d items were unplaceable, want %d — §3.8 counts these apart from skipped "+
				"files and adding them together is the mistake the two counts exist to prevent",
				name, got.Unplaceable, want.unplaceable)
		}
	}
}

// TestASecondScanOfAnUnchangedTreeReportsNoChanges is AC-2's second half, made a
// fact about the binary.
//
// The `app` half is T15's, over `RunLibrary`. This one is what an operator gets:
// two invocations of the same command line, and the second reporting three empty
// lists.
//
// # Why the lists are compared as bytes and not as lengths
//
// `[]` and `null` are the same nil slice once decoded, and "the program printed
// no list at all" is precisely the build this criterion has to exclude — a scan
// that reported nothing because it did nothing looks identical to one that
// reported nothing because nothing had changed. So the raw JSON is compared, and
// the file counts are asserted beside it: a second scan that examined no file
// would report no changes for the most uninteresting reason there is.
//
// # The control is against the declared counts, and that correction was measured
//
// It compared the second scan's counts against the **first scan's** when this
// test was written, which is the obvious spelling and is relative. A build that
// reported nothing examined for *every* scan satisfies it: both readings are
// zero and both are equal `[measurement: 003 T18, 18 mutations, this the only
// survivor of the first run, 2026-09-05]`. Another test in this file happened to
// catch that build, which is exactly the accident a control must not depend on.
// Comparing against countsOfTheFixture is absolute, and the equality with the
// first scan then follows from that test rather than standing in for it.
func TestASecondScanOfAnUnchangedTreeReportsNoChanges(t *testing.T) {
	t.Parallel()

	data := declareTheFixture(t, buildFixtureTree(t))
	first := scanEverything(t, data)
	second := scanEverything(t, data)

	if len(second) != len(first) {
		t.Fatalf("the first scan summarised %d libraries and the second %d", len(first), len(second))
	}

	for name, got := range second {
		before, found := first[name]
		if !found {
			t.Errorf("the second scan summarised %s and the first did not", name)
			continue
		}
		if got.ID != before.ID {
			t.Errorf("%s: the library's identifier moved between two scans, %s to %s", name, before.ID, got.ID)
		}
		for _, list := range []struct {
			what string
			raw  json.RawMessage
		}{
			{"added", got.Added},
			{"updated", got.Updated},
			{"removed", got.Removed},
		} {
			if string(list.raw) != "[]" {
				t.Errorf("%s: a second scan of an unchanged tree reported %s = %s, want the empty list []",
					name, list.what, list.raw)
			}
		}
		// The control: the second scan looked at the whole tree. Without it
		// "no changes" is met by a scan that walked nothing.
		want, declared := countsOfTheFixture[name]
		if !declared {
			t.Errorf("the scan summarised a library called %s, which is not one of the four the fixture holds", name)
			continue
		}
		if got.Examined != want.examined || got.Skipped != want.skipped || got.Unplaceable != want.unplaceable {
			t.Errorf("%s: the second scan examined %d/skipped %d/unplaceable %d where the tree holds "+
				"%d/%d/%d — the tree did not change, so a scan that saw something else "+
				"reported no changes for the wrong reason",
				name, got.Examined, got.Skipped, got.Unplaceable,
				want.examined, want.skipped, want.unplaceable)
		}
	}
}

// --- the check that makes the subprocess argument a check ----------------------

// TestTheFixtureReachedThisPackageWithoutAnImport runs the import check itself.
//
// Every test above has the fixture tree, and plan §3's claim is that having it
// costs this package no import: it runs the builder rather than calling it. That
// claim is checkable in one place and CI already checks it — but a reader of
// *this* file has no reason to believe CI does, and the sentence "a subprocess is
// not an import" is exactly the kind of thing that stays true until somebody
// adds one line. So the check runs here too, over the tree that now holds these
// tests.
//
// The positive line is asserted and not only the exit status, and that is the
// interesting half: the tool answers *"no conformance/ directory yet — nothing to
// check"* and **exits zero** when it is run from the wrong directory. A test that
// looked only at the status would be green for a run that checked nothing.
func TestTheFixtureReachedThisPackageWithoutAnImport(t *testing.T) {
	t.Parallel()

	command := exec.Command("go", "run", "./tools/check_conformance_imports")
	command.Dir = repositoryRoot
	command.Env = withoutAtriumEnvironment(os.Environ())

	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("conformance/ reaches into internal/, which architecture §3 forbids: %v\n%s", err, output)
	}
	const checked = "conformance/ imports nothing under internal/"
	if !strings.Contains(string(output), checked) {
		t.Fatalf("the import check exited zero without saying it checked anything.\nwant a line containing %q\ngot:\n%s",
			checked, output)
	}
}

// --- assertion 4: the L0 check, staying green with no new rows -----------------

// theRowsThisFeatureMayNotHaveAdded is the eleven rows of 001 and 002, written
// out.
//
// **This is how a feature proves it added no route**, and it is the reason plan
// §10 refuses the obvious escape from this feature's organising problem: the
// reference has a `POST /Library/Refresh` and giving 003 that route would give
// Principle VIII something to assert on. It would also be a delta added to make
// a test possible, and Principle VI keeps an endpoint out until a named consumer
// is measured calling it. A feature that adds no route has to be able to say so,
// and this list is the saying.
//
// # Why a literal, when both halves of the L0 check already derive their answer
//
// They derive it from surface.yaml, and derivation is what makes them silent
// about the change this row of the table is about. The registration half fails
// on a route with **no** row (TestARouteWithNoRowFailsTheRegistrationCheck
// proves it), so a route added on its own is already caught — but a route added
// *together with a row in surface.yaml* satisfies both halves by construction,
// and that is the change a feature makes when it decides to grow a route. The
// literal is what a reviewer of such a change has to edit, in a file that says
// why they should not.
//
// **Measured, not argued.** Registering a `POST /Library/Refresh` and declaring
// it in both copies of the surface document leaves
// TestTheServerIsReachableOnExactlyTheImplementedRowsOfTheSurfaceDocument and
// TestTheRouterServesExactlyTheImplementedRowsOfTheSurfaceDocument **green**,
// and turns this literal and its registration twin red
// `[measurement: 003 T18, 18 mutations, 2026-09-05]`. Three other checks fail on
// that build too — the sweep's route coverage, the response-model declaration
// and the embedded table's row count — and not one of them is a statement about
// *this feature having added a route*, which is what the row of plan §8.1's
// table asks for.
var theRowsThisFeatureMayNotHaveAdded = []string{
	"GET /System/Info",
	"GET /System/Info/Public",
	"GET /System/Ping",
	"POST /System/Ping",
	"GET /Sessions",
	"POST /Sessions/Capabilities/Full",
	"POST /Users/AuthenticateByName",
	"GET /Users/Me",
	"GET /Users/Public",
	"POST /Users/Configuration",
	"GET /Users/{userId}",
}

// rowsBeyond compares what a reading of the server found against what this
// feature is allowed to leave behind, and reports both directions.
//
// It is a function over two lists rather than an assertion inside a test, for
// checkRegistration's reason: **a check that has only ever been run against a
// correct server has proved nothing**, and the two tests below run this one
// against readings that are deliberately wrong.
//
// Both directions are findings. A row that appeared is 003 having grown a route;
// a row that went away is 001 or 002 having lost one, which is not this feature's
// business but is not something a check about "no new rows" should pass over in
// silence either.
func rowsBeyond(found, allowed []string) []string {
	inAllowed := map[string]bool{}
	for _, row := range allowed {
		inAllowed[row] = true
	}
	inFound := map[string]bool{}
	for _, row := range found {
		inFound[row] = true
	}

	var findings []string
	for _, row := range found {
		if !inAllowed[row] {
			findings = append(findings, fmt.Sprintf(
				"this server serves %s, which is not one of the eleven rows 001 and 002 registered: "+
					"a feature that adds a route says so in surface.yaml and in this list", row))
		}
	}
	for _, row := range allowed {
		if !inFound[row] {
			findings = append(findings, fmt.Sprintf(
				"%s is one of the eleven rows 001 and 002 registered and this server does not serve it", row))
		}
	}
	slices.Sort(findings)
	return findings
}

// TestThisServerServesExactlyTheElevenRowsOf001And002 is the fourth row of plan
// §8.1's table, over the wire.
//
// It reads the server the same way TestTheServerIsReachableOnExactlyThe
// ImplementedRowsOfTheSurfaceDocument does — a real request per row of the
// document, split by the empty refusal of behaviours §1.11 — and then asks a
// different question of the answer: not *"is what is served consistent with the
// document"* but *"is it still the eleven"*.
func TestThisServerServesExactlyTheElevenRowsOf001And002(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	served, _ := reachability(t, server, readSurfaceRows(t))

	var found []string
	for _, row := range served {
		found = append(found, row.method+" "+row.path)
	}

	if findings := rowsBeyond(found, theRowsThisFeatureMayNotHaveAdded); len(findings) != 0 {
		t.Errorf("this server does not serve exactly the eleven rows of 001 and 002:\n%s",
			strings.Join(findings, "\n"))
	}
}

// TestARowThisFeatureAddedFailsTheNoNewRowsCheck is half of the failure proof.
func TestARowThisFeatureAddedFailsTheNoNewRowsCheck(t *testing.T) {
	t.Parallel()

	grown := append(slices.Clone(theRowsThisFeatureMayNotHaveAdded), "POST /Library/Refresh")
	findings := rowsBeyond(grown, theRowsThisFeatureMayNotHaveAdded)
	if len(findings) != 1 {
		t.Fatalf("a twelfth row produced %d findings, want 1:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], "/Library/Refresh") {
		t.Errorf("the finding is %q and does not name the route that was added", findings[0])
	}
}

// TestARowThatWentAwayFailsTheNoNewRowsCheck is the other half.
//
// A check written only in the "nothing new" direction is met by a server that
// serves nothing at all, which is the vacuous green both halves of the L0 check
// already refuse in their own way.
func TestARowThatWentAwayFailsTheNoNewRowsCheck(t *testing.T) {
	t.Parallel()

	lost := theRowsThisFeatureMayNotHaveAdded[0]
	findings := rowsBeyond(theRowsThisFeatureMayNotHaveAdded[1:], theRowsThisFeatureMayNotHaveAdded)
	if len(findings) != 1 {
		t.Fatalf("a missing row produced %d findings, want 1:\n%s", len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], lost) {
		t.Errorf("the finding is %q and does not name %s, the row that went away", findings[0], lost)
	}
}
