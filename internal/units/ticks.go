package units

import (
	"math"
	"time"
)

// Ticks is a count of 100-nanosecond intervals.
//
// It is .NET's TimeSpan tick, which is the unit every duration and position on
// the wire is expressed in: RunTimeTicks, PositionTicks, PlaybackPositionTicks
// and StartPositionTicks are all ticks, at 10,000,000 per second
// (behaviours 1.3, `[source:
// MediaBrowser.MediaEncoding/Probing/ProbeResultNormalizer.cs:234 @
// v10.11.11]`).
//
// It is an int64 and not a struct so that it serialises as a JSON integer
// without a MarshalJSON of its own. That is the point of the type: the sweep
// asks whether a duration-valued field is an integer, and a type that could
// answer "string" would have to be caught rather than prevented.
//
// A negative tick count is representable and is not rejected here. Whether a
// negative position is a legal value is a question about a route, measured
// where that route is measured, and a unit type that refused one would be
// deciding it for every caller.
type Ticks int64

// TicksPerSecond is how many ticks make one second (behaviours 1.3).
const TicksPerSecond Ticks = 10_000_000

// TickDuration is one tick as a Go duration. Go counts nanoseconds and the
// wire counts hundreds of them, so this is the resolution at which the two
// agree, and the resolution Time is held at.
const TickDuration = 100 * time.Nanosecond

// TicksFromDuration converts a Go duration to ticks, rounding to the nearest
// tick rather than truncating.
//
// Rounding is behaviours 1.3's rule and not a preference: truncating loses a
// tick per conversion, and a resume position that loses a tick every time it is
// written back walks backwards through a library's worth of playback.
func TicksFromDuration(d time.Duration) Ticks {
	nanoseconds := int64(d)
	if nanoseconds >= 0 {
		return Ticks((nanoseconds + 50) / 100)
	}
	// Go's integer division truncates towards zero, so a negative value has to
	// be nudged the other way to round to nearest rather than towards zero.
	return Ticks((nanoseconds - 50) / 100)
}

// TicksFromSeconds converts a duration reported in seconds to ticks, rounding
// to the nearest tick rather than truncating.
//
// This is the conversion behaviours 1.3 names: "where a source (ffprobe)
// reports seconds as a float, the conversion happens once, at ingestion, and
// rounds rather than truncates". Doing it once, here, is what keeps it from
// happening twice with two different roundings.
//
// A value that is not a finite number of ticks — NaN, an infinity, or a span
// longer than an int64 of ticks can hold — converts to zero. This package will
// not invent a duration it was not given, and it is not the place that decides
// what an unknown duration looks like on the wire: whether that is a zero or an
// absent field belongs to the route that sends it.
func TicksFromSeconds(seconds float64) Ticks {
	ticks := math.Round(seconds * float64(TicksPerSecond))
	if math.IsNaN(ticks) || ticks >= math.MaxInt64 || ticks < math.MinInt64 {
		return 0
	}
	return Ticks(ticks)
}

// Duration returns the tick count as a Go duration. It is exact: a tick is a
// whole number of nanoseconds.
func (t Ticks) Duration() time.Duration {
	return time.Duration(t) * TickDuration
}

// Seconds returns the tick count in seconds. It is lossy above roughly 28
// years, where a float64 stops holding every tick, and it exists for the places
// that have to hand a duration to something counting seconds — never for
// storing one.
func (t Ticks) Seconds() float64 {
	return float64(t) / float64(TicksPerSecond)
}
