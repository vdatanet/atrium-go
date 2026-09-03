package httpapi

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vdatanet/atrium-go/internal/units"
)

// ResponseTimeHeader is the field name of behaviours 1.9. The reference spells
// it in exactly this casing
// [source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:17 @ v10.11.11],
// and HTTP field names are case-insensitive on the way in but not on the way
// out: L3 compares the bytes this server sends.
const ResponseTimeHeader = "X-Response-Time-ms"

// ticksPerMillisecond is the resolution the header value is quantised to.
//
// It is the reference's own resolution rather than a choice made here: the
// reference formats a .NET TimeSpan's TotalMilliseconds, and a TimeSpan counts
// the same 100-nanosecond tick internal/units does (behaviours 1.3). So a
// value it sends has at most four decimal places, which is what the measured
// `X-Response-Time-ms: 2.1329` of behaviours 1.9 shows.
const ticksPerMillisecond = units.TicksPerSecond / 1000

// ResponseTimeStamp is the stage that puts behaviours 1.9's header on every
// response.
//
// # What the reference sends
//
// Every response carries the time it took in fractional milliseconds
// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]. Its
// middleware is registered unconditionally and the two configuration flags
// beside it gate a slow-response log line rather than the header
// [source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:17,
// Jellyfin.Server/Startup.cs:163 @ v10.11.11], so there is no configuration
// under which a response arrives without it. behaviours 1.9 states why this
// project replicates a header no client reads: omitting it would be a
// difference on every response in the project, 55 rows of noise in the first
// differential run.
//
// # When the value is computed
//
// The reference measures from the moment its middleware is entered to the
// moment the response *starts* — it registers the stamp on the response's
// OnStarting callback rather than after the next delegate returns
// [source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:47-63 @
// v10.11.11]. That is the only point at which the number can be both truthful
// and sendable: a header set after the status line has gone out is dropped.
//
// Go has no OnStarting, so the stage wraps the ResponseWriter and stamps on
// the first of WriteHeader, Write, or the handler returning without either.
// The third case is a handler that leaves an empty 200 to net/http, and it is
// stamped after the fact because the header map has not been written yet.
//
// # Why the elapsed time does not come from the clock port
//
// architecture 2 makes the wall clock a port, and this stage does not use it.
// A response time is an elapsed duration, not a moment: it is read from Go's
// monotonic clock, which cannot jump or run backwards while a request is in
// flight, where a wall clock corrected by NTP mid-request can produce a
// negative one. The seam a test needs is here all the same, as the measure
// field, because a duration that changes every run cannot be asserted on.
type ResponseTimeStamp struct {
	// measure begins one measurement and returns the function that ends it.
	// It is a field so that a test can hand the stage a fixed duration; the
	// server always gets monotonicMeasure.
	measure func() func() time.Duration
}

// NewResponseTimeStamp builds the stage.
//
// It cannot fail, and so — unlike NewPathFolder, NewQueryFolder and
// NewRefusals — it returns no error: it reads neither the route table nor the
// configuration, and a constructor with an error that no input can produce
// would be a branch no test could reach.
func NewResponseTimeStamp() *ResponseTimeStamp {
	return &ResponseTimeStamp{measure: monotonicMeasure}
}

// monotonicMeasure is the real clock: time.Since reads the monotonic reading
// time.Now stored, so the result is unaffected by a wall-clock correction
// arriving mid-request.
func monotonicMeasure() func() time.Duration {
	start := time.Now()
	return func() time.Duration { return time.Since(start) }
}

// Wrap is the middleware.
//
// plan 6.7 puts this stage outside everything it claims to be timing, which
// is also what architecture 4 requires of it.
func (s *ResponseTimeStamp) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		elapsed := s.measure()
		stamped := &stampingWriter{
			ResponseWriter: w,
			stamp: func(header http.Header) {
				// Set, not Add: a stage that has already put a value here is
				// replaced rather than appended to, because two field lines
				// are one field value with a comma in it, and the reference
				// sends one number.
				header.Set(ResponseTimeHeader, formatMilliseconds(elapsed()))
			},
		}
		next.ServeHTTP(stamped, r)
		// A handler that returned without writing anything: net/http will
		// send an empty 200 built from the header map, which is still open.
		stamped.stampOnce()
	})
}

// formatMilliseconds renders an elapsed duration the way the reference renders
// one: fractional milliseconds, a full stop for the decimal separator, and no
// trailing zeros.
//
// The reference formats a double of TotalMilliseconds with the invariant
// culture [source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:61 @
// v10.11.11]. Two consequences, and only the first is measured:
//
//   - at most four decimal places, because the double is a whole number of
//     100-nanosecond ticks divided by ten thousand. behaviours 1.9's measured
//     `2.1329` is one.
//   - ⚠️ UNVERIFIED: a duration landing on a whole millisecond is sent with no
//     decimal part at all, and one landing on a tenth is sent with one digit,
//     because .NET's default formatting of a double is the shortest string
//     that round-trips. No probe in this repository caught a response fast
//     enough or round enough to show it. This implementation follows the
//     inference because a shortest form is what the reference's own formatter
//     produces for every other value, and because the cost of being wrong is
//     nil: the header is excused by name in allowlist.yaml — it moves on every
//     response by construction — so no differential run compares it.
//
// The arithmetic is integer arithmetic on ticks rather than a float format,
// which is what architecture 4 asks for: the unit types own the conversion and
// no ad-hoc format string is involved. A negative duration is sent as zero.
// The monotonic clock cannot produce one, but this function is pure and its
// argument comes from a field a test can replace, and a header claiming a
// response took less than no time is worse than one claiming it was instant.
func formatMilliseconds(elapsed time.Duration) string {
	ticks := units.TicksFromDuration(elapsed)
	if ticks < 0 {
		ticks = 0
	}

	whole := int64(ticks / ticksPerMillisecond)
	fraction := int64(ticks % ticksPerMillisecond)
	if fraction == 0 {
		return strconv.FormatInt(whole, 10)
	}

	digits := strconv.FormatInt(int64(ticksPerMillisecond)+fraction, 10)[1:]
	return strconv.FormatInt(whole, 10) + "." + strings.TrimRight(digits, "0")
}

// stampingWriter is the ResponseWriter the stage passes down: it runs stamp
// once, at the moment the response starts, and is otherwise the writer it
// wraps.
//
// # What a wrapper here must not break
//
// Unwrap is the contract net/http declares for exactly this
// [measurement: net/http documentation, Go 1.27.0, 2026-09-03]:
// http.ResponseController reaches Flush, Hijack and the deadlines through it,
// so a handler that needs one of those still gets it. ReadFrom is delegated
// for the same reason in the other direction: http.ServeContent copies through
// io.ReaderFrom when the writer has one, which is how net/http reaches
// sendfile for an *os.File, and a wrapper without it would quietly turn every
// byte of 008's streaming routes into a userspace copy.
type stampingWriter struct {
	http.ResponseWriter

	stamp   func(http.Header)
	stamped bool
}

// stampOnce writes the header if the response has not started yet.
//
// The guard is not an optimisation. Setting a header after the status line has
// been written is a no-op that reads like a bug when somebody later moves the
// call; making the second call do nothing, deliberately, is what lets Wrap
// stamp again after the handler returns without asking whether it needs to.
func (w *stampingWriter) stampOnce() {
	if w.stamped {
		return
	}
	w.stamped = true
	w.stamp(w.Header())
}

func (w *stampingWriter) WriteHeader(status int) {
	w.stampOnce()
	w.ResponseWriter.WriteHeader(status)
}

func (w *stampingWriter) Write(b []byte) (int, error) {
	w.stampOnce()
	return w.ResponseWriter.Write(b)
}

// ReadFrom keeps the wrapped writer's own copy path reachable.
func (w *stampingWriter) ReadFrom(r io.Reader) (int64, error) {
	w.stampOnce()
	if from, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		return from.ReadFrom(r)
	}
	return io.Copy(w.ResponseWriter, r)
}

// Unwrap is what http.ResponseController follows to reach the real writer.
func (w *stampingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
