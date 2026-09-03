package httpapi

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// RetryAfterHeader and MessageHeader are the two field names spec 3.5 puts on
// the 503 this stage answers.
//
// Both spellings are the pinned document's own
// [spec: every operation's 503 response in the pinned 10.11.11 document]
// [source: Jellyfin.Server/Filters/RetryOnTemporarilyUnavailableFilter.cs:19,32
// @ v10.11.11], where Retry-After is declared `integer` / `int32` and
// described as "a hint for when to retry the operation in full seconds", and
// Message is declared `string` and described as "a short plain-text reason why
// the server is not available".
const (
	RetryAfterHeader = "Retry-After"
	MessageHeader    = "Message"
)

// startingMessage is the reason this server gives while it is coming up.
//
// It is the reference's own string rather than a phrasing invented here:
// its startup middleware writes the localised StartupEmbyServerIsLoading as
// the whole body of the 503
// [source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:45-48 @
// v10.11.11], and the en-US value of that key is exactly these bytes
// [source: Emby.Server.Implementations/Localization/Core/en-US.json:79 @
// v10.11.11]. Principle I: where the reference has a string, this project
// sends that string.
const startingMessage = "Jellyfin Server is loading. Please try again shortly."

// startingRetryAfter is the hint sent while the server is starting.
//
// Five seconds is the reference's own value — the hint its setup server sends
// on the 503 it answers before the application exists
// [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:143 @ v10.11.11].
const startingRetryAfter = 5 * time.Second

// bodyContentType is the media type spec 3.5 requires of the 503 body:
// text/html, and never JSON.
//
// It carries no charset parameter, because the reference's does not: its
// startup middleware assigns the bare MediaTypeNames.Text.Html
// [source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:47 @
// v10.11.11] and its setup server writes the literal "text/html"
// [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:243 @ v10.11.11].
// internal/wire's JSON content types carry charset=utf-8 (behaviours 1.10);
// this one is a different response from a different writer and copies neither.
const bodyContentType = "text/html"

// ErrEmptyReason and ErrUnusableRetryAfter are what Withdraw refuses.
var (
	// ErrEmptyReason is returned for a withdrawal with nothing to say. A
	// Message field line with an empty value is a header a client cannot act
	// on and cannot distinguish from a server that forgot to send one.
	ErrEmptyReason = errors.New("httpapi: a withdrawal needs a reason to send in the Message header")

	// ErrUnusableRetryAfter is returned for a hint of zero or less. spec 3.5
	// makes Retry-After the difference between "starting" and "broken"; a hint
	// of zero says "retry immediately", which is the hammering the header
	// exists to prevent.
	ErrUnusableRetryAfter = errors.New("httpapi: a withdrawal needs a Retry-After hint of at least one second")
)

// ReadinessGate is the stage that answers spec 3.5's 503 until this server is
// ready to serve, and again whenever an operator withdraws it from service.
//
// # Nothing is exempt
//
// Every operation the reference declares declares a 503 — 389 of 389 in the
// pinned document, each with Retry-After, Message and a text/html body — so
// spec 3.5 exempts nothing, "not even a liveness probe", and this gate is
// therefore a stage rather than a check any handler makes
// [spec: every operation's 503 response in the pinned 10.11.11 document].
// The declaration is applied to every operation at once by an OpenAPI
// operation filter
// [source: Jellyfin.Server/Filters/RetryOnTemporarilyUnavailableFilter.cs:7-51
// @ v10.11.11], which is why the count is a property of the installation and
// the claim is the coverage rather than the number.
//
// # A contradiction this stage implements one side of, and a probe it owes
//
// The reference's *running* behaviour does not match its own declaration in
// three places, all read at the pinned tag and none of them probed:
//
//  1. the 503 answered before the application exists comes from a **separate
//     setup web server**, not from the pipeline this stage is in
//     [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:177-259 @
//     v10.11.11]. That server registers no response-time middleware, so its
//     503 very likely carries no X-Response-Time-ms at all.
//  2. that setup server answers a real /System/Info/Public body, with
//     StartupWizardCompleted false, rather than a 503
//     [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:204-237 @
//     v10.11.11]. So one route is exempt there.
//  3. the **main** pipeline's gate exempts /system/ping case-insensitively and
//     sends neither Retry-After nor Message — only the status, text/html and
//     the localised string
//     [source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:38-48
//     @ v10.11.11]. So the two headers are only ever sent together by the
//     setup server, which exempts two paths of its own.
//
// AGENTS.md 1.3 ranks these: where a probe, a source line and the pinned
// document disagree, **the running server wins**. No running reference exists
// in this repository and no CI job may start one (AGENTS.md 1.6), so the
// disagreement cannot be settled here. spec 3.5 and AC-12 are the authority on
// WHAT until it is, and this stage implements them as written: every route,
// no exemption, both headers, a text/html body. plan 6.8 records the
// contradiction and says what settles it — one probe against a starting
// reference, and 010's differential run.
//
// # State
//
// A gate is unavailable from the moment it is built. That is the whole point:
// a gate that had to be told to refuse would serve for the length of whatever
// went wrong between construction and the first MarkReady, which is exactly
// the window it exists to close.
type ReadinessGate struct {
	// unavailable holds the refusal to send, or nil when the server is
	// serving. One atomic load per request is the whole hot path, because the
	// three header values and the body are rendered at the moment the state
	// changes rather than per request.
	//
	// A pointer rather than a mutex-guarded struct because the read is on
	// every request of every route and the write happens twice in the life of
	// a process.
	unavailable atomic.Pointer[unavailability]
}

// unavailability is one refusal, fully rendered.
//
// Rendering here rather than in Wrap keeps operator-supplied text out of the
// request path in both senses: the escaping happens once, and a reason that
// could not be rendered was refused by Withdraw before it ever reached this
// struct.
type unavailability struct {
	message    string
	retryAfter string
	body       string
}

// NewReadinessGate builds the stage, unavailable, with the message and hint
// this server sends while it is starting.
//
// Like NewResponseTimeStamp and NewServerHeader, and unlike the three stages
// that read the route table, it cannot fail: its two values are constants of
// this package. Withdraw is where a caller's input is validated, and that is
// where the error return is.
func NewReadinessGate() *ReadinessGate {
	gate := &ReadinessGate{}
	gate.unavailable.Store(render(startingMessage, startingRetryAfter))
	return gate
}

// MarkReady opens the gate: every request from here on reaches the pipeline
// below it.
//
// It also restores a withdrawn server, which is the other half of spec 3.5's
// "without stopping the process" — a withdrawal that could only be undone by a
// restart would be a stop with extra steps.
func (g *ReadinessGate) MarkReady() {
	g.unavailable.Store(nil)
}

// Withdraw takes the server out of service deliberately, with its own reason
// and its own hint, and without stopping the process (spec 3.5).
//
// The response is the same shape the starting server sends, which is the point
// of the sentence in the specification: a client that already handles a
// starting server handles a rebuilding one with no new code. What differs is
// the message and the length of the hint.
//
// # Why the reason is validated rather than trusted
//
// It becomes a header field value. A reason carrying CR or LF would end the
// Message field line and let whatever followed it be read as further headers,
// or as the start of the body — response splitting, from an operator's own
// configuration string. Go's net/http refuses some of this at write time, but
// silently, by dropping the header: the refusal belongs here, where the caller
// that supplied the value is still the one being answered.
//
// The retryAfter is rounded **up** to a whole second, because the header is a
// hint about when to come back and one that under-states it invites the
// hammering it exists to prevent. Anything at or below zero is refused rather
// than rounded to one: a caller asking for no delay has not thought about the
// header, and guessing on their behalf would hide that.
func (g *ReadinessGate) Withdraw(reason string, retryAfter time.Duration) error {
	if reason == "" {
		return ErrEmptyReason
	}
	if index := indexOfUnsendableByte(reason); index >= 0 {
		return fmt.Errorf("httpapi: the withdrawal reason carries byte %#02x at offset %d, which cannot appear in a %s header field value", reason[index], index, MessageHeader)
	}
	if retryAfter <= 0 {
		return ErrUnusableRetryAfter
	}
	g.unavailable.Store(render(reason, retryAfter))
	return nil
}

// Ready reports whether the gate is open.
//
// It exists for the entry layer's log line and for a test that wants to assert
// the state without issuing a request. Nothing in the request path reads it —
// Wrap reads the pointer once, because asking "is it ready" and then "what
// does it say" would be two loads with a state change possible between them.
func (g *ReadinessGate) Ready() bool {
	return g.unavailable.Load() == nil
}

// Wrap is the middleware.
//
// plan 6.7 (amended 2026-09-03, at T13) puts this stage immediately inside the
// response-time stamp and the Server header and outside everything else, so a
// 503 from here carries both of those and nothing between this stage and
// routing can exempt a path from it.
func (g *ReadinessGate) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := g.unavailable.Load()
		if state == nil {
			next.ServeHTTP(w, r)
			return
		}

		header := w.Header()
		// Set rather than Add on all three: nothing above this stage writes
		// any of them today, and a stage that later did would be replaced
		// rather than appended to. Two Retry-After field lines are one field
		// value with a comma in it, which is not an integer.
		header.Set(RetryAfterHeader, state.retryAfter)
		header.Set(MessageHeader, state.message)
		header.Set("Content-Type", bodyContentType)
		// Explicitly, so that the response is the same length whether it was
		// asked for with GET or with HEAD: net/http computes one for a small
		// body it is given, and is given nothing for a HEAD
		// [measurement: net/http, Go 1.27.0, 2026-09-03].
		header.Set("Content-Length", strconv.Itoa(len(state.body)))
		w.WriteHeader(http.StatusServiceUnavailable)
		// The error is ignored for the reason every handler ignores it: the
		// status line has gone out, so there is nothing left to say to the
		// client, and a connection that died mid-body is the client's
		// business rather than a failure of this server.
		_, _ = w.Write([]byte(state.body))
	})
}

// render turns a reason and a hint into the three values that go on the wire.
//
// The body is the message in a minimal HTML document. That shape is this
// project's, not the reference's: the reference's main-pipeline gate writes
// the bare string with a text/html content type
// [source: Jellyfin.Api/Middleware/ServerStartupMessageMiddleware.cs:45-48 @
// v10.11.11] and its setup server renders a whole page out of the startup log
// [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:239-258 @ v10.11.11],
// so there is no single body to copy and spec 3.5 asks only for the media
// type. A document a browser can render is the useful reading of "text/html",
// and the bytes are ⚠️ UNVERIFIED against a running reference — plan 6.8
// records what a probe would settle.
//
// html.EscapeString is what keeps an operator's reason from being markup. The
// header carries the reason as written; only the body is escaped, because only
// the body is parsed as HTML.
func render(message string, retryAfter time.Duration) *unavailability {
	seconds := int64(retryAfter / time.Second)
	if retryAfter%time.Second != 0 {
		seconds++
	}
	return &unavailability{
		message: message,
		// Full seconds as an integer, never an HTTP-date (spec 3.5). The
		// reference's setup server zero-pads to three digits — "005" for its
		// five-second hint
		// [source: Jellyfin.Server/ServerSetupApp/SetupServer.cs:242 @
		// v10.11.11] — which parses as the same integer and is not what the
		// pinned document declares. plan 6.8 records it as owed a probe.
		retryAfter: strconv.FormatInt(seconds, 10),
		body:       "<!DOCTYPE html>\n<html lang=\"en\">\n<head><meta charset=\"utf-8\"><title>Service unavailable</title></head>\n<body><p>" + html.EscapeString(message) + "</p></body>\n</html>\n",
	}
}

// indexOfUnsendableByte returns the offset of the first byte that may not
// appear in a header field value, or -1.
//
// RFC 9110 5.5 allows visible ASCII, space, horizontal tab and obs-text
// (0x80-0xFF). This is stricter: obs-text is refused too, because a reason is
// plain text the pinned document describes as such, and a non-ASCII byte in a
// field value has no declared encoding — a client reading it as Latin-1 and a
// client reading it as UTF-8 see different strings. A reason that needs a
// character outside ASCII belongs in the body, which has a charset.
func indexOfUnsendableByte(s string) int {
	for i := 0; i < len(s); i++ {
		if b := s[i]; b != '\t' && (b < ' ' || b > '~') {
			return i
		}
	}
	return -1
}
