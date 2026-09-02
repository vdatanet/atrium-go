package wire

import "unicode/utf8"

// upperHexDigits is the alphabet an escape is written in. behaviours 1.16
// measured the reference's hex as upper case — `28 a\u00F1os despu\u00E9s`,
// not `\u00f1` — and it is the one detail of this pass a JSON parser cannot
// see, which is why it is asserted on bytes and not on a decoded value.
const upperHexDigits = "0123456789ABCDEF"

// escapedASCII are the seven ASCII characters behaviours 1.16 measured as
// escaped, against the ten it measured as literal (`/` `=` `:` space `!` `*`
// `(` `)` `-` `_`).
//
// The set is not guessable from a library's contents: item names give the
// accented characters and the apostrophe and say nothing about a backtick. It
// came from echoing arbitrary characters back through a validation error
// `[probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]`.
//
// Note the double quote. JSON's own escape for it is \", and the reference does
// not use it — which is the reason this pass has to know where a string starts
// and ends, since a structural quote must stay exactly one byte.
const escapedASCII = `"&'+<>` + "`"

// escape rewrites an encoded JSON document into the bytes behaviours 1.16
// measured: every non-ASCII character and the seven ASCII ones as \uXXXX with
// upper-case hex, everything else as the encoder left it.
//
// # Why it counts backslashes instead of looking for \u
//
// The hard case is a string whose *value* contains the six characters `\u00e9`.
// A pass that searched for the prefix would find them and upper-case a value
// the client sent, changing data that only looks like an escape. The encoder
// has already doubled every literal backslash by the time this runs, so the
// distinction is exact: a backslash consumes the byte after it, and the pair
// \\ is copied whole. What follows a copied pair is ordinary text, and `u00e9`
// after one is five ordinary characters.
//
// That mechanism is what withdrew behaviours 4.4, an exception taken on the
// argument that the upper-casing could not be done safely. It can.
//
// # Why it tracks whether it is inside a string
//
// Only so that a structural quote survives. Inside a string the encoder never
// emits a raw quote, so every raw quote is a delimiter and every \" is a value
// the reference would have written as ".
//
// The input is assumed to be an encoder's output, not arbitrary bytes. Where
// that assumption could fail — a truncated escape, invalid UTF-8 — the pass
// copies what it cannot interpret rather than guessing, so it never produces
// something the caller did not put in.
func escape(src []byte) []byte {
	// Nothing shrinks and most bodies grow a little; one allocation is enough
	// for a body that is mostly ASCII, which every response in 001 is.
	out := make([]byte, 0, len(src)+len(src)/8)

	inString := false

	for i := 0; i < len(src); {
		c := src[i]

		switch {
		case !inString:
			// Structure, numbers and the three bare literals. All ASCII, none
			// of them escapable, and a quote here opens a string.
			if c == '"' {
				inString = true
			}
			out = append(out, c)
			i++

		case c == '"':
			// The closing delimiter. Never a character of the value: the
			// encoder wrote those as \".
			inString = false
			out = append(out, c)
			i++

		case c == '\\':
			out, i = appendEscapeSequence(out, src, i)

		case c < utf8.RuneSelf:
			if isEscapedASCII(c) {
				out = appendCodeUnit(out, rune(c))
			} else {
				out = append(out, c)
			}
			i++

		default:
			// DecodeRune returns (RuneError, 1) on a byte that starts nothing
			// valid, so a malformed sequence becomes one `\uFFFD` and the scan
			// advances. Go's encoder replaces invalid UTF-8 before this pass
			// sees it, so this is a guard rather than a path.
			r, size := utf8.DecodeRune(src[i:])
			out = appendRuneEscape(out, r)
			i += size
		}
	}

	return out
}

// appendEscapeSequence handles the backslash at src[i] together with whatever
// it escapes, and returns the index just past the pair. Consuming the two
// together is the parity count: a run of backslashes is read as pairs, so only
// an odd one out can begin an escape.
func appendEscapeSequence(dst []byte, src []byte, i int) ([]byte, int) {
	if i+1 >= len(src) {
		// A trailing backslash is not something the encoder produces. Copy it.
		return append(dst, src[i]), i + 1
	}

	switch src[i+1] {
	case '"':
		// The one escape the reference spells differently (behaviours 1.16).
		return appendCodeUnit(dst, '"'), i + 2

	case 'u':
		// The encoder's own \uXXXX, for a control character or for U+2028 and
		// U+2029, and it writes the hex in lower case. Only the four digits
		// are touched, and only when all four are there.
		if i+6 <= len(src) && isHexDigits(src[i+2:i+6]) {
			dst = append(dst, '\\', 'u')
			for _, digit := range src[i+2 : i+6] {
				dst = append(dst, upperHexDigit(digit))
			}
			return dst, i + 6
		}
		return append(dst, src[i], src[i+1]), i + 2

	default:
		// \\ \/ \b \f \n \r \t — copied whole, which is what makes the pair
		// \\ inert and the `u00e9` that may follow it ordinary text.
		return append(dst, src[i], src[i+1]), i + 2
	}
}

// appendRuneEscape writes one rune as \uXXXX, or as a surrogate pair when it
// is outside the basic multilingual plane.
//
// ⚠️ UNVERIFIED for the pair. behaviours 1.16 measured "every non-ASCII
// character ... as \uXXXX" on characters that all fit one code unit, and a
// character above U+FFFF cannot be written as one. A pair is what a UTF-16
// encoder emits and what a JSON parser reads back to the same rune, so it is
// the only spelling that round-trips — but it is an inference, not a
// measurement, and it wants a probe that puts an emoji in a body.
func appendRuneEscape(dst []byte, r rune) []byte {
	const (
		firstSupplementary = 0x10000
		highSurrogateBase  = 0xD800
		lowSurrogateBase   = 0xDC00
	)

	if r < firstSupplementary {
		return appendCodeUnit(dst, r)
	}

	r -= firstSupplementary
	dst = appendCodeUnit(dst, highSurrogateBase+(r>>10))
	return appendCodeUnit(dst, lowSurrogateBase+(r&0x3FF))
}

// appendCodeUnit writes one UTF-16 code unit as \uXXXX with upper-case hex.
func appendCodeUnit(dst []byte, u rune) []byte {
	return append(dst, '\\', 'u',
		upperHexDigits[(u>>12)&0xF],
		upperHexDigits[(u>>8)&0xF],
		upperHexDigits[(u>>4)&0xF],
		upperHexDigits[u&0xF],
	)
}

// isEscapedASCII reports whether an ASCII byte is one of the seven.
func isEscapedASCII(c byte) bool {
	for i := 0; i < len(escapedASCII); i++ {
		if escapedASCII[i] == c {
			return true
		}
	}
	return false
}

// isHexDigits reports whether every byte is a hexadecimal digit.
func isHexDigits(b []byte) bool {
	for _, c := range b {
		switch {
		case c >= '0' && c <= '9':
		case c >= 'a' && c <= 'f':
		case c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// upperHexDigit folds one hexadecimal digit to upper case and leaves anything
// else alone.
func upperHexDigit(c byte) byte {
	if c >= 'a' && c <= 'f' {
		return c - 'a' + 'A'
	}
	return c
}
