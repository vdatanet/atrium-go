package units

import (
	"encoding/json"
	"testing"
	"time"
)

// TestTimeRoundTrip is the table T4 owes: every row is parsed, written, and
// parsed again, and the bytes are compared rather than the parsed value.
//
// Comparing bytes is Principle VIII at the smallest scale there is. The number
// of fractional digits, the Z and the four-digit year are the whole contract
// here, and every one of them is invisible the moment a value is parsed back
// into a time.
func TestTimeRoundTrip(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "the reference's own example, seven digits and a Z",
			input: "2025-06-19T00:00:00.0000000Z",
			want:  `"2025-06-19T00:00:00.0000000Z"`,
		},
		{
			// behaviours 1.2: ".0000000 is written in full on seven other
			// fields", which is what killed the trailing-zero explanation of
			// the short values. Go's own RFC 3339 formatting omits a zero
			// fraction entirely, so this is the row that fails first if the
			// layout is ever "simplified".
			name:  "a zero fraction is written in full, not omitted",
			input: "2025-06-19T00:00:00Z",
			want:  `"2025-06-19T00:00:00.0000000Z"`,
		},
		{
			// behaviours 1.2: this is how an unset date arrives, on
			// DateCreated, DateLastMediaAdded and LastPlaybackCheckIn. It is
			// .NET's DateTime.MinValue, emitted rather than omitted.
			name:  "the unset date, .NET's DateTime.MinValue",
			input: "0001-01-01T00:00:00.0000000Z",
			want:  `"0001-01-01T00:00:00.0000000Z"`,
		},
		{
			// behaviours 1.2's measured short value on LastPlayedDate. The
			// reference sent three digits; this server sends seven, and the
			// difference is declared there rather than reproduced here.
			name:  "three fractional digits in, seven out",
			input: "2026-08-13T14:01:13.061Z",
			want:  `"2026-08-13T14:01:13.0610000Z"`,
		},
		{
			// behaviours 1.2's measured short value on LastActivityDate.
			name:  "six fractional digits in, seven out",
			input: "2026-09-02T17:58:58.632188Z",
			want:  `"2026-09-02T17:58:58.6321880Z"`,
		},
		{
			name:  "a missing timezone reads as UTC",
			input: "2025-06-19T12:30:00.1234567",
			want:  `"2025-06-19T12:30:00.1234567Z"`,
		},
		{
			name:  "an offset is converted to UTC, because the wire carries Z",
			input: "2025-06-19T12:30:00.1234567+02:00",
			want:  `"2025-06-19T10:30:00.1234567Z"`,
		},
		{
			name:  "an offset without a colon",
			input: "2025-06-19T12:30:00-0500",
			want:  `"2025-06-19T17:30:00.0000000Z"`,
		},
		{
			name:  "a date on its own is midnight UTC",
			input: "2025-06-19",
			want:  `"2025-06-19T00:00:00.0000000Z"`,
		},
		{
			name:  "seconds may be absent",
			input: "2025-06-19T12:30Z",
			want:  `"2025-06-19T12:30:00.0000000Z"`,
		},
		{
			// ISO-8601 permits a comma as the decimal separator, and Go's
			// fractional layout element reads one. This row started life in the
			// rejection table and was moved here by what it measured.
			name:  "a comma as the decimal separator",
			input: "2025-06-19T12:30:00,5Z",
			want:  `"2025-06-19T12:30:00.5000000Z"`,
		},
		{
			name:  "a lower-case separator and a lower-case zone marker",
			input: "2025-06-19t12:30:00z",
			want:  `"2025-06-19T12:30:00.0000000Z"`,
		},
		{
			// Half a tick. Go's formatting truncates, which would write
			// .0000000 and lose the value; behaviours 1.3 says the conversion
			// rounds. The rounding happens once, at construction, so that the
			// value held and the value written are the same instant.
			name:  "sub-tick input rounds up rather than truncating",
			input: "2025-06-19T00:00:00.00000005Z",
			want:  `"2025-06-19T00:00:00.0000001Z"`,
		},
		{
			name:  "sub-tick input below half a tick rounds down",
			input: "2025-06-19T00:00:00.00000004Z",
			want:  `"2025-06-19T00:00:00.0000000Z"`,
		},
		{
			name:  "rounding a sub-tick fraction may carry into the second",
			input: "2025-06-19T00:00:00.99999999Z",
			want:  `"2025-06-19T00:00:01.0000000Z"`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed, err := ParseTime(c.input)
			if err != nil {
				t.Fatalf("ParseTime(%q) = %v", c.input, err)
			}

			encoded, err := json.Marshal(parsed)
			if err != nil {
				t.Fatalf("json.Marshal(%v) = %v", parsed, err)
			}
			if string(encoded) != c.want {
				t.Fatalf("json.Marshal(ParseTime(%q)) = %s, want %s", c.input, encoded, c.want)
			}

			// The second leg: what this server writes is something it can read
			// back unchanged. A format that only one of the two halves agrees
			// with would pass the first assertion alone.
			var back Time
			if err := json.Unmarshal(encoded, &back); err != nil {
				t.Fatalf("json.Unmarshal(%s) = %v", encoded, err)
			}
			if !back.Equal(parsed) {
				t.Fatalf("round trip of %q returned %v, want %v", c.input, back, parsed)
			}
			reencoded, err := json.Marshal(back)
			if err != nil {
				t.Fatalf("json.Marshal(%v) = %v", back, err)
			}
			if string(reencoded) != c.want {
				t.Fatalf("re-encoding %q gave %s, want %s", c.input, reencoded, c.want)
			}
		})
	}
}

func TestTheZeroTimeIsTheUnsetDate(t *testing.T) {
	// behaviours 1.2: an unset date arrives as .NET's DateTime.MinValue, in
	// full, rather than as an absent property. Go's zero time is the same
	// instant, so the zero value of this type is already the right answer and
	// nothing has to remember to construct it.
	encoded, err := json.Marshal(Time{})
	if err != nil {
		t.Fatalf("json.Marshal(Time{}) = %v", err)
	}
	if want := `"0001-01-01T00:00:00.0000000Z"`; string(encoded) != want {
		t.Errorf("json.Marshal(Time{}) = %s, want %s", encoded, want)
	}
	if !(Time{}).IsZero() {
		t.Error("the zero Time does not report itself as zero")
	}
}

func TestParseTimeRejects(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{name: "empty", input: ""},
		{name: "not a date at all", input: "nonsense"},
		{name: "a tick count", input: "638000000000000000"},
		{name: "a day February does not have", input: "2025-02-30T00:00:00Z"},
		{name: "the basic form, which the reference has not been seen to send", input: "20250619T123000Z"},
		{name: "trailing rubbish after a valid date", input: "2025-06-19T12:30:00Z and then some"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got, err := ParseTime(c.input); err == nil {
				t.Errorf("ParseTime(%q) = %v, want an error", c.input, got)
			}
		})
	}
}

func TestUnmarshalJSON(t *testing.T) {
	t.Run("null reads as the unset date", func(t *testing.T) {
		parsed := At(time.Date(2025, 6, 19, 0, 0, 0, 0, time.UTC))
		if err := parsed.UnmarshalJSON([]byte("null")); err != nil {
			t.Fatalf("UnmarshalJSON(null) = %v", err)
		}
		if !parsed.IsZero() {
			t.Errorf("UnmarshalJSON(null) left %v, want the zero Time", parsed)
		}
	})

	for _, input := range []string{"12345", "true", `"nonsense"`, `""`} {
		t.Run("a date that is not a date string: "+input, func(t *testing.T) {
			var parsed Time
			if err := json.Unmarshal([]byte(input), &parsed); err == nil {
				t.Errorf("json.Unmarshal(%s) = %v, want an error", input, parsed)
			}
		})
	}
}

func TestMarshalJSONRefusesAYearItCannotWrite(t *testing.T) {
	// The layout has room for four year digits. Go would write five without
	// complaining, producing a value of a length no client parses, so the type
	// refuses instead. Constructing one takes reaching past At, which is why
	// this test is in the package rather than beside it.
	outOfRange := Time{instant: time.Date(12345, 1, 1, 0, 0, 0, 0, time.UTC)}
	if got, err := json.Marshal(outOfRange); err == nil {
		t.Errorf("json.Marshal(year 12345) = %s, want an error", got)
	}
}

func TestTicksAsAnInstant(t *testing.T) {
	cases := []struct {
		name  string
		date  string
		ticks Ticks
	}{
		{
			// .NET's DateTime.MinValue is tick zero by definition.
			name:  "the unset date is tick zero",
			date:  "0001-01-01T00:00:00.0000000Z",
			ticks: 0,
		},
		{
			// The Unix epoch is where .NET's origin and Go's meet, and the
			// constant that bridges them is this number of seconds.
			name:  "the Unix epoch",
			date:  "1970-01-01T00:00:00.0000000Z",
			ticks: Ticks(ticksEpochOffsetSeconds) * TicksPerSecond,
		},
		{
			// 1750291200 is 2025-06-19T00:00:00Z as a Unix timestamp, spelled
			// out so that the arithmetic under test is not also the arithmetic
			// the expectation is built from.
			name:  "a fraction survives the conversion",
			date:  "2025-06-19T00:00:00.1234567Z",
			ticks: Ticks(ticksEpochOffsetSeconds+1750291200)*TicksPerSecond + 1234567,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parsed, err := ParseTime(c.date)
			if err != nil {
				t.Fatalf("ParseTime(%q) = %v", c.date, err)
			}
			if got := parsed.Ticks(); got != c.ticks {
				t.Errorf("ParseTime(%q).Ticks() = %d, want %d", c.date, got, c.ticks)
			}
			if got := TimeFromTicks(c.ticks); !got.Equal(parsed) {
				t.Errorf("TimeFromTicks(%d) = %v, want %v", c.ticks, got, parsed)
			}
		})
	}
}

func TestTicksAsAnInstantRoundTripsEveryTick(t *testing.T) {
	// The conversion divides and multiplies by ten million and borrows a second
	// across the epoch, which is three places an off-by-one hides. Walking a
	// span of ticks either side of a second boundary before the Unix epoch is
	// what exercises the borrow.
	base := At(time.Date(1969, 12, 31, 23, 59, 59, 0, time.UTC)).Ticks()
	for offset := Ticks(-3); offset <= 3; offset++ {
		ticks := base + offset
		if got := TimeFromTicks(ticks).Ticks(); got != ticks {
			t.Errorf("TimeFromTicks(%d).Ticks() = %d", ticks, got)
		}
	}
}

func TestIsTime(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "the shape this server writes",
			value: "2025-06-19T00:00:00.0000000Z",
			want:  true,
		},
		{
			name:  "the unset date",
			value: "0001-01-01T00:00:00.0000000Z",
			want:  true,
		},
		{
			// The sweep exists to catch this. Three digits is a date, and it is
			// not a date this server may send.
			name:  "three fractional digits is a date and is not this one",
			value: "2026-08-13T14:01:13.061Z",
			want:  false,
		},
		{
			name:  "six fractional digits",
			value: "2026-09-02T17:58:58.632188Z",
			want:  false,
		},
		{
			name:  "no fraction at all",
			value: "2025-06-19T00:00:00Z",
			want:  false,
		},
		{
			name:  "an offset instead of Z",
			value: "2025-06-19T00:00:00.0000000+02:00",
			want:  false,
		},
		{
			// Go's own parser refuses this with "day out of range", which is
			// why IsTime needs no check of its own for it. Measured, not
			// assumed: the check that was there first was removed when nothing
			// failed.
			name:  "a day February does not have",
			value: "2025-02-30T00:00:00.0000000Z",
			want:  false,
		},
		{
			name:  "an ordinary string",
			value: "Jellyfin Server",
			want:  false,
		},
		{
			name:  "empty",
			value: "",
			want:  false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsTime(c.value); got != c.want {
				t.Errorf("IsTime(%q) = %v, want %v", c.value, got, c.want)
			}
		})
	}
}

func TestIsTimeAcceptsEverythingMarshalWrites(t *testing.T) {
	// The sweep's predicate and the writer must agree, or the sweep fails on
	// this server's own output. Tying them together in a test is cheaper than
	// tying them together in code, because the predicate has to be exact about
	// a shape the writer only has to produce.
	for _, instant := range []time.Time{
		{},
		time.Date(2025, 6, 19, 0, 0, 0, 0, time.UTC),
		time.Date(2025, 6, 19, 23, 59, 59, 999999900, time.UTC),
		time.Date(9999, 12, 31, 23, 59, 59, 999999900, time.UTC),
	} {
		encoded, err := json.Marshal(At(instant))
		if err != nil {
			t.Fatalf("json.Marshal(At(%v)) = %v", instant, err)
		}
		value := string(encoded[1 : len(encoded)-1])
		if !IsTime(value) {
			t.Errorf("IsTime(%q) = false, but that is what MarshalJSON wrote", value)
		}
	}
}
