package conformance_test

import (
	"net/http"
	"testing"
)

// The `Allow` header of the two 002 paths that can produce one, asserted at the
// wire because that is where the value is a fact about this server rather than
// about a function.
//
// spec 3.6 asks for **every method the path has**, which is a property of the
// route table and not of what the router was looking for when it gave up. 001's
// T11 measured chi getting that wrong in three ways, and the measurement was
// re-run when the two 001 handlers arrived
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]:
//
//	PUT /System/Ping -> 405  Allow: GET\r\nAllow: POST   (two field lines)
//	FOO /System/Ping -> 405  (no Allow field line at all)
//
// So this file does not re-run T11's reasoning. It records that the value
// reaching a client of the real binary is the table's on the two paths feature
// 002 added, on a request a client could send and on one chi cannot answer at
// all.
//
// **002 adds a fourth way chi's value differs, and it is the largest one yet.**
// With the router's own MethodNotAllowed restored, `DELETE /Users/Configuration`
// answers `Allow: POST` then `Allow: GET`
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03] — the
// `GET` coming from `/Users/{userId}`, which is a **different row of a
// different path** that happens to match the same request path. The table
// answers `POST`, because spec 3.6 asks for the methods *that path* has. Which
// of the two the reference sends has not been probed: its 405 is raised from a
// candidate set built the same way chi's is, so `GET, POST` is the likelier
// answer there and this is a register row rather than a settled question.

// TestTheAllowHeaderOfTheTwoNewPathsComesFromTheTable is the assertion T17's
// *Verified by* line names.
//
// Each of the two paths has exactly one row in surface.yaml, so each answers a
// single comma-free `Allow` — and a single **field line**, which is the half
// chi gets wrong. behaviours 1.11 measured the reference sending one line with
// its methods joined by ", "; two lines carry the same field value to a client
// and a different one to an L3 comparison.
func TestTheAllowHeaderOfTheTwoNewPathsComesFromTheTable(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, row := range []struct {
		what   string
		method string
		path   string
		allow  string
	}{
		// A method a client could plausibly send, on a path served by one
		// other method.
		{"a delete on the configuration write", http.MethodDelete, userConfigurationPath, "POST"},
		{"a post to the session listing", http.MethodPost, sessionsPath, "GET"},

		// And a method token chi does not know, which reaches the refusal with
		// no route information at all: chi's own handler sends no Allow field
		// line here, so a value arriving is a value that came from the table.
		{"a method chi does not know, on the configuration write", "FOO", userConfigurationPath, "POST"},
		{"a method chi does not know, on the session listing", "FOO", sessionsPath, "GET"},
	} {
		t.Run(row.what, func(t *testing.T) {
			got := server.do(t, row.method, row.path, goldenHost, nil)

			if got.status != http.StatusMethodNotAllowed {
				t.Fatalf("%s %s: status %d, want %d\nbody: %s",
					row.method, row.path, got.status, http.StatusMethodNotAllowed, got.body)
			}
			if allow := got.header.Values("Allow"); len(allow) != 1 || allow[0] != row.allow {
				t.Errorf("%s %s: Allow: got %v, want exactly [%q] — every method the path has, from the route table",
					row.method, row.path, allow, row.allow)
			}

			// behaviours 1.11's empty refusal, which the 405 shares with the
			// 404: no body and no content type.
			if len(got.body) != 0 {
				t.Errorf("%s %s: body: got %s, want it empty", row.method, row.path, got.body)
			}
			if contentType := got.header.Get("Content-Type"); contentType != "" {
				t.Errorf("%s %s: Content-Type: got %q, want it absent", row.method, row.path, contentType)
			}
		})
	}
}

// TestAGetOfTheConfigurationPathIsTheParametrisedRowAndNotA405 records what
// registering /Users/Configuration and /Users/{userId} on one router costs.
//
// chi searches a static subtree first and, when nothing there answers the
// request's method, continues into the parameter node rather than stopping. So
// `GET /Users/Configuration` matches `GET /Users/{userId}` with `Configuration`
// as the identifier, and is answered by that route rather than by a 405
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03].
//
// **The reference selects the same endpoint**, by a different mechanism: its
// method policy raises a 405 only when *no* candidate for the path matches the
// method, and `Users/{userId}` does
// [source: Jellyfin.Api/Controllers/UserController.cs @ v10.11.11]. What each
// server then answers has not been probed on this path — the identifier
// grammars differ (T14's register row), so the statuses may too — and this test
// asserts only what this server does, so that the day it is measured the answer
// is written down beside the request.
//
// It is here rather than beside the Allow rows above because it is the reason
// those rows use DELETE: a GET never reaches the refusal stage at all, and a
// test that had assumed it did would have been asserting about a 401.
func TestAGetOfTheConfigurationPathIsTheParametrisedRowAndNotA405(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.get(t, userConfigurationPath, goldenHost, nil)

	if got.status == http.StatusMethodNotAllowed {
		t.Fatalf("GET %s answered 405; chi no longer falls through a static node into the parameter node, "+
			"and the measurement this file records is stale", userConfigurationPath)
	}

	// The empty 401 of the route it did reach: /Users/{userId} reads the
	// credential before it looks at the identifier (002 plan 7, T14).
	if got.status != http.StatusUnauthorized {
		t.Errorf("GET %s: status %d, want %d — the parametrised row's refusal for a request carrying no credential\nbody: %s",
			userConfigurationPath, got.status, http.StatusUnauthorized, got.body)
	}
	if len(got.body) != 0 {
		t.Errorf("GET %s: body: got %s, want it empty", userConfigurationPath, got.body)
	}
	if allow := got.header.Values("Allow"); len(allow) != 0 {
		t.Errorf("GET %s: Allow: got %v, want none — this is not a refusal of the method", userConfigurationPath, allow)
	}
}
