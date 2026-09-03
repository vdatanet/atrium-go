package httpapi_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// gated builds the pipeline as far as this task can: the response-time stamp
// and the Server header outside the gate, the gate outside path
// canonicalisation, and the router with this project's refusal handlers below
// that.
//
// The relative order is the decision plan 6.7 was amended with at T13, and it
// is here rather than in T14 because the amendment has to be *implementable*
// before T14 assembles anything: the two constraints that could not both hold
// were "a 503 from the gate carries both headers" and "the gate is outermost",
// and this chain is the shape that satisfies the first.
//
// It is still not the pipeline. Query canonicalisation and the handlers are
// absent because nothing here needs them; T14 assembles the whole thing and
// asserts the order with checks only the order can satisfy.
func gated(t *testing.T, gate *httpapi.ReadinessGate) http.Handler {
	t.Helper()

	refusals := newRefusals(t)
	folder, err := httpapi.NewPathFolder(surface.V1())
	if err != nil {
		t.Fatalf("building the path folder over the v1 table: %v", err)
	}
	return stamped(gate.Wrap(folder.Wrap(pingRouter(t, refusals))))
}

// routesOf001 is the four rows spec 3.5's "every route" means in this feature,
// read from the route table rather than written out, so that a row added to
// the feature is covered without anyone remembering to add it here.
func routesOf001(t *testing.T) []struct{ Method, Path string } {
	t.Helper()

	endpoints := surface.V1().ForFeature("001")
	if len(endpoints) != 4 {
		t.Fatalf("the v1 table names %d rows for feature 001, want the four of spec 3.1-3.3", len(endpoints))
	}
	routes := make([]struct{ Method, Path string }, 0, len(endpoints))
	for _, endpoint := range endpoints {
		routes = append(routes, struct{ Method, Path string }{endpoint.Method, endpoint.Path})
	}
	return routes
}

// assertStartingRefusal asserts everything spec 3.5 and AC-12 ask of one 503:
// the status, Retry-After in full integer seconds, a Message header, and a
// text/html body that is never JSON.
//
// Every assertion reads *every* field line of the name it is about. That is
// T11's lesson: a reader that takes the first field line of a name cannot see
// a header sent twice, and two Retry-After field lines are one field value
// with a comma in it, which is not an integer.
func assertStartingRefusal(t *testing.T, response rawResponse, wantMessage string, what string) {
	t.Helper()

	if !strings.HasPrefix(response.statusLine, "HTTP/1.1 503 ") {
		t.Errorf("%s: status line = %q, want it to begin \"HTTP/1.1 503 \"", what, response.statusLine)
	}

	switch hints := response.values(httpapi.RetryAfterHeader); {
	case len(hints) != 1:
		t.Errorf("%s: %s = %v, want exactly one field line — spec 3.5 makes it what separates \"starting\" from \"broken\"", what, httpapi.RetryAfterHeader, hints)
	default:
		assertFullIntegerSeconds(t, hints[0], what)
	}

	switch messages := response.values(httpapi.MessageHeader); {
	case len(messages) != 1:
		t.Errorf("%s: %s = %v, want exactly one field line", what, httpapi.MessageHeader, messages)
	case messages[0] != wantMessage:
		t.Errorf("%s: %s = %q, want %q", what, httpapi.MessageHeader, messages[0], wantMessage)
	}

	assertHtmlAndNeverJson(t, response, what)
}

// assertFullIntegerSeconds is AC-12's "Retry-After in full integer seconds",
// asserted as the specification words it: an integer, and **not an HTTP-date**.
//
// strconv.ParseInt alone would be too weak in one direction and too strong in
// none: it rejects "5.0" and "Wed, 03 Sep 2026 00:00:00 GMT" already, which is
// the whole of the rule. The positive check is the part worth adding — a hint
// of zero parses perfectly and tells a client to come back immediately, which
// is the hammering the header exists to prevent.
func assertFullIntegerSeconds(t *testing.T, value string, what string) {
	t.Helper()

	seconds, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		t.Errorf("%s: %s = %q, want full integer seconds and not an HTTP-date: %v", what, httpapi.RetryAfterHeader, value, err)
		return
	}
	if seconds < 1 {
		t.Errorf("%s: %s = %q, want at least one second", what, httpapi.RetryAfterHeader, value)
	}
}

// assertHtmlAndNeverJson is spec 3.5's "Body: text/html — **not** JSON",
// asserted on both halves: the declared media type, and the bytes.
//
// The bytes half is Principle VIII rather than belt and braces. A gate that
// wrote a JSON object and labelled it text/html would pass a content-type
// assertion, and a client that sniffs would parse it; a gate that wrote HTML
// and labelled it application/json would pass a body assertion. Neither is the
// response the specification describes.
func assertHtmlAndNeverJson(t *testing.T, response rawResponse, what string) {
	t.Helper()

	if got := response.values("Content-Type"); len(got) != 1 || got[0] != "text/html" {
		t.Errorf("%s: Content-Type = %v, want exactly [\"text/html\"]", what, got)
	}
	if response.body == "" {
		t.Errorf("%s: the body is empty, want the text/html document spec 3.5 declares", what)
	}
	var anything any
	if err := json.Unmarshal([]byte(response.body), &anything); err == nil {
		t.Errorf("%s: the body parses as JSON (%q), and spec 3.5 says never JSON", what, response.body)
	}
}

// TestEveryRouteAndEveryNonRouteAnswers503WhileStarting is T13's Verified by
// line and AC-12 together.
//
// The path matching no route is the row that matters most, and it is why the
// gate is a stage above routing rather than a check inside a handler: spec 3.5
// is "a property of the whole server rather than of one endpoint", so a
// request the router would refuse must be refused by the gate first. A gate
// installed as chi middleware *below* the router would answer 404 here, and
// every other row in this test would still pass.
func TestEveryRouteAndEveryNonRouteAnswers503WhileStarting(t *testing.T) {
	gate := httpapi.NewReadinessGate()
	handler := gated(t, gate)

	for _, route := range routesOf001(t) {
		what := route.Method + " " + route.Path
		assertStartingRefusal(t, send(t, handler, route.Method, route.Path), "Jellyfin Server is loading. Please try again shortly.", what)
	}

	// Four requests the router would otherwise answer itself, each reaching a
	// different piece of code once the gate is open: NotFound, NotFound after
	// canonicalisation, MethodNotAllowed, and canonicalisation's own
	// doubled-slash 404. While the gate is shut, all four are the same 503.
	for _, unrouted := range []struct{ what, method, target string }{
		{"a path matching no route", "GET", "/Nowhere"},
		{"a path matching no route, in another casing", "GET", "/nowhere/at/all"},
		{"a method the path does not have", "PUT", "/System/Ping"},
		{"the doubled trailing slash canonicalisation refuses", "GET", "/System/Ping//"},
	} {
		assertStartingRefusal(t, send(t, handler, unrouted.method, unrouted.target), "Jellyfin Server is loading. Please try again shortly.", unrouted.what)
	}
}

// TestAGateIsShutBeforeAnythingOpensIt asserts the default, which is the one
// thing about this stage that cannot be fixed later.
//
// A gate constructed open would serve every request that arrived between
// construction and the first MarkReady — the whole of the start, which is the
// window spec 3.5 exists to describe.
func TestAGateIsShutBeforeAnythingOpensIt(t *testing.T) {
	gate := httpapi.NewReadinessGate()
	if gate.Ready() {
		t.Error("a newly built gate reports itself ready, want it shut until MarkReady is called")
	}
	assertStartingRefusal(t, send(t, gated(t, gate), "GET", "/System/Info/Public"), "Jellyfin Server is loading. Please try again shortly.", "a newly built gate")
}

// TestOnceReadyTheGateStopsAnswering is the other half: a gate that never
// opened would refuse for ever, and every assertion above would still pass.
//
// The unknown path is here too, because it is the request that tells "the gate
// is open" from "the gate is answering a 503 that happens to be a 404".
func TestOnceReadyTheGateStopsAnswering(t *testing.T) {
	gate := httpapi.NewReadinessGate()
	handler := gated(t, gate)
	gate.MarkReady()

	if !gate.Ready() {
		t.Error("the gate reports itself shut after MarkReady")
	}
	for _, route := range routesOf001(t) {
		response := send(t, handler, route.Method, route.Path)
		if !strings.HasPrefix(response.statusLine, "HTTP/1.1 200 ") {
			t.Errorf("%s %s: status line = %q, want the request to reach the router", route.Method, route.Path, response.statusLine)
		}
		if response.has(httpapi.RetryAfterHeader) || response.has(httpapi.MessageHeader) {
			t.Errorf("%s %s: an open gate sent %s / %s", route.Method, route.Path, httpapi.RetryAfterHeader, httpapi.MessageHeader)
		}
	}
	assertEmptyRefusal(t, send(t, handler, "GET", "/Nowhere"), http.StatusNotFound, "an unrouted path once the gate is open")
}

// TestTheGateAnswersBeforeTheStampSoA503CarriesBothHeaders is the constraint
// that decided plan 6.7's order, asserted rather than described.
//
// T12 shipped TestAStageOutsideTheStampIsNotStamped, which proves the other
// direction: a stage that answers without calling the next handler is never
// reached by anything below it, so the gate as originally ordered — outermost,
// above the stamp — would answer a 503 carrying neither X-Response-Time-ms nor
// Server, while T14's own acceptance asks for both. The two could not both
// hold. This test is what the resolution has to satisfy.
func TestTheGateAnswersBeforeTheStampSoA503CarriesBothHeaders(t *testing.T) {
	handler := gated(t, httpapi.NewReadinessGate())

	assertBothHeaders(t, send(t, handler, "GET", "/System/Info/Public"), "the gate's 503 on a route")
	assertBothHeaders(t, send(t, handler, "GET", "/Nowhere"), "the gate's 503 on a path matching no route")
}

// TestTheStartingBodyIsTheReferencesOwnMessage pins the bytes that came from
// the reference rather than from a choice made here.
//
// The message is the reference's localised StartupEmbyServerIsLoading, which
// its own startup middleware writes as the whole body of the 503
// [source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:45-48 @
// v10.11.11] [source: Emby.Server.Implementations/Localization/Core/en-US.json:79
// @ v10.11.11]. The document around it is this project's, because there is no
// single body to copy — see plan 6.8 — so what is asserted here is that the
// message survives into both the header and the body.
func TestTheStartingBodyIsTheReferencesOwnMessage(t *testing.T) {
	const message = "Jellyfin Server is loading. Please try again shortly."

	response := send(t, gated(t, httpapi.NewReadinessGate()), "GET", "/System/Ping")
	if !strings.Contains(response.body, message) {
		t.Errorf("the body does not carry the reference's own message.\n got: %q\nwant it to contain: %q", response.body, message)
	}
	if !strings.HasPrefix(response.body, "<!DOCTYPE html>") {
		t.Errorf("the body = %q, want a document a browser will render as HTML", response.body)
	}
	if got := response.values(httpapi.RetryAfterHeader); len(got) != 1 || got[0] != "5" {
		t.Errorf("%s = %v, want [\"5\"] — the reference's own five-second hint [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:143 @ v10.11.11]", httpapi.RetryAfterHeader, got)
	}
}

// TestTheDeclaredLengthIsThisProjectsAndNotTheRuntimes asserts that the 503
// declares its own length, on a GET, on a HEAD, and on a body too big for
// net/http to measure on this server's behalf.
//
// The long reason is the row with teeth. net/http computes Content-Length for
// a response whose whole body fits in its buffer, and falls back to chunked
// transfer encoding for one that does not
// [measurement: net/http, Go 1.27.0, 2026-09-03] — so a gate that let the
// runtime declare the length would declare one for the starting message and
// none for a withdrawal an operator wrote a paragraph of. The short rows on
// their own prove nothing about the Set: they pass either way, which is what a
// mutation run showed before this case was written.
func TestTheDeclaredLengthIsThisProjectsAndNotTheRuntimes(t *testing.T) {
	handler := gated(t, httpapi.NewReadinessGate())

	get := send(t, handler, "GET", "/System/Info/Public")
	lengths := get.values("Content-Length")
	if len(lengths) != 1 {
		t.Fatalf("GET: Content-Length = %v, want exactly one field line", lengths)
	}
	if lengths[0] != strconv.Itoa(len(get.body)) {
		t.Errorf("GET: Content-Length = %q, but the body is %d bytes", lengths[0], len(get.body))
	}

	head := send(t, handler, "HEAD", "/System/Info/Public")
	if got := head.values("Content-Length"); len(got) != 1 || got[0] != lengths[0] {
		t.Errorf("HEAD: Content-Length = %v, want the same %q the GET declared", got, lengths[0])
	}
	if head.body != "" {
		t.Errorf("HEAD: body = %q, want it empty", head.body)
	}

	// Longer than net/http's own response buffer, so the runtime would send
	// this one chunked and declare nothing.
	long := httpapi.NewReadinessGate()
	if err := long.Withdraw("Rebuilding. "+strings.Repeat("This will take a while. ", 200), time.Hour); err != nil {
		t.Fatalf("withdrawing with a long reason: %v", err)
	}
	response := send(t, gated(t, long), "GET", "/System/Ping")
	if got := response.values("Content-Length"); len(got) != 1 || got[0] != strconv.Itoa(len(response.body)) {
		t.Errorf("a %d-byte body declared Content-Length = %v, want exactly [%q]", len(response.body), got, strconv.Itoa(len(response.body)))
	}
	if response.has("Transfer-Encoding") {
		t.Errorf("a %d-byte body was sent %v, want a declared length instead", len(response.body), response.values("Transfer-Encoding"))
	}
}

// TestTheGateReplacesWhatAStageAboveItWrote is why all three headers are Set
// and not Add.
//
// Two field lines of one name are one field value with a comma in it: a client
// reading Retry-After would then see "3600, 5", which is not an integer, and a
// client reading Message would see two reasons. Nothing in this pipeline
// writes either name today, so the rule is untestable without a stage that
// does — T12 learned the same thing about Server, where the Set-versus-Add
// mutation survived until a test put a stage above it.
func TestTheGateReplacesWhatAStageAboveItWrote(t *testing.T) {
	gate := httpapi.NewReadinessGate()
	inner := gated(t, gate)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(httpapi.RetryAfterHeader, "3600")
		w.Header().Set(httpapi.MessageHeader, "Something else entirely.")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		inner.ServeHTTP(w, r)
	})

	assertStartingRefusal(t, send(t, handler, "GET", "/System/Ping"), "Jellyfin Server is loading. Please try again shortly.", "a 503 under a stage that had already written all three headers")
}

// TestAWithdrawalIsTheSameShapeWithADifferentMessageAndALongerHint is spec
// 3.5's last paragraph.
//
// "The same response serves a deliberate withdrawal from service — a long
// rebuild, say — with a different message and a longer hint, without stopping
// the process." All four clauses are asserted: the same shape, a different
// message, a longer hint, and a process that is still answering — the last one
// by making the same handler serve again afterwards.
func TestAWithdrawalIsTheSameShapeWithADifferentMessageAndALongerHint(t *testing.T) {
	const reason = "Atrium is rebuilding its library index."

	gate := httpapi.NewReadinessGate()
	handler := gated(t, gate)
	gate.MarkReady()

	if err := gate.Withdraw(reason, 30*time.Minute); err != nil {
		t.Fatalf("withdrawing from service: %v", err)
	}
	if gate.Ready() {
		t.Error("the gate reports itself ready after a withdrawal")
	}

	for _, route := range routesOf001(t) {
		assertStartingRefusal(t, send(t, handler, route.Method, route.Path), reason, "withdrawn: "+route.Method+" "+route.Path)
	}
	assertStartingRefusal(t, send(t, handler, "GET", "/Nowhere"), reason, "withdrawn: a path matching no route")

	withdrawn := send(t, handler, "GET", "/System/Ping")
	hint, err := strconv.Atoi(withdrawn.values(httpapi.RetryAfterHeader)[0])
	if err != nil {
		t.Fatalf("the withdrawal's hint is not an integer: %v", err)
	}
	if hint <= 5 {
		t.Errorf("the withdrawal's %s = %d, want a longer hint than the starting five seconds", httpapi.RetryAfterHeader, hint)
	}
	if !strings.Contains(withdrawn.body, reason) {
		t.Errorf("the withdrawal's body = %q, want it to carry the operator's reason", withdrawn.body)
	}
	if strings.Contains(withdrawn.body, "Jellyfin Server is loading") {
		t.Errorf("the withdrawal's body = %q, want the operator's reason and not the starting one", withdrawn.body)
	}

	// "Without stopping the process": the same handler, the same server,
	// serving again the moment the operator says so.
	gate.MarkReady()
	if response := send(t, handler, "GET", "/System/Ping"); !strings.HasPrefix(response.statusLine, "HTTP/1.1 200 ") {
		t.Errorf("after the withdrawal was lifted: status line = %q, want the request served", response.statusLine)
	}
}

// TestAHintIsRoundedUpToAWholeSecond asserts the direction of the rounding,
// which is not arbitrary: a hint that under-states when to come back invites
// the retry storm Retry-After exists to prevent, and truncation is what Go's
// integer division does if nobody says otherwise.
func TestAHintIsRoundedUpToAWholeSecond(t *testing.T) {
	for _, testCase := range []struct {
		hint time.Duration
		want string
	}{
		{time.Nanosecond, "1"},
		{500 * time.Millisecond, "1"},
		{time.Second, "1"},
		{1500 * time.Millisecond, "2"},
		{90 * time.Second, "90"},
		{time.Hour, "3600"},
	} {
		gate := httpapi.NewReadinessGate()
		if err := gate.Withdraw("Down for maintenance.", testCase.hint); err != nil {
			t.Fatalf("withdrawing with a hint of %v: %v", testCase.hint, err)
		}
		response := send(t, gated(t, gate), "GET", "/System/Ping")
		if got := response.values(httpapi.RetryAfterHeader); len(got) != 1 || got[0] != testCase.want {
			t.Errorf("a hint of %v sent %s = %v, want [%q]", testCase.hint, httpapi.RetryAfterHeader, got, testCase.want)
		}
	}
}

// TestWithdrawRefusesWhatCannotBeSent covers the three refusals, and the first
// of them is a security property rather than tidiness.
//
// The reason becomes a header field value. One carrying CR or LF would end the
// Message field line and let the rest be read as further headers or as the
// start of the body — response splitting out of an operator's own
// configuration string. Go drops such a header silently at write time, which
// is the wrong place and the wrong answer: the caller that supplied the value
// is gone by then.
func TestWithdrawRefusesWhatCannotBeSent(t *testing.T) {
	for _, testCase := range []struct {
		what   string
		reason string
		hint   time.Duration
		want   error
	}{
		{"nothing to say", "", time.Minute, httpapi.ErrEmptyReason},
		{"a hint of zero", "Down.", 0, httpapi.ErrUnusableRetryAfter},
		{"a hint that runs backwards", "Down.", -time.Minute, httpapi.ErrUnusableRetryAfter},
		{"a carriage return", "Down.\r\nX-Injected: yes", time.Minute, nil},
		{"a line feed", "Down.\nX-Injected: yes", time.Minute, nil},
		{"a NUL", "Down.\x00", time.Minute, nil},
		{"a byte above ASCII", "Down for maintenance é.", time.Minute, nil},
	} {
		gate := httpapi.NewReadinessGate()
		err := gate.Withdraw(testCase.reason, testCase.hint)
		if err == nil {
			t.Errorf("%s: Withdraw(%q, %v) was accepted, want it refused", testCase.what, testCase.reason, testCase.hint)
			continue
		}
		if testCase.want != nil && !errors.Is(err, testCase.want) {
			t.Errorf("%s: Withdraw returned %v, want %v", testCase.what, err, testCase.want)
		}
		// A refused withdrawal changes nothing: the gate is still saying
		// whatever it was saying before.
		response := send(t, gated(t, gate), "GET", "/System/Ping")
		if got := response.values(httpapi.MessageHeader); len(got) != 1 || got[0] != "Jellyfin Server is loading. Please try again shortly." {
			t.Errorf("%s: a refused withdrawal changed the message to %v", testCase.what, got)
		}
	}
}

// TestAReasonThatLooksLikeMarkupIsTextInBothPlaces is the other half of the
// injection question, and it is why the header value and the body are rendered
// separately.
//
// The body is HTML, so a reason carrying `<` has to be escaped or it becomes
// markup — an operator writing a status message would otherwise be writing the
// page. The header is not HTML and must carry the reason as written, or a
// client reading the Message header would see the escaping.
func TestAReasonThatLooksLikeMarkupIsTextInBothPlaces(t *testing.T) {
	const reason = "Rebuilding <b>everything</b> & then some."

	gate := httpapi.NewReadinessGate()
	if err := gate.Withdraw(reason, time.Hour); err != nil {
		t.Fatalf("withdrawing with markup in the reason: %v", err)
	}
	response := send(t, gated(t, gate), "GET", "/System/Ping")

	if got := response.values(httpapi.MessageHeader); len(got) != 1 || got[0] != reason {
		t.Errorf("%s = %v, want the reason as written, %q", httpapi.MessageHeader, got, reason)
	}
	if strings.Contains(response.body, "<b>") {
		t.Errorf("the body = %q, want the reason escaped rather than rendered as markup", response.body)
	}
	if !strings.Contains(response.body, "&lt;b&gt;everything&lt;/b&gt; &amp; then some.") {
		t.Errorf("the body = %q, want it to carry the reason as escaped text", response.body)
	}
}

// TestTheGateIsSafeToChangeWhileItIsServing is why the state is an atomic
// pointer and not a bool nobody guards.
//
// MarkReady is called by the entry layer at the end of a start, and Withdraw
// by whatever an operator drives, while requests are in flight. Run with -race
// this fails on a plain field; without it, it fails on nothing, which is said
// here rather than left for somebody to discover.
func TestTheGateIsSafeToChangeWhileItIsServing(t *testing.T) {
	gate := httpapi.NewReadinessGate()
	handler := gated(t, gate)

	var waiting sync.WaitGroup
	for worker := range 8 {
		waiting.Add(1)
		go func() {
			defer waiting.Done()
			for round := range 16 {
				switch (worker + round) % 3 {
				case 0:
					gate.MarkReady()
				case 1:
					_ = gate.Withdraw(fmt.Sprintf("Worker %d, round %d.", worker, round), time.Minute)
				default:
					_ = gate.Ready()
				}
			}
		}()
	}
	for range 8 {
		waiting.Add(1)
		go func() {
			defer waiting.Done()
			for range 16 {
				// A recorder rather than the wire, because send reports a
				// failure with t.Fatalf and that may only be called from the
				// goroutine running the test. Nothing here asserts on the
				// response: what is under test is that reading the state
				// while it changes is defined.
				handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/System/Ping", nil))
			}
		}()
	}
	waiting.Wait()
}
