package httpapi

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// The three empty refusal shapes of behaviours 1.11.
//
// # Why they are three functions and not three handlers
//
// A refusal shape is defined as much by the headers it does *not* carry as by
// the status it does, and an absence cannot be restored by a later stage. So
// the shape is written in one place, by the code that decides to refuse, and
// every caller that refuses calls one of these rather than assembling a status
// and a header map of its own. Path canonicalisation's doubled-slash 404 is a
// caller; so is the router's NotFound, so is the router's MethodNotAllowed,
// and so will the authentication stage's 401 be.
//
// # What the reference sends
//
// behaviours 1.11 measures the refusal shape as a property of *where* the
// refusal happened, and the three that 001 can reach are the empty ones
// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28]
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]:
//
//	401  empty body, Content-Length: 0, no Content-Type, no WWW-Authenticate
//	404  empty body, no Content-Type
//	405  empty body, no Content-Type, Allow naming every method that path has
//
// The four remaining shapes in that table — problem details, the fixed 25-byte
// text/plain body, the JSON-encoded bare string, the 415 — all belong to a
// handler that got as far as looking something up. No route in 001 has one.
//
// **Three of those four are written below, added by 002 T11 on 2026-09-03.**
// The sentence above stands as 001 wrote it — it was true of 001's routes and
// it is why these were not written then — and what changed is the feature, not
// the reading: 002 serves routes that read a store, so the shapes became
// reachable. The 415 is still nobody's, and problem details is a model rather
// than a shape.
//
// A fourth function joined the three at T16, and it is deliberately not one of
// the measured shapes: WriteInternalServerError. behaviours 1.11 has no 500
// row, plan 7 records the shape as owed, and it is written here anyway because
// a handler that cannot build its response has to answer something. It says so
// itself; do not read it as a fourth measurement.
//
// # Content-Length is set here, and not only inherited
//
// Go's net/http adds `Content-Length: 0`, and no content type, to a body-less
// response — but **not to a body-less response to a HEAD request**
// [measurement: net/http, Go 1.27.0, 2026-09-03]. So the length behaviours
// 1.11 declares would arrive on the wire on its own for a GET and go missing
// for a HEAD, and `HEAD /System/Ping` is a 405 this feature really answers
// (spec 3.6: nothing is automatic).
//
// Setting it here removes that distinction, and it is the right place for a
// second reason: a shape this project depends on should be a property of this
// project's code rather than of the runtime under it. The same argument makes
// it uniform across all three shapes rather than a special case on the one
// behaviours 1.11 happens to spell out.
//
// It also has a practical edge. httptest.ResponseRecorder synthesises none of
// what net/http adds on the wire, so a test on a recorder sees no
// Content-Length at all; setting it here is what lets the recorder and the
// wire agree.

// WriteNotFound answers the 404 of behaviours 1.11: empty body, no content
// type.
//
// This is the refusal for a path matching no route (spec 3.6), which is both
// the router's NotFound and path canonicalisation's answer to two or more
// trailing slashes. The two are one shape on purpose — plan 6.1 records that
// making them two would mean two shapes to keep the same.
func WriteNotFound(w http.ResponseWriter) {
	refuse(w, http.StatusNotFound)
}

// WriteMethodNotAllowed answers the 405 of behaviours 1.11: empty body, no
// content type, and the given Allow header.
//
// The allow argument is the finished header value — every method the *path*
// has, sorted alphabetically and joined with ", " (plan 6.5). Refusals.Allow
// is what computes one; this function does not, so that a caller holding the
// route table for another reason is not forced to hand it over again.
func WriteMethodNotAllowed(w http.ResponseWriter, allow string) {
	// Set rather than Add: chi's own default 405 handler adds one Allow field
	// line per method it thinks applies, and Add would append this project's
	// value to whatever a stage before it left behind instead of replacing it.
	w.Header().Set("Allow", allow)
	refuse(w, http.StatusMethodNotAllowed)
}

// WriteUnauthorized answers the 401 of behaviours 1.11: empty body,
// Content-Length: 0, no content type and **no WWW-Authenticate**.
//
// The missing challenge is the part worth stating. RFC 9110 requires a 401 to
// carry WWW-Authenticate and the reference sends none
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28];
// Principle I settles which of the two this project follows, and a framework
// that adds the header for correctness's sake would put a difference on every
// authenticated route at once. Hence the Del below rather than a comment
// saying nothing sets it.
//
// This is not the only 401 the reference sends: behaviours 1.11 measures
// DELETE /Items/{itemId} answering a caller who may not delete the item with
// the JSON-encoded string "Unauthorized access" instead. That is a handler
// that read the item first, and it belongs to the feature that owns the route.
// This one is the refusal for a request that arrived with no usable
// credential at all.
func WriteUnauthorized(w http.ResponseWriter) {
	refuse(w, http.StatusUnauthorized)
}

// WriteInternalServerError answers a handler that could not build its response
// at all: the status, an empty body, and nothing else.
//
// **This shape is not measured, and that is stated rather than hidden.**
// behaviours 1.11's table has seven rows and none of them is a 500; plan 7
// records the 500 as owed, under the risks. So this writes the one thing that
// is certainly right — the status — and invents no body, because a body this
// project made up is a difference on a response the reference sends something
// else for, and an empty one is at least a difference nobody can mistake for a
// measurement.
//
// A caller reaches it only when something below the wire failed: a store that
// cannot be read, or a model that cannot be serialised. Both are exceptional
// and neither is a client's fault, so there is nothing here for a client to
// act on. It carries no log line because this package holds no logger; that is
// a gap worth closing when a feature gives the edge one, and it is named here
// rather than left as a silence.
func WriteInternalServerError(w http.ResponseWriter) {
	refuse(w, http.StatusInternalServerError)
}

// The three shapes 001 could not reach, added by 002 T11.
//
// 001 wrote the empty shapes because no route it served ever got as far as
// looking something up: a refusal decided before routing has nothing to say.
// 002 serves routes that read a store, and behaviours 1.11's remaining rows
// become reachable one at a time. Three of them are shapes rather than models
// and live here, beside the four above, for the reason the file opens with —
// a shape is defined as much by the headers it does *not* carry as by the
// status it does, and an absence cannot be restored by a later stage.
//
// The fourth remaining row, RFC 9457 problem details, is deliberately not
// here. It carries a traceId, which makes it a model with fields rather than a
// header decision, and plan 7 puts it with the handler that binds the request
// that failed.
//
// What each of these three is, measured:
//
//	the controller's own refusal   4xx, `text/plain` with **no charset**, the
//	                               fixed 25 bytes `Error processing request.`
//	the controller's own message   4xx, the message as a JSON-encoded bare
//	                               string under `application/json; charset=utf-8`
//	an authorization policy         403, **no content type and no body at all**
//
// The last two rows of that table are the one worth reading twice. A `403` is
// two shapes, split by *how* the refusal was expressed rather than by which
// layer expressed it: a refusal thrown as an exception is rendered by the
// error middleware and carries the 25 bytes
// [source: Jellyfin.Api/Helpers/RequestHelpers.cs:77-81 @ v10.11.11], while
// one returned as a result carries nothing at all
// [source: Jellyfin.Api/Controllers/PlaylistsController.cs:421-427 @ v10.11.11]
// — both measured on one status
// [probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31]
// [probe: tools/probe_playlist_shares.py, Jellyfin 10.11.11, 2026-08-31].
// So WriteControllerRefusal(w, 403) and WriteForbidden(w) are not two spellings
// of the same thing, and a caller that reaches for whichever it remembers
// first is wrong half the time.

// controllerRefusalBody is the fixed body of behaviours 1.11's controller
// refusal: twenty-five bytes, the same on every status and every route
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]
// [probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31].
//
// It is a constant for two reasons. It is the same bytes wherever it is sent,
// so spec 3.3's four refusals differ in nothing but their status — and a
// constant is a body no error path can interpolate a password into, which is
// half of what plan 7 says AC-11 stands on.
const controllerRefusalBody = "Error processing request."

// WriteControllerRefusal answers behaviours 1.11's controller refusal: the
// given status, `text/plain` with **no charset parameter**, and the fixed
// twenty-five bytes `Error processing request.`
//
// The status is an argument because the shape is not one status. Spec 3.3
// sends it on `401` for an unknown username or a wrong password, on `403` for
// a disabled or locked-out account and on `400` for a missing
// `X-Emby-Authorization`, and spec 3.8 sends it on the `403` for a
// non-administrator naming somebody else in `controllableByUserId`. **The
// status is the whole of the difference between them**, which is why the
// bytes are written once here rather than four times at four call sites.
//
// # The missing charset is the detail a framework will add for you
//
// The measured content type is `text/plain`, bare
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]. Go
// sniffs `text/plain; charset=utf-8` for bytes like these, so leaving the
// header to net/http would put a parameter on the wire that the reference does
// not send — invisible to every client, visible to L3, and a difference on
// every refusal this feature sends at once. Hence the explicit Set.
//
// # The length is declared here, and that Set is currently unassertable
//
// 001's reason for setting a length explicitly does not carry over, and this
// says so rather than borrowing it. That finding was about a **body-less**
// response: net/http adds no `Content-Length` to one answering a HEAD. This
// shape writes a body, and net/http computes the length from it — **including
// for a HEAD, where the body is then discarded and the header kept**
// [measurement: net/http, Go 1.27.0, 2026-09-03]. Removing the Set below
// changes no byte on any request, and a mutation that removed it survived the
// whole suite.
//
// It stays for the reason refuse() gives, which is not an assertion about the
// wire: a shape this project depends on should be a property of this project's
// code rather than of the runtime under it, and the next ResponseWriter
// wrapper in this pipeline is one buffering middleware away from making that
// difference real. What is recorded rather than tested is that the two agree
// today — 001 recorded an unassertable distinction the same way, in the pull
// request rather than in a test that would have passed either way.
func WriteControllerRefusal(w http.ResponseWriter, status int) {
	header := w.Header()
	// Not Del("Content-Type"): this shape has one, and Set replaces whatever
	// an earlier stage left behind — which is refuse()'s rule, one line
	// further on.
	header.Set("Content-Type", "text/plain")
	header.Set("Content-Length", strconv.Itoa(len(controllerRefusalBody)))
	header.Del("WWW-Authenticate")
	w.WriteHeader(status)
	// The write can only fail once the status line and the headers are already
	// on the wire, and there is no second response to send instead. It is
	// discarded rather than handled for that reason, and not because nothing
	// can go wrong.
	_, _ = io.WriteString(w, controllerRefusalBody)
}

// WriteJSONMessage answers behaviours 1.11's fourth shape: the given status
// and the message as a **JSON-encoded bare string** — quotes included — under
// `application/json; charset=utf-8`.
//
// Three routes are measured sending it, and the one this feature owns is
// `GET /Users/{userId}` naming nobody: sixteen bytes of `"User not found"`,
// the same body to an administrator and to a non-administrator
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]. It is not
// only a `404` shape — `DELETE /Items/{itemId}` refuses a caller who may not
// delete with `401` and `"Unauthorized access"`
// [probe: tools/probe_item_deletion.py, Jellyfin 10.11.11, 2026-09-01] — which
// is why the status is an argument here too.
//
// # The body goes through internal/wire and is not fmt.Fprintf'd
//
// A JSON-encoded string is still a JSON document, so behaviours 1.16's escape
// pass applies to it: a message carrying an apostrophe or an ampersand — an
// item name, a playlist name — is escaped by the reference and would not be by
// a hand-rolled `"` + message + `"`. Writing it through wire is what keeps the
// two agreeing, and it is the rule this project states as calling the thing
// rather than copying its pattern.
//
// # The profile is pinned to plain, and that is a decision rather than an
// oversight
//
// wire.Write echoes the content type of the profile it is handed, and this
// hands it ProfilePlain unconditionally: `application/json; charset=utf-8`,
// which is exactly what the three probes above measured. What the reference
// answers to a refusal on a request that *negotiated* `profile="CamelCase"` is
// unmeasured — a bare string carries no property names, so only the echoed
// content type could differ — and this signature takes no request, so there is
// nothing here to negotiate from. A future measurement that finds an echo is a
// change to this line and to the callers that would have to pass a profile in.
func WriteJSONMessage(w http.ResponseWriter, status int, message string) {
	header := w.Header()
	// wire.Write sets the content type itself; the challenge is deleted for
	// the reason refuse() deletes it, since this shape is sent on 401 too.
	header.Del("WWW-Authenticate")
	if err := wire.Write(w, status, message, wire.ProfilePlain); err != nil {
		// Unreachable for a string — encoding one cannot fail and the plain
		// profile is always known — and handled anyway, the way Ping handles
		// it: wire.Write writes nothing to w unless the whole body was
		// produced, so there is still a refusal to send.
		WriteInternalServerError(w)
	}
}

// WriteForbidden answers behaviours 1.11's *policy* refusal: `403`, no body,
// and **no content type at all**.
//
// This is the shape a refusal takes when it is returned rather than thrown
// [probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31],
// and plan 7 gives it one row in this feature: a live token whose account was
// disabled after it was issued, refused by the authenticator before any
// handler runs. It is **not** the shape of spec 3.3's `403` for a disabled
// account attempting to log in, which is a controller refusal carrying the
// twenty-five bytes; one status, two shapes, and the login route and the
// authenticator are on opposite sides of the split.
//
// # Content-Length: 0 is sent here, and behaviours 1.11 does not measure it
//
// This goes through refuse(), so it declares a zero length like 001's four.
// The measurement behind this row is *no content type, no body*; it says
// nothing either way about the length, and a silent measurement is not a
// measured absence — the earlier reading of that same probe could not even
// tell an empty body from a body-less refusal.
//
// Sending it is the deliberate choice, for three reasons. Omitting the header
// would not produce an absence on the wire in the first place: net/http adds
// `Content-Length: 0` to a body-less response of its own accord, and the only
// request that would see the difference is a HEAD, which no route refused by
// policy answers. Making this shape the exception would mean two spellings of
// an empty refusal in one file, which is the thing this file exists to prevent.
// And 001 took the same decision for the same reason on the `404` and the
// `405`, where 1.11 does not name a length either.
//
// So what is unmeasured here is one header on one shape, and it belongs in the
// register of things this project asserts and has never measured rather than
// in a comment nobody will find. Changing the answer means changing refuse()
// and 001's assertEmptyRefusal together, not this function alone.
func WriteForbidden(w http.ResponseWriter) {
	refuse(w, http.StatusForbidden)
}

// refuse writes an empty refusal, and is where the three shapes agree.
//
// Nothing is written to the body. That is what keeps the content type off in
// the first place — net/http sniffs one only for bytes it is given — and the
// Del is for the case the sniffing rule does not cover: a stage that already
// put a content type in the header map before something downstream decided to
// refuse. A refusal that inherited it would be a body-less response
// advertising a media type, which is neither of the two shapes 1.11 measured.
func refuse(w http.ResponseWriter, status int) {
	header := w.Header()
	header.Del("Content-Type")
	header.Del("WWW-Authenticate")
	header.Set("Content-Length", "0")
	w.WriteHeader(status)
}

// Refusals is the stage that answers a request the router could not route.
//
// It exists because chi's own Allow header is wrong in three ways, and every
// one of them is a difference on the wire
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]:
//
//	PUT /System/Ping -> 405  Allow: GET\r\nAllow: POST
//	FOO /System/Ping -> 405  (no Allow field line at all)
//
// One field line per method, where the reference sends one comma-joined line —
// behaviours 1.11 measured `Allow: DELETE, POST`. HTTP combines the two into
// the same field value, which is why the difference is invisible to a client
// and visible to L3.
//
// The order is Go map-iteration order, because chi builds the list by ranging
// over its endpoints map. Over 200 identical requests it was `GET` then `POST`
// 171 times and `POST` then `GET` 29. behaviours 1.11 measured alphabetical,
// and Principle VII wants an order derived from a stable input.
//
// And a method token chi does not know reaches the 405 branch with no methods
// at all, so the header goes missing entirely. spec 3.6 puts an Allow on every
// 405.
//
// The deeper reason is the one plan 1 and plan 10 give: spec 3.6 asks for
// every method the *path* has, which is a fact about the route table rather
// than about what the router was looking for when it gave up. So Allow is
// computed here, and chi is told to call these handlers instead of its own.
//
// plan 1's own record of this measurement was corrected on 2026-09-03; the
// amendment there says what changed and why the conclusion did not.
type Refusals struct {
	// folder maps a request path to the table row it belongs to. Allow is a
	// property of a path, and a request never carries the pattern that
	// matched it: /Items/abc has to become /Items/{itemId} before the table
	// can be asked anything about it.
	folder *PathFolder

	// allow holds the finished header value per canonical path, joined once
	// at construction rather than per refusal.
	allow map[string]string
}

// NewRefusals builds the stage from a route table.
//
// It builds its own PathFolder from the same table rather than taking one,
// for the reason NewQueryFolder does: a stage owns what it reads, and
// threading one folder through three constructors makes the order they are
// built in part of the contract. The cost is one more fold of a 51-path table
// at start.
func NewRefusals(table *surface.Table) (*Refusals, error) {
	folder, err := NewPathFolder(table)
	if err != nil {
		return nil, err
	}

	paths := table.Paths()
	refusals := &Refusals{folder: folder, allow: make(map[string]string, len(paths))}
	for _, path := range paths {
		// Sorted alphabetically by the table, which is where that ordering
		// lives: behaviours 1.11 measured PUT /UserFavoriteItems/{itemId}
		// answering `Allow: DELETE, POST`, and one place that sorts is one
		// place to be wrong.
		methods := table.Methods(path)
		if len(methods) == 0 {
			return nil, fmt.Errorf("httpapi: the route table names path %q with no methods, so there is no Allow header for it", path)
		}
		refusals.allow[path] = strings.Join(methods, ", ")
	}
	return refusals, nil
}

// Allow returns the header value for a request path: every method that path
// has, sorted alphabetically, joined with ", ".
//
// It reports false for a path the table does not describe, which is not the
// same as a path with no methods — the table has no such row.
//
// The argument is an escaped path, as r.URL.EscapedPath answers one, and it is
// expected to have been through path canonicalisation already (plan 6.7 puts
// that stage before routing). A path that has not been folded is looked up
// exactly and may well miss, which is deliberate: covering for a stage that
// did not run would hide the fact that it did not run.
func (rf *Refusals) Allow(escapedPath string) (string, bool) {
	pattern, ok := rf.folder.pattern(escapedPath)
	if !ok {
		return "", false
	}
	allow, ok := rf.allow[pattern]
	return allow, ok
}

// NotFoundHandler is what a router's NotFound is set to.
func (rf *Refusals) NotFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		WriteNotFound(w)
	}
}

// MethodNotAllowedHandler is what a router's MethodNotAllowed is set to.
//
// # A path with no rows is a 404, not a 405
//
// chi answers 405 before it routes at all when the request's method is not one
// of the nine it knows, so `FOO /Nowhere` reaches this handler rather than
// NotFound [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0,
// 2026-09-03]. spec 3.6 keys its 404 on the path — "a path matching no route"
// — and says nothing about the method, so a path the table does not have is
// answered 404 here whatever arrived on it. That is a reading of the
// specification rather than a measurement: what the reference answers to an
// unknown method token has not been probed, and this is the shape that follows
// from the row 3.6 does state. A 405 would additionally have to carry an Allow
// header naming methods that do not exist.
func (rf *Refusals) MethodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allow, ok := rf.Allow(r.URL.EscapedPath())
		if !ok {
			WriteNotFound(w)
			return
		}
		WriteMethodNotAllowed(w, allow)
	}
}
