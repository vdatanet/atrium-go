package httpapi

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/vdatanet/atrium-go/internal/surface"
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
