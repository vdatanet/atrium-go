package library

import (
	"strings"
)

// This file turns a filename into a title and a year, and it is the one place
// in this package where Principle IV has a sharp edge.
//
// The reference does the same job with six regular expressions and two more
// `[source: Emby.Naming/Common/NamingOptions.cs:147-161 @ v10.11.11]`. Those
// expressions are GPL-licensed source and translating one would be forking it,
// which Principle IV forbids and [ADR-0005] turns into a licensing argument. So
// what is reimplemented here is the **rule** each of them states, read at the
// pinned tag and written out in this project's own terms, and what is asserted
// is a corpus of names rather than an expression: 003 §3.3 says so in terms —
// *"Atrium reimplements the rules, not the expressions. The acceptance test is
// behavioural: given a corpus of real-world names, the same title and year come
// out."*
//
// The one thing that **is** transcribed is the release-tag vocabulary, and it
// is transcribed the way this package already transcribes the extras folder
// names and the extras suffixes: as measured data with the line it came from
// beside it. A vocabulary is a list of facts about what the reference
// recognises; an expression is a program.
//
// [ADR-0005]: ../../docs/decisions/0005-licence.md

// nameSeparators are the characters that delimit a year and a release tag from
// the title around them.
//
// The set is the reference's, and the **comma is in it and the space is not**
// for one of the two places it is used: the year's own separator class excludes
// the comma and includes the space, while the trailing run stripped off a title
// excludes the space and includes the comma
// `[source: Emby.Naming/Common/NamingOptions.cs:147-151 @ v10.11.11]`. The
// asymmetry looks like an oversight upstream and is load-bearing here: it is
// the whole reason `S.W.A.T. (2003)` keeps its final full stop while
// `S.W.A.T. 2003` loses it. Both sets are written out separately below rather
// than derived from one another, because a reader has to be able to see that
// they differ.
const (
	// yearSeparators is the run that may stand between a title and the year
	// that follows it. The space is in it; the comma is not.
	yearSeparators = " _.()[]-"

	// yearSingleSeparators is the same run's single-character form, and it is
	// the space that is missing rather than the comma. Both are the
	// reference's, one per date expression, and [titleBeforeYear] says why
	// having two matters.
	yearSingleSeparators = "_.()[]-"

	// titleTrailingSeparators are stripped off the end of a title once the
	// year has been taken out. The space is deliberately absent: the
	// reference's own capture may end in one, and trimming it here would
	// expose the full stop behind it to the same strip.
	titleTrailingSeparators = "_,.()[]-"

	// tagSeparators delimit a release tag. This is the class the reference's
	// first clean-string expression uses on both sides of the token
	// `[source: Emby.Naming/Common/NamingOptions.cs:153 @ v10.11.11]`.
	tagSeparators = " _,.()[]-"
)

// releaseTags is the vocabulary of release-tag noise, transcribed from the
// alternation of the reference's first clean-string expression, in that
// expression's own order and with `cd[1-9]` written out
// `[source: Emby.Naming/Common/NamingOptions.cs:153 @ v10.11.11]`.
//
// 003 §3.3 names the categories rather than the tokens — *"resolution, source,
// codec, audio format, language, release-group brackets"* — and every entry
// here belongs to one of them. Matching ignores ASCII case, which is why the
// entries the source spells in capitals are spelled in lower case here and are
// not duplicated.
//
// A token containing a separator character (`blu-ray`, `read.nfo`,
// `www.www`) is matched **literally**, where the expression's `.` would match
// any character. That is a narrowing, and it is the safe direction: it refuses
// a name the reference would have cleaned rather than cleaning one it would
// have left alone.
//
// The list is never iterated to produce output. It is read in order and the
// first entry that matches at a position wins, and because every match must be
// delimited on both sides the order between two entries sharing a prefix
// (`bd`, `bd5`, `bdrip`) changes nothing.
var releaseTags = []string{
	"3d", "sbs", "tab", "hsbs", "htab", "mvc", "hdr", "hdc", "uhd", "ultrahd",
	"4k", "ac3", "dts", "custom", "dc", "divx", "divx5", "dsr", "dsrip",
	"dutch", "dvd", "dvdrip", "dvdscr", "dvdscreener", "screener", "dvdivx",
	"cam", "fragment", "fs", "hdtv", "hdrip", "hdtvrip", "internal", "limited",
	"multi", "subs", "ntsc", "ogg", "ogm", "pal", "pdtv", "proper", "repack",
	"rerip", "retail",
	"cd1", "cd2", "cd3", "cd4", "cd5", "cd6", "cd7", "cd8", "cd9",
	"r5", "bd5", "bd", "se", "svcd", "swedish", "german", "read.nfo",
	"nfofix", "unrated", "ws", "telesync", "ts", "telecine", "tc", "brrip",
	"bdrip", "480p", "480i", "576p", "576i", "720p", "720i", "1080p", "1080i",
	"2160p", "hrhd", "hrhdtv", "hddvd", "bluray", "blu-ray", "x264", "x265",
	"h264", "h265", "xvid", "xvidvd", "xxx", "www.www", "aac",
}

// cleanVideoName turns a filename's stem — or a directory's name — into the
// title and the year 003 §3.3 asks for.
//
// The two steps run in the reference's own order, and the order is not
// editorial: the year is taken out first and the release tags second
// `[source: Emby.Naming/Video/VideoResolver.cs:88-96 @ v10.11.11]`. Reversed,
// `The.Film.2019.1080p.BluRay` loses its year with the noise, because the
// noise-cutting rule cuts at `1080p` and the year is behind it.
//
// The returned name is trimmed of surrounding whitespace, which is 003 §3.5's
// own word for a path-derived name and a **declared difference** from the
// reference: the reference's reading of this project's own fixture names the
// padded film `"  Padded"` with its two leading spaces intact
// `[probe: tools/probe_reference_scan.py, Jellyfin 10.11.11, 2026-09-02]`.
//
// A name that cleans away to nothing keeps its stem: an item with no name is
// not a better answer than an item named after a file nobody can tidy.
func cleanVideoName(stem string) (string, *int) {
	title, year := takeYear(stem)
	title = removeReleaseTags(title)
	title = strings.TrimSpace(title)
	if title == "" {
		title = strings.TrimSpace(stem)
	}
	return title, year
}

// takeYear removes a four-digit year and everything after it, returning what is
// left and the year.
//
// The rule, read at the pinned tag
// `[source: Emby.Naming/Common/NamingOptions.cs:147-151 @ v10.11.11]`:
//
//   - The year is four digits whose value is **1900–2099**, so `1899` and
//     `2100` are part of the title and not a year. 003 §3.3 states the range.
//   - It is preceded by at least one character of [yearSeparators] and by at
//     least one character of title before that, so `1917` is a film called
//     `1917` and not a film with no name made in 1917.
//   - It is not adjacent to another digit, so `19999` is not a year.
//   - It is not the leading year of a date — two digits and two digits behind
//     it — so `The Daily Show - 2024-01-31` has no production year, which is
//     what keeps 003 §3.4's date-named episodes out of this rule.
//   - The **last** such year in the name wins, which is what makes
//     `Blade Runner 2049 (2017)` a 2017 film called `Blade Runner 2049` rather
//     than a 2049 film called `Blade Runner`.
//   - Everything after the year is discarded with it.
//
// What is left has its trailing [titleTrailingSeparators] removed and its
// whitespace left alone. That is the asymmetry [titleTrailingSeparators]
// documents, and it is why `S.W.A.T. (2003)` is `S.W.A.T.` here.
func takeYear(s string) (string, *int) {
	found := -1
	value := 0

	for i := 0; i+4 <= len(s); i++ {
		if !isYearRun(s, i) {
			continue
		}
		v := int(s[i]-'0')*1000 + int(s[i+1]-'0')*100 + int(s[i+2]-'0')*10 + int(s[i+3]-'0')
		if v < 1900 || v > 2099 {
			continue
		}
		if i < 2 || !strings.ContainsRune(yearSeparators, rune(s[i-1])) {
			continue
		}
		if looksLikeDateTail(s[i+4:]) {
			continue
		}
		found = i
		value = v
	}

	if found < 0 {
		return s, nil
	}
	title, ok := titleBeforeYear(s, found)
	if !ok {
		return s, nil
	}
	year := value
	return title, &year
}

// titleBeforeYear is what is left of a name once the year at found and the
// separator run before it are taken off.
//
// It is two rules and not one, and that is the reference's shape rather than a
// refinement: the first of its two date expressions takes a **single**
// separator character which may not be a space, and the second takes a run
// which may `[source: Emby.Naming/Common/NamingOptions.cs:147-151 @ v10.11.11]`.
// Both require what is left to end in a character that is not one of
// [titleTrailingSeparators] — a set which excludes the space and includes the
// comma.
//
// The consequence is the pair worth having in a corpus:
//
//	S.W.A.T. (2003)  →  "S.W.A.T. "     the first rule, one separator "("
//	The Film - 1999  →  "The Film"      the second, the run "- "
//
// Under one rule instead of two, the first of those loses its final full stop.
func titleBeforeYear(s string, found int) (string, bool) {
	if strings.ContainsRune(yearSingleSeparators, rune(s[found-1])) {
		if head := s[:found-1]; endsOutsideTitleSeparators(head) {
			return head, true
		}
	}
	if head := strings.TrimRight(s[:found], yearSeparators); endsOutsideTitleSeparators(head) {
		return head, true
	}
	return "", false
}

// endsOutsideTitleSeparators reports whether s is non-empty and does not end in
// one of [titleTrailingSeparators].
func endsOutsideTitleSeparators(s string) bool {
	if s == "" {
		return false
	}
	return !strings.ContainsRune(titleTrailingSeparators, rune(s[len(s)-1]))
}

// isYearRun reports whether s[i:i+4] is four digits with no digit on either
// side of the run.
func isYearRun(s string, i int) bool {
	for j := i; j < i+4; j++ {
		if !isASCIIDigit(s[j]) {
			return false
		}
	}
	if i > 0 && isASCIIDigit(s[i-1]) {
		return false
	}
	if i+4 < len(s) && isASCIIDigit(s[i+4]) {
		return false
	}
	return true
}

// looksLikeDateTail reports whether what follows a four-digit run is the rest
// of a date: a non-word character, two digits, a non-word character and two
// digits. It is the reference's own guard, and without it a date-named episode
// acquires a production year from its own air date.
func looksLikeDateTail(rest string) bool {
	if len(rest) < 6 {
		return false
	}
	return !isWordByte(rest[0]) &&
		isASCIIDigit(rest[1]) && isASCIIDigit(rest[2]) &&
		!isWordByte(rest[3]) &&
		isASCIIDigit(rest[4]) && isASCIIDigit(rest[5])
}

func isASCIIDigit(b byte) bool { return b >= '0' && b <= '9' }

// isWordByte is the reference's `\w` for the bytes this guard can meet: an
// ASCII letter, a digit or an underscore. A byte of a multi-byte rune is not
// one, which is the conservative answer — it makes the date guard fire less
// often, and the guard's failure mode is a title losing a year it should have
// kept.
func isWordByte(b byte) bool {
	return b == '_' || isASCIIDigit(b) ||
		(b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// removeReleaseTags cuts release-tag noise off a title.
//
// It applies the three rules 003 §3.3 names, in the reference's own order and
// feeding each one's output into the next
// `[source: Emby.Naming/Video/CleanStringParser.cs:29-38 @ v10.11.11]` —
// the reference applies every expression in turn rather than stopping at the
// first that matches:
//
//  1. **A delimited release tag** cuts the title at the separator before it.
//     The earliest such tag wins, because the reference's capture is
//     non-greedy: `The Film 1080p BluRay` is `The Film`, not `The Film 1080p`.
//  2. **A release-group bracket** cuts the title at the bracket.
//  3. **A leading release-group bracket** is removed and what follows is kept,
//     which is the one of the three that does not cut a title short.
//
// The reference's other three clean-string expressions are not here and each
// has an owner: an episode range and a trailing number are 003 §3.4's, and the
// extras suffixes are §3.2's and were implemented at extras.go. §3.3's list is
// *"resolution, source, codec, audio format, language, release-group
// brackets"* and this is all six.
func removeReleaseTags(s string) string {
	s = cutAtReleaseTag(s)
	s = cutAtReleaseGroupBracket(s)
	s = dropLeadingReleaseGroupBracket(s)
	return s
}

// cutAtReleaseTag returns the title up to the earliest separator that is
// followed by a delimited release tag.
//
// There must be at least one character of title before the separator, which is
// the reference's `.+?`: a name that is nothing but a tag keeps it, because
// cutting it leaves an item with no name at all.
func cutAtReleaseTag(s string) string {
	for i := 1; i < len(s); i++ {
		if !strings.ContainsRune(tagSeparators, rune(s[i])) {
			continue
		}
		if !releaseTagAt(s, i+1) {
			continue
		}
		return s[:i]
	}
	return s
}

// releaseTagAt reports whether a release tag begins at i and is delimited by a
// separator or the end of the string.
func releaseTagAt(s string, i int) bool {
	rest := foldASCIICase(s[i:])
	for _, tag := range releaseTags {
		if !strings.HasPrefix(rest, tag) {
			continue
		}
		if len(rest) == len(tag) {
			return true
		}
		if strings.ContainsRune(tagSeparators, rune(rest[len(tag)])) {
			return true
		}
	}
	return false
}

// cutAtReleaseGroupBracket returns the title up to the first `[` that has a
// `]` behind it, when there is title before the bracket.
func cutAtReleaseGroupBracket(s string) string {
	open := strings.IndexByte(s, '[')
	if open < 1 {
		return s
	}
	if !strings.Contains(s[open+1:], "]") {
		return s
	}
	return s[:open]
}

// dropLeadingReleaseGroupBracket removes a `[…]` the title begins with and
// returns what follows it, when anything does.
func dropLeadingReleaseGroupBracket(s string) string {
	trimmed := strings.TrimLeft(s, " ")
	if !strings.HasPrefix(trimmed, "[") {
		return s
	}
	end := strings.IndexByte(trimmed, ']')
	if end < 2 {
		return s
	}
	rest := strings.TrimLeft(trimmed[end+1:], " ")
	if rest == "" {
		return s
	}
	return rest
}

// stemOfPath is the filename with its extension removed. It is [stemOf] under
// another name for a caller holding a path rather than a base name, and it
// exists so that no caller has to remember to take the base first.
func stemOfPath(relPath string) string {
	return stemOf(baseName(relPath))
}

// baseName is the last element of a slash-separated path. It is `path.Base`
// without the special cases: a reading's path is never empty, never absolute
// and never ends in a separator.
func baseName(relPath string) string {
	if i := strings.LastIndexByte(relPath, '/'); i >= 0 {
		return relPath[i+1:]
	}
	return relPath
}

// dirName is everything before the last separator, and "." when there is none.
// A path directly under a library root therefore has "." for its directory,
// which is the value that keeps a root from ever being read as a film's own
// folder.
func dirName(relPath string) string {
	i := strings.LastIndexByte(relPath, '/')
	if i < 0 {
		return "."
	}
	return relPath[:i]
}
