package library

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/text/unicode/norm"
)

// The tests in this file are 003 T3's, and they sit at the `library` level of
// 003 tasks.md's three: a Go test beside the package, asserting a function at
// the layer of the function.
//
// # What none of them proves
//
// **That the string a client receives is the string that was stored.** Nothing
// in this feature can: 003 registers no route, `surface.yaml` has no row for it,
// and there is no wire representation of an item anywhere in the build. This is
// [003 plan §8.3]'s first row, and it is discharged at **005** by one `/Items`
// listing whose `Id` and `ParentId` are compared byte for byte against a golden.
//
// A green run here is not evidence for that claim, and 005 must not read it as
// any. What is proven here is the derivation; T15 proves the string the store
// holds, across three scans; the step from the store to a body belongs to the
// feature that has one.
//
// [003 plan §8.3]: ../../specs/003-library-configuration-and-scanning/plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven

// testLibraryID is a library's allocated identity, as a literal. It is not
// derived from anything, which is the point of it: 003 §3.6 allocates a
// library's identity and keeps it, so that a rename and a remount cost nothing.
const testLibraryID = "0123456789abcdef0123456789abcdef"

// TestADecomposedAndAPrecomposedSpellingOfOneNameDeriveOneIdentifier is the
// second of 003 §3.6's three normalisation steps, asserted over the character
// the fixture actually contains.
//
// The name is `Amélie (2001).mkv`, and the two spellings are U+00E9 LATIN SMALL
// LETTER E WITH ACUTE and `e` followed by U+0301 COMBINING ACUTE ACCENT. One
// filesystem hands back the first and another the second — they are the same
// file, and an identifier that differed between them would discard every
// favourite a user has for that film the day the library moved between two
// machines.
//
// It is asserted for **both** settings of case sensitivity, because the Unicode
// form is normalised whatever the operator said about case: a case-sensitive
// library still gets one item for one file.
func TestADecomposedAndAPrecomposedSpellingOfOneNameDeriveOneIdentifier(t *testing.T) {
	const (
		precomposed = "Am\u00e9lie (2001).mkv"
		decomposed  = "Ame\u0301lie (2001).mkv"
	)

	if precomposed == decomposed {
		t.Fatalf("the two spellings are the same string, so this test asserts nothing")
	}

	for _, caseSensitive := range []bool{false, true} {
		first, err := Normalise(precomposed, caseSensitive)
		if err != nil {
			t.Fatalf("Normalise(%q, %t): %v", precomposed, caseSensitive, err)
		}
		second, err := Normalise(decomposed, caseSensitive)
		if err != nil {
			t.Fatalf("Normalise(%q, %t): %v", decomposed, caseSensitive, err)
		}
		if first != second {
			t.Errorf("caseSensitive=%t: the two spellings normalise apart: %q against %q",
				caseSensitive, first, second)
		}

		a := DeriveID(testLibraryID, KindMovie, first)
		b := DeriveID(testLibraryID, KindMovie, second)
		if a != b {
			t.Errorf("caseSensitive=%t: one file, two identifiers: %s against %s", caseSensitive, a, b)
		}
	}
}

// TestCapitalisationIsTheWholeOfWhatCaseSensitiveDecides asserts both halves of
// 003 §3.6's per-library setting, which is why that section freezes it.
//
// A name differing only in its capitalisation derives **one** identifier when
// the library is case-insensitive — Atrium's default, and its stated
// divergence from the reference's own (spec OQ-9, 003 plan §6.3) — and **two**
// when the operator declared the library case-sensitive because they have two
// files that differ only that way.
//
// Asserting only the first half passes on a build that ignores the parameter
// entirely, which is the whole setting doing nothing.
//
// The name carries an accent on purpose. `THE MATRIX` against `the matrix` is
// met by an ASCII fold, and this package has one — `foldASCIICase`, which every
// extension comparison uses because the reference compares those ordinally. It
// is the wrong fold here and a case pair that is entirely ASCII would not say
// so: `AMÉLIE` and `amélie` are the same directory, and an ASCII fold gives them
// two identifiers.
func TestCapitalisationIsTheWholeOfWhatCaseSensitiveDecides(t *testing.T) {
	const (
		lower = "the matrix (1999)/am\u00e9lie the matrix (1999).mkv"
		upper = "The Matrix (1999)/AM\u00c9LIE The Matrix (1999).mkv"
	)

	insensitiveA, err := Normalise(lower, false)
	if err != nil {
		t.Fatalf("Normalise(%q, false): %v", lower, err)
	}
	insensitiveB, err := Normalise(upper, false)
	if err != nil {
		t.Fatalf("Normalise(%q, false): %v", upper, err)
	}
	if got, want := DeriveID(testLibraryID, KindMovie, insensitiveA), DeriveID(testLibraryID, KindMovie, insensitiveB); got != want {
		t.Errorf("case-insensitive: two identifiers for one directory renamed in its capitalisation: %s against %s", got, want)
	}

	sensitiveA, err := Normalise(lower, true)
	if err != nil {
		t.Fatalf("Normalise(%q, true): %v", lower, err)
	}
	sensitiveB, err := Normalise(upper, true)
	if err != nil {
		t.Fatalf("Normalise(%q, true): %v", upper, err)
	}
	if a, b := DeriveID(testLibraryID, KindMovie, sensitiveA), DeriveID(testLibraryID, KindMovie, sensitiveB); a == b {
		t.Errorf("case-sensitive: two files differing only in case share one identifier %s", a)
	}
}

// TestTheUnicodeFormIsNormalisedBeforeTheCaseIsFolded asserts 003 §3.6's two
// last steps in the order that section states them, and it names the character
// it relies on.
//
// **Case folding is not closed over normalisation forms.** The input is `I`
// followed by U+0307 COMBINING DOT ABOVE, and the two orders answer differently:
//
//	NFC then fold  →  U+0130 →  "i"             — one rune
//	fold then NFC  →  "i" U+0307                — two runes, nothing to compose
//
// U+0130 LATIN CAPITAL LETTER I WITH DOT ABOVE is what NFC composes the pair
// into, and its simple lowercase mapping is a plain `i`; there is no precomposed
// character for a lowercase `i` with a dot above, so the other order has nothing
// to compose and keeps the combining mark.
//
// The test computes both orders itself and **fails if they agree**, because an
// input over which the order is invisible would make the rest of this test
// green whatever the implementation did — an assertion that looks thorough and
// asserts nothing. That is the trap this test exists to avoid rather than to
// demonstrate.
func TestTheUnicodeFormIsNormalisedBeforeTheCaseIsFolded(t *testing.T) {
	// "I" U+0307, inside a name so that the key is a plausible one.
	const decomposedCapital = "I\u0307stanbul (2019).mkv"

	formThenFold := strings.ToLower(norm.NFC.String(decomposedCapital))
	foldThenForm := norm.NFC.String(strings.ToLower(decomposedCapital))

	if formThenFold == foldThenForm {
		t.Fatalf("the two orders agree over %q, so this test could not distinguish them; "+
			"it needs an input where U+0307 makes the order observable", decomposedCapital)
	}

	got, err := Normalise(decomposedCapital, false)
	if err != nil {
		t.Fatalf("Normalise(%q, false): %v", decomposedCapital, err)
	}
	if got != formThenFold {
		t.Errorf("Normalise folded before it composed: got %q (%s), want %q (%s)",
			got, runeNames(got), formThenFold, runeNames(formThenFold))
	}
	if got == foldThenForm {
		t.Errorf("Normalise answers the wrong order's answer %q (%s)", got, runeNames(got))
	}
}

// TestSeparatorsAreReducedToOneFormBeforeAnythingElse is 003 §3.6's first step:
// a walker on one platform yields one separator and on another the other, and
// the same directory must be one key either way. The `.` element, the repeated
// separator and the trailing one are the same rule — two spellings of one path.
func TestSeparatorsAreReducedToOneFormBeforeAnythingElse(t *testing.T) {
	const want = "shows/the series/season 01"

	for _, spelling := range []string{
		`shows\the series\season 01`,
		"shows/the series/season 01/",
		"shows//the series/season 01",
		"./shows/the series/season 01",
		`shows\the series/season 01`,
		"shows/the series/season 02/../season 01",
	} {
		got, err := Normalise(spelling, false)
		if err != nil {
			t.Fatalf("Normalise(%q, false): %v", spelling, err)
		}
		if got != want {
			t.Errorf("Normalise(%q) = %q, want %q", spelling, got, want)
		}
	}
}

// TestAnAbsoluteKeyAndOneThatClimbsAboveItsRootAreErrorsAndNotNormalisations is
// 003 §3.6's refusal, and 003 plan §7's row: the library's scan fails and
// nothing is written, because *"a caller holding a path it believes is relative
// and is not has computed the wrong root, not the wrong file"*.
//
// The drive-letter form is here because [Normalise] must answer the same on
// every platform: `path/filepath.IsAbs` says `C:\Movies` is relative on Linux,
// and a key that normalised on one build host and not another would be the
// worst available failure.
func TestAnAbsoluteKeyAndOneThatClimbsAboveItsRootAreErrorsAndNotNormalisations(t *testing.T) {
	cases := []struct {
		key  string
		want error
	}{
		{"/Movies/The Matrix (1999).mkv", ErrPathAbsolute},
		{`\Movies\The Matrix (1999).mkv`, ErrPathAbsolute},
		{`C:\Movies\The Matrix (1999).mkv`, ErrPathAbsolute},
		{`c:/Movies/The Matrix (1999).mkv`, ErrPathAbsolute},
		{`\\host\share\The Matrix (1999).mkv`, ErrPathAbsolute},
		{"../Movies/The Matrix (1999).mkv", ErrPathClimbsAboveRoot},
		{"..", ErrPathClimbsAboveRoot},
		{"Movies/../../The Matrix (1999).mkv", ErrPathClimbsAboveRoot},
		{`Movies\..\..\The Matrix (1999).mkv`, ErrPathClimbsAboveRoot},
	}

	for _, c := range cases {
		for _, caseSensitive := range []bool{false, true} {
			got, err := Normalise(c.key, caseSensitive)
			if err == nil {
				t.Errorf("Normalise(%q, %t) normalised to %q; it must refuse", c.key, caseSensitive, got)
				continue
			}
			if !errors.Is(err, c.want) {
				t.Errorf("Normalise(%q, %t) = %v, want it to be %v", c.key, caseSensitive, err, c.want)
			}
			if got != "" {
				t.Errorf("Normalise(%q, %t) returned the key %q beside its error; "+
					"a caller must not be able to carry on with what it got back", c.key, caseSensitive, got)
			}

			var pathErr *PathError
			if !errors.As(err, &pathErr) {
				t.Fatalf("Normalise(%q, %t) returned %T, want a *PathError", c.key, caseSensitive, err)
			}
			if pathErr.Path != c.key {
				t.Errorf("the error names %q, want the key as it was passed, %q", pathErr.Path, c.key)
			}
			// The message quotes the key, which is what makes a key with a
			// trailing space or a backslash in it readable in a log line at
			// all, so the assertion is over the quoted form.
			if !strings.Contains(err.Error(), strconv.Quote(c.key)) {
				t.Errorf("the message %q does not name the key %q the operator has to find", err.Error(), c.key)
			}
		}
	}

	if errors.Is(ErrPathAbsolute, ErrPathClimbsAboveRoot) || errors.Is(ErrPathClimbsAboveRoot, ErrPathAbsolute) {
		t.Errorf("the two refusals are one sentinel, so a caller cannot tell them apart")
	}
}

// TestARefusedKeyIsNotAFileThatWasSkipped is the clause T13 turns on, asserted
// as the difference it actually is rather than as a message.
//
// T13 has to **fail a whole library** on a key that will not normalise, while a
// [Skip] refuses one file and is not a fault at all (003 plan §7). So the two
// must be distinguishable, and the sharpest way to say that is this: the very
// same strings are *candidates* as far as every path rule in §3.2 is concerned.
// [CollectionType.Candidate] answers `(true, NotSkipped)` for
// `/Movies/The Matrix (1999).mkv` — nothing about it is hidden, an extra or a
// refused extension — and [Normalise] refuses it. A caller that consulted only
// the skip vocabulary would carry on and write items under a root it does not
// have.
//
// The companion assertion is that [Skip] is not an error: the two channels
// cannot be crossed by accident, because a skip reason cannot be returned in an
// error position.
func TestARefusedKeyIsNotAFileThatWasSkipped(t *testing.T) {
	for _, key := range []string{
		"/Movies/The Matrix (1999).mkv",
		"../Movies/The Matrix (1999).mkv",
	} {
		candidate, skip := Movies.Candidate(key)
		if !candidate || skip != NotSkipped {
			t.Fatalf("Candidate(%q) = (%t, %v); this test needs a key the skip vocabulary passes, "+
				"or it is not asserting that the two channels differ", key, candidate, skip)
		}
		if _, err := Normalise(key, false); err == nil {
			t.Errorf("Normalise(%q) accepted a key that must fail the library's scan", key)
		}
	}

	var asError any = NotSkipped
	if _, isError := asError.(error); isError {
		t.Errorf("Skip implements error, so a normalisation failure and a skipped file " +
			"can be returned through one channel; 003 plan §7 makes them different outcomes")
	}
}

// TestAnInteriorParentElementThatStaysInsideTheRootIsANormalisation is the
// other side of the refusal, and it is what makes ErrPathClimbsAboveRoot's name
// true rather than approximate. `a/../b` does not climb above anything.
func TestAnInteriorParentElementThatStaysInsideTheRootIsANormalisation(t *testing.T) {
	got, err := Normalise("Movies/The Matrix (1999)/../The Matrix (1999).mkv", true)
	if err != nil {
		t.Fatalf("Normalise: %v", err)
	}
	if want := "Movies/The Matrix (1999).mkv"; got != want {
		t.Errorf("Normalise = %q, want %q", got, want)
	}
}

// TestTheSeparatorsAreWhatStopTheThreeInputsColliding is the collision a
// concatenation without separators produces, and `internal/sessions` had to
// assert exactly this once already for a session identifier (002 T9).
//
// Both joins are asserted, not just the first: a build that separated the
// library from the kind and then ran the kind straight into the key would pass
// a test that only checked `("ab", k, "c")` against `("a", k, "bc")`.
func TestTheSeparatorsAreWhatStopTheThreeInputsColliding(t *testing.T) {
	if a, b := DeriveID("ab", KindMovie, "c"), DeriveID("a", KindMovie, "bc"); a == b {
		t.Errorf(`DeriveID("ab", Movie, "c") and DeriveID("a", Movie, "bc") are one identifier %s`, a)
	}

	// The first join, where the library identifier runs into the kind. The
	// pair above cannot see this one: with only the first NUL missing, its two
	// arguments still differ on either side of the second. Only the golden
	// would have caught it, and a golden says *what* moved rather than *why*.
	if a, b := DeriveID("libMov", Kind("ie"), "x.mkv"), DeriveID("lib", KindMovie, "x.mkv"); a == b {
		t.Errorf("the library identifier runs into the kind: both derive %s", a)
	}

	// The second join, where the kind runs into the key. `Kind` is a string
	// type and a caller can spell one that is not among the eight, which is
	// exactly what a resolver with a typo does.
	if a, b := DeriveID("lib", Kind("Movi"), "e/x.mkv"), DeriveID("lib", KindMovie, "/x.mkv"); a == b {
		t.Errorf("the kind runs into the key: both derive %s", a)
	}
}

// TestEveryKindDerivesADifferentIdentifierFromOneKey is why the kind is in the
// key at all: a directory and the item it backs must not collide.
//
// 003's own instance of it is a `Series` at `Shows/The Series` and a `Season`
// keyed on the same string — the fixture has that directory, and an inferred
// season with no directory of its own would otherwise be able to land on the
// series' identifier. The loop generalises it to all eight of plan §4.2's
// types, because the pair is not special.
func TestEveryKindDerivesADifferentIdentifierFromOneKey(t *testing.T) {
	const key = "shows/the series"

	if DeriveID(testLibraryID, KindSeries, key) == DeriveID(testLibraryID, KindSeason, key) {
		t.Errorf("a Series and a Season on %q derive one identifier", key)
	}

	seen := make(map[string]Kind, len(AllKinds()))
	for _, kind := range AllKinds() {
		id := DeriveID(testLibraryID, kind, key)
		if other, ok := seen[id]; ok {
			t.Errorf("%s and %s both derive %s from %q", other, kind, id, key)
		}
		seen[id] = kind
	}
	if len(seen) != 8 {
		t.Errorf("plan §4.2 names eight types and AllKinds gave %d", len(seen))
	}
}

// TestTheLibraryIsInTheKeyAndTheRootPathIsNot asserts 003 plan §6.3's sentence
// as the pair of inequalities it is, because either half alone is met by a
// build that gets the other wrong.
//
//   - **Two libraries configured over the same tree derive different
//     identifiers.** The library's allocated identity is an input, so one tree
//     scanned by two libraries is two sets of items and not one shared set.
//   - **A library whose root path moved derives the same ones.** The root path
//     is not an input, and cannot become one: the key is relative to the root,
//     and [Normalise] refuses an absolute key outright — which is the structural
//     reason a root cannot leak into an identifier by accident.
//
// The reference has the second half wrong and it is measured: every one of 448
// identifiers is reproducible from the file's absolute path alone
// `[probe: tools/probe_item_identity.py, Jellyfin 10.11.11, 2026-08-27]`, so
// moving a root there silently discards every favourite and resume position in
// the library ([behaviours §1.4]). Atrium does not inherit that, and AC-10 is
// the criterion; this is its `library`-level half, and the scan-move-scan half
// is AC-10's own at the subcommand.
//
// [behaviours §1.4]: ../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters
func TestTheLibraryIsInTheKeyAndTheRootPathIsNot(t *testing.T) {
	const (
		before   = "/mnt/a/media/movies"
		after    = "/srv/library/films"
		absolute = "The Matrix (1999)/The Matrix (1999).mkv"
	)
	const secondLibraryID = "fedcba9876543210fedcba9876543210"

	// The move, as a scan sees it: the same file, under two roots.
	movedKey, err := Normalise(keyUnderRoot(t, before, before+"/"+absolute), false)
	if err != nil {
		t.Fatalf("Normalise: %v", err)
	}
	sameKey, err := Normalise(keyUnderRoot(t, after, after+"/"+absolute), false)
	if err != nil {
		t.Fatalf("Normalise: %v", err)
	}
	if a, b := DeriveID(testLibraryID, KindMovie, movedKey), DeriveID(testLibraryID, KindMovie, sameKey); a != b {
		t.Errorf("moving the root from %q to %q changed the identifier: %s became %s", before, after, a, b)
	}

	// And the root cannot become an input by accident, because the absolute
	// path is not a key at all.
	if _, err := Normalise(before+"/"+absolute, false); err == nil {
		t.Errorf("an absolute path normalised; the root would then be inside every identifier under it")
	}

	if a, b := DeriveID(testLibraryID, KindMovie, movedKey), DeriveID(secondLibraryID, KindMovie, movedKey); a == b {
		t.Errorf("two libraries over one tree derive one identifier %s for %q", a, movedKey)
	}
}

// keyUnderRoot is what a walk hands [Normalise]: the path of a file, relative
// to the root it was found under. It is in the test rather than in the package
// because the walk is T8's and this is only the shape of its output.
func keyUnderRoot(t *testing.T, root, absolute string) string {
	t.Helper()
	rest, found := strings.CutPrefix(absolute, root+"/")
	if !found {
		t.Fatalf("%q is not under %q", absolute, root)
	}
	return rest
}

// TestDifferentKeysDeriveDifferentIdentifiers is the assertion the golden below
// cannot make, and 002 found out twice why it has to be written separately: a
// `DeriveID` returning a **constant equal to the pinned literal** passes a
// golden, because a constant is stable across runs. Determinism and
// distinctness are two assertions.
func TestDifferentKeysDeriveDifferentIdentifiers(t *testing.T) {
	keys := []string{
		"the matrix (1999)/the matrix (1999).mkv",
		"the long film (1998)/the long film (1998) - part1.mkv",
		"the long film (1998)/the long film (1998) - part2.mkv",
		"wall-e (2008).mkv",
		"irobot (2004).mkv",
		"",
	}

	seen := make(map[string]string, len(keys))
	for _, key := range keys {
		id := DeriveID(testLibraryID, KindMovie, key)
		if other, ok := seen[id]; ok {
			t.Errorf("%q and %q both derive %s", other, key, id)
		}
		seen[id] = key
	}
}

// TestTheSameInputsDeriveTheSameIdentifier is Principle VII stated at the
// smallest scale there is: a rescan calls this again and must not invalidate a
// client's caches, favourites and resume positions.
func TestTheSameInputsDeriveTheSameIdentifier(t *testing.T) {
	const key = "the matrix (1999)/the matrix (1999).mkv"
	first := DeriveID(testLibraryID, KindMovie, key)
	for i := 0; i < 100; i++ {
		if got := DeriveID(testLibraryID, KindMovie, key); got != first {
			t.Fatalf("call %d derived %s where the first derived %s", i, got, first)
		}
	}
}

// TestTheDerivationIsPinnedAgainstAGolden pins the bytes, so that the
// derivation is fixed rather than merely shaped.
//
// A test asserting only *"32 lowercase hexadecimal characters"* is met by every
// SHA-256 truncation there is, and by an MD5, and by a build that reordered the
// three inputs. The day any of those lands, these eight literals move — and
// every golden body in this project that carries an item's `Id` moves with them,
// which is the cost this pin exists to make visible.
//
// The library identifier is the literal [testLibraryID] and the keys are already
// normalised, because [DeriveID] does not normalise for itself.
func TestTheDerivationIsPinnedAgainstAGolden(t *testing.T) {
	golden := []struct {
		kind Kind
		key  string
		want string
	}{
		{KindCollectionFolder, testLibraryID, "066742a2bc4e45c060d72ad3543f7afe"},
		{KindMovie, "the matrix (1999)/the matrix (1999).mkv", "0a9b6fc62ecead526ddc968b9500877c"},
		{KindSeries, "the series", "471853fe7e504280864c874a267c10d1"},
		{KindSeason, "the series/2", "f6acb9373c48cb16e480f1068257be09"},
		{KindEpisode, "the series/season 01/the series - s01e01 - pilot.mkv", "f3b42d9e1b9e58095ba0131b2531e944"},
		{KindMusicArtist, "the artist", "a148b6bc03badcba25e870c980c10813"},
		{KindMusicAlbum, "the artist/first album", "2ef15f9fae88b95dfc69d2b90314eb4f"},
		{KindAudio, "the artist/first album/01 first track.flac", "aa920479c055e86b8f014a47883fd019"},
	}

	for _, g := range golden {
		if got := DeriveID(testLibraryID, g.kind, g.key); got != g.want {
			t.Errorf("DeriveID(%s, %q) = %s, want %s", g.kind, g.key, got, g.want)
		}
	}
}

// TestAnIdentifierIsThirtyTwoLowercaseHexadecimalCharacters is
// [behaviours §1.4]'s shape, asserted over inputs that could break it — an
// empty key, a key of one byte, a key holding characters outside ASCII, and a
// long one — because the shape is what every client parses.
//
// [behaviours §1.4]: ../../docs/compatibility/behaviours.md#14-item-identifiers-are-32-lowercase-hex-characters
func TestAnIdentifierIsThirtyTwoLowercaseHexadecimalCharacters(t *testing.T) {
	keys := []string{
		"",
		"a",
		"am\u00e9lie (2001).mkv",
		strings.Repeat("a very long directory name/", 40) + "x.mkv",
	}

	for _, kind := range AllKinds() {
		for _, key := range keys {
			id := DeriveID(testLibraryID, kind, key)
			if len(id) != 32 {
				t.Errorf("DeriveID(%s, %q) is %d characters, want 32", kind, key, len(id))
			}
			for i := 0; i < len(id); i++ {
				c := id[i]
				if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
					t.Errorf("DeriveID(%s, %q) = %q, which is not lowercase hexadecimal", kind, key, id)
					break
				}
			}
		}
	}
}

// TestTheEightKindsAreSpelledAsTheStoredColumnHoldsThem pins the eight strings
// of 003 plan §4.2's `items.type`.
//
// They are not internal names. They are what the column holds, what
// `ports.ScannedItem.Type` carries, what the expected item set's rows say, and
// what 005 eventually puts on a wire — and, because the kind is an input to
// every identifier, renaming one silently rewrites every identifier of that
// type.
func TestTheEightKindsAreSpelledAsTheStoredColumnHoldsThem(t *testing.T) {
	want := []Kind{
		"CollectionFolder", "Movie", "Series", "Season",
		"Episode", "MusicArtist", "MusicAlbum", "Audio",
	}

	got := AllKinds()
	if len(got) != len(want) {
		t.Fatalf("AllKinds returned %d kinds, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("AllKinds()[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	first := AllKinds()
	first[0] = "mutated"
	if AllKinds()[0] != KindCollectionFolder {
		t.Errorf("AllKinds returns the package's own slice, so one caller can reorder it under another")
	}
}

// runeNames spells a string as its code points, so that a failure over
// combining characters is readable in the test output rather than being two
// strings that look identical.
func runeNames(s string) string {
	var b strings.Builder
	for i, r := range []rune(s) {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(codePoint(r))
	}
	return b.String()
}

func codePoint(r rune) string {
	const hexDigits = "0123456789ABCDEF"
	out := []byte{'U', '+'}
	for shift := 12; shift >= 0; shift -= 4 {
		out = append(out, hexDigits[(r>>uint(shift))&0xf])
	}
	return string(out)
}
