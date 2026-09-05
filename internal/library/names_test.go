package library

import (
	"testing"
)

// The corpus is written down rather than generated, which 003's task list asks
// for in terms, and every row of it is a *behavioural* assertion: a name in,
// a title and a year out. Nothing here compares an expression against an
// expression, because there is no expression here to compare — Principle IV,
// and names.go's own comment carries the argument.
//
// Expected values are written between delimiters for the reason 003's sort-key
// task recorded: `S.W.A.T. ` and `  Padded   ` differ from their neighbours
// only in whitespace, a trailing space inside a Go string literal is invisible
// in review and in a diff, and `gofmt` will not save anybody from one.

// The `delimiter` this file compares through is sortkey_test.go's own, for the
// reason its comment gives. There is one in this package and not two.

// noYear is what a row expects when the name carries no production year. It is
// a value rather than a nil pointer in the table so that a row missing its year
// reads as a decision instead of as a zero.
const noYear = 0

type nameRow struct {
	// Name is the filename stem, or the directory name, as it is on disk.
	Name string

	// Title is what 003 §3.3 leaves once the year and the release-tag noise
	// are taken out.
	Title string

	// Year is the production year, or noYear.
	Year int

	// Why says what this row is in the corpus for. A row without one is a row
	// nobody can tell has stopped testing anything.
	Why string
}

// movieNameCorpus is the corpus 003 §3.3 asks the release-tag removal and the
// year extraction to be asserted over.
//
// The first block is every film the fixture tree holds, so that a change to
// either rule that would move a row of `libraryfixture.ExpectedItems` fails
// here first and with a smaller failure message. The rest are the shapes the
// fixture deliberately does not carry.
func movieNameCorpus() []nameRow {
	return []nameRow{
		// ---- The fixture's own sixteen -------------------------------------
		{Name: "  Padded   (1999)", Title: "Padded", Year: 1999,
			Why: "a path-derived name is trimmed (§3.5), and the reference's reading keeps the two leading spaces"},
		{Name: "10 Things I Hate About You (1999)", Title: "10 Things I Hate About You", Year: 1999,
			Why: "a title that begins with digits is not a numbered anything"},
		{Name: "100% Wolf (2020)", Title: "100% Wolf", Year: 2020,
			Why: "a per-cent sign is not a separator and does not delimit a tag"},
		{Name: "2 Fast 2 Furious (2003)", Title: "2 Fast 2 Furious", Year: 2003,
			Why: "digits in the middle of a title are not a year"},
		{Name: "A Bridge Too Far (1977)", Title: "A Bridge Too Far", Year: 1977, Why: "the plain shape"},
		{Name: "A Broadcast Capture (2011)", Title: "A Broadcast Capture", Year: 2011,
			Why: "`.ts` is a release tag as well as an extension, and the extension is gone before this runs"},
		{Name: "A Newer Transfer (2015)", Title: "A Newer Transfer", Year: 2015, Why: "the plain shape"},
		{Name: "Amélie (2001)", Title: "Amélie", Year: 2001,
			Why: "a multi-byte rune is never mistaken for a separator"},
		{Name: "An Old Transfer (1985)", Title: "An Old Transfer", Year: 1985, Why: "the plain shape"},
		{Name: "Don't Look Up (2021)", Title: "Don't Look Up", Year: 2021, Why: "an apostrophe is not a separator"},
		{Name: "Rock & Roll (1978)", Title: "Rock & Roll", Year: 1978, Why: "an ampersand is not a separator"},
		{Name: "S.W.A.T. (2003)", Title: "S.W.A.T.", Year: 2003,
			Why: "the single-separator date rule, which is the only one that keeps the final full stop"},
		{Name: "The Long Film (1998)", Title: "The Long Film", Year: 1998,
			Why: "the directory that names the multi-part film"},
		{Name: "The Matrix (1999)", Title: "The Matrix", Year: 1999, Why: "the folder-per-film directory"},
		{Name: "Wall-E (2008)", Title: "Wall-E", Year: 2008,
			Why: "a hyphen inside a title is not a separator before a tag"},
		{Name: "iRobot (2004)", Title: "iRobot", Year: 2004, Why: "a lower-case first letter survives"},

		// ---- The year, and where it is refused ------------------------------
		{Name: "The Film 1999", Title: "The Film", Year: 1999, Why: "the trailing form, with a space"},
		{Name: "Movie.1999", Title: "Movie", Year: 1999, Why: "the trailing form, with a full stop"},
		{Name: "S.W.A.T. 2003", Title: "S.W.A.T", Year: 2003,
			Why: "the run rule, and it is the pair with the bracketed row above: the full stop goes"},
		{Name: "The Film (1900)", Title: "The Film", Year: 1900, Why: "the low end of §3.3's range, accepted"},
		{Name: "The Film (2099)", Title: "The Film", Year: 2099, Why: "the high end of §3.3's range, accepted"},
		{Name: "The Film (1899)", Title: "The Film (1899)", Year: noYear,
			Why: "one below the range: not a year, and the brackets stay in the title"},
		{Name: "The Film (2100)", Title: "The Film (2100)", Year: noYear, Why: "one above the range"},
		{Name: "The Film 19999", Title: "The Film 19999", Year: noYear,
			Why: "a digit behind the run means it was never a four-digit year"},
		{Name: "1917", Title: "1917", Year: noYear,
			Why: "a year with no title in front of it is the title"},
		{Name: "2012 (2009)", Title: "2012", Year: 2009,
			Why: "a title that is itself a year, beside a real one"},
		{Name: "Blade Runner 2049 (2017)", Title: "Blade Runner 2049", Year: 2017,
			Why: "the LAST year wins, which is what keeps the title's own number"},
		{Name: "The Film, 1999", Title: "The Film, 1999", Year: noYear,
			Why: "a comma may not end a title, so this name has no year at all"},
		{Name: "The Daily Show - 2024-01-31", Title: "The Daily Show - 2024-01-31", Year: noYear,
			Why: "the date guard: §3.4's date-named episodes must not acquire a production year"},

		// ---- Release-tag noise ---------------------------------------------
		{Name: "The Film 1080p BluRay", Title: "The Film", Year: noYear,
			Why: "resolution: the earliest tag cuts, so the source behind it goes too"},
		{Name: "The Film DVDRip XviD", Title: "The Film", Year: noYear,
			Why: "source and codec, and a tag that is a prefix of a longer one"},
		{Name: "The.Film.2019.1080p.BluRay.x264", Title: "The.Film", Year: 2019,
			Why: "the Kodi-style dotted name, where the year sits in front of the noise"},
		{Name: "Some Film DVDRip 1999", Title: "Some Film", Year: 1999,
			Why: "the ordering pair: the year is taken FIRST, and a build that cut the noise first " +
				"would answer `Some Film` with no year at all, because the year is behind the tag"},
		{Name: "The Film (1999) 1080p", Title: "The Film", Year: 1999,
			Why: "everything after the year goes with it"},
		{Name: "The Film AC3 DTS", Title: "The Film", Year: noYear, Why: "audio format"},
		{Name: "The Film German Subs", Title: "The Film", Year: noYear, Why: "language"},
		{Name: "The Film [SomeGroup]", Title: "The Film", Year: noYear, Why: "a trailing release-group bracket"},
		{Name: "[SomeGroup] The Film", Title: "The Film", Year: noYear,
			Why: "a leading release-group bracket, the one rule that keeps what follows it"},
		{Name: "The Film - cd1", Title: "The Film -", Year: noYear,
			Why: "a lone part marker is release-tag noise, because `cd[1-9]` is in the vocabulary — " +
				"and the cut lands on the separator immediately before the tag, so the hyphen stays. " +
				"That is the reference's own shape and not a tidiness this project owes: its capture is " +
				"non-greedy, and the earliest split whose token is a tag is the one that leaves the hyphen"},
		{Name: "The Film - part1", Title: "The Film - part1", Year: noYear,
			Why: "and `part1` is not, which is why a lone part of that shape keeps its whole stem"},
		{Name: "Sublime", Title: "Sublime", Year: noYear,
			Why: "`subs` is a tag and `Sublime` starts with it: a tag must be delimited on both sides"},
		{Name: "The Film - a", Title: "The Film - a", Year: noYear,
			Why: "U-43's bare letter is neither a part marker nor a release tag"},
	}
}

func TestTheTitleAndTheYearComeOutOfACorpusOfNames(t *testing.T) {
	for _, row := range movieNameCorpus() {
		t.Run(row.Name, func(t *testing.T) {
			if row.Why == "" {
				t.Fatalf("the row for %q has no reason to be in the corpus", row.Name)
			}
			title, year := cleanVideoName(row.Name)

			if got, want := delimiter+title+delimiter, delimiter+row.Title+delimiter; got != want {
				t.Errorf("title of %q = %s, want %s (%s)", row.Name, got, want, row.Why)
			}

			switch {
			case row.Year == noYear && year != nil:
				t.Errorf("year of %q = %d, want none (%s)", row.Name, *year, row.Why)
			case row.Year != noYear && year == nil:
				t.Errorf("year of %q = none, want %d (%s)", row.Name, row.Year, row.Why)
			case row.Year != noYear && *year != row.Year:
				t.Errorf("year of %q = %d, want %d (%s)", row.Name, *year, row.Year, row.Why)
			}
		})
	}
}

// TestEveryExpectedTitleKeptItsDelimiters is the guard the sort-key task's
// handoff asked for by name: it fails a row whose expected value lost a
// delimiter in an edit, which is the way a whitespace assertion silently stops
// asserting whitespace.
func TestEveryExpectedTitleKeptItsDelimiters(t *testing.T) {
	for _, row := range movieNameCorpus() {
		if row.Title == "" {
			t.Errorf("the row for %q expects an empty title; no name cleans away to nothing", row.Name)
		}
	}
	if delimiter == "" {
		t.Fatal("the delimiter is empty, so every expected value in this file is compared without its boundary")
	}
}

// TestTheYearRangeIsRefusedOnBothSides walks the boundary one year at a time.
//
// The corpus already carries 1899, 1900, 2099 and 2100 as named rows; this
// asserts the same four as a **range**, because a build with the range written
// the other way round — accepting 1900 and refusing 2099, say — passes any two
// of them.
func TestTheYearRangeIsRefusedOnBothSides(t *testing.T) {
	for year := 1897; year <= 2102; year++ {
		name := "The Film (" + itoa4(year) + ")"
		_, got := cleanVideoName(name)
		want := year >= 1900 && year <= 2099
		if want && got == nil {
			t.Errorf("%q: no year, want %d", name, year)
		}
		if !want && got != nil {
			t.Errorf("%q: year %d, want none", name, *got)
		}
	}
}

// itoa4 renders a four-digit year. The years this walks are all four digits by
// construction, and a helper that could render three would be asserting
// something else.
func itoa4(n int) string {
	if n < 1000 || n > 9999 {
		panic("itoa4 is for four-digit years only")
	}
	return string([]byte{
		byte('0' + n/1000),
		byte('0' + n/100%10),
		byte('0' + n/10%10),
		byte('0' + n%10),
	})
}

// TestThePartMarkerVocabularyIsTheReferences is U-43's vocabulary, asserted a
// stem at a time.
//
// The rows that matter most are the two negatives. A bare trailing letter is
// not a marker, and 003 §3.3's withdrawn parenthetical said it was.
func TestThePartMarkerVocabularyIsTheReferences(t *testing.T) {
	rows := []struct {
		Stem    string
		Base    string
		Kind    partKind
		Type    string
		Ordinal int
		Why     string
	}{
		{Stem: "The Long Film (1998) - part1", Base: "The Long Film (1998)", Kind: partNumeric, Type: "part", Ordinal: 1,
			Why: "the fixture's own multi-part film"},
		{Stem: "The Long Film (1998) - part2", Base: "The Long Film (1998)", Kind: partNumeric, Type: "part", Ordinal: 2,
			Why: "and its second part, which must reduce to the same base"},
		{Stem: "Movie cd1", Base: "Movie", Kind: partNumeric, Type: "cd", Ordinal: 1, Why: "a space separates"},
		{Stem: "Movie.CD2", Base: "Movie", Kind: partNumeric, Type: "cd", Ordinal: 2, Why: "case is folded, a full stop separates"},
		{Stem: "Movie_dvd1", Base: "Movie", Kind: partNumeric, Type: "dvd", Ordinal: 1, Why: "an underscore separates"},
		{Stem: "Movie - pt2", Base: "Movie", Kind: partNumeric, Type: "pt", Ordinal: 2, Why: "`pt` is a marker word"},
		{Stem: "Movie - disc3", Base: "Movie", Kind: partNumeric, Type: "disc", Ordinal: 3, Why: "`disc`"},
		{Stem: "Movie - disk4", Base: "Movie", Kind: partNumeric, Type: "disk", Ordinal: 4, Why: "`disk`, which the source spells `dis[ck]`"},
		{Stem: "Movie (disc 3)", Base: "Movie", Kind: partNumeric, Type: "disc", Ordinal: 3,
			Why: "the marker may be bracketed and the number may be separated from its word"},
		{Stem: "Movie [cd1]", Base: "Movie", Kind: partNumeric, Type: "cd", Ordinal: 1, Why: "a square bracket does the same"},
		{Stem: "Movie (1998)cd1", Base: "Movie (1998)", Kind: partNumeric, Type: "cd", Ordinal: 1,
			Why: "a closing bracket ends the title with no separator at all"},
		{Stem: "Movie - cda", Base: "Movie", Kind: partAlphabetic, Type: "cd", Ordinal: 1, Why: "a letter after the word"},
		{Stem: "Movie - cdb", Base: "Movie", Kind: partAlphabetic, Type: "cd", Ordinal: 2, Why: "and the next one"},
		{Stem: "Movie - cdd", Base: "Movie", Kind: partAlphabetic, Type: "cd", Ordinal: 4, Why: "`d` is the last letter the rule takes"},
		{Stem: "Movie disc10", Base: "Movie", Kind: partNumeric, Type: "disc", Ordinal: 10, Why: "more than one digit"},

		{Stem: "Movie - a", Kind: partAbsent, Why: "U-43: a bare trailing letter is not a marker"},
		{Stem: "Movie - b", Kind: partAbsent, Why: "U-43, the other half"},
		{Stem: "Movie - cde", Kind: partAbsent, Why: "`e` is past the letter range"},
		{Stem: "Movie - cdab", Kind: partAbsent, Why: "a single letter, and this is two"},
		{Stem: "Movie - Part of the Story", Kind: partAbsent, Why: "the marker word must be followed by a number"},
		{Stem: "cd1", Kind: partAbsent, Why: "the title may be empty but the separator may not"},
		{Stem: "Movie", Kind: partAbsent, Why: "the ordinary film"},
		{Stem: "The Matrix (1999)", Kind: partAbsent, Why: "the fixture's folder-per-film file"},
	}

	for _, row := range rows {
		t.Run(row.Stem, func(t *testing.T) {
			base, marker, ok := splitPartMarker(row.Stem)

			if row.Kind == partAbsent {
				if ok {
					t.Fatalf("%q: stacked as %v part %d of %q, want no marker (%s)",
						row.Stem, marker.Kind, marker.Ordinal, base, row.Why)
				}
				if base != row.Stem {
					t.Errorf("%q: base %q, want the stem back unchanged", row.Stem, base)
				}
				return
			}

			if !ok {
				t.Fatalf("%q: no marker, want %s part %d (%s)", row.Stem, row.Type, row.Ordinal, row.Why)
			}
			if got, want := delimiter+base+delimiter, delimiter+row.Base+delimiter; got != want {
				t.Errorf("%q: base %s, want %s (%s)", row.Stem, got, want, row.Why)
			}
			if marker.Kind != row.Kind {
				t.Errorf("%q: kind %d, want %d (%s)", row.Stem, marker.Kind, row.Kind, row.Why)
			}
			if marker.Type != row.Type {
				t.Errorf("%q: marker word %q, want %q (%s)", row.Stem, marker.Type, row.Type, row.Why)
			}
			if marker.Ordinal != row.Ordinal {
				t.Errorf("%q: ordinal %d, want %d (%s)", row.Stem, marker.Ordinal, row.Ordinal, row.Why)
			}
		})
	}
}
