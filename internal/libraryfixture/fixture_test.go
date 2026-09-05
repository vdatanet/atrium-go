// Package libraryfixture_test asserts the four properties the tree is built
// for, from outside the package: everything here is reachable through the
// declaration's own surface, which is the surface
// tools/build_library_fixture and every later task use.
package libraryfixture_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"image/jpeg"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/vdatanet/atrium-go/internal/libraryfixture"
)

// repositoryRoot is where this package sits, so a test can name the reference's
// recorded reading from the package directory the test runs in. The same
// constant, for the same reason, is in internal/surface.
const repositoryRoot = "../.."

// reading is docs/compatibility/reference-fixture-reading.json, parsed.
//
// Only the members a path can be read out of are declared. The file also
// carries the probe's citation, the image digest and the counts, and a struct
// that reproduced them would be a transcription of exactly the kind the rest of
// this file exists to avoid.
type reading struct {
	Totals struct {
		Libraries int `json:"libraries"`
		Items     int `json:"items"`
	} `json:"totals"`
	Libraries []readLibrary `json:"libraries"`
}

type readLibrary struct {
	Name           string     `json:"name"`
	CollectionType string     `json:"collection_type"`
	ItemCount      int        `json:"item_count"`
	Items          []readItem `json:"items"`
}

type readItem struct {
	Type string  `json:"type"`
	Name string  `json:"name"`
	File *string `json:"file"`
	Path *string `json:"path"`
}

func theReferencesReading(t *testing.T) reading {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(libraryfixture.ReferenceReading)))
	if err != nil {
		t.Fatalf("reading %s: %v", libraryfixture.ReferenceReading, err)
	}

	var parsed reading
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parsing %s: %v", libraryfixture.ReferenceReading, err)
	}
	if len(parsed.Libraries) == 0 {
		t.Fatalf("%s parsed to no libraries at all", libraryfixture.ReferenceReading)
	}
	return parsed
}

// build writes the tree into a directory of this test's own and returns it.
func build(t *testing.T) string {
	t.Helper()

	root := filepath.Join(t.TempDir(), "fixture")
	if err := libraryfixture.Build(root); err != nil {
		t.Fatalf("building the fixture: %v", err)
	}
	return root
}

// TestEveryPathTheReferencesReadingNamesExistsInAFreshlyBuiltTree is the check
// 003 plan §8.5 asks for, and the reason it is first.
//
// docs/compatibility/reference-fixture-reading.json was taken by mounting
// *this* tree into a container, so a fixture that drifts from it makes 003's
// comparison against that reading meaningless while leaving it green. The
// expected paths are read out of the JSON rather than transcribed into this
// file: transcribing them is how the two stop being the same tree, and the
// transcription would still pass.
//
// The direction is one-way on purpose. Every path the reading names has to
// exist here; the tree holds ten files the reading cannot name, because both
// servers drop them, and the declaration says which rule drops each.
func TestEveryPathTheReferencesReadingNamesExistsInAFreshlyBuiltTree(t *testing.T) {
	root := build(t)
	reading := theReferencesReading(t)

	built := make(map[string]libraryfixture.Library, len(libraryfixture.Libraries()))
	for _, library := range libraryfixture.Libraries() {
		built[library.Name] = library
	}

	// Scoped to the four this feature builds, and it says so: a check that
	// silently skipped a third of the libraries would read exactly like a
	// check over all six.
	var checked, skipped []string
	for _, library := range reading.Libraries {
		declared, isBuilt := built[library.Name]
		if !isBuilt {
			why, named := libraryfixture.NotBuiltHere[library.Name]
			if !named {
				t.Errorf("%s names the library %q, which this package neither builds nor accounts for in NotBuiltHere.\n"+
					"A library the reading holds and nothing here decides about is exactly the drift this test exists to catch.",
					libraryfixture.ReferenceReading, library.Name)
				continue
			}
			skipped = append(skipped, library.Name+" — "+why)
			continue
		}
		checked = append(checked, library.Name)

		if declared.CollectionType != library.CollectionType {
			t.Errorf("%s is configured %q here and was read as %q there: the reading is over a differently configured library and the comparison against it means less than it looks",
				library.Name, declared.CollectionType, library.CollectionType)
		}

		for _, item := range library.Items {
			if item.File != nil {
				assertRegularFile(t, root, library.Name, *item.File, item)
				continue
			}
			if item.Path != nil {
				assertDirectory(t, root, library.Name, *item.Path, item)
			}
			// An item with neither is a container the reference inferred —
			// "Season Unknown" is the one — and there is no path to check.
		}
	}

	if len(checked) != len(libraryfixture.LibraryNames()) {
		t.Errorf("the reading was checked against %v, and this package builds %v", checked, libraryfixture.LibraryNames())
	}
	if len(skipped) != len(libraryfixture.NotBuiltHere) {
		t.Errorf("%d libraries were skipped and NotBuiltHere names %d", len(skipped), len(libraryfixture.NotBuiltHere))
	}
	t.Logf("checked %v; skipped %v", checked, skipped)
}

func assertRegularFile(t *testing.T, root, library, path string, item readItem) {
	t.Helper()

	full := filepath.Join(root, library, filepath.FromSlash(path))
	info, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s/%s is named by the reference's reading (%s %q) and the built tree has no such file.\n"+
			"The reading was taken over this tree; a path it names and the builder does not write is drift, and drift is invisible in the comparison it feeds.",
			library, path, item.Type, item.Name)
		return
	}
	if err != nil {
		t.Errorf("%s/%s: %v", library, path, err)
		return
	}
	if !info.Mode().IsRegular() {
		t.Errorf("%s/%s is named as the file behind %s %q and is not a regular file", library, path, item.Type, item.Name)
	}
}

func assertDirectory(t *testing.T, root, library, path string, item readItem) {
	t.Helper()

	full := filepath.Join(root, library, filepath.FromSlash(path))
	info, err := os.Stat(full)
	if errors.Is(err, fs.ErrNotExist) {
		t.Errorf("%s/%s backs %s %q in the reference's reading and the built tree has no such directory",
			library, path, item.Type, item.Name)
		return
	}
	if err != nil {
		t.Errorf("%s/%s: %v", library, path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("%s/%s backs the container %s %q and is not a directory", library, path, item.Type, item.Name)
	}
}

// TestTheZeroByteFilmAndTheIgnoreMarkerMeasureZero asserts the two files whose
// emptiness *is* the rule, as lengths.
//
// Each of them exists to exercise a rule: 003 §3.2 ignores a zero-byte file
// because it is an incomplete copy, and only an empty `.ignore` marker excludes
// anything. A builder that wrote one byte into either would disable a rule
// silently and cost nothing visible.
//
// The control is the other half: every other file in the tree is asserted to
// carry bytes, so a builder that wrote nothing anywhere cannot pass this test
// by being empty everywhere.
func TestTheZeroByteFilmAndTheIgnoreMarkerMeasureZero(t *testing.T) {
	root := build(t)

	mustMeasureZero := map[string]string{
		"Movies/An Incomplete Copy (2000).mkv": "an incomplete copy, which 003 §3.2 ignores for being zero bytes",
		"Movies/Excluded/.ignore":              "the marker, of which only an empty one excludes anything (003 §3.2)",
	}

	var withBytes int
	for _, library := range libraryfixture.Libraries() {
		for _, file := range library.Files {
			path := library.Name + "/" + file.Path
			info, err := os.Stat(filepath.Join(root, library.Name, filepath.FromSlash(file.Path)))
			if err != nil {
				t.Fatalf("%s: %v", path, err)
			}

			if why, isRule := mustMeasureZero[path]; isRule {
				if info.Size() != 0 {
					t.Errorf("%s measures %d bytes and must measure 0: it is %s", path, info.Size(), why)
				}
				continue
			}
			if info.Size() == 0 {
				t.Errorf("%s measures 0 bytes and is not one of the two files whose emptiness is a rule.\n"+
					"An accidentally empty file is skipped by the scanner as an incomplete copy, which is a rule firing where nothing declared it.", path)
				continue
			}
			withBytes++
		}
	}

	if withBytes < 30 {
		t.Errorf("only %d files carry bytes: the control on this test is that most of the tree is not empty, and it is not holding", withBytes)
	}
}

// TestTwoBuildsAgreeOnPathsAndSizesAndDifferOnModificationTimes is the property
// a committed tree cannot have, and it is 003 plan §8.5's first reason for
// generating the fixture rather than checking it in: a zero-byte file survives
// a checkout and a modification time does not, while the whole of §6.4's change
// signal is one.
func TestTwoBuildsAgreeOnPathsAndSizesAndDifferOnModificationTimes(t *testing.T) {
	first := build(t)

	// Two builds a microsecond apart can share a modification time on a
	// filesystem whose timestamps are coarser than that, and the test would
	// then fail for a reason that is not about the builder. So the clock the
	// filesystem actually uses is measured, and the second build starts once
	// it has moved.
	waitForTheFilesystemsClockToMove(t)
	second := build(t)

	firstTree := walk(t, first)
	secondTree := walk(t, second)

	if len(firstTree) == 0 {
		t.Fatal("the first build wrote nothing")
	}

	firstPaths := sortedKeys(firstTree)
	secondPaths := sortedKeys(secondTree)
	if !slices.Equal(firstPaths, secondPaths) {
		t.Fatalf("two builds wrote different trees:\nfirst:  %v\nsecond: %v", firstPaths, secondPaths)
	}

	for _, path := range firstPaths {
		before, after := firstTree[path], secondTree[path]
		if before.size != after.size {
			t.Errorf("%s measures %d bytes in one build and %d in the other: a size that depends on the run is a signal that depends on the run", path, before.size, after.size)
		}
		if !after.modified.After(before.modified) {
			t.Errorf("%s carries %s in the first build and %s in the second: the two trees have to differ in modification time, which is the one thing git could not have given us",
				path, before.modified.Format(time.RFC3339Nano), after.modified.Format(time.RFC3339Nano))
		}
	}
}

type entry struct {
	size     int64
	modified time.Time
}

func walk(t *testing.T, root string) map[string]entry {
	t.Helper()

	tree := map[string]entry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		tree[filepath.ToSlash(relative)] = entry{size: info.Size(), modified: info.ModTime()}
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return tree
}

func sortedKeys(tree map[string]entry) []string {
	keys := make([]string, 0, len(tree))
	for key := range tree {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// waitForTheFilesystemsClockToMove writes the same file until its modification
// time changes, so the caller knows the timestamp a later write receives is a
// later timestamp. It measures the resolution rather than assuming one, which
// is the same reason 003 §6.4 stores a modification time that may not be a
// whole multiple of a tick.
func waitForTheFilesystemsClockToMove(t *testing.T) {
	t.Helper()

	probe := filepath.Join(t.TempDir(), "probe")
	stamp := func() time.Time {
		if err := os.WriteFile(probe, []byte("probe"), 0o644); err != nil {
			t.Fatalf("writing the timestamp probe: %v", err)
		}
		info, err := os.Stat(probe)
		if err != nil {
			t.Fatalf("reading the timestamp probe: %v", err)
		}
		return info.ModTime()
	}

	start := stamp()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if stamp().After(start) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("this filesystem's modification times did not move in five seconds, so the two builds cannot be told apart by one")
		}
		time.Sleep(time.Millisecond)
	}
}

// TestEmptyIsADirectoryThatExistsAndHoldsNothing keeps the two states apart
// that AC-12 needs kept apart: a library with nothing in it reads clean
// (behaviours §5.7), and a root that is *missing* fails the scan and removes
// nothing. A builder that expressed the first as the second would make AC-12's
// test pass over a library that was never built.
func TestEmptyIsADirectoryThatExistsAndHoldsNothing(t *testing.T) {
	root := build(t)

	// The premise, stated rather than assumed: the declaration puts nothing in
	// this library at all. So the directory in the built tree can only have
	// come from the rule that every library gets one, which is the half a
	// builder that created the directories its files imply would not have.
	for _, library := range libraryfixture.Libraries() {
		if library.Name != "Empty" {
			continue
		}
		if len(library.Files) != 0 || len(library.Dirs) != 0 {
			t.Fatalf("the Empty library declares %d files and %d directories, so this test no longer asserts what it says",
				len(library.Files), len(library.Dirs))
		}
	}

	empty := filepath.Join(root, "Empty")
	info, err := os.Stat(empty)
	if err != nil {
		t.Fatalf("the Empty library: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("the Empty library is not a directory: %s", info.Mode())
	}

	entries, err := os.ReadDir(empty)
	if err != nil {
		t.Fatalf("reading the Empty library: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("the Empty library holds %d entries: %v", len(entries), entries)
	}
}

// TestTheExpectedItemSetIsReachableWithoutBuildingTheTree is 003 plan §8.5's
// "must not assert a count it computed from the same declaration", asserted as
// the property that makes it true: the expected set is a literal in a file of
// its own, and this test reaches it without the builder having run.
//
// A count derived from the builder is a test of arithmetic. A count derived
// from a literal is a test of the scanner, and a change to the tree is then a
// change to two files with a reviewer seeing both.
func TestTheExpectedItemSetIsReachableWithoutBuildingTheTree(t *testing.T) {
	items := libraryfixture.ExpectedItems()
	if len(items) == 0 {
		t.Fatal("the expected item set is empty")
	}

	names := libraryfixture.LibraryNames()
	byLibrary := map[string]int{}
	roots := map[string]int{}
	paths := map[string]string{}

	for _, item := range items {
		if !slices.Contains(names, item.Library) {
			t.Errorf("%s %q names the library %q, which this package does not build", item.Type, item.Name, item.Library)
			continue
		}
		byLibrary[item.Library]++

		if item.Type == "CollectionFolder" {
			roots[item.Library]++
			if item.Path != "" || item.Parent != "" {
				t.Errorf("the %s row carries path %q and parent %q: a library's own row has neither (003 plan §4.2)", item.Library, item.Path, item.Parent)
			}
			continue
		}

		if item.Path == "" {
			t.Errorf("%s %q in %s has no path, and every item in this tree but a library's own row is backed by one", item.Type, item.Name, item.Library)
		}
		if previous, repeated := paths[item.Library+"/"+item.Path]; repeated {
			t.Errorf("%s/%s backs both %s and %s: two items cannot derive an identity from one path (003 §3.6)", item.Library, item.Path, previous, item.Type)
		}
		paths[item.Library+"/"+item.Path] = item.Type
		if item.Name == "" {
			t.Errorf("%s %s/%s has no name", item.Type, item.Library, item.Path)
		}
	}

	for _, library := range names {
		if roots[library] != 1 {
			t.Errorf("%s has %d rows of its own: each library is one CollectionFolder item (003 §3.1)", library, roots[library])
		}
	}
	if byLibrary["Empty"] != 1 {
		t.Errorf("Empty holds %d expected items and must hold exactly its own row", byLibrary["Empty"])
	}

	// Every parent is an item of the same library, so the structure closes.
	for _, item := range items {
		if item.Parent == "" || item.Parent == libraryfixture.LibraryRoot {
			continue
		}
		if _, known := paths[item.Library+"/"+item.Parent]; !known {
			t.Errorf("%s %q in %s names the parent %q, and no expected item has that path", item.Type, item.Name, item.Library, item.Parent)
		}
	}
}

// TestEveryExpectedItemThatNamesAPathHasOneInTheBuiltTree ties the literal to
// the tree in the one direction that is safe to tie them in.
//
// The literal is written from 003's specification and the tree from the
// reference's reading, and neither is derived from the other — that is what
// makes AC-1 an assertion rather than a restatement. What they must agree on is
// this: an item the scanner is expected to produce has to have a file or a
// directory to produce it from.
func TestEveryExpectedItemThatNamesAPathHasOneInTheBuiltTree(t *testing.T) {
	root := build(t)

	for _, item := range libraryfixture.ExpectedItems() {
		if item.Path == "" {
			continue
		}
		full := filepath.Join(root, item.Library, filepath.FromSlash(item.Path))
		if _, err := os.Lstat(full); err != nil {
			t.Errorf("the expected %s %q names %s/%s, which the built tree does not hold: %v",
				item.Type, item.Name, item.Library, item.Path, err)
		}
	}
}

// TestTheTwoFilesWhoseBytesAreTheirReasonAreWrittenAsDeclared covers the two
// inputs conformance §L2 keeps in this world because no remote request reaches
// either: a subtitle in a legacy single-byte encoding (behaviours §5.11) and an
// image carrying an EXIF orientation (006's resize edge).
//
// Both are inert to 003, which never opens a file. They are asserted here
// because a fixture that shipped a subtitle in UTF-8, or an image with no
// orientation in it, would be a fixture whose one input for each of those rules
// had quietly stopped being one.
func TestTheTwoFilesWhoseBytesAreTheirReasonAreWrittenAsDeclared(t *testing.T) {
	root := build(t)

	subtitle, err := os.ReadFile(filepath.Join(root, "Movies", "An Old Transfer (1985).srt"))
	if err != nil {
		t.Fatalf("the legacy-encoded subtitle: %v", err)
	}
	if utf8.Valid(subtitle) {
		t.Error("the legacy-encoded subtitle is valid UTF-8, so it is no longer the one input behaviours §5.11 has")
	}
	if !bytes.ContainsFunc(subtitle, func(r rune) bool { return r > 0x7F }) {
		t.Error("the legacy-encoded subtitle is pure ASCII, so nothing about it is a legacy encoding")
	}

	poster := filepath.Join(root, "Movies", "The Matrix (1999)", "poster.jpg")
	raw, err := os.ReadFile(poster)
	if err != nil {
		t.Fatalf("the image beside the film: %v", err)
	}
	if !bytes.Contains(raw, []byte(libraryfixture.ExifHeader)) {
		t.Error("the image beside the film carries no EXIF segment, so it carries no orientation either")
	}
	if _, err := jpeg.DecodeConfig(bytes.NewReader(raw)); err != nil {
		t.Errorf("the image beside the film does not decode as a JPEG: %v.\n"+
			"An image 006 cannot open is not an input to a resize edge, it is a file with the right extension.", err)
	}
}

// TestBuildRefusesATreeThatIsAlreadyThere states the builder's one refusal.
// Every test and every run gets its own copy, and a build that wrote over a
// tree somebody had mutated between two scans would hand the next scan a
// mixture of the two.
func TestBuildRefusesATreeThatIsAlreadyThere(t *testing.T) {
	root := build(t)

	err := libraryfixture.Build(root)
	if err == nil {
		t.Fatal("a second build into the same directory succeeded")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("the refusal does not say what is wrong: %v", err)
	}
}

// TestEveryDeclaredFileSaysWhyItIsThere is the rule that keeps the tree
// reviewable. A file nobody can state the reason for is a file nobody can
// decide to remove — and for the ten the reference's reading does not name, the
// reason has to say which rule drops it at *both* servers, since a file only
// this scanner drops would be a difference the reading has no row for.
func TestEveryDeclaredFileSaysWhyItIsThere(t *testing.T) {
	for _, library := range libraryfixture.Libraries() {
		for _, file := range library.Files {
			if strings.TrimSpace(file.Why) == "" {
				t.Errorf("%s/%s has no Why", library.Name, file.Path)
			}
		}
	}
}

// TestTheExpectedItemSetIsALiteralAndNotADerivation asserts the property the
// test above demonstrates, rather than leaving it to be true by the way that
// test happens to be written.
//
// 003 plan §8.5's rule is that the fixture "must not be scanned by a test that
// then asserts a count it computed from the same declaration" — so the expected
// set has to be written down, not worked out. A file that reached for the
// declaration, or for the filesystem, would be computing the answer it is
// supposed to be checking, and every test built on it would pass by
// construction.
func TestTheExpectedItemSetIsALiteralAndNotADerivation(t *testing.T) {
	const literal = "expected.go"

	source, err := os.ReadFile(literal)
	if err != nil {
		t.Fatalf("reading %s: %v", literal, err)
	}

	for _, derivation := range []string{"Libraries()", "Build(", "os.", "filepath.", "range "} {
		if bytes.Contains(source, []byte(derivation)) {
			t.Errorf("%s contains %q, so the expected set is worked out from something rather than written down.\n"+
				"A count derived from the declaration is a test of arithmetic (003 plan §8.5).", literal, derivation)
		}
	}
}
