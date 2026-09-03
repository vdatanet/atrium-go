package httpapi_test

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/build"
	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// stamped wraps a handler in the two stages this task ships, in the relative
// order plan 6.7 declares: the response-time stamp outside the Server header.
//
// It is not the pipeline. Assembling that — and asserting the order with
// checks only the order can satisfy — is T14's, and these tests deliberately
// stop at the two stages plus whatever produces the response under test.
func stamped(handler http.Handler) http.Handler {
	return httpapi.NewResponseTimeStamp().Wrap(httpapi.NewServerHeader().Wrap(handler))
}

// responseTimeValue is the shape of the header the reference sends: fractional
// milliseconds, a full stop, no exponent, no unit, and at most four decimal
// places because the reference's own value is a whole number of
// 100-nanosecond ticks (behaviours 1.9, 1.3).
var responseTimeValue = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,4})?$`)

// assertBothHeaders asserts everything the Verified by line of T12 asks of one
// response: both headers present, each exactly once, and each carrying the
// value it is supposed to.
//
// "Each exactly once" is T11's lesson rather than pedantry. chi answers a 405
// with two Allow field lines where the reference sends one, and a reader that
// takes the first field line of a name cannot see the difference
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]. So
// every assertion here reads every field line of the name.
func assertBothHeaders(t *testing.T, response rawResponse, what string) {
	t.Helper()

	times := response.values("X-Response-Time-ms")
	switch {
	case len(times) == 0:
		t.Errorf("%s: no X-Response-Time-ms field line — behaviours 1.9 puts one on every response", what)
	case len(times) > 1:
		t.Errorf("%s: X-Response-Time-ms = %v, want exactly one field line", what, times)
	case !responseTimeValue.MatchString(times[0]):
		t.Errorf("%s: X-Response-Time-ms = %q, want fractional milliseconds like 2.1329", what, times[0])
	}

	want := "Atrium/" + build.Version()
	servers := response.values("Server")
	switch {
	case len(servers) == 0:
		t.Errorf("%s: no Server field line — behaviours 4.1 puts Atrium/<version> on every response", what)
	case len(servers) > 1:
		t.Errorf("%s: Server = %v, want exactly one field line", what, servers)
	case servers[0] != want:
		t.Errorf("%s: Server = %q, want %q", what, servers[0], want)
	}
}

// TestBothHeadersAreOnEveryResponseThisFeatureAnswers is the task's Verified
// by line, minus the row nothing can reach yet.
//
// The four statuses are asked for by name because they are produced by four
// different pieces of code: a handler, the router's NotFound, the router's
// MethodNotAllowed, and path canonicalisation refusing before the router is
// reached. **The refusals are where a header added by the wrong layer goes
// missing** — a stage that stamped inside the handler, or that set the header
// only where a body was written, passes the 200 and fails the other three.
//
// The 503 of spec 3.5 is not here because nothing in this repository answers
// one yet: the readiness gate is T13. TestTheHeadersReachAStatusNoRouteProduced
// below covers the shape of that response as far as this task can, and says
// what it does not prove.
func TestBothHeadersAreOnEveryResponseThisFeatureAnswers(t *testing.T) {
	refusals := newRefusals(t)
	folder, err := httpapi.NewPathFolder(surface.V1())
	if err != nil {
		t.Fatalf("building the path folder over the v1 table: %v", err)
	}
	handler := stamped(folder.Wrap(pingRouter(t, refusals)))

	cases := []struct {
		what   string
		method string
		target string
		status int
	}{
		{"a handler's 200", http.MethodGet, "/System/Ping", http.StatusOK},
		{"the router's empty 404", http.MethodGet, "/Nowhere", http.StatusNotFound},
		{"canonicalisation's doubled-slash 404", http.MethodGet, "/System//Ping", http.StatusNotFound},
		{"the router's 405", http.MethodPut, "/System/Ping", http.StatusMethodNotAllowed},
		// A body-less HEAD is the request T11 found could tell an explicit
		// shape from an inherited one, and it is a real 405 in this feature.
		{"the 405 answering a HEAD", http.MethodHead, "/System/Ping", http.StatusMethodNotAllowed},
	}

	for _, testCase := range cases {
		response := send(t, handler, testCase.method, testCase.target)
		if got := statusOf(t, response); got != testCase.status {
			t.Errorf("%s: %s %s answered %d, want %d", testCase.what, testCase.method, testCase.target, got, testCase.status)
		}
		assertBothHeaders(t, response, testCase.what)
	}
}

// TestTheHeadersReachAStatusNoRouteProduced covers the fourth row of the
// Verified by line as far as this task can, and is explicit about the half it
// cannot.
//
// **What it proves:** a response written by a stage rather than by a route —
// a status, a content type and a body that never went through the router —
// carries both headers, because the stamp is a writer decorator rather than
// something a handler opts into.
//
// **What it does not prove:** that the 503 of spec 3.5 carries them. The gate
// that answers it is T13's, and plan 6.7 puts that gate *outside* both of
// these stages, where — as TestAStageOutsideTheStampIsNotStamped shows — a
// refusal carries neither. T14's own Verified by line requires the opposite,
// so the two cannot both hold as written. Whoever takes T13 or T14 closes it;
// this row is a stand-in and is named one.
func TestTheHeadersReachAStatusNoRouteProduced(t *testing.T) {
	gate := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "10")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("<html>starting</html>"))
	})

	response := send(t, stamped(gate), http.MethodGet, "/System/Info/Public")
	if got := statusOf(t, response); got != http.StatusServiceUnavailable {
		t.Fatalf("the stand-in gate answered %d, want 503", got)
	}
	assertBothHeaders(t, response, "a 503 written by a stage inside the stamp")
}

// TestAStageOutsideTheStampIsNotStamped is the finding T13 and T14 need, and
// it is asserted rather than described.
//
// A middleware that answers without calling the next handler is never reached
// by anything below it. plan 6.7 orders the pipeline "readiness gate →
// response-time stamp → Server header → …", which puts the gate above both,
// while T14's Verified by line asks that "a 503 from the gate still carries
// the response-time stamp and Server". Only one of those can be true of a
// chain, and this test says which: the gate has to be *inside* the two stages,
// or it has to write both headers itself.
//
// The reference resolves it the first way. Its response-time middleware is
// registered at the outside of the main pipeline and its startup gate well
// inside it [source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11], so the
// 503 that pipeline answers while the server is loading is stamped.
func TestAStageOutsideTheStampIsNotStamped(t *testing.T) {
	refusing := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		})
	}

	response := send(t, refusing(stamped(http.NotFoundHandler())), http.MethodGet, "/System/Ping")
	if response.has("X-Response-Time-ms") || response.has("Server") {
		t.Fatalf("a stage that refuses above the stamp sent X-Response-Time-ms=%v Server=%v; the point of this test is that it cannot, so plan 6.7's order and T14's Verified by line disagree and one of them has to move",
			response.values("X-Response-Time-ms"), response.values("Server"))
	}
}

// TestTheRefusalShapesSurviveTheTwoStages is the other half of "a header added
// by the wrong layer".
//
// Adding two headers must not add a third. A writer decorator that wrote a
// byte of its own, or a stage that let net/http sniff a content type, would
// turn every empty refusal of behaviours 1.11 into a shape the reference does
// not send — and would do it on every route at once, which is what
// architecture 4 means by "the refusal shapes must survive every layer above
// them".
func TestTheRefusalShapesSurviveTheTwoStages(t *testing.T) {
	handler := stamped(pingRouter(t, newRefusals(t)))

	notFound := send(t, handler, http.MethodGet, "/Nowhere")
	assertEmptyRefusal(t, notFound, http.StatusNotFound, "the 404 under the two stages")

	methodNotAllowed := send(t, handler, http.MethodPut, "/System/Ping")
	assertEmptyRefusal(t, methodNotAllowed, http.StatusMethodNotAllowed, "the 405 under the two stages")
	if got := methodNotAllowed.values("Allow"); len(got) != 1 || got[0] != "GET, POST" {
		t.Errorf("Allow = %v, want exactly [\"GET, POST\"] — the stages must not disturb T11's header", got)
	}
}

// TestTheServerHeaderNamesTheBuildStamp asserts the value, not merely its
// presence.
//
// architecture 5 is why the version half matters: a differential run refuses
// to start unless the two servers differ on this header, because ProductName
// is "Jellyfin Server" on both by design (behaviours 4.1). A stage that sent a
// bare "Atrium" would pass every presence check in this file and leave a
// report unable to say which binary produced it.
func TestTheServerHeaderNamesTheBuildStamp(t *testing.T) {
	stage := httpapi.NewServerHeader()

	if want := "Atrium/" + build.Version(); stage.Value() != want {
		t.Errorf("Server = %q, want %q", stage.Value(), want)
	}
	if build.Version() == "" {
		t.Fatal("build.Version is empty, so the header carries a product token with an empty version")
	}
}

// TestTheServerHeaderIsAValidProductToken checks the guarantee
// NewServerHeader relies on to have no error return.
//
// RFC 9110's product token is a token, optionally a "/" and a version that is
// also a token. build.Version promises one; this asserts it, over the value
// this binary actually reports rather than over a fixture, because the stamp
// arrives from the linker and from the toolchain's VCS information and a test
// with its own string would not be looking at either.
func TestTheServerHeaderIsAValidProductToken(t *testing.T) {
	value := httpapi.NewServerHeader().Value()

	product, version, found := strings.Cut(value, "/")
	if !found {
		t.Fatalf("Server = %q, want a product token of the form Atrium/<version>", value)
	}
	for _, part := range []struct {
		what string
		text string
	}{{"the product", product}, {"the version", version}} {
		if part.text == "" {
			t.Errorf("Server = %q: %s is empty", value, part.what)
			continue
		}
		for i := 0; i < len(part.text); i++ {
			if !isTokenByte(part.text[i]) {
				t.Errorf("Server = %q: %s carries %q, which may not appear in an HTTP token", value, part.what, part.text[i])
			}
		}
	}
}

// TestTheResponseCarriesTheDateHeaderTheReferenceAlsoSends closes the question
// T1 left for this task.
//
// Go's net/http adds a Date header to every response it sends, which is a
// header this project never asked for and cannot remove through the
// ResponseWriter interface. That would be a divergence somebody had to
// declare — except that the reference sends one too, and this repository
// already measured it moving on 19 of 19 read cases of the surface
// [probe: tools/probe_reference_determinism.py, Jellyfin 10.11.11,
// 2026-09-01]: it is one of the two header values allowlist.yaml excuses on
// every endpoint, beside the response time itself, and 010 §7 OQ-3 records the
// reading. So no divergence is owed and nothing is left to declare.
//
// The test asserts the property that argument depends on — exactly one Date
// field line, parsing as an HTTP date — so that a later stage setting its own
// Date, or setting it twice, is caught here rather than in a differential run
// nobody runs automatically.
func TestTheResponseCarriesTheDateHeaderTheReferenceAlsoSends(t *testing.T) {
	response := send(t, stamped(pingRouter(t, newRefusals(t))), http.MethodGet, "/System/Ping")

	dates := response.values("Date")
	if len(dates) != 1 {
		t.Fatalf("Date = %v, want exactly one field line", dates)
	}
	if _, err := http.ParseTime(dates[0]); err != nil {
		t.Errorf("Date = %q, which does not parse as an HTTP date: %v", dates[0], err)
	}
}

// isTokenByte reports whether c is one of RFC 9110's tchar.
//
// The rule is written out here rather than borrowed from the package under
// test: a test that asks the implementation what a valid byte is agrees with
// it by construction.
func isTokenByte(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	}
	return strings.IndexByte("!#$%&'*+-.^_`|~", c) >= 0
}

// statusOf reads the status code out of a raw status line.
func statusOf(t *testing.T, response rawResponse) int {
	t.Helper()

	// "HTTP/1.1 404 Not Found" — the code is the second field.
	fields := strings.SplitN(response.statusLine, " ", 3)
	if len(fields) < 2 {
		t.Fatalf("status line = %q, want an HTTP status line", response.statusLine)
	}
	status, err := strconv.Atoi(fields[1])
	if err != nil {
		t.Fatalf("status line = %q: %v", response.statusLine, err)
	}
	return status
}

// TestAnEarlierValueIsReplacedRatherThanAppended is why both stages use Set.
//
// Two field lines of the same name are one field value with a comma in it, so
// a stage that appended to a value already in the header map would send
// `Server: Kestrel, Atrium/1.2.3` and a response time with two numbers in it —
// a shape no reference sends and no client expects. T11 wrote the same test
// for Allow, and for the same reason: the header map is shared by the whole
// chain, so what a stage does to a name a previous one wrote is a decision
// rather than an accident.
func TestAnEarlierValueIsReplacedRatherThanAppended(t *testing.T) {
	earlier := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", "Kestrel")
			w.Header().Set("X-Response-Time-ms", "9999")
			next.ServeHTTP(w, r)
		})
	}

	response := send(t, earlier(stamped(pingRouter(t, newRefusals(t)))), http.MethodGet, "/System/Ping")
	assertBothHeaders(t, response, "a response whose headers a stage above had already written")
}
