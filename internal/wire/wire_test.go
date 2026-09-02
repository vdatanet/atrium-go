package wire_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// body is the smallest model that carries a string to the wire. Its field is
// PascalCase because spec 3.0.1 says every property name is.
type body struct {
	Value string
}

// write runs one body through the package and hands back the bytes that
// reached the client.
//
// Everything below asserts on those bytes and not on a decoded value
// (Principle VIII). A JSON parser turns `\u00F1` and `ñ` into the same string,
// and lower-case hex into the same string again, so a test that unmarshalled
// could not fail on the one thing behaviours 1.16 is about.
func write(t *testing.T, v any) *httptest.ResponseRecorder {
	t.Helper()

	recorder := httptest.NewRecorder()
	if err := wire.Write(recorder, http.StatusOK, v, wire.ProfilePlain); err != nil {
		t.Fatalf("Write(...) = %v, want no error", err)
	}
	return recorder
}

// TestWriteEscapesABodyAsTheReferenceDoes is the table behaviours 1.16
// describes, read as bytes.
func TestWriteEscapesABodyAsTheReferenceDoes(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  string
	}{
		// The three strings behaviours 1.16 and 4.4 were measured on. They are
		// the only rows here that are quotations rather than constructions.
		{
			name:  "a Spanish title",
			value: "28 años después",
			want:  `{"Value":"28 a\u00F1os despu\u00E9s"}`,
		},
		{
			name:  "an apostrophe in a title",
			value: "Abraham's Boys",
			want:  `{"Value":"Abraham\u0027s Boys"}`,
		},
		{
			name:  "a culture name",
			value: "Occitan (post 1500); Provençal",
			want:  `{"Value":"Occitan (post 1500); Proven\u00E7al"}`,
		},

		// The seven ASCII characters, one row each, so a failure names which.
		{name: "a double quote", value: `"`, want: `{"Value":"\u0022"}`},
		{name: "an ampersand", value: "&", want: `{"Value":"\u0026"}`},
		{name: "an apostrophe", value: "'", want: `{"Value":"\u0027"}`},
		{name: "a plus", value: "+", want: `{"Value":"\u002B"}`},
		{name: "a less-than", value: "<", want: `{"Value":"\u003C"}`},
		{name: "a greater-than", value: ">", want: `{"Value":"\u003E"}`},
		{name: "a backtick", value: "`", want: "{\"Value\":\"\\u0060\"}"},

		// The ten left literal, in one row: the point is that none of them
		// moves, and a row each would say the same thing ten times.
		{
			name:  "the ten characters left literal",
			value: `/=: !*()-_`,
			want:  `{"Value":"/=: !*()-_"}`,
		},

		// The hard case, and the reason plan 6.4 counts backslash parity.
		{
			name:  "a value that is itself the six characters of an escape",
			value: `\u00e9`,
			want:  `{"Value":"\\u00e9"}`,
		},
		{
			name:  "the six characters beside the character they spell",
			value: "\\u00e9é",
			want:  `{"Value":"\\u00e9\u00E9"}`,
		},
		{
			name: "two literal backslashes before an escape's letters",
			// Four bytes reach the encoder as eight; the pass reads them as
			// four pairs and `u00e9` as text. A pass that searched for the
			// prefix would find one at every odd offset.
			value: `\\\\u00e9`,
			want:  `{"Value":"\\\\\\\\u00e9"}`,
		},
		{
			name:  "a lone backslash",
			value: `\`,
			want:  `{"Value":"\\"}`,
		},

		// Non-ASCII beyond the measured examples.
		{
			name:  "a character outside the basic multilingual plane",
			value: "\U0001F600",
			want:  `{"Value":"\uD83D\uDE00"}`,
		},
		{
			name: "a control character the encoder escapes itself",
			// ⚠️ UNVERIFIED that the reference spells a control character
			// \uXXXX at all — behaviours 1.16 measured the seven printable
			// ones and every non-ASCII one, not these. What is asserted here
			// is only that this pass applies 1.16's casing rule to an escape
			// the encoder produced, rather than leaving two spellings of hex
			// in one body.
			value: "\x1f",
			want:  `{"Value":"\u001F"}`,
		},

		// Structure. The quotes and colons that hold the document together are
		// not values and must not move.
		{
			name:  "the empty string",
			value: "",
			want:  `{"Value":""}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := write(t, body{Value: c.value}).Body.String()
			if got != c.want {
				t.Errorf("Write(%q) body =\n\t%s\nwant\n\t%s", c.value, got, c.want)
			}
		})
	}
}

// TestWriteEscapesInsideNestedValuesAndDictionaryKeys asserts the pass reaches
// everything, because it works on the finished document and not on a field.
//
// A dictionary key is escaped like any other string. That is not in tension
// with spec 3.0.2's "dictionary keys are never converted": conversion is the
// naming policy, which happens where a field is still a field; escaping is a
// property of the bytes and applies to every string in the body.
func TestWriteEscapesInsideNestedValuesAndDictionaryKeys(t *testing.T) {
	type inner struct {
		Deep string
	}
	type outer struct {
		Nested inner
		List   []string
		Map    map[string]string
	}

	got := write(t, outer{
		Nested: inner{Deep: "é"},
		List:   []string{"<", "ñ"},
		Map:    map[string]string{"clé": "ç"},
	}).Body.String()

	want := `{"Nested":{"Deep":"\u00E9"},` +
		`"List":["\u003C","\u00F1"],` +
		`"Map":{"cl\u00E9":"\u00E7"}}`

	if got != want {
		t.Errorf("Write(...) body =\n\t%s\nwant\n\t%s", got, want)
	}
}

// TestWriteLeavesNoNonASCIIByteInABody is the "every" in behaviours 1.16's
// title, checked rather than sampled: every code point in the basic
// multilingual plane, plus the ends of the supplementary planes.
//
// It asserts two things at once — that nothing above ASCII survives, and that
// what replaced it decodes back to the same rune, which is what stops a pass
// that escaped everything to `\u0000` from passing.
func TestWriteLeavesNoNonASCIIByteInABody(t *testing.T) {
	runes := make([]rune, 0, 0x10000)
	for r := rune(0x80); r <= 0xFFFF; r++ {
		if r >= 0xD800 && r <= 0xDFFF {
			continue // Surrogates are not characters and cannot be encoded.
		}
		runes = append(runes, r)
	}
	runes = append(runes, 0x10000, 0x1F600, 0x10FFFF)

	for _, r := range runes {
		encoded := write(t, body{Value: string(r)}).Body.String()

		for i := 0; i < len(encoded); i++ {
			if encoded[i] >= utf8.RuneSelf {
				t.Fatalf("Write(%U) body = %q, want no byte above ASCII", r, encoded)
			}
		}

		want := fmt.Sprintf(`{"Value":"%s"}`, expectedEscape(r))
		if encoded != want {
			t.Fatalf("Write(%U) body = %s, want %s", r, encoded, want)
		}
	}
}

// expectedEscape spells one rune the way UTF-16 does, computed independently of
// the package under test so that a wrong shared helper cannot make both agree.
func expectedEscape(r rune) string {
	if r < 0x10000 {
		return fmt.Sprintf(`\u%04X`, r)
	}
	offset := r - 0x10000
	return fmt.Sprintf(`\u%04X\u%04X`, 0xD800+(offset>>10), 0xDC00+(offset&0x3FF))
}

// TestWriteEscapesExactlySevenPrintableASCIICharacters walks the printable
// ASCII range and asserts the set that moves is the set behaviours 1.16
// measured — no more and no fewer.
//
// The ten characters the table calls literal are covered by this: they are ten
// of the eighty-eight this test requires to come through untouched. What the
// table does not cover, and this does, is everything in between — a semicolon,
// a percent sign, a digit.
func TestWriteEscapesExactlySevenPrintableASCIICharacters(t *testing.T) {
	const measured = "\"&'+<>`"

	for c := byte(0x20); c <= 0x7E; c++ {
		value := string(rune(c))
		got := write(t, body{Value: value}).Body.String()

		want := fmt.Sprintf(`{"Value":"%s"}`, value)
		switch {
		case strings.IndexByte(measured, c) >= 0:
			want = fmt.Sprintf(`{"Value":"\u%04X"}`, c)
		case c == '\\':
			// Not a choice this package makes: JSON has no other spelling for
			// a backslash inside a string.
			want = `{"Value":"\\"}`
		}

		if got != want {
			t.Errorf("Write(%q) body = %s, want %s", value, got, want)
		}
	}
}

// TestWriteSetsTheContentTypeAndTheStatusTogether is behaviours 1.10, and the
// reason plan 5 puts the status and the value in one call: the content type
// belongs to the thing that produced the body.
func TestWriteSetsTheContentTypeAndTheStatusTogether(t *testing.T) {
	recorder := httptest.NewRecorder()

	if err := wire.Write(recorder, http.StatusCreated, body{Value: "x"}, wire.ProfilePlain); err != nil {
		t.Fatalf("Write(...) = %v, want no error", err)
	}

	if got, want := recorder.Code, http.StatusCreated; got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
	if got, want := recorder.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

// TestWriteSendsNoTrailingNewline guards the one artefact of encoding through
// an Encoder rather than Marshal, which is how the HTML escaping gets switched
// off. A trailing newline is invisible to a parser and is a byte the reference
// does not send.
func TestWriteSendsNoTrailingNewline(t *testing.T) {
	got := write(t, body{Value: "x"}).Body.String()

	if strings.HasSuffix(got, "\n") {
		t.Errorf("Write(...) body = %q, want no trailing newline", got)
	}
}

// TestWriteSendsNothingWhenTheValueCannotBeSerialised covers the failure T4
// handed over: units.Time refuses a year outside 1 to 9999, and wire.Write must
// propagate that rather than swallow it.
//
// The stronger half of the assertion is that nothing reached the client. A
// writer that had already sent a status and a content type could only follow
// them with more body, so the caller would have lost the chance to send the
// refusal that plan 7 says it should.
func TestWriteSendsNothingWhenTheValueCannotBeSerialised(t *testing.T) {
	type dated struct {
		When units.Time
	}
	unwritable := units.At(time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC))

	recorder := httptest.NewRecorder()
	err := wire.Write(recorder, http.StatusOK, dated{When: unwritable}, wire.ProfilePlain)

	if err == nil {
		t.Fatalf("Write(...) = nil, want the error units.Time raises for year 10000")
	}
	if got := recorder.Body.Len(); got != 0 {
		t.Errorf("body = %q, want nothing written", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want it unset", got)
	}
}
