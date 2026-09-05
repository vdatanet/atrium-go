// Package scan's walk tests sit at the `scan` level of 003 tasks' three: a Go
// test beside the package, over a real tree in a real temporary directory,
// asserting about the function that produced the reading and nothing
// downstream of it.
//
// **What none of them proves** is anything about what was stored or about what
// a client would receive. 003 serves no route, so Principle VIII has no
// boundary here (003 plan §8.1), and a green run in this package is evidence
// about a walk of a filesystem and about nothing further.
//
// The trees are built here rather than taken from `internal/libraryfixture`
// wherever the shape a rule needs is one the fixture deliberately does not
// carry — T1's handoff rule: a file added to the fixture must be one *both*
// servers drop. `TestTheWalkOverTheFixturesMoviesLibrary…` is the one test that
// walks the fixture itself, and it is what ties this package to the tree the
// reference's recorded reading was taken over.
package scan

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/library"
	"github.com/vdatanet/atrium-go/internal/libraryfixture"
)

// repositoryRoot is where this package sits, so a test can name the
// reference's recorded reading from the package directory the test runs in.
// The same constant, for the same reason, is in internal/libraryfixture and in
// internal/surface.
const repositoryRoot = "../.."

// file is one file of a tree a test builds, with the bytes it holds. The
// content is a string rather than a size because two of the rules here are
// decided on bytes: a marker excludes only when it is empty or whitespace, and
// a candidate is refused when it measures zero.
type file struct {
	path    string
	content string
}

// treeInto writes files into root, in the order they are given, creating the
// directories they imply.
//
// The order is a parameter because
// TestTwoTreesWhoseEntriesWereCreatedInOppositeOrders… varies it. It is
// deliberately *not* enough on its own to vary anything — see that test.
func treeInto(t *testing.T, root string, files ...file) string {
	t.Helper()
	for _, f := range files {
		full := filepath.Join(root, filepath.FromSlash(f.path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("creating the directories above %s: %v", f.path, err)
		}
		if err := os.WriteFile(full, []byte(f.content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", f.path, err)
		}
	}
	return root
}

// tree writes files into a fresh temporary directory and answers its path.
func tree(t *testing.T, files ...file) string {
	t.Helper()
	return treeInto(t, t.TempDir(), files...)
}

// walkMovies walks dir as a `movies` library's only root and fails the test on
// any error. `movies` is the type most of these trees use because its
// admitted list is the longest; the rules under test are not the collection
// type's.
func walkMovies(t *testing.T, dir string) Result {
	t.Helper()
	return walkFS(t, os.DirFS(dir))
}

func walkFS(t *testing.T, fsys fs.FS) Result {
	t.Helper()
	result, err := Walk(fsys, 0, library.Movies)
	if err != nil {
		t.Fatalf("Walk: %v", err)
	}
	return result
}

// candidates is the reading's paths, in the order the reading carries them.
func candidates(result Result) []string {
	out := make([]string, len(result.Reading.Entries))
	for i, entry := range result.Reading.Entries {
		out[i] = entry.Path
	}
	return out
}

// refusals is every skipped path with the reason, as "path\treason", so a
// mismatch prints both halves.
func refusals(result Result) []string {
	out := make([]string, len(result.Skipped))
	for i, note := range result.Skipped {
		out[i] = note.Path + "\t" + note.Reason
	}
	return out
}

func assertCandidates(t *testing.T, result Result, want ...string) {
	t.Helper()
	if got := candidates(result); !slices.Equal(got, want) {
		t.Errorf("the reading holds\n\t%v\nwant\n\t%v", got, want)
	}
}

func assertRefusals(t *testing.T, result Result, want ...string) {
	t.Helper()
	if got := refusals(result); !slices.Equal(got, want) {
		t.Errorf("the walk refused\n\t%q\nwant\n\t%q", got, want)
	}
}

func refusal(path string, reason library.Skip) string { return path + "\t" + reason.String() }

// -----------------------------------------------------------------------------
// The marker
// -----------------------------------------------------------------------------

// TestAnEmptyMarkerExcludesTheDirectoryHoldingItAndEverythingBeneath is the
// first half of 003 §3.2's `.ignore` rule, and the whitespace-only subtest is
// the amendment's own words.
//
// The exclusion is reported as **one** note naming the directory, not one per
// file under it: the walk does not descend, so it never saw them. A build that
// reported the files would be reporting paths it had to open the directory to
// learn, which is the descent this rule exists to avoid.
func TestAnEmptyMarkerExcludesTheDirectoryHoldingItAndEverythingBeneath(t *testing.T) {
	for _, marker := range []struct{ name, content string }{
		{"empty", ""},
		{"whitespace only", " \t\r\n  \n"},
	} {
		t.Run(marker.name, func(t *testing.T) {
			dir := tree(t,
				file{"A Kept Film (2001).mkv", "kept"},
				file{"Excluded/" + IgnoreMarker, marker.content},
				file{"Excluded/An Excluded Film (1994).mkv", "excluded"},
				file{"Excluded/Deeper/Beneath It Too (1995).mkv", "excluded"},
			)

			result := walkMovies(t, dir)
			assertCandidates(t, result, "A Kept Film (2001).mkv")
			assertRefusals(t, result, refusal("Excluded", library.SkipIgnoreMarker))
		})
	}
}

// TestTheAncestorSearchIsWhatExcludesAndNotTheMarkersOwnDirectory is the clause
// written to defeat a build that asks *"does this file's own directory carry a
// marker"*.
//
// **There is no candidate in the directory holding the marker.** Every file it
// must exclude is one or two levels below it, so the per-directory build
// answers with both of them and only the ancestor search refuses them. The
// difference is why 003 plan §6.1 states the rule as a search rather than as a
// test: an operator who wrote a marker at the top of a season directory would
// otherwise find it excluded the top directory only.
//
// The test above is written so that it fails on that build as well — its tree
// carries a file beneath the marker's directory too — and this one is the tree
// where **nothing but** the ancestor rule can answer.
func TestTheAncestorSearchIsWhatExcludesAndNotTheMarkersOwnDirectory(t *testing.T) {
	dir := tree(t,
		file{"A Kept Film (2001).mkv", "kept"},
		file{"Excluded/" + IgnoreMarker, ""},
		file{"Excluded/Deeper/One Level Below (1995).mkv", "excluded"},
		file{"Excluded/Deeper/Deeper Still/Two Levels Below (1996).mkv", "excluded"},
	)

	result := walkMovies(t, dir)
	assertCandidates(t, result, "A Kept Film (2001).mkv")
	assertRefusals(t, result, refusal("Excluded", library.SkipIgnoreMarker))
}

// TestAMarkerAtTheLibraryRootExcludesTheWholeLibrary is the inclusive end of
// *"up to the library root and no further"*.
//
// It is asserted beside the test below because the two are the same sentence
// read from its two ends: at the root the marker applies, one directory above
// it does not, and a build that got the boundary off by one fails exactly one
// of the pair.
func TestAMarkerAtTheLibraryRootExcludesTheWholeLibrary(t *testing.T) {
	dir := tree(t,
		file{IgnoreMarker, ""},
		file{"A Film (2001).mkv", "bytes"},
		file{"A Directory/Another Film (2002).mkv", "bytes"},
	)

	result := walkMovies(t, dir)
	assertCandidates(t, result)
	assertRefusals(t, result, refusal(".", library.SkipIgnoreMarker))
}

// TestAMarkerAboveTheLibraryRootExcludesNothing is a **deliberate divergence**,
// asserted so that the day U-42 is measured the measurement lands on a failing
// test rather than on a rediscovery.
//
// The reference searches for the marker from a file's own directory upwards to
// the **filesystem** root
// `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:41-68 @ v10.11.11]`,
// so a stray `.ignore` in a home directory empties every library beneath it.
// Atrium stops at the library root: 003 §3.2's amendment calls that a foot-gun
// rather than a feature, and the narrowing shows more rather than less, which
// is the safe direction for a scanner.
//
// **The marker one directory lower is the control**, and it is what keeps this
// test from passing on a build where markers do nothing at all. The same bytes,
// written one level down, empty the library — so what is asserted is the
// *boundary* and not the marker's impotence. Without it the test is green on a
// build that never reads a marker, which is the failure this whole file is
// written against.
func TestAMarkerAboveTheLibraryRootExcludesNothing(t *testing.T) {
	above := t.TempDir()
	root := filepath.Join(above, "Movies")

	treeInto(t, above, file{IgnoreMarker, ""})
	treeInto(t, root, file{"A Film (2001).mkv", "bytes"})

	// The premise, asserted rather than assumed: the marker really is there
	// and really is the empty kind. A test whose fixture quietly stopped
	// being an exclusion would prove nothing and say nothing.
	info, err := os.Stat(filepath.Join(above, IgnoreMarker))
	if err != nil {
		t.Fatalf("the marker above the root: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("the marker above the root measures %d bytes; only an empty one excludes anything", info.Size())
	}

	assertCandidates(t, walkMovies(t, root), "A Film (2001).mkv")

	// The control: the identical marker inside the root does exclude.
	treeInto(t, root, file{IgnoreMarker, ""})
	assertCandidates(t, walkMovies(t, root))
}

// TestANonEmptyMarkerExcludesNothing is the **accepted shortfall**, asserted
// rather than left as a comment.
//
// At the reference a non-empty marker is a set of `.gitignore`-style patterns
// of which only the matching paths are excluded, with the fallback that a
// marker whose every pattern fails to parse excludes everything
// `[source: Emby.Server.Implementations/Library/DotIgnoreIgnoreRule.cs:95-131 @ v10.11.11]`.
// v1 implements none of it: the matcher would be this project's own, for a
// shape no measurement shows anybody using, and getting it subtly wrong hides
// files an operator expects to see (003 plan §6.1, U-42).
//
// The three contents are the three ways a build might be tempted to read a
// non-empty marker as an exclusion: a pattern that matches the file, a pattern
// that matches nothing, and a line that is not a pattern at all.
//
// The **control** is the same construction as the test above: emptying the
// marker excludes, so what is asserted is the emptiness rule and not a build
// where markers do nothing.
func TestANonEmptyMarkerExcludesNothing(t *testing.T) {
	for _, content := range []string{"*.mkv\n", "nothing-here\n", "  a line that is not a pattern\n"} {
		t.Run(content, func(t *testing.T) {
			dir := tree(t,
				file{"Kept/" + IgnoreMarker, content},
				file{"Kept/A Film (2001).mkv", "bytes"},
			)

			result := walkMovies(t, dir)
			assertCandidates(t, result, "Kept/A Film (2001).mkv")
			// The marker itself is a hidden file, and that is the rule
			// that refuses it — never SkipIgnoreMarker, which would mean
			// the walk had read it as an exclusion of something.
			assertRefusals(t, result, refusal("Kept/"+IgnoreMarker, library.SkipHidden))

			// The control.
			treeInto(t, dir, file{"Kept/" + IgnoreMarker, ""})
			assertCandidates(t, walkMovies(t, dir))
		})
	}
}

// TestAMarkerThatIsNotARegularFileExcludesNothingAndIsNotAnError.
//
// A directory somebody created called `.ignore` is not an operator's exclusion,
// and it is not a media file either. The two answers a build might give instead
// are both worse: excluding the subtree acts on something nobody wrote, and
// failing the read fails the library's whole scan over a directory entry.
//
// It is here because [Walk] says so in a doc comment, and 003's audits have
// found three times now that a decision stated in a comment and asserted
// nowhere is a decision nothing holds.
func TestAMarkerThatIsNotARegularFileExcludesNothingAndIsNotAnError(t *testing.T) {
	dir := tree(t,
		file{"Kept/" + IgnoreMarker + "/not a marker.txt", "bytes"},
		file{"Kept/A Film (2001).mkv", "bytes"},
	)

	result := walkMovies(t, dir)
	assertCandidates(t, result, "Kept/A Film (2001).mkv")
	// The directory is hidden, so the dot rule refuses it — never
	// SkipIgnoreMarker, which would mean the walk had read it as an
	// exclusion of the directory holding it.
	assertRefusals(t, result, refusal("Kept/"+IgnoreMarker, library.SkipHidden))
}

// -----------------------------------------------------------------------------
// The dot rule, asserted apart for the file and for the directory
// -----------------------------------------------------------------------------

// TestAHiddenFileIsSkipped is 003 §3.2's dot rule over a file, and the file it
// uses is the shape the rule was written for: a macOS resource fork, which is a
// hidden file carrying an admitted extension beside the film it shadows.
func TestAHiddenFileIsSkipped(t *testing.T) {
	dir := tree(t,
		file{"._Wall-E (2008).mkv", "resource fork"},
		file{"Wall-E (2008).mkv", "the film"},
	)

	result := walkMovies(t, dir)
	assertCandidates(t, result, "Wall-E (2008).mkv")
	assertRefusals(t, result, refusal("._Wall-E (2008).mkv", library.SkipHidden))
}

// TestAHiddenDirectoryIsNotDescendedInto is the other half, and it is asserted
// apart from the one above for the reason the task list gives: **a walk that
// skipped the file and descended anyway gives the same reading**, because
// [library.CollectionType.Candidate] reads every path component and refuses a
// candidate under a hidden ancestor on its own. The two builds differ in what
// the walk *did*, not in what it answered, until something inside the hidden
// directory is not hidden.
//
// So the assertion is on the descent itself. The tree is wrapped in an
// [fs.FS] that records every ReadDir, and the hidden directory must not appear
// among them. The second, independent half is the refusals: one note naming the
// directory, and none naming anything under it — a walk that descended has
// three notes and has learned three paths it had to open the directory to see.
func TestAHiddenDirectoryIsNotDescendedInto(t *testing.T) {
	dir := tree(t,
		file{".hidden/A Hidden Film (1990).mkv", "bytes"},
		file{".hidden/Deeper/Another (1991).mkv", "bytes"},
		file{"A Kept Film (2001).mkv", "bytes"},
	)

	recorder := &recordingFS{FS: os.DirFS(dir)}
	result := walkFS(t, recorder)

	assertCandidates(t, result, "A Kept Film (2001).mkv")

	if got, want := recorder.read, []string{"."}; !slices.Equal(got, want) {
		t.Errorf("the walk read the directories %q, want %q — a hidden directory is not descended into", got, want)
	}

	assertRefusals(t, result, refusal(".hidden", library.SkipHidden))
}

// TestTheRecordingFilesystemSeesADescentWhenThereIsOne keeps the assertion
// above from being green because the instrument is broken.
//
// A `recordingFS` that recorded nothing would pass that test on every build,
// including one that descends into every hidden directory in the tree. So the
// same tree with the same walk, minus the leading dot on the directory name,
// must show the descent.
func TestTheRecordingFilesystemSeesADescentWhenThereIsOne(t *testing.T) {
	dir := tree(t,
		file{"not-hidden/A Film (1990).mkv", "bytes"},
		file{"not-hidden/Deeper/Another (1991).mkv", "bytes"},
	)

	recorder := &recordingFS{FS: os.DirFS(dir)}
	result := walkFS(t, recorder)

	assertCandidates(t, result, "not-hidden/A Film (1990).mkv", "not-hidden/Deeper/Another (1991).mkv")

	want := []string{".", "not-hidden", "not-hidden/Deeper"}
	if got := recorder.read; !slices.Equal(got, want) {
		t.Errorf("the walk read the directories %q, want %q", got, want)
	}
}

// -----------------------------------------------------------------------------
// The zero-byte rule
// -----------------------------------------------------------------------------

// TestAZeroByteFileYieldsNoCandidate is 003 §3.2's incomplete copy.
//
// The sibling that measures one byte is what keeps the assertion from passing
// on a build that refuses every file: the rule is about the size, so the test
// asserts a size on both sides of it.
func TestAZeroByteFileYieldsNoCandidate(t *testing.T) {
	dir := tree(t,
		file{"A Complete Copy (2001).mkv", "x"},
		file{"An Incomplete Copy (2000).mkv", ""},
	)

	result := walkMovies(t, dir)
	assertCandidates(t, result, "A Complete Copy (2001).mkv")
	assertRefusals(t, result, refusal("An Incomplete Copy (2000).mkv", library.SkipZeroBytes))

	if got := result.Reading.Entries[0].Size; got != 1 {
		t.Errorf("the kept file measures %d bytes in the reading, want 1", got)
	}
	if result.Reading.Entries[0].ModifiedAt.IsZero() {
		t.Error("the kept file carries no modification time; 003 §6.4's change signal is the size and that time")
	}
}

// TestTheReferenceMakesAnItemOfTheZeroByteFilm is the declared inequality
// beside the rule above, and it is asserted rather than commented for the
// reason 003 plan §8.2 gives: a difference that has **gone away** has to fail as
// loudly as an undeclared one.
//
// The reference's recorded reading of this repository's own fixture tree names
// `An Incomplete Copy (2000).mkv` as a `Movie`
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`, and
// Atrium's walk of the same tree yields no candidate for it. That is one of the
// forty-seven differences 003 declares over that tree; T17 owns the count, and
// this is the one row the walk produces on its own.
func TestTheReferenceMakesAnItemOfTheZeroByteFilm(t *testing.T) {
	const zeroByteFilm = "An Incomplete Copy (2000).mkv"

	referenced := referenceMoviePaths(t)
	if !slices.Contains(referenced, zeroByteFilm) {
		t.Fatalf("%s no longer names %q under Movies; the declared difference has gone away, "+
			"which fails as loudly as an undeclared one (003 plan §8.2)",
			libraryfixture.ReferenceReading, zeroByteFilm)
	}

	result := walkMovies(t, fixtureLibrary(t, "Movies"))
	if slices.Contains(candidates(result), zeroByteFilm) {
		t.Errorf("the walk yields a candidate for %q; 003 §3.2 ignores an incomplete copy", zeroByteFilm)
	}
}

// -----------------------------------------------------------------------------
// Entries that are not files
// -----------------------------------------------------------------------------

// TestASymbolicLinkIsFollowedAndWhatIsNotAFileAfterwardsIsRefused is a shape
// 003 §3.2 does not mention and a walk cannot avoid taking a position on.
//
// The reference reads a library through a filesystem that follows links, so a
// linked film is an item there; refusing one here would show **fewer** items
// than the reference, which is the unsafe direction for a scanner. So a link to
// a file is the file it points at, **size included** — and the size is what
// makes the assertion more than a count: reading the directory entry rather
// than following it gives a candidate measuring the length of the target's
// path, which is a plausible non-zero number and would pass every other
// assertion in this file.
//
// A link pointing at a **directory** and a link pointing at **nothing** are
// both refused, and neither fails the walk. The second is the one worth
// arguing: a dangling link, or a file moved between the directory read and the
// stat, is a race a walk of a live tree really has, and failing the library's
// whole scan over one would mean a download completing during a scan costs the
// operator every item in that library.
func TestASymbolicLinkIsFollowedAndWhatIsNotAFileAfterwardsIsRefused(t *testing.T) {
	dir := tree(t,
		file{"A Directory/Inside It (2003).mkv", "bytes"},
		file{"The Film (2001).mkv", "twelve bytes"},
	)

	links := []struct{ from, to string }{
		{"A Link To The Film (2001).mkv", "The Film (2001).mkv"},
		{"A Link To A Directory (2002).mkv", "A Directory"},
		{"A Link To Nothing (2004).mkv", "Nowhere At All.mkv"},
	}
	for _, link := range links {
		if err := os.Symlink(link.to, filepath.Join(dir, link.from)); err != nil {
			t.Skipf("this filesystem does not do symbolic links: %v", err)
		}
	}

	result := walkMovies(t, dir)

	assertCandidates(t, result,
		"A Directory/Inside It (2003).mkv",
		"A Link To The Film (2001).mkv",
		"The Film (2001).mkv",
	)
	assertRefusals(t, result,
		refusal("A Link To A Directory (2002).mkv", library.SkipNotARegularFile),
		refusal("A Link To Nothing (2004).mkv", library.SkipNotARegularFile),
	)

	const throughTheLink = "A Link To The Film (2001).mkv"
	for _, entry := range result.Reading.Entries {
		if entry.Path == throughTheLink && entry.Size != int64(len("twelve bytes")) {
			t.Errorf("%s measures %d in the reading, want %d — the link was read rather than followed",
				throughTheLink, entry.Size, len("twelve bytes"))
		}
	}
}

// -----------------------------------------------------------------------------
// Determinism
// -----------------------------------------------------------------------------

// TestTheReadingIsSortedOnThePathAndNotInWalkOrder pins the sort, and the tree
// is the smallest one where the two orders differ.
//
// [fs.WalkDir] visits a directory's children before a sibling that sorts after
// the directory's own *name*, so a directory `A` and a file `A.mkv` come out in
// the order `A/x.mkv`, `A.mkv`. Sorted on the whole path they are the other way
// round, because `.` is 0x2E and `/` is 0x2F. Every tree whose directories sort
// after their sibling files hides this, which is most of them.
func TestTheReadingIsSortedOnThePathAndNotInWalkOrder(t *testing.T) {
	dir := tree(t,
		file{"A/Inside The Directory (2001).mkv", "bytes"},
		file{"A.mkv", "bytes"},
	)

	assertCandidates(t, walkMovies(t, dir), "A.mkv", "A/Inside The Directory (2001).mkv")
}

// TestTwoWalksWhoseDirectoryEntriesArriveInOppositeOrdersYieldTheIdenticalReading
// is Principle VII asserted at the layer where a filesystem's own ordering
// could still reach the answer.
//
// **The creation order of the files is not what varies it, and saying so is the
// point.** [os.DirFS] implements [fs.ReadDirFS] and its ReadDir sorts, so two
// trees whose entries were created in opposite orders are the *same input* by
// the time [fs.WalkDir] has read them — the assertion would be satisfied by the
// standard library and would survive the removal of everything this package
// does. That is 003 T6's finding a second time: a "both orders" loop that
// varies nothing because something sorts first.
//
// So the order is varied where it can actually reach the walk: an [fs.FS] whose
// ReadDir answers in the opposite order, which is a shape [fs.ReadDir] hands
// straight through to the caller without sorting. The reading must be identical
// — the same paths, sizes and modification times, over the same tree — and the
// mutation that removes the sort in [Walk] turns it red.
//
// The companion over two trees is here as well, asserted on paths and sizes and
// **not** on modification times: two builds of one declaration differ in every
// modification time, which is exactly why the fixture is generated rather than
// committed (003 plan §8.5).
func TestTwoWalksWhoseDirectoryEntriesArriveInOppositeOrdersYieldTheIdenticalReading(t *testing.T) {
	files := []file{
		{"A/Inside The Directory (2001).mkv", "one"},
		{"A.mkv", "two"},
		{"B (2002).mkv", "three"},
		{"B/Also Inside (2003).mkv", "four"},
		// Two refusals, so that Skipped is a list with an order too. It
		// is sorted for the same reason the reading is: 003 §3.8 puts it
		// in front of an operator, and a list that moves between two runs
		// of one scan is a list nothing can be compared against.
		{".hidden (2005).mkv", "five"},
		{"poster.jpg", "six"},
	}

	dir := tree(t, files...)

	forwards := walkFS(t, os.DirFS(dir))
	backwards := walkFS(t, reversedFS{FS: os.DirFS(dir)})

	if !slices.Equal(forwards.Reading.Entries, backwards.Reading.Entries) {
		t.Errorf("read forwards the entries are\n\t%v\nand backwards\n\t%v", forwards.Reading.Entries, backwards.Reading.Entries)
	}
	if forwards.Reading.Root != backwards.Reading.Root {
		t.Errorf("the root ordinal moved: %d against %d", forwards.Reading.Root, backwards.Reading.Root)
	}
	if !slices.Equal(forwards.Skipped, backwards.Skipped) {
		t.Errorf("read forwards the refusals are\n\t%v\nand backwards\n\t%v", forwards.Skipped, backwards.Skipped)
	}
	if len(forwards.Skipped) < 2 {
		t.Fatalf("the tree produced %d refusals; with fewer than two there is no order to assert", len(forwards.Skipped))
	}

	// The reversal has to reach the walk, or the assertion above is another
	// loop that varies nothing. Read through the wrapper directly and check
	// that it really does answer backwards.
	plain, err := fs.ReadDir(os.DirFS(dir), ".")
	if err != nil {
		t.Fatalf("reading the root: %v", err)
	}
	reversed, err := fs.ReadDir(reversedFS{FS: os.DirFS(dir)}, ".")
	if err != nil {
		t.Fatalf("reading the root backwards: %v", err)
	}
	if len(plain) < 2 || plain[0].Name() == reversed[0].Name() {
		t.Fatalf("the reversing filesystem answers %d entries beginning with %q, the same as the plain one; "+
			"this test varies nothing", len(reversed), reversed[0].Name())
	}

	// And the two trees, built in opposite creation orders.
	other := slices.Clone(files)
	slices.Reverse(other)
	second := walkMovies(t, tree(t, other...))

	if !slices.Equal(candidates(forwards), candidates(second)) {
		t.Errorf("two trees built in opposite orders read as\n\t%v\nand\n\t%v", candidates(forwards), candidates(second))
	}
	for i, entry := range forwards.Reading.Entries {
		if entry.Size != second.Reading.Entries[i].Size {
			t.Errorf("%s measures %d in one tree and %d in the other", entry.Path, entry.Size, second.Reading.Entries[i].Size)
		}
	}
}

// -----------------------------------------------------------------------------
// The whole tree, and the errors
// -----------------------------------------------------------------------------

// TestTheWalkOverTheFixturesMoviesLibraryYieldsExactlyTheseCandidates is the
// one test here that walks the tree the reference's recorded reading was taken
// over, and it asserts both halves of what a walk answers: every candidate, and
// every refusal with the rule that refused it.
//
// Both lists are literals. A list computed from the fixture's own declaration
// would be a test of arithmetic (003 plan §8.5), and it would agree with a walk
// that had stopped applying a rule the moment the declaration changed.
func TestTheWalkOverTheFixturesMoviesLibraryYieldsExactlyTheseCandidates(t *testing.T) {
	result := walkMovies(t, fixtureLibrary(t, "Movies"))

	assertCandidates(t, result,
		"  Padded   (1999).mkv",
		"10 Things I Hate About You (1999).mkv",
		"100% Wolf (2020).mkv",
		"2 Fast 2 Furious (2003).mkv",
		"A Bridge Too Far (1977).mkv",
		"A Broadcast Capture (2011).ts",
		"A Newer Transfer (2015).mp4",
		"Amélie (2001).mkv",
		"An Old Transfer (1985).avi",
		"Don't Look Up (2021).mkv",
		"Rock & Roll (1978).mkv",
		"S.W.A.T. (2003).mkv",
		"The Long Film (1998)/The Long Film (1998) - part1.mkv",
		"The Long Film (1998)/The Long Film (1998) - part2.mkv",
		"The Matrix (1999)/The Matrix (1999).mkv",
		"Wall-E (2008).mkv",
		"iRobot (2004).mkv",
	)

	assertRefusals(t, result,
		refusal("._Wall-E (2008).mkv", library.SkipHidden),
		refusal(".hidden", library.SkipHidden),
		refusal("An Incomplete Copy (2000).mkv", library.SkipZeroBytes),
		refusal("An Old Transfer (1985).srt", library.SkipExtension),
		refusal("Excluded", library.SkipIgnoreMarker),
		refusal("Not A Film (1999).mp3", library.SkipExtension),
		refusal("The Matrix (1999)/poster.jpg", library.SkipExtension),
	)
}

// TestEveryFixtureLibraryWalksWithoutError is the shallow half, and it is here
// because the three collection types admit different lists and a walk that
// worked only for `movies` would pass every test above.
//
// `Empty` is walked too, and it answers a reading with nothing in it and **no
// error**: a library with nothing in it is not an unreadable root, and 003
// §3.8's guard against reading the second as the first is T13's.
func TestEveryFixtureLibraryWalksWithoutError(t *testing.T) {
	root := t.TempDir()
	if err := libraryfixture.Build(root); err != nil {
		t.Fatalf("building the fixture: %v", err)
	}

	for _, declared := range libraryfixture.Libraries() {
		t.Run(declared.Name, func(t *testing.T) {
			collection, ok := library.ParseCollectionType(declared.CollectionType)
			if !ok {
				t.Fatalf("the fixture declares %s as %q, which is not one of 003 §3.1's three",
					declared.Name, declared.CollectionType)
			}
			result, err := Walk(os.DirFS(filepath.Join(root, declared.Name)), 0, collection)
			if err != nil {
				t.Fatalf("Walk: %v", err)
			}
			if declared.Name == "Empty" {
				if len(result.Reading.Entries) != 0 || len(result.Skipped) != 0 {
					t.Errorf("the Empty library read %d candidates and %d refusals, want none of either",
						len(result.Reading.Entries), len(result.Skipped))
				}
				return
			}
			if len(result.Reading.Entries) == 0 {
				t.Errorf("%s read as holding no candidate at all", declared.Name)
			}
		})
	}
}

// TestAnUnknownCollectionTypeIsAnErrorAndNotAnEmptyReading.
//
// An unknown type admits nothing, so a walk that carried on would read a whole
// library as empty and hand a reconciliation the deletion of everything in it —
// which 003 §3.8 calls the single most destructive thing a scanner can do. It
// is a caller's error and it fails at the top rather than one file at a time.
func TestAnUnknownCollectionTypeIsAnErrorAndNotAnEmptyReading(t *testing.T) {
	dir := tree(t, file{"A Film (2001).mkv", "bytes"})

	result, err := Walk(os.DirFS(dir), 0, library.CollectionType("books"))
	if err == nil {
		t.Fatalf("Walk answered a reading of %d entries, want an error", len(result.Reading.Entries))
	}
	if !errors.Is(err, library.ErrCollectionTypeUnknown) {
		t.Errorf("Walk answered %v, want it to wrap ErrCollectionTypeUnknown", err)
	}
	if len(result.Reading.Entries) != 0 || len(result.Skipped) != 0 {
		t.Errorf("Walk answered %d entries and %d refusals beside the error, want none of either",
			len(result.Reading.Entries), len(result.Skipped))
	}
}

// TestAnErrorAnywhereInTheWalkFailsTheWholeWalkAndReturnsNoPartialReading is
// 003 plan §7's row, and the assertion that matters is the **second** one.
//
// A build that returned what it had read before the error would hand a
// reconciliation a reading that is missing a subtree, and every item under that
// subtree would be computed as deleted. The error alone does not prevent that:
// a caller can ignore an error, and a partial reading beside one is a loaded
// gun. There is nothing to be tempted by, and this is what says so.
func TestAnErrorAnywhereInTheWalkFailsTheWholeWalkAndReturnsNoPartialReading(t *testing.T) {
	dir := tree(t,
		file{"A Film Read Before The Failure (2001).mkv", "bytes"},
		file{"Unreadable/A Film Under It (2002).mkv", "bytes"},
	)

	broken := failingFS{FS: os.DirFS(dir), failOn: "Unreadable"}
	result, err := Walk(broken, 0, library.Movies)
	if err == nil {
		t.Fatalf("Walk answered a reading of %d entries, want the error the filesystem raised", len(result.Reading.Entries))
	}
	if !strings.Contains(err.Error(), "Unreadable") {
		t.Errorf("Walk answered %v, which does not name the directory that failed", err)
	}
	if len(result.Reading.Entries) != 0 {
		t.Errorf("Walk answered %d entries beside the error: %v", len(result.Reading.Entries), candidates(result))
	}
}

// -----------------------------------------------------------------------------
// The three filesystems these tests wrap the real one in
// -----------------------------------------------------------------------------

// recordingFS records the name of every directory the walk read, in the order
// it read them. It is an observer over a real tree and not a fake filesystem:
// 003 plan §3 argues at some length why there is no filesystem port, and what
// these tests want is a real directory rather than a double.
type recordingFS struct {
	fs.FS
	read []string
}

func (r *recordingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	r.read = append(r.read, name)
	return fs.ReadDir(r.FS, name)
}

// reversedFS answers every ReadDir in the opposite order.
//
// [fs.ReadDir] sorts only when the filesystem does not implement
// [fs.ReadDirFS]; one that does is handed straight through. So this is the one
// way a directory's own ordering can reach a walk of a real tree, and it is
// what the determinism test varies.
type reversedFS struct {
	fs.FS
}

func (r reversedFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(r.FS, name)
	slices.Reverse(entries)
	return entries, err
}

// failingFS refuses to read one named directory, which is a permission refused
// deep inside a tree — the failure 003 plan §7 says fails the library's whole
// scan.
type failingFS struct {
	fs.FS
	failOn string
}

func (f failingFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.failOn {
		return nil, fmt.Errorf("reading %s: %w", name, fs.ErrPermission)
	}
	return fs.ReadDir(f.FS, name)
}

// -----------------------------------------------------------------------------
// The fixture, and the reference's recorded reading of it
// -----------------------------------------------------------------------------

// fixtureLibrary builds the fixture tree into a fresh temporary directory and
// answers the path of one library's root.
func fixtureLibrary(t *testing.T, name string) string {
	t.Helper()
	root := t.TempDir()
	if err := libraryfixture.Build(root); err != nil {
		t.Fatalf("building the fixture: %v", err)
	}
	return filepath.Join(root, name)
}

// referenceMoviePaths is every path the reference's recorded reading names
// under `Movies`, read out of the JSON rather than transcribed.
//
// Transcribing it is how the two stop being the same reading, and the
// transcription would still pass — T1's rule for the fixture applies to the
// reading it was measured against.
func referenceMoviePaths(t *testing.T) []string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join(repositoryRoot, filepath.FromSlash(libraryfixture.ReferenceReading)))
	if err != nil {
		t.Fatalf("reading %s: %v", libraryfixture.ReferenceReading, err)
	}

	var parsed struct {
		Libraries []struct {
			Name  string `json:"name"`
			Items []struct {
				Path *string `json:"path"`
			} `json:"items"`
		} `json:"libraries"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("parsing %s: %v", libraryfixture.ReferenceReading, err)
	}

	for _, library := range parsed.Libraries {
		if library.Name != "Movies" {
			continue
		}
		var paths []string
		for _, item := range library.Items {
			if item.Path != nil {
				paths = append(paths, *item.Path)
			}
		}
		if len(paths) == 0 {
			t.Fatalf("%s names no path at all under Movies", libraryfixture.ReferenceReading)
		}
		return paths
	}

	t.Fatalf("%s names no Movies library", libraryfixture.ReferenceReading)
	return nil
}
