package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestTheResponseTimeIsFractionalMilliseconds is the value the header carries,
// asserted where it can be deterministic.
//
// behaviours 1.9 measured `X-Response-Time-ms: 2.1329`. The reference computes
// it as a .NET TimeSpan's TotalMilliseconds formatted with the invariant
// culture [source: Jellyfin.Api/Middleware/ResponseTimeMiddleware.cs:49-61 @
// v10.11.11], and a TimeSpan counts the same 100-nanosecond tick this project
// counts (behaviours 1.3) — which is why four decimal places is the most the
// reference can send and why this is integer arithmetic on units.Ticks rather
// than a float format string.
func TestTheResponseTimeIsFractionalMilliseconds(t *testing.T) {
	cases := []struct {
		what    string
		elapsed time.Duration
		want    string
	}{
		{"behaviours 1.9's own measured value", 2*time.Millisecond + 132900*time.Nanosecond, "2.1329"},
		{"one tick, the smallest value that is not zero", 100 * time.Nanosecond, "0.0001"},
		{"a whole millisecond keeps no decimal part", 2 * time.Millisecond, "2"},
		{"a tenth of a millisecond keeps one digit", time.Millisecond + 100*time.Microsecond, "1.1"},
		{"trailing zeros are dropped, as the reference's formatter drops them", 2*time.Millisecond + 133*time.Microsecond, "2.133"},
		{"nothing measurable at all", 0, "0"},
		{"below half a tick rounds down to nothing", 49 * time.Nanosecond, "0"},
		{"above half a tick rounds up to one", 51 * time.Nanosecond, "0.0001"},
		{"a slow response is still plain decimal, never an exponent", time.Hour, "3600000"},
		{"a negative duration the monotonic clock cannot produce", -time.Second, "0"},
	}

	for _, testCase := range cases {
		if got := formatMilliseconds(testCase.elapsed); got != testCase.want {
			t.Errorf("%s: formatMilliseconds(%v) = %q, want %q", testCase.what, testCase.elapsed, got, testCase.want)
		}
	}
}

// TestTheStampCarriesTheDurationThatWasMeasured checks the wiring between the
// measurement and the header, which the shape test above cannot see: a stage
// that formatted the right number and sent a different one would pass it.
func TestTheStampCarriesTheDurationThatWasMeasured(t *testing.T) {
	stage := fixedStamp(2*time.Millisecond + 132900*time.Nanosecond)
	recorder := httptest.NewRecorder()

	stage.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/System/Ping", nil))

	if got := recorder.Header().Values(ResponseTimeHeader); len(got) != 1 || got[0] != "2.1329" {
		t.Errorf("%s = %v, want exactly [\"2.1329\"]", ResponseTimeHeader, got)
	}
}

// TestAHandlerThatWritesNothingIsStillStamped covers the third of the three
// moments a response can start.
//
// A handler that returns without touching the ResponseWriter still produces a
// response: net/http sends an empty 200 built from the header map. Nothing in
// the wrapper has run at that point, so the stamp has to happen after the
// handler returns as well — and behaviours 1.9 says every response, which
// includes that one.
func TestAHandlerThatWritesNothingIsStillStamped(t *testing.T) {
	recorder := httptest.NewRecorder()

	fixedStamp(time.Millisecond).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/System/Ping", nil))

	if got := recorder.Header().Values(ResponseTimeHeader); len(got) != 1 || got[0] != "1" {
		t.Errorf("%s = %v, want exactly [\"1\"] on a response the handler left to net/http", ResponseTimeHeader, got)
	}
}

// TestTheResponseIsStampedOnceAndOnlyBeforeItStarts asserts the guard that
// makes the previous test safe.
//
// The header is a measurement of when the response started, so a second stamp
// after the body has begun would be both a lie and — on a real connection —
// silently discarded. The recorder cannot show the discarding, so the count is
// what is asserted: one field line, holding the first value.
func TestTheResponseIsStampedOnceAndOnlyBeforeItStarts(t *testing.T) {
	elapsed := time.Duration(0)
	stage := &ResponseTimeStamp{measure: func() func() time.Duration {
		return func() time.Duration {
			elapsed += time.Millisecond
			return elapsed
		}
	}}

	recorder := httptest.NewRecorder()
	stage.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("first"))
		_, _ = w.Write([]byte("second"))
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/System/Ping", nil))

	if got := recorder.Header().Values(ResponseTimeHeader); len(got) != 1 || got[0] != "1" {
		t.Errorf("%s = %v, want exactly [\"1\"] — the stamp is taken once, when the response starts", ResponseTimeHeader, got)
	}
}

// TestTheWrapperKeepsTheWritersOwnCopyPath is about what a ResponseWriter
// decorator quietly costs.
//
// http.ServeContent copies a body through io.ReaderFrom when the writer has
// one, which is how net/http reaches sendfile for an *os.File. A wrapper that
// does not forward ReadFrom is invisible in every test in this repository and
// turns every byte of 008's streaming routes into a userspace copy. The
// response is still stamped, because the copy is the moment it starts.
func TestTheWrapperKeepsTheWritersOwnCopyPath(t *testing.T) {
	underlying := &readerFromWriter{ResponseWriter: httptest.NewRecorder()}
	stamped := &stampingWriter{ResponseWriter: underlying, stamp: func(header http.Header) {
		header.Set(ResponseTimeHeader, "1")
	}}

	// A plain reader, because io.Copy prefers the *source's* WriteTo when it
	// has one — strings.Reader does — and would then never ask the
	// destination for its copy path at all.
	written, err := io.Copy(stamped, plainReader{strings.NewReader("a body")})
	if err != nil {
		t.Fatalf("copying through the wrapper: %v", err)
	}
	if written != int64(len("a body")) {
		t.Errorf("copied %d bytes, want %d", written, len("a body"))
	}
	if !underlying.readFromCalled {
		t.Error("the wrapped writer's ReadFrom was not reached, so a body would be copied byte by byte where net/http would use sendfile")
	}
	if got := stamped.Header().Get(ResponseTimeHeader); got != "1" {
		t.Errorf("%s = %q after a copy, want the response stamped by it", ResponseTimeHeader, got)
	}
}

// TestTheWrapperCanBeUnwrapped covers the other direction: everything a
// handler reaches through http.ResponseController — Flush, Hijack, the
// deadlines — is found by following Unwrap.
func TestTheWrapperCanBeUnwrapped(t *testing.T) {
	recorder := httptest.NewRecorder()
	stamped := &stampingWriter{ResponseWriter: recorder, stamp: func(http.Header) {}}

	if stamped.Unwrap() != http.ResponseWriter(recorder) {
		t.Error("Unwrap did not answer the writer that was wrapped")
	}
	if err := http.NewResponseController(stamped).Flush(); err != nil {
		t.Errorf("flushing through the wrapper: %v", err)
	}
	if !recorder.Flushed {
		t.Error("the flush did not reach the wrapped writer")
	}
}

// fixedStamp is the stage with the clock replaced by a constant, which is the
// seam the measure field exists for: a duration that changes every run cannot
// be asserted on.
func fixedStamp(elapsed time.Duration) *ResponseTimeStamp {
	return &ResponseTimeStamp{measure: func() func() time.Duration {
		return func() time.Duration { return elapsed }
	}}
}

// plainReader hides whatever else a reader can do, leaving io.Copy with
// nothing but Read to work from.
type plainReader struct{ io.Reader }

// readerFromWriter is a ResponseWriter that also has its own copy path, the
// way net/http's real one does.
type readerFromWriter struct {
	http.ResponseWriter

	readFromCalled bool
}

func (w *readerFromWriter) ReadFrom(r io.Reader) (int64, error) {
	w.readFromCalled = true
	return io.Copy(w.ResponseWriter, r)
}
