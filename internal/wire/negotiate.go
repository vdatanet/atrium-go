package wire

import (
	"strconv"
	"strings"
)

// jsonMediaType is the one media type this server has a representation for.
// Every route in v1 answers JSON or answers nothing (behaviours 1.11).
const jsonMediaType = "application/json"

// Negotiate chooses the content profile one request is answered under, from the
// value of its `Accept` header (spec 3.0.2, plan 6.3).
//
// It takes the header value rather than a request, because the domain of this
// function is a string and taking an *http.Request would let a caller believe
// it also reads the method, the path or a cookie. Where a request carries more
// than one `Accept` field line, join them with a comma first: RFC 9110 §5.3
// makes that the same header.
//
// # The four rules
//
// Each is behaviours 1.13's, and each has a row in the table test:
//
//  1. The `profile` parameter is compared **case-insensitively and unquoted**,
//     so `profile=CamelCase` and `profile="camelcase"` both match.
//  2. A `charset` parameter **beside** `profile` stops the profile match, and
//     the range falls back to the plain type.
//  3. An unknown `profile` value falls back to the plain type too.
//  4. Ranking is by `q` descending, and **on a tie the client's order wins** —
//     which is why `application/json, application/json; profile="CamelCase"`
//     answers plain and the reverse order answers camelCase.
//
// # What it does when nothing matches
//
// ProfilePlain. No `Accept` at all, an `Accept` naming only media types this
// server has no representation for, and an `Accept` in which every acceptable
// range carries `q=0` all end there. This server never answers 406: behaviours
// 1.11 measured four refusal shapes and none of them is one, and the plain type
// is a declared representation the client can read whatever it asked for.
func Negotiate(accept string) Profile {
	winner, winnerQuality, found := ProfilePlain, 0.0, false

	for _, text := range splitOutsideQuotes(accept, ',') {
		media, ok := parseMediaRange(text)
		if !ok || !media.acceptsJSON() {
			continue
		}

		// `q=0` is "not acceptable" and not a vote for anything, so the range
		// is skipped rather than ranked last.
		if media.quality <= 0 {
			continue
		}

		// Strictly greater is where rule 4's tie-break lives: an equal quality
		// leaves the earlier range in place, and the ranges arrive in the
		// order the client wrote them.
		if !found || media.quality > winnerQuality {
			winner, winnerQuality, found = media.profile(), media.quality, true
		}
	}

	return winner
}

// mediaRange is one comma-separated entry of an `Accept` header, parsed.
type mediaRange struct {
	// mediaType is lower-cased: `type/subtype` is case-insensitive.
	mediaType string

	// parameters holds the **media type's** parameters — those written before
	// `q` — with lower-cased names and unquoted values. Anything after `q` is
	// an accept-extension and belongs to the negotiation rather than to the
	// media type (RFC 9110 §12.5.1), so it is not here and cannot select a
	// profile. ⚠️ UNVERIFIED against the reference: no probe has sent a
	// `profile` after a `q`, and the rule is the RFC's reading rather than a
	// measurement.
	parameters map[string]string

	// quality defaults to 1 when the range names no `q`.
	quality float64
}

// parseMediaRange reads one range. It reports false for an entry with no media
// type at all, which is what a trailing or doubled comma produces.
func parseMediaRange(text string) (mediaRange, bool) {
	pieces := splitOutsideQuotes(text, ';')

	mediaType := strings.ToLower(strings.TrimSpace(pieces[0]))
	if mediaType == "" {
		return mediaRange{}, false
	}

	media := mediaRange{
		mediaType:  mediaType,
		parameters: make(map[string]string, len(pieces)-1),
		quality:    1,
	}

	for _, piece := range pieces[1:] {
		name, value, ok := parseParameter(piece)
		if !ok {
			continue
		}

		if name == "q" {
			// An unparseable quality is treated as absent rather than as zero:
			// a malformed `q` is a client that meant to ask for something, and
			// refusing to consider the range would answer it with a profile it
			// did not name. ⚠️ UNVERIFIED — no probe has sent one.
			if quality, err := strconv.ParseFloat(value, 64); err == nil {
				media.quality = quality
			}
			break
		}

		media.parameters[name] = value
	}

	return media, true
}

// acceptsJSON reports whether this range covers the one representation the
// server has.
//
// A wildcard counts. The reference sets `RespectBrowserAcceptHeader = true`
// with the comment "Allow requester to change between camelCase and
// PascalCase" `[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:125-126
// @ v10.11.11]`, so a `*/*` is not discarded before ranking the way the
// framework's default would discard it — it ranks as an ordinary range that
// names no profile.
func (m mediaRange) acceptsJSON() bool {
	switch m.mediaType {
	case jsonMediaType, "application/*", "*/*":
		return true
	}
	return false
}

// profile is what this range asks to be answered with.
func (m mediaRange) profile() Profile {
	// A wildcard names no profile. `*/*; profile="CamelCase"` is not a request
	// for camelCase JSON; it is a request for anything, with a parameter that
	// belongs to a media type it did not name.
	if m.mediaType != jsonMediaType {
		return ProfilePlain
	}

	value, named := m.parameters["profile"]
	if !named {
		return ProfilePlain
	}

	// Rule 2. The charset is checked before the value is read, because it does
	// not matter what the profile said: the range falls back to plain whether
	// the profile was one of the two or not.
	if _, charset := m.parameters["charset"]; charset {
		return ProfilePlain
	}

	// Rule 1: unquoted already, and compared without regard to case. Rule 3 is
	// the default arm.
	switch strings.ToLower(value) {
	case "pascalcase":
		return ProfilePascal
	case "camelcase":
		return ProfileCamel
	}
	return ProfilePlain
}

// parseParameter reads one `name=value` of a media range. A piece with no `=`
// is not a parameter and is dropped.
func parseParameter(text string) (name, value string, ok bool) {
	name, value, found := strings.Cut(text, "=")
	if !found {
		return "", "", false
	}

	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return "", "", false
	}

	return name, unquote(strings.TrimSpace(value)), true
}

// unquote strips a quoted-string's delimiters and its backslash escapes. A
// value that is not a well-formed quoted string is returned as it stands, which
// is how an unquoted `profile=CamelCase` survives rule 1.
func unquote(text string) string {
	if len(text) < 2 || text[0] != '"' || text[len(text)-1] != '"' {
		return text
	}

	var unquoted strings.Builder
	escaped := false
	for _, char := range text[1 : len(text)-1] {
		switch {
		case escaped:
			unquoted.WriteRune(char)
			escaped = false
		case char == '\\':
			escaped = true
		default:
			unquoted.WriteRune(char)
		}
	}
	return unquoted.String()
}

// splitOutsideQuotes splits on a separator, ignoring the ones inside a quoted
// string.
//
// It is why the parse is written by hand rather than as two calls to
// strings.Split. A parameter value is allowed to be a quoted string, and a
// quoted string is allowed to contain a comma or a semicolon — so a naive split
// would cut one range in half and leave the tail looking like a range of its
// own. Every such tail this project could construct is a media type the server
// has no representation for, and is therefore dropped, so the defect would not
// show in an answer; that is an argument for parsing correctly rather than for
// not bothering, because the next parameter this API meets may not be so kind.
func splitOutsideQuotes(text string, separator byte) []string {
	pieces := make([]string, 0, strings.Count(text, string(separator))+1)

	start, quoted, escaped := 0, false, false
	for i := 0; i < len(text); i++ {
		switch {
		case escaped:
			escaped = false
		case quoted && text[i] == '\\':
			escaped = true
		case text[i] == '"':
			quoted = !quoted
		case !quoted && text[i] == separator:
			pieces = append(pieces, text[start:i])
			start = i + 1
		}
	}

	return append(pieces, text[start:])
}
