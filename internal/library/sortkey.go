package library

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vdatanet/atrium-go/internal/ports"
	"golang.org/x/text/unicode/norm"
)

// The sort key is what every ordered list a client draws is ordered by, and
// there is not one rule for it. There are two, and the second is not a
// refinement of the first (003 §3.7,
// [behaviours §2.6](../../docs/compatibility/behaviours.md#26-sortname-has-two-derivations-and-three-types-use-the-second)).
//
// # Why there is one entry point that takes an item and one that takes a name
//
// behaviours §2.6 names two temptations, and the second one is a *shape of
// codebase* rather than a mistake in a line: *"using one sort-name function for
// everything"*. Its symptom is not a wrong field somewhere — it is a track
// called `The Song` sorting under `s` instead of `T`, which reorders every
// album in the library and which no client can correct or even recognise as
// wrong.
//
// So [SortKeyFor] is the only function in this package that takes an item, and
// the three type-specific derivations it dispatches to are unexported. A caller
// holding a [ports.ScannedItem] has exactly one way to key it and cannot reach
// the base derivation by accident. [SortKeyBase] takes a bare `string` because
// its callers hold a bare name and nothing else: 005 keys a by-name row, which
// has no item behind it. A caller that has only a name in its hand cannot know
// a type, so it cannot be the caller that got the type wrong.
//
// # Why nothing here tidies whitespace
//
// `Rock & Roll` keeps the double space its removed `&` leaves and `S.W.A.T.`
// keeps the trailing space its last replaced `.` leaves, because 003 §3.7.1's
// steps 3 to 5 neither trim nor collapse. An implementation that tidied them
// would sort differently from the reference — quietly, and only for names
// containing those characters. [strings.Fields], [strings.TrimSpace] and
// [strings.Join] are each a way to lose that by accident, so every step below
// walks and appends, and the one trim there is happens at step 1 where the
// specification puts it.

// sortArticles, sortRemovedCharacters and sortReplacedCharacters are the
// measured defaults of the three lists 003 §3.7.1 configures, in its order
// `[probe: tools/probe_sort_names.py, Jellyfin 10.11.11, 2026-08-26]`.
//
// They are server configuration in the reference rather than protocol, and
// Atrium exposes them with the same defaults and honours them the same way
// (§3.7.1). Until something exposes them there is nothing to configure them
// with, which is the same argument `admittedExtensions` makes for itself.
var (
	sortArticles           = []string{"the", "a", "an"}
	sortRemovedCharacters  = []rune{',', '&', '-', '{', '}', '\''}
	sortReplacedCharacters = []rune{'.', '+', '%'}
)

// sortPadWidth is 003 §3.7.1's step 5: every run of digits is left-padded with
// zeros to this width.
//
// **It is part of the contract and not a tuning knob.** Numeric ordering here
// is not numeric comparison — it is lexical comparison over zero-padded runs,
// so a width that differs from the reference's orders two names whose digit
// runs differ in length differently.
//
// `TestThePadWidthIsPinnedByBytesAndNotByTheOrderingItExistsFor` is the check,
// and it records the thing a comment cannot: the `2 Fast 2 Furious` against
// `10 Things` ordering this width exists for holds at 9, 10 and 11 alike, so
// that ordering does **not** pin the width. Only the bytes do.
const sortPadWidth = 10

// SortKeyBase is 003 §3.7.1's six ordered steps, and the order is the
// specification's rather than any convenient one.
//
//  1. Trim surrounding whitespace, then lowercase
//  2. Remove each configured article: from the start when followed by a
//     space, from anywhere when surrounded by spaces, from the end when
//     preceded by a space
//  3. Remove each configured character outright
//  4. Replace each configured character with a space
//  5. Left-pad every run of digits with zeros to a fixed width
//  6. Fold diacritics; transliterate anything still outside ASCII
//
// **Step 2 running before steps 3 and 4 is load-bearing.** `S.W.A.T.` keys as
// `s w a t ` precisely because the article `a` is not surrounded by spaces at
// the moment articles are removed; run step 4 first and the `a` disappears,
// giving `s w  t ` and a different place in every list.
//
// # It takes a name and never an item, deliberately
//
// Movies, series, albums, artists and playlists use this. `Audio`, `Episode`
// and `Season` do **not** use it at all, and calling it with one of their names
// is the failure behaviours §2.6 names. That is why this signature is a
// `string`: [SortKeyFor] is what an item is keyed with, and a caller holding
// only a name — 005's by-name row, 004's own re-derivation after it replaces a
// name — cannot get the type wrong because it never had one.
func SortKeyBase(name string) string {
	// Step 1.
	s := strings.ToLower(trimSurroundingWhitespace(name))

	// Step 2.
	for _, article := range sortArticles {
		s = removeArticle(s, article)
	}

	// Step 3.
	s = removeCharacters(s, sortRemovedCharacters)

	// Step 4.
	s = replaceCharactersWithSpace(s, sortReplacedCharacters)

	// Step 5.
	s = padDigitRuns(s, sortPadWidth)

	// Step 6.
	return foldToASCII(s)
}

// SortKeyFor is the single entry point for keying an item, and the only
// function here that reads a type.
//
// It answers, in this order:
//
//   - an explicit sort title, when the item carries one, by 003 §3.7.3 — for
//     **every** type, the three that override included;
//   - the numeric-prefix derivation of §3.7.2, for `Audio`, `Episode` and
//     `Season`, which append the **raw** name with none of §3.7.1 applied;
//   - [SortKeyBase] for everything else.
//
// The three overriding derivations are unexported and this is the only thing
// that reaches them, which is behaviours §2.6's second named temptation made
// structurally hard rather than warned about: there is no way to hand an item
// to the wrong derivation, because there is no other function an item fits.
func SortKeyFor(item *ports.ScannedItem) string {
	if item.SortTitle != "" {
		return sortKeyFromTitle(item.SortTitle)
	}

	switch Kind(item.Type) {
	case KindAudio:
		// Disc padded to 4, track padded to 4, then the raw name
		// `[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:94-98 @ v10.11.11]`.
		return numberSegment(item.ParentIndexNumber, 4) +
			numberSegment(item.IndexNumber, 4) +
			item.Name

	case KindEpisode:
		// Season padded to **3** and episode padded to **4**. The asymmetry is
		// real and is not a transcription error
		// `[source: MediaBrowser.Controller/Entities/TV/Episode.cs:238-242 @ v10.11.11]`.
		return numberSegment(item.ParentIndexNumber, 3) +
			numberSegment(item.IndexNumber, 4) +
			item.Name

	case KindSeason:
		// Four digits and nothing else — and the raw name when there is no
		// number, which 003 §3.7.2 did not state and the source settles
		// `[source: MediaBrowser.Controller/Entities/TV/Season.cs:149-152 @ v10.11.11]`.
		// A season with no number is not hypothetical: §3.4 infers one from an
		// episode's filename, and the reference's own reading of this
		// project's fixture tree contains a `Season Unknown`.
		if item.IndexNumber == nil {
			return item.Name
		}
		return padNumber(*item.IndexNumber, 4)

	default:
		return SortKeyBase(item.Name)
	}
}

// sortKeyFromTitle is 003 §3.7.3: an explicit sort title replaces the
// derivation entirely, for every type, and is lowercased and digit-padded but
// **not** article-stripped.
//
// The specification names those three properties and is silent about the other
// three steps. The source settles the silence, and settles it the same way the
// specification's sentence reads — the forced name is passed through the
// padding and folding step and then lowercased, and through nothing else
// `[source: MediaBrowser.Controller/Entities/BaseItem.cs:535-536,964-1005 @ v10.11.11]`.
// So no trim, no article removal, no character removal and no character
// replacement: a sort title an operator or a tag wrote is taken as written.
//
// The order here is the source's — pad, fold, then lowercase — rather than
// §3.7.1's, which lowercases first. For every input either order agrees; it is
// written this way so that the two derivations cannot be read as one with a
// flag, because they are not.
func sortKeyFromTitle(title string) string {
	return strings.ToLower(foldToASCII(padDigitRuns(title, sortPadWidth)))
}

// numberSegment is one number of §3.7.2's prefix, with its separator, or the
// empty string.
//
// **A missing number contributes no segment at all** — not a run of zeros. The
// separator belongs to the segment rather than to a join between segments,
// which is what makes that true without a special case: the reference formats
// the number and the ` - ` together
// `[source: MediaBrowser.Controller/Entities/Audio/Audio.cs:96-97 @ v10.11.11]`.
//
// The difference matters more than it looks. A run of zeros for an absent
// number sorts everything unnumbered ahead of everything numbered, in every
// album and every season, and it is what the obvious implementation produces.
func numberSegment(n *int, width int) string {
	if n == nil {
		return ""
	}
	return padNumber(*n, width) + " - "
}

// padNumber left-pads a number with zeros to width, and never truncates one
// that is already longer. Episode 12345 keys as `12345`, because a library
// holding one is not a library whose ordering should collapse.
func padNumber(n, width int) string {
	digits := strconv.Itoa(n)
	if len(digits) >= width {
		return digits
	}
	return strings.Repeat("0", width-len(digits)) + digits
}

// trimSurroundingWhitespace is §3.7.1's step 1, and it exists rather than
// [strings.TrimSpace] so that the only trim in this file is the one the
// specification asks for, at the step it asks for it.
//
// That is not a stylistic preference. `s w a t ` and `rock  roll` are the
// contract; a trim reached for anywhere later silently destroys them, and a
// trailing space is invisible in a diff.
func trimSurroundingWhitespace(s string) string {
	start := 0
	for start < len(s) {
		r, size := utf8.DecodeRuneInString(s[start:])
		if !unicode.IsSpace(r) {
			break
		}
		start += size
	}

	end := len(s)
	for end > start {
		r, size := utf8.DecodeLastRuneInString(s[:end])
		if !unicode.IsSpace(r) {
			break
		}
		end -= size
	}

	return s[start:end]
}

// removeArticle is §3.7.1's step 2 for one article: from the start when
// followed by a space, from anywhere when surrounded by spaces, from the end
// when preceded by a space — in that order.
//
// The middle rule replaces ` article ` with a single space, which is why
// `Once The Time` keys as `once time` and not `once  time`: the two spaces
// around the article become the one space that was between the two words.
//
// The scan is over bytes and that is safe rather than sloppy: an article and a
// space are ASCII, UTF-8 is self-synchronising, and no multi-byte rune contains
// an ASCII byte. Matches are non-overlapping and left to right.
func removeArticle(s, article string) string {
	if strings.HasPrefix(s, article+" ") {
		s = s[len(article)+1:]
	}

	surrounded := " " + article + " "
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], surrounded) {
			b.WriteByte(' ')
			i += len(surrounded)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	s = b.String()

	if strings.HasSuffix(s, " "+article) {
		s = s[:len(s)-len(article)-1]
	}
	return s
}

// removeCharacters is §3.7.1's step 3: each configured character disappears and
// nothing takes its place. `Wall-E` becomes `walle`; `Rock & Roll` becomes
// `rock  roll`, with the two spaces that surrounded the `&` still there.
func removeCharacters(s string, set []rune) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if containsRune(set, r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// replaceCharactersWithSpace is §3.7.1's step 4: each configured character
// becomes one space, including a trailing one. `S.W.A.T.` becomes `s w a t `
// and the space at the end is the contract.
func replaceCharactersWithSpace(s string, set []rune) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if containsRune(set, r) {
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func containsRune(set []rune, r rune) bool {
	for _, c := range set {
		if c == r {
			return true
		}
	}
	return false
}

// padDigitRuns is §3.7.1's step 5: **every** run of digits is left-padded with
// zeros to width, not only the leading one — which is why `2 Fast 2 Furious`
// keys as `0000000002 fast 0000000002 furious`.
//
// A digit is Unicode's decimal-digit category rather than ASCII `0`-`9`, which
// is the reference's own predicate
// `[source: MediaBrowser.Controller/Entities/BaseItem.cs:964-996 @ v10.11.11]`.
// The zeros appended are ASCII, and step 6 drops whatever a non-ASCII digit
// leaves behind.
//
// width is a parameter rather than a read of [sortPadWidth] so that the test
// can put the contract's width beside two neighbouring ones and show what the
// ordering does and does not pin.
func padDigitRuns(s string, width int) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !unicode.IsDigit(r) {
			b.WriteRune(r)
			i += size
			continue
		}

		run := 0
		j := i
		for j < len(s) {
			d, dsize := utf8.DecodeRuneInString(s[j:])
			if !unicode.IsDigit(d) {
				break
			}
			run++
			j += dsize
		}
		for k := run; k < width; k++ {
			b.WriteByte('0')
		}
		b.WriteString(s[i:j])
		i = j
	}

	return b.String()
}

// latinReadings is 003 §3.7.1 step 6's *"transliterate anything still outside
// ASCII"*, as far as this project is entitled to take it: a short table of the
// obvious Latin readings, applied after the fold and before the drop.
//
// **It is a decision and not a reproduction, and that is OQ-7.** The step was
// measured over one name — `Amélie` — whose `é` decomposes, so folding alone
// reaches it. What the reference does with a character that has no ASCII
// decomposition was never sent to it; the source shows it calling a
// transliteration of its own
// `[source: MediaBrowser.Controller/Entities/BaseItem.cs:999-1003 @ v10.11.11]`,
// and a source reading is not a measurement. So this table claims nothing about
// the reference. What it claims is that the answer is **stable**: the same name
// keys the same way twice, which a partial guess would not guarantee and which
// is the whole of what §3.7.1's OQ-7 note asks for.
//
// Both cases are listed. §3.7.1 lowercases before it reaches step 6 and §3.7.3
// lowercases after, so an uppercase form does reach here.
var latinReadings = map[rune]string{
	'æ': "ae", 'Æ': "AE",
	'œ': "oe", 'Œ': "OE",
	'ø': "o", 'Ø': "O",
	'ß': "ss", 'ẞ': "SS",
	'đ': "d", 'Đ': "D",
	'ð': "d", 'Ð': "D",
	'þ': "th", 'Þ': "TH",
	'ł': "l", 'Ł': "L",
	'ħ': "h", 'Ħ': "H",
	'ŋ': "ng", 'Ŋ': "NG",
	'ı': "i", 'İ': "I",
}

// foldToASCII is §3.7.1's step 6: fold diacritics, then transliterate what is
// still outside ASCII, then drop what remains.
//
// The fold is canonical decomposition with the combining marks discarded, which
// is what turns `Amélie` into `amelie` and is the reference's own
// `RemoveDiacritics`
// `[source: MediaBrowser.Controller/Entities/BaseItem.cs:999 @ v10.11.11]`.
// Dropping is the last resort and it is chosen because it is stable: a name in
// a script this table does not know keys the same way every time, and every
// time it is scanned.
func foldToASCII(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range norm.NFD.String(s) {
		switch {
		case r < utf8.RuneSelf:
			b.WriteRune(r)
		case unicode.Is(unicode.Mn, r):
			// A combining mark: this is the fold.
		default:
			// A reading if there is one, and otherwise nothing — OQ-7.
			b.WriteString(latinReadings[r])
		}
	}

	return b.String()
}
