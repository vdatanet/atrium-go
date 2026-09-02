package units

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// TestTicksSerialiseAsAJSONInteger asserts the bytes, not the value.
//
// behaviours 1.3 makes every duration on the wire a tick count, and
// conformance L1's unit sweep asks that such a field be an *integer*. A tick
// that serialised as a string, or as a float with a trailing ".0", would parse
// back to the same number in every test that looked at a parsed body — which is
// exactly the class of failure Principle VIII says to assert bytes for.
func TestTicksSerialiseAsAJSONInteger(t *testing.T) {
	cases := []struct {
		name  string
		ticks Ticks
		want  string
	}{
		{name: "zero", ticks: 0, want: "0"},
		{name: "one second", ticks: TicksPerSecond, want: "10000000"},
		{name: "a negative position", ticks: -1, want: "-1"},
		{
			// Longer than a float64 holds every tick of, which is the point:
			// the type is an integer all the way to the wire.
			name:  "a value beyond a float64's exact range",
			ticks: 9007199254740993,
			want:  "9007199254740993",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded, err := json.Marshal(c.ticks)
			if err != nil {
				t.Fatalf("json.Marshal(%d) = %v", c.ticks, err)
			}
			if string(encoded) != c.want {
				t.Fatalf("json.Marshal(%d) = %s, want %s", c.ticks, encoded, c.want)
			}

			var back Ticks
			if err := json.Unmarshal(encoded, &back); err != nil {
				t.Fatalf("json.Unmarshal(%s) = %v", encoded, err)
			}
			if back != c.ticks {
				t.Fatalf("round trip of %d returned %d", c.ticks, back)
			}
		})
	}
}

func TestTicksFromDuration(t *testing.T) {
	cases := []struct {
		name string
		in   time.Duration
		want Ticks
	}{
		{name: "zero", in: 0, want: 0},
		{name: "one second is ten million ticks", in: time.Second, want: TicksPerSecond},
		{name: "one tick", in: 100 * time.Nanosecond, want: 1},
		{
			// The case the task singles out. Truncating gives 1; behaviours 1.3
			// says the conversion rounds, so this is 2.
			name: "sub-tick input rounds rather than truncating",
			in:   150 * time.Nanosecond,
			want: 2,
		},
		{name: "just under half a tick rounds down", in: 149 * time.Nanosecond, want: 1},
		{name: "just over half a tick rounds up", in: 151 * time.Nanosecond, want: 2},
		{name: "a negative sub-tick value rounds away from zero too", in: -150 * time.Nanosecond, want: -2},
		{name: "a negative sub-tick value below half rounds towards zero", in: -149 * time.Nanosecond, want: -1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TicksFromDuration(c.in); got != c.want {
				t.Errorf("TicksFromDuration(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestTicksFromSeconds(t *testing.T) {
	cases := []struct {
		name string
		in   float64
		want Ticks
	}{
		{name: "zero", in: 0, want: 0},
		{name: "a whole second", in: 1, want: TicksPerSecond},
		{
			// The ffprobe shape behaviours 1.3 names: a duration in seconds,
			// converted once, at ingestion.
			name: "an ffprobe duration",
			in:   1234.5678901,
			want: 12345678901,
		},
		{
			name: "sub-tick input rounds rather than truncating",
			in:   0.00000015,
			want: 2,
		},
		{name: "just under half a tick rounds down", in: 0.00000014, want: 1},
		{name: "a negative value", in: -1, want: -TicksPerSecond},
		{name: "a duration ffprobe could not determine", in: math.NaN(), want: 0},
		{name: "an infinite duration", in: math.Inf(1), want: 0},
		{name: "a span no tick count can hold", in: 1e30, want: 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := TicksFromSeconds(c.in); got != c.want {
				t.Errorf("TicksFromSeconds(%v) = %d, want %d", c.in, got, c.want)
			}
		})
	}
}

func TestTicksBackToDurationAndSeconds(t *testing.T) {
	if got, want := Ticks(TicksPerSecond).Duration(), time.Second; got != want {
		t.Errorf("Ticks(%d).Duration() = %v, want %v", TicksPerSecond, got, want)
	}
	if got, want := Ticks(1).Duration(), 100*time.Nanosecond; got != want {
		t.Errorf("Ticks(1).Duration() = %v, want %v", got, want)
	}
	if got, want := Ticks(12345678901).Seconds(), 1234.5678901; got != want {
		t.Errorf("Ticks(12345678901).Seconds() = %v, want %v", got, want)
	}
}
