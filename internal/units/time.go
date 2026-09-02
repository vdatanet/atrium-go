package units

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"time"
)

// wireLayout is the one spelling of a date this server writes:
// .NET's round-trip ISO-8601, seven fractional digits and a Z
// (behaviours 1.2, e.g. "2025-06-19T00:00:00.0000000Z").
//
// The trailing Z is a literal, not a zone element — Go reads "Z07:00" and
// "Z0700" as the zone and everything else as text — which is correct here
// because a Time is held in UTC and can be written no other way.
const wireLayout = "2006-01-02T15:04:05.0000000Z"

// parseLayouts is what an incoming date may look like, tried in order.
//
// behaviours 1.2 says input is "anything ISO-8601, with or without a timezone;
// a missing timezone is read as UTC", and Go has no single layout for that:
// a layout with a zone element will not match a value that has none, and a
// layout without one will not match a value that has. The fractional element is
// ".999999999", which matches any number of fractional digits including none,
// so each row below covers a whole family of values rather than one shape.
var parseLayouts = []string{
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999Z0700",
	"2006-01-02T15:04:05.999999999Z07",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02T15:04Z07:00",
	"2006-01-02T15:04Z0700",
	"2006-01-02T15:04Z07",
	"2006-01-02T15:04",
	"2006-01-02",
}

// ticksEpochOffsetSeconds is the distance from 0001-01-01T00:00:00Z, which is
// where .NET counts ticks from, to the Unix epoch.
const ticksEpochOffsetSeconds = 62135596800

// Time is an instant as the wire carries one: seven fractional digits and a Z
// (behaviours 1.2).
//
// # It is held in UTC and rounded to a whole tick
//
// Both are normalisations done once, at construction, rather than at every
// write. A Time carrying an offset would have to be converted before every
// format, and the day one is not is the day a body carries "+02:00" where every
// measured reference value carries "Z". A Time carrying nanoseconds the wire
// cannot express would compare unequal to the value it serialises as, which is
// the same bug one layer further in.
//
// Rounding rather than truncating is behaviours 1.3's rule for ticks, and it
// applies here for the same reason: Go's own formatting truncates, so a value
// half a tick short of a second would be written as the second before it.
//
// # The zero value is not a missing date
//
// The zero Time writes as "0001-01-01T00:00:00.0000000Z", which is exactly what
// the reference sends for an unset date — .NET's DateTime.MinValue, emitted in
// full rather than omitted, on DateCreated, DateLastMediaAdded and
// LastPlaybackCheckIn `[probe: tools/probe_wire_format, Jellyfin 10.11.11,
// 2026-09-02]`. So the zero value is a legal wire value and not a signal, and a
// field that is genuinely absent is a *Time that is nil (ADR-0002: optional
// fields are pointers, and "omitempty" on a non-pointer is banned).
type Time struct {
	// instant is unexported so that every Time in the process has been through
	// At: there is no way to hold one that is not UTC and not a whole tick.
	instant time.Time
}

// At returns the instant t as a Time, in UTC and rounded to the nearest tick.
func At(t time.Time) Time {
	return Time{instant: t.UTC().Round(TickDuration)}
}

// TimeFromTicks returns the instant that many ticks after 0001-01-01T00:00:00Z,
// which is .NET's DateTime.Ticks and the integer the store keeps a date as
// (plan 4).
func TimeFromTicks(ticks Ticks) Time {
	seconds := int64(ticks) / int64(TicksPerSecond)
	remainder := int64(ticks) % int64(TicksPerSecond)
	if remainder < 0 {
		// Go truncates towards zero, so an instant before the epoch lands a
		// second late with a negative remainder. Borrowing a second puts the
		// remainder back in range.
		seconds--
		remainder += int64(TicksPerSecond)
	}
	return At(time.Unix(seconds-ticksEpochOffsetSeconds, remainder*100).UTC())
}

// ParseTime reads a date.
//
// It accepts anything ISO-8601 in extended form — with or without a timezone,
// with or without seconds, with any number of fractional digits, and a date on
// its own — and reads a missing timezone as UTC (behaviours 1.2). A separating
// "t" or space is accepted for "T", and a trailing lower-case "z" for "Z".
//
// A comma as the decimal separator is accepted, which ISO-8601 permits and Go's
// fractional layout element already reads — measured rather than assumed, in
// the test that first expected it to be refused.
//
// It does not accept the basic form ("20250619T123000Z"). That form has not
// been observed from the reference, and a layout list that grows past what the
// reference sends grows cases nothing exercises.
func ParseTime(s string) (Time, error) {
	normalised := normaliseForParse(s)
	for _, layout := range parseLayouts {
		if parsed, err := time.Parse(layout, normalised); err == nil {
			return At(parsed), nil
		}
	}
	return Time{}, fmt.Errorf("units: %q is not an ISO-8601 date", s)
}

// normaliseForParse folds the two spellings ISO-8601 permits that Go's layouts
// do not, so that the layout list stays a list of shapes rather than a cross
// product of shapes and separators.
func normaliseForParse(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 10 && (s[10] == 't' || s[10] == ' ') {
		s = s[:10] + "T" + s[11:]
	}
	if strings.HasSuffix(s, "z") {
		s = s[:len(s)-1] + "Z"
	}
	return s
}

// Instant returns the underlying instant, in UTC and rounded to a whole tick.
func (t Time) Instant() time.Time { return t.instant }

// IsZero reports whether this is the zero Time — which is a real wire value,
// the unset date the reference sends, and not an absent one. See the type's
// documentation.
func (t Time) IsZero() bool { return t.instant.IsZero() }

// Equal reports whether two instants are the same. Two Times that serialise to
// the same bytes are equal, because both have been rounded to a whole tick.
func (t Time) Equal(other Time) bool { return t.instant.Equal(other.instant) }

// Ticks returns the instant as .NET's DateTime.Ticks: 100-nanosecond intervals
// since 0001-01-01T00:00:00Z.
//
// It shares the Ticks type with a duration, as .NET shares Int64 between
// DateTime.Ticks and TimeSpan.Ticks. The unit is the same; the origin is what
// this method adds, and it is named here rather than left to a caller because
// the store column is this number (plan 4).
func (t Time) Ticks() Ticks {
	// Not a subtraction of two times: a time.Duration is nanoseconds in an
	// int64 and overflows after about 292 years, so the distance from year 1
	// cannot be expressed as one. Seconds and the nanosecond remainder can.
	return Ticks((t.instant.Unix()+ticksEpochOffsetSeconds)*int64(TicksPerSecond) +
		int64(t.instant.Nanosecond())/100)
}

// String returns the wire form, so that a log line and a response body spell a
// date the same way.
func (t Time) String() string { return t.instant.Format(wireLayout) }

// MarshalJSON writes the date as behaviours 1.2 measured it: seven fractional
// digits and a Z, quoted.
//
// Seven is what 346 of 352 date values on eleven routes carry `[probe:
// tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`. The six that do not
// are LastPlayedDate and LastActivityDate, whose mechanism behaviours 1.2
// records as unidentified; writing seven always is still right, because it is
// what a client parses either way and because inventing the short form would
// mean reproducing a rule nobody has found. What it costs is a differential
// difference on those two fields, and behaviours 1.2 says so rather than this
// package inventing an exception.
//
// A zero fraction is written in full — ".0000000Z", not an omitted fraction.
// The reference writes it that way, which is the evidence that killed the
// trailing-zero-trimming explanation of the short values.
func (t Time) MarshalJSON() ([]byte, error) {
	// The layout has room for four year digits and no more, so a year outside
	// .NET's own DateTime range would silently produce a value of a different
	// length. Refusing is the only answer that keeps the shape a shape.
	if year := t.instant.Year(); year < 1 || year > 9999 {
		return nil, fmt.Errorf("units: %s is outside the range a date can be written in (years 1 to 9999)",
			t.instant.Format(time.RFC3339Nano))
	}
	b := make([]byte, 0, len(wireLayout)+2)
	b = append(b, '"')
	b = t.instant.AppendFormat(b, wireLayout)
	return append(b, '"'), nil
}

// UnmarshalJSON reads a date on the terms ParseTime states.
//
// A JSON null reads as the zero Time. behaviours 1.7 has the reference omitting
// a null property rather than sending one, so a null here is something else's
// output; reading it as the unset date is the reading that loses nothing, and a
// field that must tell absent from unset is a *Time.
func (t *Time) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		*t = Time{}
		return nil
	}
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return errors.New("units: a date must be a JSON string")
	}
	parsed, err := ParseTime(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*t = parsed
	return nil
}

// IsTime reports whether s is a date in the form this package writes.
//
// This is the unit sweep's question, and it is asked of a *value* rather than
// of a field name on purpose: conformance L1's own correction is that
// DateCreated, DateLastMediaAdded and LastPlaybackCheckIn are dates whose names
// do not end in Date, so a sweep keyed on the suffix checks six of the nine
// date-valued fields observed `[probe: tools/probe_wire_format, Jellyfin
// 10.11.11, 2026-09-02]`. The name is a heuristic for finding a candidate; this
// is the rule.
//
// It is deliberately exact rather than lenient. A value with three fractional
// digits is a date, and it is not a date this server may send, so the sweep has
// to fail on it — which means this may not accept it.
//
// It is one parse against the one layout and nothing else. A re-format check
// was written beside it first, on the assumption that time.Parse rolls an
// out-of-range day forward the way time.Date does; removing the check made no
// test fail, and Go in fact refuses "2025-02-30T00:00:00.0000000Z" with "day
// out of range" `[measurement: Go 1.27.0, 2026-09-03]`. A guard no case can
// reach is a guard that has proved nothing, so it is gone and the case it was
// written for is in the table beside this function instead.
func IsTime(s string) bool {
	_, err := time.Parse(wireLayout, s)
	return err == nil
}
