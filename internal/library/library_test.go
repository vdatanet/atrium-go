package library

import (
	"slices"
	"testing"
)

// measuredLists is 003 §3.2's table, transcribed once and used by every test
// below that needs it.
//
// `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`
var measuredLists = map[CollectionType][]string{
	Movies: {".mkv", ".mp4", ".avi", ".ts"},
	Shows:  {".mkv", ".avi", ".mp4"},
	Music:  {".flac", ".m4a", ".dsf"},
}

// TestTheThreeCollectionTypeNamesAreSpelledAsAnOperatorConfiguresThem pins the
// three strings of 003 §3.1.
//
// They are not internal names: they are what an operator writes and what the
// reference's own collection_type reads, and the fixture's libraries are
// declared with them. A constant renamed to `tv` would leave every test in this
// package green and every library in the fixture unconfigurable.
func TestTheThreeCollectionTypeNamesAreSpelledAsAnOperatorConfiguresThem(t *testing.T) {
	if got, want := string(Movies), "movies"; got != want {
		t.Errorf("Movies is %q, want %q", got, want)
	}
	if got, want := string(Shows), "tvshows"; got != want {
		t.Errorf("Shows is %q, want %q", got, want)
	}
	if got, want := string(Music), "music"; got != want {
		t.Errorf("Music is %q, want %q", got, want)
	}

	if got, want := AllCollectionTypes(), []CollectionType{Movies, Shows, Music}; !slices.Equal(got, want) {
		t.Errorf("AllCollectionTypes() = %v, want %v", got, want)
	}

	for _, name := range []string{"movies", "tvshows", "music"} {
		if _, ok := ParseCollectionType(name); !ok {
			t.Errorf("ParseCollectionType(%q) refused a name 003 §3.1 states", name)
		}
	}
	// Mixed-content roots are not supported in v1 (003 §3.1), and neither is
	// a fourth type, a blank one, or a spelling that only looks right.
	for _, name := range []string{"", "Movies", "TVShows", "tv", "shows", "musicvideos", "mixed", "books"} {
		if got, ok := ParseCollectionType(name); ok {
			t.Errorf("ParseCollectionType(%q) accepted it as %q; 003 §3.1 states three names", name, got)
		}
	}
}

// TestAllCollectionTypesReturnsAFreshSlice asserts that a caller cannot reorder
// the list every other caller reads.
func TestAllCollectionTypesReturnsAFreshSlice(t *testing.T) {
	first := AllCollectionTypes()
	first[0] = Music
	if got, want := AllCollectionTypes(), []CollectionType{Movies, Shows, Music}; !slices.Equal(got, want) {
		t.Errorf("after a caller wrote into the returned slice, AllCollectionTypes() = %v, want %v", got, want)
	}
}

// TestEachTypeAdmitsExactlyItsMeasuredListAndNothingElse is 003 §3.2's table
// asserted in both directions.
//
// The "nothing else" half is the one with teeth. It asks each type about every
// extension on the *other two* types' lists as well as about a spread of things
// a real library root contains — artwork, subtitles, sidecars and
// operating-system detritus, none of which is an error (003 §3.2).
//
// `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`
func TestEachTypeAdmitsExactlyItsMeasuredListAndNothingElse(t *testing.T) {
	// Every extension named anywhere in the table, so that each type is
	// asked about the other two types' lists and not only about detritus.
	var everyMeasured []string
	for _, c := range AllCollectionTypes() {
		everyMeasured = append(everyMeasured, measuredLists[c]...)
	}
	// A library root's ordinary contents, plus the two audio extensions
	// behaviours §2.15 measured under a video root.
	refused := []string{
		".mp3", ".mka", ".srt", ".ass", ".nfo", ".jpg", ".png", ".txt",
		".lrc", ".cue", ".m3u", ".DS_Store", ".part", ".iso", ".mov",
		".webm", ".wav", ".ogg", ".m4v", ".mpg", ".wmv", ".aac", ".opus",
	}

	for _, c := range AllCollectionTypes() {
		t.Run(string(c), func(t *testing.T) {
			want := measuredLists[c]
			if got := c.Extensions(); !slices.Equal(got, want) {
				t.Errorf("Extensions() = %v, want the measured list %v", got, want)
			}
			for _, ext := range want {
				if !c.Admits(ext) {
					t.Errorf("Admits(%q) is false; 003 §3.2 measured it admitted under %s", ext, c)
				}
			}
			for _, ext := range append(append([]string{}, everyMeasured...), refused...) {
				if slices.Contains(want, ext) {
					continue
				}
				if c.Admits(ext) {
					t.Errorf("Admits(%q) is true; it is not on %s's measured list %v", ext, c, want)
				}
			}
		})
	}

	// An unknown type is not a fourth list with different contents; it is
	// no list at all.
	unknown := CollectionType("books")
	if unknown.Valid() {
		t.Fatal("CollectionType(\"books\").Valid() is true")
	}
	if got := unknown.Extensions(); len(got) != 0 {
		t.Errorf("an unknown collection type offers %v, want nothing", got)
	}
	for _, ext := range everyMeasured {
		if unknown.Admits(ext) {
			t.Errorf("an unknown collection type admits %q", ext)
		}
	}
}

// TestExtensionsReturnsAFreshSlice asserts that a caller cannot edit the
// measured list in place.
//
// The lists are a measured contract and not configuration (003 plan §4.3).
// There is nothing to configure them with, and a returned slice that aliased
// the package's own is a way to configure them by accident.
func TestExtensionsReturnsAFreshSlice(t *testing.T) {
	got := Movies.Extensions()
	got[0] = ".mp3"
	if Movies.Admits(".mp3") {
		t.Fatal("writing into the slice Extensions() returned changed what Movies admits")
	}
	if again := Movies.Extensions(); !slices.Equal(again, measuredLists[Movies]) {
		t.Errorf("Extensions() = %v after a caller wrote into an earlier one, want %v", again, measuredLists[Movies])
	}
}

// TestAnAudioFileUnderAVideoRootIsAdmittedByNoTypeAtAll is behaviours §2.15,
// and it is asserted the way that measurement is worded rather than the way it
// is convenient to test.
//
// The measurement is about **promotion**. Under the `movies` and `tvshows`
// roots of a library holding 8,288 items, 89 `.mp3` files and 3 `.mka` files
// produced *no item of any type* — not a `Movie`, not an `Episode`, and not an
// `Audio` either
// `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`.
//
// A test written as *"movies refuses `.mp3`"* passes on a build that consults
// every list and quietly files theme music beside a film as `Audio`, which is
// the exact bug the measurement exists to prevent. So the assertion here is
// over **every collection type at once**: there is no list in this package that
// would take either extension, so there is nothing for a promoting
// implementation to reach for.
func TestAnAudioFileUnderAVideoRootIsAdmittedByNoTypeAtAll(t *testing.T) {
	// The two the probe measured, at the paths the fixture plants them at:
	// `Movies/Not A Film (1999).mp3` is theme music beside a film and
	// `Shows/The Series/Season 01/Not An Episode.mka` is its counterpart
	// under a video root of the other kind.
	cases := []struct {
		ext     string
		library CollectionType
		path    string
	}{
		{".mp3", Movies, "Not A Film (1999).mp3"},
		{".mka", Shows, "The Series/Season 01/Not An Episode.mka"},
	}

	for _, tc := range cases {
		t.Run(tc.ext, func(t *testing.T) {
			// The whole point: not one of the three lists takes it.
			var admittedBy []CollectionType
			for _, c := range AllCollectionTypes() {
				if c.Admits(tc.ext) {
					admittedBy = append(admittedBy, c)
				}
			}
			if len(admittedBy) != 0 {
				t.Errorf("%q is admitted by %v; behaviours §2.15 measured it becoming no item of any type", tc.ext, admittedBy)
			}

			// And the file at its real path is a candidate under no
			// type either — including the two that are not its
			// library's, which is where a promoting build would
			// find its answer.
			for _, c := range AllCollectionTypes() {
				ok, why := c.Candidate(tc.path)
				if ok {
					t.Errorf("%s.Candidate(%q) is a candidate", c, tc.path)
				}
				if why != SkipExtension {
					t.Errorf("%s.Candidate(%q) refused with %v, want the extension rule", c, tc.path, why)
				}
			}
		})
	}
}

// TestTheDecidingListIsTheLibrarysOwnAndNeverTheUnion is the other half of
// behaviours §2.15, and it is the half that can fail on its own.
//
// The `.mp3` and `.mka` above are on no list at all, so a build with one shared
// union of every extension would pass that test. This one cannot be passed that
// way: `.flac` is admitted under `music` and must still be refused under
// `movies` and `tvshows`, and `.mkv` the other way round. The measurement says
// it in those terms — *"the same server admits `.flac`, `.m4a` and `.dsf` under
// its music root, so the extensions are recognised; the collection type is what
// refuses them"*
// `[probe: tools/probe_library_extensions.py, Jellyfin 10.11.11, 2026-08-27]`.
func TestTheDecidingListIsTheLibrarysOwnAndNeverTheUnion(t *testing.T) {
	cases := []struct {
		path          string
		admittedUnder CollectionType
	}{
		{"Fake Album/01 - A Track.flac", Music},
		{"Fake Album/01 - A Track.m4a", Music},
		{"Fake Album/01 - A Track.dsf", Music},
		{"The Matrix (1999)/The Matrix (1999).mkv", Movies},
		{"The Matrix (1999)/The Matrix (1999).ts", Movies},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			if ok, why := tc.admittedUnder.Candidate(tc.path); !ok {
				t.Fatalf("%s.Candidate(%q) refused with %v; it is on that type's own list", tc.admittedUnder, tc.path, why)
			}
			for _, c := range AllCollectionTypes() {
				if c == tc.admittedUnder {
					continue
				}
				// Two types share `.mkv` and `.mp4`, and those
				// are not cross cases. Which type shares which
				// extension is read out of the test's own
				// transcription of 003 §3.2's table and never
				// out of Admits — asking the code under test
				// which cases to skip is how a build that
				// admits everything skips every assertion and
				// stays green.
				if slices.Contains(measuredLists[c], pathExt(tc.path)) {
					continue
				}
				ok, why := c.Candidate(tc.path)
				if ok {
					t.Errorf("%s.Candidate(%q) admitted a file only %s's list carries", c, tc.path, tc.admittedUnder)
				}
				if why != SkipExtension {
					t.Errorf("%s.Candidate(%q) refused with %v, want the extension rule", c, tc.path, why)
				}
			}
		})
	}
}

// TestAsciiCaseIsIgnoredWhenAnExtensionIsMatched asserts the one leniency the
// reference has and this package copies.
//
// The reference compares an extension with `StringComparison.OrdinalIgnoreCase`
// `[source: Emby.Naming/Video/VideoResolver.cs:119-123 @ v10.11.11]`, so a
// library holding `.MKV` reads the same on both servers and a case-sensitive
// implementation would be missing items the reference has.
func TestAsciiCaseIsIgnoredWhenAnExtensionIsMatched(t *testing.T) {
	for _, ext := range []string{".MKV", ".Mkv", ".mKV", ".TS", ".FLAC"} {
		c := Movies
		if ext == ".FLAC" {
			c = Music
		}
		if !c.Admits(ext) {
			t.Errorf("%s.Admits(%q) is false; the reference's comparison ignores case", c, ext)
		}
	}
	// The fold is ASCII-only and not a Unicode one. U+212A KELVIN SIGN
	// lowercases to `k` under a Unicode fold, so `.m<KELVIN>v` would be
	// admitted by a strings.ToLower implementation and is refused by the
	// reference's ordinal comparison.
	if Movies.Admits(".mKv") {
		t.Error("a Kelvin sign spelled .mkv; the fold is meant to be ASCII-only")
	}
}

// TestAnExtensionWithoutItsLeadingDotIsNotAnExtension pins Admits's contract.
//
// It takes what path.Ext yields. A caller that hands it `mkv` has a bug, and a
// forgiving implementation would hide the bug in every caller at once.
func TestAnExtensionWithoutItsLeadingDotIsNotAnExtension(t *testing.T) {
	for _, ext := range []string{"", ".", "mkv", "MKV", "kv", "x.mkv", " .mkv"} {
		if Movies.Admits(ext) {
			t.Errorf("Movies.Admits(%q) is true; Admits takes an extension with its leading dot", ext)
		}
	}
	// And a file with no extension at all is a candidate nowhere.
	for _, c := range AllCollectionTypes() {
		if ok, _ := c.Candidate("A Film Without An Extension"); ok {
			t.Errorf("%s made a candidate of a file with no extension", c)
		}
	}
}

// TestAHiddenComponentAnywhereRefusesThePath is 003 §3.2's dot rule.
//
// It is any component and not only the last, because the fixture carries both
// shapes and they are different bugs: `.hidden/A Hidden Film (1990).mkv` is a
// candidate inside a hidden directory, and `._Wall-E (2008).mkv` is a macOS
// resource fork, which is a hidden *file* carrying an admitted extension.
func TestAHiddenComponentAnywhereRefusesThePath(t *testing.T) {
	hidden := []string{
		".hidden/A Hidden Film (1990).mkv",
		"._Wall-E (2008).mkv",
		"The Series/.git/objects/A Film.mkv",
		"a/b/.c/d.mkv",
		"Excluded/.ignore",
	}
	for _, p := range hidden {
		ok, why := Movies.Candidate(p)
		if ok {
			t.Errorf("Candidate(%q) is a candidate; 003 §3.2 refuses any component beginning with a dot", p)
		}
		if why != SkipHidden {
			t.Errorf("Candidate(%q) refused with %v, want the dot rule", p, why)
		}
	}

	// A dot inside a name is not a hidden component, and neither is a
	// version number.
	for _, p := range []string{"The.Matrix.1999.mkv", "S.W.A.T. (2003)/S.W.A.T. (2003).mkv"} {
		if ok, why := Movies.Candidate(p); !ok {
			t.Errorf("Candidate(%q) refused with %v; the dot rule is about a component's first byte", p, why)
		}
	}

	// `.` and `..` are path syntax rather than names. A walker asks this of
	// the directory it is standing in, and a root that answered "hidden"
	// would exclude the whole library.
	for _, name := range []string{".", ".."} {
		if IsHiddenName(name) {
			t.Errorf("IsHiddenName(%q) is true; it is path syntax, not a hidden name", name)
		}
	}
	for _, name := range []string{".hidden", "._Wall-E (2008).mkv", ".ignore", ".DS_Store"} {
		if !IsHiddenName(name) {
			t.Errorf("IsHiddenName(%q) is false", name)
		}
	}
}

// TestSkipStringNamesEveryRule keeps the reason a scan reports readable, and
// keeps a new reason from silently reading as "unknown".
func TestSkipStringNamesEveryRule(t *testing.T) {
	seen := map[string]Skip{}
	for _, s := range []Skip{NotSkipped, SkipHidden, SkipExtrasFolder, SkipExtrasFilename, SkipExtrasSuffix, SkipExtension} {
		got := s.String()
		if got == "unknown" || got == "" {
			t.Errorf("Skip(%d).String() = %q", s, got)
		}
		if other, dup := seen[got]; dup {
			t.Errorf("Skip(%d) and Skip(%d) both describe themselves as %q", s, other, got)
		}
		seen[got] = s
	}
	if got := Skip(200).String(); got != "unknown" {
		t.Errorf("Skip(200).String() = %q, want %q", got, "unknown")
	}
}

// pathExt is path.Ext, spelled here so the test file does not import path only
// to ask one question.
func pathExt(p string) string {
	for i := len(p) - 1; i >= 0 && p[i] != '/'; i-- {
		if p[i] == '.' {
			return p[i:]
		}
	}
	return ""
}
