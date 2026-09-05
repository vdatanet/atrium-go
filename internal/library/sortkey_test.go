package library

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// The tests in this file are 003 T4's, and they sit at the `library` level of
// 003 tasks.md's three: a Go test beside the package, asserting a function at
// the layer of the function.
//
// # What none of them proves
//
// **That any list a client receives is ordered by this key.** This is
// [003 plan §8.3]'s second row. 003 registers no route and produces no wire
// representation of an item, so nothing here — and nothing anywhere in this
// feature — checks that `ORDER BY` names this column, or that it compares it
// with `BINARY`. T12 carries the collation half, at the store; 005 carries the
// `ORDER BY` and the body it orders.
//
// A green run in this file is evidence about a derivation and about nothing
// downstream of it. AC-13 is written as *"sort ordering matches the table"* and
// is discharged here against the **key**, one level below the ordering it
// names; plan §8.4 says so rather than leaving it to be discovered.
//
// [003 plan §8.3]: ../../specs/003-library-configuration-and-scanning/plan.md#83-what-only-becomes-observable-at-005-and-what-005-must-not-accept-as-proven

// delimiter wraps every expected value in the measured table below.
//
// It is not decoration. `s w a t ` ends in a space and `rock  roll` carries two
// in the middle: inside a bare Go string literal the first is invisible in a
// diff, invisible in review, and `gofmt` will not save it — it will not even
// notice. Wrapping both sides puts the boundary of the string on the screen, so
// a lost trailing space is a visible change to a literal rather than a change
// to nothing.
const delimiter = "|"

// TestTheBaseDerivationReproducesEveryMeasuredCase is 003 §3.7.1's table,
// verbatim and in its order, and it is the whole of what OQ-3 was closed on
// `[probe: tools/probe_sort_names.py, Jellyfin 10.11.11, 2026-08-26]`.
//
// The comparison is on the **whole string**. A per-character or per-word
// assertion is exactly the assertion that cannot see the two artefacts this
// derivation exists to preserve.
func TestTheBaseDerivationReproducesEveryMeasuredCase(t *testing.T) {
	cases := []struct {
		name      string
		delimited string // the expected key, between delimiters
		shows     string
	}{
		{"The Matrix", "|matrix|", "article at the start"},
		{"Matrix The", "|matrix|", "and at the end"},
		{"Once The Time", "|once time|", "and in the middle"},
		{"A Bridge", "|bridge|", "single-letter article"},
		{"Amélie", "|amelie|", "diacritics folded"},
		{"iRobot", "|irobot|", "case normalised"},
		{"2 Fast 2 Furious", "|0000000002 fast 0000000002 furious|", "every digit run, not just the leading one"},
		{"10 Things", "|0000000010 things|", "which is what makes 2 sort before 10"},
		{"Wall-E", "|walle|", "character removed"},
		{"Rock & Roll", "|rock  roll|", "two spaces — nothing collapses them"},
		{"Don't Look Up", "|dont look up|", "apostrophe removed"},
		{"S.W.A.T.", "|s w a t |", "trailing space — nothing trims it"},
		{"100% Wolf", "|0000000100  wolf|", "replacement and padding together"},
		{"  Padded  ", "|padded|", "trimmed at step 1, before anything else"},
	}

	if len(cases) != 14 {
		t.Fatalf("the table has %d rows, and §3.7.1's has 14", len(cases))
	}

	for _, c := range cases {
		// A row whose expected value lost a delimiter would compare a
		// differently shaped string and could pass by accident.
		if !strings.HasPrefix(c.delimited, delimiter) || !strings.HasSuffix(c.delimited, delimiter) {
			t.Fatalf("the expected value for %q is not delimited: %s", c.name, c.delimited)
		}

		got := delimiter + SortKeyBase(c.name) + delimiter
		if got != c.delimited {
			t.Errorf("SortKeyBase(%q) = %s, want %s (%s)", c.name, got, c.delimited, c.shows)
		}
	}
}

// TestTheThreeOverridingTypesUseTheirOwnAsymmetricWidths is 003 §3.7.2, and the
// asymmetry is the assertion.
//
// An episode's **season** is three digits while its **episode number** is four,
// and an audio track's disc and track are both four. That is not a
// transcription error and it is not tidied here
// `[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98,
// MediaBrowser.Controller/Entities/TV/Episode.cs:238-242,
// MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`.
//
// The numbers chosen are small on purpose: 1 and 2 padded to three and to four
// are different strings, so a build that used one width for both fails.
func TestTheThreeOverridingTypesUseTheirOwnAsymmetricWidths(t *testing.T) {
	cases := []struct {
		what      string
		item      ports.ScannedItem
		delimited string
	}{
		{
			what: "Audio pads disc and track to 4",
			item: ports.ScannedItem{
				Type: string(KindAudio), Name: "The Song",
				ParentIndexNumber: intp(1), IndexNumber: intp(3),
			},
			delimited: "|0001 - 0003 - The Song|",
		},
		{
			what: "Episode pads season to 3 and episode to 4",
			item: ports.ScannedItem{
				Type: string(KindEpisode), Name: "Pilot",
				ParentIndexNumber: intp(1), IndexNumber: intp(2),
			},
			delimited: "|001 - 0002 - Pilot|",
		},
		{
			what: "Season is four digits and nothing else",
			item: ports.ScannedItem{
				Type: string(KindSeason), Name: "Season 4",
				IndexNumber: intp(4),
			},
			delimited: "|0004|",
		},
	}

	for _, c := range cases {
		got := delimiter + SortKeyFor(&c.item) + delimiter
		if got != c.delimited {
			t.Errorf("%s: SortKeyFor = %s, want %s", c.what, got, c.delimited)
		}
	}
}

// TestAMissingNumberContributesNoSegmentAtAll is the clause of §3.7.2 that the
// obvious implementation gets wrong, and the assertion is the **absence of a
// run of zeros** rather than the presence of some other string.
//
// A build that formatted an absent number as `0000` produces a key that is
// perfectly well-formed and sorts every unnumbered track ahead of every
// numbered one, in every album at once. Asserting only the expected string
// would catch it; asserting the absence says what is being caught, and catches
// it for a width this table does not happen to use.
func TestAMissingNumberContributesNoSegmentAtAll(t *testing.T) {
	cases := []struct {
		what      string
		item      ports.ScannedItem
		delimited string
	}{
		{
			what: "an audio track with no disc number",
			item: ports.ScannedItem{
				Type: string(KindAudio), Name: "The Song", IndexNumber: intp(3),
			},
			delimited: "|0003 - The Song|",
		},
		{
			what: "an audio track with neither number",
			item: ports.ScannedItem{
				Type: string(KindAudio), Name: "The Song",
			},
			delimited: "|The Song|",
		},
		{
			what: "an episode with no season number",
			item: ports.ScannedItem{
				Type: string(KindEpisode), Name: "Pilot", IndexNumber: intp(2),
			},
			delimited: "|0002 - Pilot|",
		},
		{
			what: "an episode with no episode number",
			item: ports.ScannedItem{
				Type: string(KindEpisode), Name: "Pilot", ParentIndexNumber: intp(1),
			},
			delimited: "|001 - Pilot|",
		},
		{
			// Not hypothetical: §3.4 infers a season from an episode's
			// filename, and the reference's recorded reading of the fixture
			// tree names a `Season Unknown`. §3.7.2 states four digits and
			// nothing else and says nothing about having no number; the source
			// settles it as the raw name
			// `[source: MediaBrowser.Controller/Entities/TV/Season.cs:151 @ v10.11.11]`.
			what: "a season with no number",
			item: ports.ScannedItem{
				Type: string(KindSeason), Name: "Season Unknown",
			},
			delimited: "|Season Unknown|",
		},
	}

	for _, c := range cases {
		got := SortKeyFor(&c.item)
		if delimiter+got+delimiter != c.delimited {
			t.Errorf("%s: SortKeyFor = %s, want %s", c.what, delimiter+got+delimiter, c.delimited)
		}
		for _, zeros := range []string{"0000 - ", "000 - "} {
			if strings.HasPrefix(got, zeros) {
				t.Errorf("%s: SortKeyFor = %q, which begins with a run of zeros where the number was absent", c.what, got)
			}
		}
	}
}

// TestAnAudioKeyEndsInTheRawName is behaviours §2.6's second named temptation,
// asserted as the **raw name** and not as the absence of an article.
//
// *"Using one sort-name function for everything"* makes a track called
// `The Song` sort under `s` instead of `T` and reorders every album in the
// library. A test asserting `!strings.HasPrefix(key, "song")` would pass on a
// build that lowercased the name, or folded it, or stripped a character from
// it — every one of which is the same failure one step smaller. So the
// assertion is that the name arrives untouched, and the cost of the temptation
// is put beside it: the two derivations are shown to disagree over this name.
func TestAnAudioKeyEndsInTheRawName(t *testing.T) {
	const rawName = "The Song"

	item := ports.ScannedItem{
		Type: string(KindAudio), Name: rawName,
		ParentIndexNumber: intp(1), IndexNumber: intp(3),
	}

	got := SortKeyFor(&item)
	if !strings.HasSuffix(got, rawName) {
		t.Errorf("SortKeyFor over an Audio = %q, want it to end in the raw %q", got, rawName)
	}
	if got != "0001 - 0003 - "+rawName {
		t.Errorf("SortKeyFor over an Audio = %q, want %q", got, "0001 - 0003 - "+rawName)
	}

	// The temptation, priced. Nothing in this package lets a caller holding an
	// item reach the left-hand side, which is the design; this asserts that the
	// design is worth having rather than assuming it.
	if base := SortKeyBase(rawName); base == got {
		t.Fatalf("the two derivations agree over %q, so this test could not tell them apart", rawName)
	} else if base != "song" {
		t.Errorf("SortKeyBase(%q) = %q, want %q — the base derivation is what would sort this track under `s`", rawName, base, "song")
	}
}

// TestThePadWidthIsPinnedByBytesAndNotByTheOrderingItExistsFor is the check
// that 003 plan §6.6 asks for, plus the finding that writing it produced.
//
// The plan says *"the width is a named constant with the measured value and a
// test that asserts the ordering of `2 Fast 2 Furious` against `10 Things`,
// which is the pair the width exists for"*. That ordering is asserted here.
// **It does not pin the width**, and this test says so by demonstration: the
// same pair orders the same way at 9, at 10 and at 11, because padding to any
// width of at least two already makes the shorter run compare low. What pins
// the width is the byte-exact table above, and it is pinned again here at three
// widths so the demonstration cannot rot into a comment.
//
// What the padding *is* for is asserted too: with no padding at all the pair
// reverses, `10 Things` sorting first because `1` precedes `2`. That is the
// failure the step exists to prevent, and it is the only mutation of the width
// the ordering alone can see.
func TestThePadWidthIsPinnedByBytesAndNotByTheOrderingItExistsFor(t *testing.T) {
	const (
		two = "2 Fast 2 Furious"
		ten = "10 Things"
	)

	if sortPadWidth != 10 {
		t.Fatalf("sortPadWidth is %d, and §3.7.1's measured width is 10", sortPadWidth)
	}

	// The contract's own bytes, at the contract's own width.
	if got := delimiter + SortKeyBase(two) + delimiter; got != "|0000000002 fast 0000000002 furious|" {
		t.Errorf("SortKeyBase(%q) = %s", two, got)
	}
	if got := delimiter + SortKeyBase(ten) + delimiter; got != "|0000000010 things|" {
		t.Errorf("SortKeyBase(%q) = %s", ten, got)
	}
	if !(SortKeyBase(two) < SortKeyBase(ten)) {
		t.Errorf("%q does not sort before %q by bytes", two, ten)
	}

	// Nine and eleven: the ordering survives both, the bytes do not.
	atWidth := func(name string, width int) string {
		return foldToASCII(padDigitRuns(strings.ToLower(name), width))
	}
	if atWidth(two, sortPadWidth) != SortKeyBase(two) {
		t.Fatalf("the width-parameterised derivation disagrees with SortKeyBase at width %d, so the widths below prove nothing", sortPadWidth)
	}

	for _, width := range []int{9, 10, 11} {
		if !(atWidth(two, width) < atWidth(ten, width)) {
			t.Errorf("at pad width %d, %q does not sort before %q — the ordering was expected to survive every width of at least two", width, two, ten)
		}
		if width == sortPadWidth {
			continue
		}
		if atWidth(two, width) == SortKeyBase(two) {
			t.Errorf("pad width %d produces the same bytes as %d, so moving the width would not be visible", width, sortPadWidth)
		}
	}

	// Width one is no padding, and it is what the step exists to prevent.
	if !(atWidth(ten, 1) < atWidth(two, 1)) {
		t.Errorf("with no padding, %q was expected to sort before %q — that reversal is what the step is for", ten, two)
	}
}

// TestAnExplicitSortTitleReplacesTheDerivationForEveryType is AC-15 and
// 003 §3.7.3.
//
// Every one of the eight types is asked, the three that override included,
// because *"replaces the derivation entirely, for every type"* is a claim about
// the dispatch and not about one branch of it — and the item carries a disc and
// a track number, so a build that let the override win would produce a numeric
// prefix and be caught.
//
// The title begins `The `, which is the **only** clause distinguishing §3.7.3
// from §3.7.1: both lowercase and both pad digits, and only §3.7.1 strips
// articles. A title without an article asserts nothing about which of the two
// ran.
func TestAnExplicitSortTitleReplacesTheDerivationForEveryType(t *testing.T) {
	const (
		title = "The Ninth Gate 2"
		want  = "the ninth gate 0000000002"
	)

	if SortKeyBase(title) == want {
		t.Fatalf("the two derivations agree over %q, so this test cannot tell them apart", title)
	}

	for _, kind := range AllKinds() {
		item := ports.ScannedItem{
			Type: string(kind), Name: "Some Other Name", SortTitle: title,
			ParentIndexNumber: intp(1), IndexNumber: intp(3),
		}

		got := SortKeyFor(&item)
		if delimiter+got+delimiter != delimiter+want+delimiter {
			t.Errorf("SortKeyFor over a %s carrying an explicit sort title = %s, want %s",
				kind, delimiter+got+delimiter, delimiter+want+delimiter)
		}
		if !strings.HasPrefix(got, "the ") {
			t.Errorf("SortKeyFor over a %s stripped the article from the explicit sort title: %q", kind, got)
		}
		if !strings.Contains(got, "0000000002") {
			t.Errorf("SortKeyFor over a %s did not pad the digits of the explicit sort title: %q", kind, got)
		}
		if strings.HasPrefix(got, "0001 - ") {
			t.Errorf("SortKeyFor over a %s let the type's own derivation win over the explicit sort title: %q", kind, got)
		}
	}
}

// TestTheTailOfStep6IsStableAndNotCorrect is OQ-7, asserted as the open
// question it is.
//
// §3.7.1's step 6 says *"transliterate anything still outside ASCII"*, and the
// only character the measurement contains is the `é` of `Amélie` — which
// decomposes, so the fold alone reaches it. What the reference does with `ø`,
// `ß`, `æ` or a non-Latin script **has never been measured**, and the register
// holds the row open.
//
// So this test asserts nothing about the reference. It asserts the two
// properties v1's answer was chosen for: the output is ASCII, and the same
// input keys the same way twice. The third assertion is the price — two names
// this table has no reading for key alike, which is what dropping costs — and
// it is written down rather than left to be discovered, so that the day OQ-7 is
// measured a failing test is the notification.
func TestTheTailOfStep6IsStableAndNotCorrect(t *testing.T) {
	names := []string{"Amélie", "Ørsted", "Straße", "Æon Flux", "日本語", "Ω"}

	for _, name := range names {
		first := SortKeyBase(name)
		if second := SortKeyBase(name); first != second {
			t.Errorf("SortKeyBase(%q) answered %q and then %q", name, first, second)
		}
		if !isASCII(first) {
			t.Errorf("SortKeyBase(%q) = %q, which is not ASCII", name, first)
		}
	}

	// The cost of dropping, stated. Both of these key as the empty string
	// because the readings table has nothing for either script.
	if a, b := SortKeyBase("日本語"), SortKeyBase("Ω"); a != b {
		t.Errorf("two names outside the readings table keyed as %q and %q; v1 drops what it cannot read, so both were expected to be the same and empty", a, b)
	} else if a != "" {
		t.Errorf("a name outside the readings table keyed as %q, want the empty string — v1 drops what it cannot read", a)
	}
}

// TestTheReadingsTableIsAppliedBeforeTheDrop asserts the middle step of step
// 6's tail: the short table of obvious Latin readings runs, and it runs on the
// characters §3.7.1's OQ-7 note names.
//
// It is separated from the stability test above because the two claim different
// things. This one claims that a reading was applied; it still claims nothing
// about the reference agreeing with it, which is why the expected values are
// this project's decision and are named as such in the failure message.
func TestTheReadingsTableIsAppliedBeforeTheDrop(t *testing.T) {
	cases := []struct{ name, want string }{
		{"Ørsted", "orsted"},
		{"Straße", "strasse"},
		{"Æon Flux", "aeon flux"},
		{"Þorn", "thorn"},
	}

	for _, c := range cases {
		if got := SortKeyBase(c.name); got != c.want {
			t.Errorf("SortKeyBase(%q) = %q, want %q — this project's chosen reading, not a measured one (OQ-7)", c.name, got, c.want)
		}
	}
}

func intp(n int) *int { return &n }

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

// TestStep1LowercasesBeyondASCII is the distinction 003 T3 left for this task
// and `identity.go` argues at length: the ASCII fold every extension comparison
// in this package uses is right there and wrong here.
//
// §3.7.1's measured row is `Amélie`, whose `é` is already lowercase, so the
// table above cannot tell an ASCII fold from a Unicode one. `AMÉLIE` can: an
// ASCII fold leaves the `É` standing, step 6 then folds it to `E`, and the
// film sorts among the capitals. The two spellings are one film and must be one
// key.
func TestStep1LowercasesBeyondASCII(t *testing.T) {
	for _, name := range []string{"Amélie", "AMÉLIE", "amélie"} {
		if got := SortKeyBase(name); got != "amelie" {
			t.Errorf("SortKeyBase(%q) = %q, want %q", name, got, "amelie")
		}
	}
}
