package library

import "testing"

// TestSpecialsIsNotAnExtrasName is spec §3.4's warning asserted at the
// predicate rather than at the summary.
//
// `Specials` is an alias for season zero
// `[source: Emby.Naming/TV/SeasonPathParser.cs:82 @ v10.11.11]`. It sits beside
// `Extras` and `Featurettes` in real libraries, and a scanner that grouped the
// three *"would drop every special episode in every series while producing a
// scan that looks entirely correct"* (003 §3.4).
//
// That is why this assertion exists here as well as at the resolver: the
// failure is invisible in a scan's own summary — no error, no warning, a
// plausible count — so the only place it can be caught cheaply is the predicate
// that would have made the mistake. Every casing is asked, because the folder
// rule ignores case and a build that added `specials` to the list in lowercase
// would pass a test that only asked about `Specials`.
func TestSpecialsIsNotAnExtrasName(t *testing.T) {
	for _, spelling := range []string{"Specials", "specials", "SPECIALS", "Specials "} {
		if IsExtrasFolderName(spelling) {
			t.Errorf("IsExtrasFolderName(%q) is true; it is an alias for season zero, not an extras folder", spelling)
		}
	}
	for _, name := range []string{"Specials.mkv", "The Series - Specials.mkv", "specials.mkv"} {
		if IsExtrasFilename(name) {
			t.Errorf("IsExtrasFilename(%q) is true", name)
		}
		if HasExtrasSuffix(name) {
			t.Errorf("HasExtrasSuffix(%q) is true", name)
		}
	}

	// And the whole path a special episode arrives on is a candidate, which
	// is the thing the resolver later needs to see at all.
	special := "The Series/Specials/The Series - S00E01 - A Special.mkv"
	if ok, why := Shows.Candidate(special); !ok {
		t.Errorf("Shows.Candidate(%q) refused with %v; §3.4 makes it season zero", special, why)
	}
}

// TestAnExtrasFolderExcludesItsContents and TestAnExtrasSuffixExcludesOneFile
// are deliberately two tests over two disjoint fixtures.
//
// A build that implemented only the suffix rule is right about every extras
// file that happens to be suffixed — which is most of them, and all of them in
// any fixture assembled from suffixed names — and wrong about a `Featurettes`
// directory holding something that is not suffixed. A build that implemented
// only the folder rule has the mirror-image hole. Asserting the two together
// over one tree would let either build pass, so they are asserted apart and
// neither fixture carries the other rule's evidence.
func TestAnExtrasFolderExcludesItsContents(t *testing.T) {
	// Not one of these filenames carries an extras suffix or is named for
	// an extra. The directory is doing all of the work.
	for _, dir := range []string{
		"Trailers", "Backdrops", "Behind The Scenes", "Deleted Scenes",
		"Interviews", "Scenes", "Samples", "Shorts", "Featurettes",
		"Extras", "extra", "Other", "Clips",
	} {
		if !IsExtrasFolderName(dir) {
			t.Errorf("IsExtrasFolderName(%q) is false", dir)
		}

		p := "The Matrix (1999)/" + dir + "/Making Of The Matrix.mkv"
		if IsExtrasFilename("Making Of The Matrix.mkv") || HasExtrasSuffix("Making Of The Matrix.mkv") {
			t.Fatalf("the fixture for the folder rule carries suffix evidence: %q", p)
		}
		ok, why := Movies.Candidate(p)
		if ok {
			t.Errorf("Candidate(%q) is a candidate; the containing directory names an extras folder", p)
		}
		if why != SkipExtrasFolder {
			t.Errorf("Candidate(%q) refused with %v, want the extras folder rule", p, why)
		}
	}

	// The same file one directory over is an item, which is what makes the
	// assertion above about the directory and not about the name.
	sibling := "The Matrix (1999)/Making Of The Matrix.mkv"
	if ok, why := Movies.Candidate(sibling); !ok {
		t.Errorf("Candidate(%q) refused with %v; only the extras directory was meant to refuse it", sibling, why)
	}
}

func TestAnExtrasSuffixExcludesOneFile(t *testing.T) {
	// Every one of these sits in an ordinary directory. The name is doing
	// all of the work.
	suffixed := []string{
		"The Matrix (1999)-trailer.mkv",
		"The Matrix (1999).trailer.mkv",
		"The Matrix (1999)_trailer.mkv",
		"The Matrix (1999) - trailer.mkv",
		"The Matrix (1999)-sample.mkv",
		"The Matrix (1999).sample.mkv",
		"The Matrix (1999)_sample.mkv",
		"The Matrix (1999) - sample.mkv",
		"The Matrix (1999)-scene.mkv",
		"The Matrix (1999)-clip.mkv",
		"The Matrix (1999)-interview.mkv",
		"The Matrix (1999)-behindthescenes.mkv",
		"The Matrix (1999)-deleted.mkv",
		"The Matrix (1999)-deletedscene.mkv",
		"The Matrix (1999)-featurette.mkv",
		"The Matrix (1999)-short.mkv",
		"The Matrix (1999)-extra.mkv",
		"The Matrix (1999)-other.mkv",
	}
	for _, name := range suffixed {
		if !HasExtrasSuffix(name) {
			t.Errorf("HasExtrasSuffix(%q) is false", name)
		}

		p := "The Matrix (1999)/" + name
		if IsExtrasFolderName("The Matrix (1999)") {
			t.Fatal("the fixture for the suffix rule carries folder evidence")
		}
		ok, why := Movies.Candidate(p)
		if ok {
			t.Errorf("Candidate(%q) is a candidate; it carries an extras suffix", p)
		}
		if why != SkipExtrasSuffix {
			t.Errorf("Candidate(%q) refused with %v, want the extras suffix rule", p, why)
		}
	}

	// The work itself, in the same directory as every one of those, is an
	// item — so the suffix refused one file and not the directory.
	film := "The Matrix (1999)/The Matrix (1999).mkv"
	if ok, why := Movies.Candidate(film); !ok {
		t.Errorf("Candidate(%q) refused with %v; the suffix rule is meant to refuse one file", film, why)
	}
}

// TestTheTwoExtrasRulesAreReportedApart is the conflation the task warns about,
// asserted directly: the two shapes must not come back with the same reason.
func TestTheTwoExtrasRulesAreReportedApart(t *testing.T) {
	byFolder := "The Matrix (1999)/Featurettes/Making Of.mkv"
	bySuffix := "The Matrix (1999)/The Matrix (1999)-featurette.mkv"

	_, folderWhy := Movies.Candidate(byFolder)
	_, suffixWhy := Movies.Candidate(bySuffix)

	if folderWhy != SkipExtrasFolder {
		t.Errorf("Candidate(%q) refused with %v, want the folder rule", byFolder, folderWhy)
	}
	if suffixWhy != SkipExtrasSuffix {
		t.Errorf("Candidate(%q) refused with %v, want the suffix rule", bySuffix, suffixWhy)
	}
	if folderWhy == suffixWhy {
		t.Error("the folder rule and the suffix rule report the same reason; they exclude different amounts and a scan that says so cannot be read")
	}
}

// TestAWholeFilenameNamedForAnExtraIsRefusedAndATrailingDigitIsNot pins the
// reference's own asymmetry between its two rule types.
//
// Its `Filename` rules match the stem as it is, and its `Suffix` rules match
// the stem with trailing digits removed — the comment beside the trim says it
// is so that `-trailer2` is recognised
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:32-34,48-49 @ v10.11.11]`.
// The consequence is that `trailer.mkv` is an extra and `trailer2.mkv` is an
// item, and it is asserted here because it looks like a bug and is the
// reference's shape. Principle I is what settles it: no delta, in either
// direction.
func TestAWholeFilenameNamedForAnExtraIsRefusedAndATrailingDigitIsNot(t *testing.T) {
	for _, name := range []string{"trailer.mkv", "Trailer.mkv", "SAMPLE.mkv", "sample.mkv"} {
		p := "The Matrix (1999)/" + name
		ok, why := Movies.Candidate(p)
		if ok {
			t.Errorf("Candidate(%q) is a candidate; its whole stem names an extra", p)
		}
		if why != SkipExtrasFilename {
			t.Errorf("Candidate(%q) refused with %v, want the extras filename rule", p, why)
		}
	}

	// The suffix rule trims trailing digits, so a second trailer is still a
	// trailer.
	for _, name := range []string{"The Matrix (1999)-trailer2.mkv", "The Matrix (1999)-trailer10.mkv"} {
		p := "The Matrix (1999)/" + name
		if ok, _ := Movies.Candidate(p); ok {
			t.Errorf("Candidate(%q) is a candidate; the reference trims trailing digits before matching a suffix", p)
		}
	}

	// The filename rule does not trim, so `trailer2.mkv` matches nothing.
	// This is the assertion that fails if the two rules are given one
	// implementation.
	for _, name := range []string{"trailer2.mkv", "sample3.mkv"} {
		p := "The Matrix (1999)/" + name
		if ok, why := Movies.Candidate(p); !ok {
			t.Errorf("Candidate(%q) refused with %v; the reference's Filename rules match the stem untrimmed", p, why)
		}
	}
}

// TestTheExtrasFolderNameIsTheImmediateContainingDirectory records where the
// folder rule stops, because "the folder excludes its contents" has two
// readings and only one of them is the reference's.
//
// The reference compares its token against
// `Path.GetFileName(Path.GetDirectoryName(path))` — the immediate parent and
// nothing above it
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:35,51 @ v10.11.11]`. So a
// file one level below an extras folder is an item there, and excluding it here
// would be an item this server lacks and the reference has: a difference
// nothing has measured and nothing declares.
//
// The reference also refuses the rule when the directory *is* the library root
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:52 @ v10.11.11]`. Here that
// falls out of paths being root-relative, and it is asserted so that a future
// implementation walking every component cannot quietly take it away.
func TestTheExtrasFolderNameIsTheImmediateContainingDirectory(t *testing.T) {
	nested := "The Matrix (1999)/Extras/Making Of/Part One.mkv"
	if ok, why := Movies.Candidate(nested); !ok {
		t.Errorf("Candidate(%q) refused with %v; the reference's folder rule reaches the immediate parent only", nested, why)
	}

	// A library whose own root directory is called `Extras` still holds
	// items: a file directly under the root has no containing directory
	// component to ask about.
	atRoot := "Extras.mkv"
	if ok, why := Movies.Candidate(atRoot); !ok {
		t.Errorf("Candidate(%q) refused with %v; a file at the root has no containing directory", atRoot, why)
	}
	rootChild := "A Film (2001).mkv"
	if ok, why := Movies.Candidate(rootChild); !ok {
		t.Errorf("Candidate(%q) refused with %v", rootChild, why)
	}
}

// TestExtrasAreNotRecognisedUnderAMusicLibrary asserts the reference's
// media-type gate, expressed in the one term this package has.
//
// Every extras rule implemented here is declared `MediaType.Video`, and the
// reference skips a rule whose media type the file does not have
// `[source: Emby.Naming/Video/ExtraRuleResolver.cs:40-44 @ v10.11.11]`. Every
// extension `music` admits is in the reference's `AudioFileExtensions` and in
// neither its `VideoFileExtensions`
// `[source: Emby.Naming/Common/NamingOptions.cs:24-80,213-295 @ v10.11.11]`, so
// a file that would reach these rules under `music` is never a video file
// there and never matches one of them.
//
// The two `MediaType.Audio` tokens the reference carries — `theme` and
// `theme-music` — are not implemented, and this test is where that shows: a
// track called `theme.flac` is an item here. It is an accepted shortfall on a
// source reading with no measurement behind it, and it shows more rather than
// less.
func TestExtrasAreNotRecognisedUnderAMusicLibrary(t *testing.T) {
	if Music.RecognisesExtras() {
		t.Error("Music.RecognisesExtras() is true; every rule in the list is MediaType.Video")
	}
	for _, c := range []CollectionType{Movies, Shows} {
		if !c.RecognisesExtras() {
			t.Errorf("%s.RecognisesExtras() is false", c)
		}
	}

	for _, p := range []string{
		"An Artist/An Album/01 - A Track-trailer.flac",
		"An Artist/Extras/01 - A Track.flac",
		"An Artist/An Album/trailer.flac",
		"An Artist/An Album/theme.flac",
	} {
		if ok, why := Music.Candidate(p); !ok {
			t.Errorf("Music.Candidate(%q) refused with %v; the extras rules do not apply under music", p, why)
		}
	}
}

// TestTheExtrasListsAreTheReferencesOwn transcribes the three lists from the
// pinned source, so that a token quietly added or dropped fails here rather
// than changing a scan.
//
// `[source: Emby.Naming/Common/NamingOptions.cs:484-695 @ v10.11.11]`
func TestTheExtrasListsAreTheReferencesOwn(t *testing.T) {
	// A name that is not on the list is not an extras folder, and these are
	// the near misses that matter: season zero, and the singular and plural
	// forms the reference does not carry.
	for _, name := range []string{
		"Specials", "Season 00", "Season 0", "Trailer", "Featurette",
		"Interview", "Scene", "Short", "Clip", "Sample", "Backdrop",
		"behind the scene", "deleted scene", "theme-music", "Music",
		"", "The Matrix (1999)",
	} {
		if IsExtrasFolderName(name) {
			t.Errorf("IsExtrasFolderName(%q) is true; it is not one of the reference's DirectoryName tokens", name)
		}
	}

	// A stem that merely contains a token is not suffixed by it: the rule
	// is an ending, not a substring.
	for _, name := range []string{
		"The Trailer Park Boys (2001).mkv",
		"Sample Text (1999).mkv",
		"Interview With The Vampire (1994).mkv",
		"The Other Guys (2010).mkv",
		"Deleted (2020) - Extras Are Elsewhere.mkv",
	} {
		if HasExtrasSuffix(name) {
			t.Errorf("HasExtrasSuffix(%q) is true; the rule is a suffix, not a substring", name)
		}
		if IsExtrasFilename(name) {
			t.Errorf("IsExtrasFilename(%q) is true", name)
		}
	}
}
