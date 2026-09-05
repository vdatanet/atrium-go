package library

import (
	"strconv"
	"strings"
	"time"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// This file is 003 §3.4's three levels: a series, a season that is often not a
// directory, and an episode that may span two numbers.
//
// # What is transcribed and what is not
//
// The reference does this job with about thirty regular expressions for
// episodes, eight more for multi-episode files and five for season folders
// `[source: Emby.Naming/Common/NamingOptions.cs:320-470,754-765,
// Emby.Naming/TV/SeasonPathParser.cs:13-22 @ v10.11.11]`. Those expressions are
// GPL-licensed source and translating one would be forking it (Principle IV,
// [ADR-0005]). What is written out here is the **rule** each states, read at the
// pinned tag, in this project's own terms — the same treatment `names.go`
// already gives the film rules. The one thing transcribed literally is a
// **vocabulary**: the words a season directory may spell "season" with, which
// is a list of facts rather than a program.
//
// # Which of the reference's rules are deliberately absent
//
// 003 §3.4 names the family it wants: "`S01E02` and its separators, `1x02`,
// `E02`/`EP02`, and date-based naming (`2024-01-31`) for daily shows". That is
// what is implemented, and the reference's remaining expressions are **not**:
//
//   - its optimistic bare-number forms — `01.avi`, `01 - blah.avi`,
//     `blah - 01.avi`, `Foo Bar 889`
//     `[source: Emby.Naming/Common/NamingOptions.cs:364,424-452 @ v10.11.11]`;
//   - `Episode 16` with no season beside it `[…:398 @ v10.11.11]`;
//   - the part and chapter forms `[…:390 @ v10.11.11]`.
//
// One field is absent for the same reason: **an episode carries no production
// year here**. The reference runs an episode's filename through the same
// year-and-tag cleaner a film's goes through, so `… - S01E01 - Pilot (2019)` is
// a 2019 episode there and a year-less one here. A series does get one, out of
// its directory's name, because a series is named by a directory exactly as a
// film in its own folder is (003 §3.3).
//
// Every one of these shows **fewer** placed episodes here than there — a file
// the reference numbers is unplaceable here, and 003 §3.8 counts an unplaceable
// item apart from a skipped one precisely so that an operator can see it. None
// is exercised by the fixture tree, so none of them moves 003 plan §8.2's
// forty-seven; the day one is planted it stops being invisible.
//
// [ADR-0005]: ../../docs/decisions/0005-licence.md

// numberSeparators is the run that may stand between the parts of one numbering
// token — between `S01` and `E02` in `S01 - E02`, and between a season word and
// its digits `[source: Emby.Naming/Common/NamingOptions.cs:324,356 @ v10.11.11]`.
//
// It is not `partSeparators` and not `tagSeparators`: those two are `names.go`'s
// and belong to the film rules, and writing one set in terms of another would
// make a change to either silently change the other.
const numberSeparators = "][ ._-"

// nameTrimSeparators are taken off both ends of a name cut out of a stem.
//
// The reference trims exactly whitespace, `_`, `.` and `-` off the series name
// it captures `[source: Emby.Naming/TV/EpisodePathParser.cs:85-88 @ v10.11.11]`,
// and the same trim is applied to the episode name that follows the numbering,
// which is Atrium's own rule and not the reference's — see [episodeName].
const nameTrimSeparators = " \t_.-"

// seasonZeroName is what a season numbered zero is called.
//
// It is `LibraryOptions.SeasonZeroDisplayName`, whose default is `Specials`
// `[source: Emby.Server.Implementations/Library/Resolvers/TV/SeasonResolver.cs:85-86,
// MediaBrowser.Model/Configuration/LibraryOptions.cs:57 @ v10.11.11]`. It is a
// constant here for the same reason the extension lists are: there is nothing
// in v1 to configure it with.
const seasonZeroName = "Specials"

// seasonNamePrefix is the other half of a season's name: the reference formats
// the localised `NameSeasonNumber` string, whose English value is `Season {0}`
// `[source: Emby.Server.Implementations/Library/Resolvers/TV/SeasonResolver.cs:87-91 @ v10.11.11]`.
//
// So `Season 01` on disk is an item called `Season 1`: the name comes from the
// **number**, not from the directory, which is why a zero-padded directory and
// an inferred season with no directory at all read the same in a client.
const seasonNamePrefix = "Season "

// ReasonNoEpisodeNumber and ReasonNoSeries are the two ways a file under a
// `tvshows` library becomes an item nothing can place.
//
// They are [Note.Reason] values and not [Skip] values, and the distinction is
// 003 §3.8's: a skipped file is **not** in the library and an unplaceable item
// **is**, so "an operator told that both were skipped would go looking for
// something that is not missing". [Plan.Unplaceable] is where they are counted.
const (
	// ReasonNoEpisodeNumber is `blob.mkv` in a season directory: a real file,
	// of an admitted extension, under a series and a season, whose name says
	// nothing about which episode it is.
	ReasonNoEpisodeNumber = "the name carries neither an episode number nor a date"

	// ReasonNoSeries is a candidate directly under a library root whose own
	// name does not say which series it belongs to. There is no directory
	// above it to ask.
	ReasonNoSeries = "nothing names the series this file belongs to"
)

// ---------------------------------------------------------------------------
// The numbering a filename carries
// ---------------------------------------------------------------------------

// episodeNumbering is what one name said about where an episode sits, and the
// span of the name that said it.
//
// The span is the point of the type. 003 §3.4's naming rule — an episode is
// called what **follows** the numbering — needs to know where the numbering
// ended, and a parser that returned only the numbers would leave the caller to
// find it a second time and disagree.
type episodeNumbering struct {
	season     *int
	episode    *int
	episodeEnd *int
	premiere   *units.Time

	// start and end bracket the matched numbering within the stem.
	start, end int

	// found reports that some rule matched. It is not the same question as
	// "can this be placed": a date-named episode has a premiere date, no
	// episode number, and is perfectly placeable.
	found bool
}

// placeable reports whether the name said enough to make this an episode of
// something rather than a file that merely sits in the right place.
//
// A date is enough and an episode number is enough; a season number on its own
// is not, which is the reference's own test
// `[source: Emby.Naming/TV/EpisodePathParser.cs:169 @ v10.11.11]` widened by the
// date rule beside it.
func (n episodeNumbering) placeable() bool {
	return n.episode != nil || n.premiere != nil
}

// parseEpisodeNumbering reads 003 §3.4's family of patterns out of a filename's
// stem.
//
// **The rules are tried in order and the first that matches wins**, which is
// the reference's own shape — it walks its expression list and stops at the
// first success `[source: Emby.Naming/TV/EpisodePathParser.cs:53-77 @ v10.11.11]`
// — and the order is the reference's too, with one deliberate change: the
// `Season 1 Episode 2` spelling is folded into the `S01E02` rule rather than
// sitting six places further down, because here they are one rule. For a name
// to be answered differently by that it would have to carry both a date and a
// spelled-out season and episode.
func parseEpisodeNumbering(stem string) episodeNumbering {
	for _, rule := range []func(string) episodeNumbering{
		seasonAndEpisodeNumbering,
		prefixedEpisodeNumbering,
		dateNumbering,
		crossEpisodeNumbering,
	} {
		if n := rule(stem); n.found {
			return n
		}
	}
	return episodeNumbering{}
}

// seasonAndEpisodeNumbering is `S01E02` and everything spelled like it:
// `S01 - E02`, `s01.e02`, `S01xE02`, and `Season 1 Episode 2`
// `[source: Emby.Naming/Common/NamingOptions.cs:324,356,418 @ v10.11.11]`.
//
// The **earliest** position that matches wins, so a series whose own title
// contains a later `s…e…` cannot take the numbering away from the real one.
func seasonAndEpisodeNumbering(stem string) episodeNumbering {
	for i := 0; i < len(stem); i++ {
		j, ok := keywordAt(stem, i, "s", "eason")
		if !ok {
			continue
		}
		j = skipRun(stem, j, numberSeparators)
		season, j, ok := readNumber(stem, j)
		if !ok || !plausibleSeason(season) {
			continue
		}

		// `S01xE02` is the reference's own spelling and the `x` is neither a
		// separator nor part of a number `[…:418 @ v10.11.11]`.
		k := skipRun(stem, j, numberSeparators)
		if k < len(stem) && (stem[k] == 'x' || stem[k] == 'X') {
			k = skipRun(stem, k+1, numberSeparators)
		}
		k, ok = keywordAt(stem, k, "e", "pisode")
		if !ok {
			continue
		}
		k = skipRun(stem, k, numberSeparators)
		episode, k, ok := readNumber(stem, k)
		if !ok {
			continue
		}

		end, endNumber := readEpisodeRange(stem, k)
		return episodeNumbering{
			season:     &season,
			episode:    &episode,
			episodeEnd: endNumber,
			start:      i,
			end:        end,
			found:      true,
		}
	}
	return episodeNumbering{}
}

// crossEpisodeNumbering is `1x02` — a season, the letter `x`, an episode
// `[source: Emby.Naming/Common/NamingOptions.cs:369,403,413 @ v10.11.11]`.
//
// The digits must begin the stem or follow one of the reference's own
// delimiters, so `1920x1080` inside `Series Special (1920x1080)` is refused by
// the season plausibility rule rather than read as season 1920.
func crossEpisodeNumbering(stem string) episodeNumbering {
	for i := 0; i < len(stem); i++ {
		if i > 0 && !strings.ContainsRune(numberSeparators+"(", rune(stem[i-1])) {
			continue
		}
		season, j, ok := readNumber(stem, i)
		if !ok || !plausibleSeason(season) {
			continue
		}
		if j >= len(stem) || (stem[j] != 'x' && stem[j] != 'X') {
			continue
		}
		episode, k, ok := readNumber(stem, j+1)
		if !ok {
			continue
		}

		end, endNumber := readEpisodeRange(stem, k)
		return episodeNumbering{
			season:     &season,
			episode:    &episode,
			episodeEnd: endNumber,
			start:      i,
			end:        end,
			found:      true,
		}
	}
	return episodeNumbering{}
}

// prefixedEpisodeNumbering is `EP02` and `E02` — an episode number with no
// season beside it
// `[source: Emby.Naming/Common/NamingOptions.cs:329,331 @ v10.11.11]`.
//
// Two narrowings of the reference, both in the direction of placing fewer
// files:
//
//   - a separator is required before the letter. The reference requires one
//     before `ep` and not before `e`, and the `e` form with none matches inside
//     ordinary words.
//   - the `e` form's digits must end the stem or be followed by a full stop,
//     because the reference's expression requires a `.` after them and over a
//     path that dot is very often the extension's. So `foo - E01.mkv` carries
//     episode 1 and `foo - E01 - Title.mkv` does not, there and here.
func prefixedEpisodeNumbering(stem string) episodeNumbering {
	for i := 0; i < len(stem); i++ {
		if i > 0 && !strings.ContainsRune(numberSeparators, rune(stem[i-1])) {
			continue
		}
		if j, ok := keywordAt(stem, i, "ep", ""); ok {
			j = skipRun(stem, j, "_")
			if episode, k, ok := readNumber(stem, j); ok {
				end, endNumber := readEpisodeRange(stem, k)
				return episodeNumbering{episode: &episode, episodeEnd: endNumber, start: i, end: end, found: true}
			}
			continue
		}
		j, ok := keywordAt(stem, i, "e", "")
		if !ok {
			continue
		}
		episode, k, ok := readNumber(stem, j)
		if !ok {
			continue
		}
		if k != len(stem) && stem[k] != '.' {
			continue
		}
		return episodeNumbering{episode: &episode, start: i, end: k, found: true}
	}
	return episodeNumbering{}
}

// dateNumbering is 003 §3.4's daily show: `2024-01-31`, and the reference's
// day-first spelling beside it
// `[source: Emby.Naming/Common/NamingOptions.cs:332-351 @ v10.11.11]`.
//
// The separators either side must be the same character, and the result must be
// a real calendar date, because the reference validates the whole run against
// four exact formats each of which repeats one separator
// `[…:336-339,346-349 @ v10.11.11]`. It is a narrowing in one respect the
// reference itself flags in a `TODO`: there, a run that matches the expression
// but parses as no date is still counted a success and contributes no date at
// all `[source: Emby.Naming/TV/EpisodePathParser.cs:136-137 @ v10.11.11]`.
//
// A date-named episode has **no** episode number, which is not an error: 003
// §3.4 lists date-based naming beside the numbered forms rather than under
// them.
func dateNumbering(stem string) episodeNumbering {
	for i := 0; i < len(stem); i++ {
		if i > 0 && isASCIIDigit(stem[i-1]) {
			continue
		}
		for _, yearFirst := range []bool{true, false} {
			t, end, ok := readDate(stem, i, yearFirst)
			if !ok {
				continue
			}
			at := units.At(t)
			return episodeNumbering{premiere: &at, start: i, end: end, found: true}
		}
	}
	return episodeNumbering{}
}

// readDate reads `yyyy<sep>mm<sep>dd` or `dd<sep>mm<sep>yyyy` at i.
func readDate(stem string, i int, yearFirst bool) (time.Time, int, bool) {
	first, second := 4, 2
	if !yearFirst {
		first, second = 2, 4
	}

	a, ok := fixedDigits(stem, i, first)
	if !ok {
		return time.Time{}, 0, false
	}
	j := i + first
	if j >= len(stem) || !strings.ContainsRune("._ -", rune(stem[j])) {
		return time.Time{}, 0, false
	}
	sep := stem[j]

	month, ok := fixedDigits(stem, j+1, 2)
	if !ok {
		return time.Time{}, 0, false
	}
	k := j + 3
	if k >= len(stem) || stem[k] != sep {
		return time.Time{}, 0, false
	}

	b, ok := fixedDigits(stem, k+1, second)
	if !ok {
		return time.Time{}, 0, false
	}
	end := k + 1 + second
	if end < len(stem) && isASCIIDigit(stem[end]) {
		return time.Time{}, 0, false
	}

	year, day := a, b
	if !yearFirst {
		year, day = b, a
	}
	when := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if when.Year() != year || int(when.Month()) != month || when.Day() != day {
		return time.Time{}, 0, false
	}
	return when, end, true
}

// readEpisodeRange reads the second number of a multi-episode file, which 003
// §3.4 requires to stay **one** item spanning two numbers rather than becoming
// two `[source: Emby.Naming/Common/NamingOptions.cs:754-765 @ v10.11.11]`.
//
// The forms are `-E03`, `- E03`, `-03`, `-x03` and `E03` with no separator at
// all, and the reference's alternation reduces to one rule with **two** cases
// rather than to "a hyphen or a letter":
//
//   - a **bare** hyphen, with no spaces around it, may stand alone —
//     `(-[xX]?[eE]?(?<endingepnumber>[0-9]{1,3}))+`;
//   - a spaced ` - `, or no separator at all, requires the letter —
//     `((-| - )?[xXeE](?<endingepnumber>[0-9]{1,3}))+`.
//
// The distinction is not a nicety, and the fixture tree fails without it:
// `24 - S01E01 - 12-00 AM` ends in a spaced hyphen followed by digits, so a
// rule that accepted any hyphen makes that file episodes 1 **to 12**, with
// every per-field assertion about season and episode still passing.
//
// And the guard that is the whole reason this is not three lines: the ending
// number is refused when another digit, a `p` or an `i` follows it, so
// `series - s09e14-1080p` is episode 14 and not episodes 14 to 108
// `[source: Emby.Naming/TV/EpisodePathParser.cs:154-165 @ v10.11.11]`. The
// comment there says so in as many words, and the failure it prevents makes a
// range the size of a hundred episodes out of a resolution the size of a
// screen. The reference caps the ending number at three digits where this reads
// the whole run; the guard absorbs the difference, because a run longer than
// three digits has a digit behind the third.
func readEpisodeRange(stem string, i int) (int, *int) {
	j := i
	spaced := false
	switch {
	case strings.HasPrefix(stem[i:], " - "):
		j, spaced = i+3, true
	case i < len(stem) && stem[i] == '-':
		j = i + 1
	}
	hyphen := j > i

	letter := false
	if j < len(stem) && strings.ContainsRune("eExX", rune(stem[j])) {
		letter = true
		j++
	}
	if !letter && (spaced || !hyphen) {
		return i, nil
	}

	end, k, ok := readNumber(stem, j)
	if !ok {
		return i, nil
	}
	if k < len(stem) && strings.ContainsRune("0123456789iIpP", rune(stem[k])) {
		return i, nil
	}
	return k, &end
}

// plausibleSeason refuses a season number the reference refuses: 200 to 1927,
// and anything above 2500
// `[source: Emby.Naming/TV/EpisodePathParser.cs:186-193 @ v10.11.11]`.
//
// The reference's comment gives the case and it is the one that matters:
// without it `Series Special (1920x1080)` is season 1920 episode 1080. The
// window that stays open — 1928 to 2500 — is what leaves a daily show's
// `Season 2024` alone.
//
// It applies to a number read out of a **name**. A season *directory* goes
// through [parseSeasonDirectoryName], which is a different code path in the
// reference too and carries no such guard.
func plausibleSeason(n int) bool {
	return !((n >= 200 && n < 1928) || n > 2500)
}

// keywordAt matches a single letter at i, optionally followed by the rest of
// the word it abbreviates — `s` or `season`, `e` or `episode` — ignoring ASCII
// case, and returns the index just past what it matched.
//
// The fold is ASCII because every letter it compares is one. A season word that
// is not — `sæson`, `сезон` — belongs to a directory's name and goes through
// [seasonDirectoryWords], which folds with [strings.ToLower] for the reason
// `identity.go` gives.
func keywordAt(s string, i int, letter, rest string) (int, bool) {
	if i >= len(s) || foldASCIICase(s[i:i+1]) != letter[:1] {
		return 0, false
	}
	if len(letter) > 1 {
		if i+len(letter) > len(s) || foldASCIICase(s[i:i+len(letter)]) != letter {
			return 0, false
		}
	}
	j := i + len(letter)
	if rest != "" && j+len(rest) <= len(s) && foldASCIICase(s[j:j+len(rest)]) == rest {
		j += len(rest)
	}
	return j, true
}

// readNumber reads a run of ASCII digits at i.
func readNumber(s string, i int) (int, int, bool) {
	start := i
	value := 0
	for i < len(s) && isASCIIDigit(s[i]) {
		value = value*10 + int(s[i]-'0')
		i++
		if i-start > 9 {
			// A run this long is not a number anybody wrote on a file, and
			// carrying on would overflow rather than answer.
			return 0, 0, false
		}
	}
	if i == start {
		return 0, 0, false
	}
	return value, i, true
}

// fixedDigits reads exactly n digits at i.
func fixedDigits(s string, i, n int) (int, bool) {
	if i < 0 || i+n > len(s) {
		return 0, false
	}
	value := 0
	for j := i; j < i+n; j++ {
		if !isASCIIDigit(s[j]) {
			return 0, false
		}
		value = value*10 + int(s[j]-'0')
	}
	return value, true
}

// skipRun advances past a run of any of the given bytes.
func skipRun(s string, i int, set string) int {
	for i < len(s) && strings.ContainsRune(set, rune(s[i])) {
		i++
	}
	return i
}

// ---------------------------------------------------------------------------
// The number a season directory carries
// ---------------------------------------------------------------------------

// seasonDirectoryWords is the vocabulary a season directory may spell the word
// "season" with, transcribed from the alternation of the reference's two
// season-folder expressions
// `[source: Emby.Naming/TV/SeasonPathParser.cs:15,18 @ v10.11.11]`.
//
// It is **data**, and is transcribed for the same reason the release tags and
// the extras folder names are: a vocabulary is a list of facts about what the
// reference recognises, where an expression is a program (Principle IV).
//
// Longest first, because the alternation's bare single letters — `s`, `t`, `k`
// and Cyrillic `с`, each of which the reference accepts on its own — would
// otherwise match the first letter of a longer word and leave the rest of it
// standing where digits have to be.
var seasonDirectoryWords = []string{
	"temporada",
	"stagione", "sezonul", "seizoen", "seasong",
	"staffel", "säsong", "sezona", "sezóna",
	"saison", "series", "season", "kausi", "сезон",
	"sæson", "sezon",
	"シーズン", "시즌",
	"s", "t", "k", "с",
}

// specialsDirectoryName is the alias 003 §3.4 makes load-bearing: a directory
// called `Specials` is **season zero**
// `[source: Emby.Naming/TV/SeasonPathParser.cs:81-86 @ v10.11.11]`.
//
// # The reference accepts a second alias here and Atrium does not
//
// The same two lines accept `extras` as season zero as well. Atrium does not,
// and the reason is that the two rules would contradict each other: 003 §3.2
// makes `Extras` an extras directory whose files are not candidates, so under
// this server nothing is ever placed in a season made of one. A season nothing
// is placed in is never created — the same rule that leaves the fixture's empty
// `The Series/Season 03` with no item — so implementing the alias could change
// exactly one thing: it could let an `Extras` directory beside a `Specials` one
// **supply season zero's path**, which is the wrong directory to name.
//
// So the alias implemented is the one 003 §3.4 states, and this is a narrowing
// of a source reading rather than a measurement being ignored. Nothing in the
// fixture tree exercises either half.
const specialsDirectoryName = "specials"

// parseSeasonDirectoryName reads a season number out of one directory's name,
// given the name of the series directory above it.
//
// The rules are the reference's, in its order
// `[source: Emby.Naming/TV/SeasonPathParser.cs:64-104 @ v10.11.11]`:
//
//  1. an `S` followed by one to four digits that are not followed by another
//     digit or by `E<digit>`, and that end the name or are followed by one of
//     `. _ - [ ]` — so `S01` is season 1 and `S01E01` is not a season at all;
//  2. otherwise the name has ` . _ - [ ]` removed from it, and the series
//     directory's name — cleaned the same way — is removed from what is left,
//     so `The Series Season 1` under `The Series` reads as `Season 1`;
//  3. `specials` is season zero;
//  4. a name that is nothing but digits is that number;
//  5. digits then a season word (`1 Season`, `1st Season`);
//  6. a season word then digits (`Season 01`, `Season 2024`).
//
// Rules 5 and 6 both refuse a number followed by `E<digit>`, and rule 6 stops a
// digit run early where the rest of it is a resolution — `Season 1 1080p` is
// season 1 and not season 11080.
func parseSeasonDirectoryName(name, seriesName string) (int, bool) {
	if n, ok := seasonPrefixNumber(name); ok {
		return n, true
	}

	cleaned := removeAny(name, numberSeparators)
	if seriesName != "" {
		if folded := removeAny(seriesName, numberSeparators); folded != "" {
			cleaned = removeFoldedSubstring(cleaned, folded)
		}
	}
	if cleaned == "" {
		return 0, false
	}

	if strings.ToLower(cleaned) == specialsDirectoryName {
		return 0, true
	}
	if n, j, ok := readNumber(cleaned, 0); ok && j == len(cleaned) {
		return n, true
	}
	if n, ok := numberThenSeasonWord(cleaned); ok {
		return n, true
	}
	return seasonWordThenNumber(cleaned)
}

// seasonPrefixNumber is rule 1: `S01`, `S1-something`, and never `S01E01`
// `[source: Emby.Naming/TV/SeasonPathParser.cs:21,66-71 @ v10.11.11]`.
func seasonPrefixNumber(name string) (int, bool) {
	for i := 0; i+1 < len(name); i++ {
		if foldASCIICase(name[i:i+1]) != "s" {
			continue
		}
		value, j, ok := readNumber(name, i+1)
		if !ok || j-(i+1) > 4 {
			continue
		}
		if j < len(name) {
			if isASCIIDigit(name[j]) {
				continue
			}
			if foldASCIICase(name[j:j+1]) == "e" && j+1 < len(name) && isASCIIDigit(name[j+1]) {
				continue
			}
			if !strings.ContainsRune("._-[] ", rune(name[j])) {
				continue
			}
		}
		return value, true
	}
	return 0, false
}

// numberThenSeasonWord is rule 5: `1 Season`, `1st Season`, cleaned to
// `1Season` and `1stSeason` by the time it gets here
// `[source: Emby.Naming/TV/SeasonPathParser.cs:15 @ v10.11.11]`.
func numberThenSeasonWord(cleaned string) (int, bool) {
	value, j, ok := readNumber(cleaned, 0)
	if !ok {
		return 0, false
	}
	for _, ordinal := range []string{"st", "nd", "rd", "th"} {
		for j+2 <= len(cleaned) && foldASCIICase(cleaned[j:j+2]) == ordinal {
			j += 2
		}
	}
	if followedByEpisodeLetter(cleaned, j) {
		return 0, false
	}
	if _, ok := seasonWordAt(cleaned, j); !ok {
		return 0, false
	}
	return value, true
}

// seasonWordThenNumber is rule 6: `Season 01`, `Season2024`, `Staffel 3`.
func seasonWordThenNumber(cleaned string) (int, bool) {
	j, ok := seasonWordAt(cleaned, 0)
	if !ok {
		return 0, false
	}
	run := j
	for run < len(cleaned) && isASCIIDigit(cleaned[run]) {
		run++
	}
	if run == j {
		return 0, false
	}

	// The reference's number is non-greedy with a `\d{3,4}p` lookahead behind
	// it, which is a resolution guard and not an accident: `Season 1 1080p`
	// cleans to `Season11080p` and is season 1
	// `[source: Emby.Naming/TV/SeasonPathParser.cs:18 @ v10.11.11]`.
	end := run
	for k := j + 1; k < run; k++ {
		if isResolutionRun(cleaned, k, run) {
			end = k
			break
		}
	}
	if followedByEpisodeLetter(cleaned, end) {
		return 0, false
	}
	value, _, ok := readNumber(cleaned[:end], j)
	return value, ok
}

// isResolutionRun reports whether the digits from k to the end of the run are
// three or four of them followed by a `p`.
func isResolutionRun(cleaned string, k, run int) bool {
	width := run - k
	if width != 3 && width != 4 {
		return false
	}
	return run < len(cleaned) && foldASCIICase(cleaned[run:run+1]) == "p"
}

// followedByEpisodeLetter is the reference's `(?!\s*[Ee]\d)`: a season number
// with an episode number behind it is not a season directory's number
// `[source: Emby.Naming/TV/SeasonPathParser.cs:15,18 @ v10.11.11]`.
func followedByEpisodeLetter(s string, i int) bool {
	i = skipRun(s, i, " ")
	return i+1 < len(s) && foldASCIICase(s[i:i+1]) == "e" && isASCIIDigit(s[i+1])
}

// seasonWordAt matches one of [seasonDirectoryWords] at i and returns the index
// just past it.
func seasonWordAt(s string, i int) (int, bool) {
	lowered := strings.ToLower(s[i:])
	for _, word := range seasonDirectoryWords {
		if strings.HasPrefix(lowered, word) {
			return i + len(word), true
		}
	}
	return 0, false
}

// removeAny deletes every byte of set from s.
func removeAny(s, set string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if strings.ContainsRune(set, rune(s[i])) {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// removeFoldedSubstring deletes every case-insensitive occurrence of sub from
// s, which is the reference's `Replace(cleanParent, "", OrdinalIgnoreCase)`
// `[source: Emby.Naming/TV/SeasonPathParser.cs:78 @ v10.11.11]`.
func removeFoldedSubstring(s, sub string) string {
	lowered := strings.ToLower(s)
	target := strings.ToLower(sub)
	var b strings.Builder
	for i := 0; i < len(s); {
		if strings.HasPrefix(lowered[i:], target) {
			i += len(target)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// ---------------------------------------------------------------------------
// The three passes
// ---------------------------------------------------------------------------

// showCandidate is one file, after the classify pass read everything its own
// path yields.
type showCandidate struct {
	root  int
	entry Entry

	// seriesDir is the first path component, and is empty for a file directly
	// under a library root.
	seriesDir string

	// seasonDir is the file's immediate containing directory when that
	// directory is **below** the series directory, and empty otherwise.
	//
	// The exclusion is 003 §3.4's `24` rule at the level a directory can break
	// it: a series called `24` is a directory whose name is nothing but
	// digits, which is exactly the shape the reference reads as a numeric
	// season folder `[source: Emby.Naming/TV/SeasonPathParser.cs:88-92 @ v10.11.11]`.
	// A series directory is never asked, there — the reference resolves it as
	// a Series and its season parser only ever runs on a directory whose
	// parent is one `[source: Emby.Server.Implementations/Library/Resolvers/TV/SeasonResolver.cs:45 @ v10.11.11]`
	// — or here.
	seasonDir string

	seriesName string
	seriesYear *int

	numbers episodeNumbering
	name    string
}

// classifyShow is pass 1 for one entry: everything the path alone yields.
func classifyShow(root int, entry Entry) showCandidate {
	stem := stemOfPath(entry.Path)
	dir := dirName(entry.Path)

	candidate := showCandidate{root: root, entry: entry}
	if dir != "." {
		parts := strings.Split(dir, "/")
		candidate.seriesDir = parts[0]
		if len(parts) > 1 {
			candidate.seasonDir = dir
		}
	}

	candidate.numbers = parseEpisodeNumbering(stem)
	candidate.name = episodeName(stem, candidate.numbers)

	// A directory names the series; a file directly under a root has none, so
	// the reference's own `seriesname` capture — everything before the
	// numbering — is what is left to ask
	// `[source: Emby.Naming/Common/NamingOptions.cs:324 @ v10.11.11]`.
	source := candidate.seriesDir
	if source == "" && candidate.numbers.found {
		source = strings.Trim(stem[:candidate.numbers.start], nameTrimSeparators)
	}
	if source != "" {
		candidate.seriesName, candidate.seriesYear = cleanVideoName(source)
	}
	return candidate
}

// episodeName is 003 §3.4's naming rule as this project reads it: **an episode
// is called what follows the numbering**, and where nothing follows it the
// whole stem stays.
//
// That is a **declared difference** from the reference, which names an episode
// after its whole cleaned filename — all nine of the fixture tree's episodes
// differ by it `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11,
// 2026-09-02]` — and it is the rule `libraryfixture.ExpectedItems` is written
// against.
//
// It is one rule and not two: `The Daily Show - 2024-01-31` keeps its whole
// name because its date ends the stem and nothing follows it, and
// `… - S01E01 - Pilot` becomes `Pilot`. A build that took the last hyphenated
// segment instead agrees about `Pilot` and calls the daily show's episode
// `2024-01-31`.
func episodeName(stem string, n episodeNumbering) string {
	if n.found {
		if after := strings.Trim(stem[n.end:], nameTrimSeparators); after != "" {
			return after
		}
	}
	return strings.TrimSpace(stem)
}

// resolveShows is 003 plan §6.2's three passes for a `tvshows` library.
//
// Classify reads what one path yields, group folds the candidates into series
// and seasons, and place derives the identifiers and the sort keys. The middle
// pass needs siblings for the same reason the film resolver's does: a season
// discovered from an episode's filename and a season directory naming the same
// number are **one** item (003 §3.6 keys a season on its series plus its
// number), and which of the two supplies the season's path cannot be decided
// from either one alone.
//
// No map is iterated to produce output. Series and seasons are appended to
// slices in the sorted entry order and read back out of them by key, and the
// items are sorted once, on the way out of [Resolve].
func resolveShows(lib ports.Library, readings []Reading, parentID string) ([]ports.ScannedItem, []Note, error) {
	var (
		candidates  []showCandidate
		series      = map[string]*seriesGroup{}
		seriesOrder []*seriesGroup
	)

	for _, reading := range readings {
		for _, entry := range reading.Entries {
			candidate := classifyShow(reading.Root, entry)
			candidates = append(candidates, candidate)

			if candidate.seriesName == "" {
				continue
			}
			key, err := Normalise(candidate.seriesName, lib.CaseSensitive)
			if err != nil {
				return nil, nil, err
			}
			if _, seen := series[key]; !seen {
				group := &seriesGroup{
					id:      DeriveID(lib.ID, KindSeries, key),
					key:     key,
					name:    candidate.seriesName,
					year:    candidate.seriesYear,
					path:    candidate.seriesDir,
					root:    candidate.root,
					seasons: map[int]*seasonGroup{},
				}
				series[key] = group
				seriesOrder = append(seriesOrder, group)
			}
		}
	}

	items := make([]ports.ScannedItem, 0, len(candidates)+len(seriesOrder))
	var unplaceable []Note

	episodes := make([]ports.ScannedItem, 0, len(candidates))
	for _, candidate := range candidates {
		item, note, err := placeEpisode(lib, candidate, series, parentID)
		if err != nil {
			return nil, nil, err
		}
		if note != nil {
			unplaceable = append(unplaceable, *note)
		}
		episodes = append(episodes, item)
	}

	for _, group := range seriesOrder {
		items = append(items, group.item(lib, parentID))
		for _, season := range group.order {
			items = append(items, season.item(lib, group))
		}
	}
	return append(items, episodes...), unplaceable, nil
}

// seriesGroup is one series and the seasons found beneath it.
type seriesGroup struct {
	id   string
	key  string
	name string
	year *int
	path string
	root int

	seasons map[int]*seasonGroup
	order   []*seasonGroup
}

// seasonGroup is one season. Its identity is its series' identity plus its
// number (003 §3.6), which is what makes a season inferred from an episode's
// filename and a directory naming the same number **one** item — and what lets
// an inferred season have an identifier at all, having no path to derive one
// from.
type seasonGroup struct {
	id     string
	number int
	path   string
	root   int
}

// season returns the group for a number, creating it the first time.
//
// The identifier is derived here and read back from the group rather than
// derived a second time where an episode needs its parent: a derivation
// written twice is a derivation that can disagree with itself, and nothing
// downstream of a `parent_id` could tell.
func (s *seriesGroup) season(libID string, number int) *seasonGroup {
	if existing, ok := s.seasons[number]; ok {
		return existing
	}
	group := &seasonGroup{
		id:     DeriveID(libID, KindSeason, joinKey(s.id, strconv.Itoa(number))),
		number: number,
		root:   s.root,
	}
	s.seasons[number] = group
	s.order = append(s.order, group)
	return group
}

func (s *seriesGroup) item(lib ports.Library, parentID string) ports.ScannedItem {
	item := ports.ScannedItem{
		ID:             s.id,
		LibraryID:      lib.ID,
		ParentID:       parentID,
		Type:           string(KindSeries),
		Name:           s.name,
		Path:           s.path,
		RootOrdinal:    s.root,
		ProductionYear: s.year,
	}
	item.SortKey = SortKeyFor(&item)
	return item
}

func (g *seasonGroup) item(lib ports.Library, series *seriesGroup) ports.ScannedItem {
	number := g.number
	name := seasonNamePrefix + strconv.Itoa(number)
	if number == 0 {
		name = seasonZeroName
	}
	item := ports.ScannedItem{
		ID:          g.id,
		LibraryID:   lib.ID,
		ParentID:    series.id,
		Type:        string(KindSeason),
		Name:        name,
		Path:        g.path,
		RootOrdinal: g.root,
		IndexNumber: &number,
	}
	item.SortKey = SortKeyFor(&item)
	return item
}

// placeEpisode is pass 3 for one candidate: it settles which season the file
// belongs to, creates that season if it is the first to need it, and builds the
// item.
//
// # Where the season number comes from, in order
//
// 003 §3.4: **"Ambiguity is resolved by position, not by preference: the
// pattern is matched against the filename first, then against the parent
// directory."** So:
//
//  1. the filename's own numbering;
//  2. failing that, the containing directory's name — but only a directory
//     **below** the series, see [showCandidate.seasonDir];
//  3. failing both, season 1, when the name gave an episode number or a date
//     and there was no season directory to consult. That is the reference's own
//     rule and not a guess
//     `[source: Emby.Server.Implementations/Library/Resolvers/TV/EpisodeResolver.cs:78-82 @ v10.11.11]`;
//  4. failing all three, no season at all, and the episode's parent is its
//     series.
//
// A season's **path** is filled only by a directory whose parsed number is the
// number the season ended up with, which is what makes the answer independent
// of the order the entries arrived in: a season met first through a filename
// and later through its directory still gets the directory.
func placeEpisode(lib ports.Library, candidate showCandidate, series map[string]*seriesGroup, parentID string) (ports.ScannedItem, *Note, error) {
	key, err := Normalise(candidate.entry.Path, lib.CaseSensitive)
	if err != nil {
		return ports.ScannedItem{}, nil, err
	}

	item := ports.ScannedItem{
		ID:                DeriveID(lib.ID, KindEpisode, key),
		LibraryID:         lib.ID,
		ParentID:          parentID,
		Type:              string(KindEpisode),
		Name:              candidate.name,
		Path:              candidate.entry.Path,
		RootOrdinal:       candidate.root,
		IndexNumber:       candidate.numbers.episode,
		IndexNumberEnd:    candidate.numbers.episodeEnd,
		PremiereDate:      candidate.numbers.premiere,
		ParentIndexNumber: candidate.numbers.season,
		Files: []ports.ScannedFile{{
			Ordinal:    0,
			Path:       candidate.entry.Path,
			Size:       candidate.entry.Size,
			ModifiedAt: candidate.entry.ModifiedAt,
		}},
	}

	var note *Note
	reason := ""
	switch {
	case candidate.seriesName == "":
		reason = ReasonNoSeries
	case !candidate.numbers.placeable():
		reason = ReasonNoEpisodeNumber
	}
	if reason != "" {
		item.Unplaceable = true
		note = &Note{Root: candidate.root, Path: candidate.entry.Path, Reason: reason}
	}

	if candidate.seriesName == "" {
		item.SortKey = SortKeyFor(&item)
		return item, note, nil
	}

	seriesKey, err := Normalise(candidate.seriesName, lib.CaseSensitive)
	if err != nil {
		return ports.ScannedItem{}, nil, err
	}
	group := series[seriesKey]
	item.ParentID = group.id

	directoryNumber, hasDirectory := 0, false
	if candidate.seasonDir != "" {
		directoryNumber, hasDirectory = parseSeasonDirectoryName(baseName(candidate.seasonDir), candidate.seriesDir)
	}

	number, ok := 0, false
	switch {
	case candidate.numbers.season != nil:
		number, ok = *candidate.numbers.season, true
	case hasDirectory:
		number, ok = directoryNumber, true
	case candidate.seasonDir == "" && candidate.numbers.placeable():
		number, ok = 1, true
	}
	if !ok {
		item.SortKey = SortKeyFor(&item)
		return item, note, nil
	}

	season := group.season(lib.ID, number)
	if hasDirectory && directoryNumber == number && season.path == "" {
		season.path = candidate.seasonDir
		season.root = candidate.root
	}
	item.ParentID = season.id
	item.ParentIndexNumber = &number
	item.SortKey = SortKeyFor(&item)
	return item, note, nil
}
