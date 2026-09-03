package conformance_test

import (
	"net/http"
	"testing"
	"time"
)

// AC-11's first clause against the running binary.
//
// # Why this file exists, and what it is not repeating
//
// T9 and T11 prove path canonicalisation and the two refusal shapes
// thoroughly, in internal/httpapi, over a stand-in router whose handler echoes
// the route it was reached through. That is the right place for the mechanism:
// a fold is a property of a table and a request line, and proving it over
// twenty synthesised paths is worth more than proving it over four real ones.
//
// What it cannot prove is the criterion. AC-11 says a spelling reaches "the
// same route" and "returns the same bytes", and the bytes of an echo handler
// are the route's name — so every assertion of that shape holds equally on a
// server whose real handlers are wired to the wrong rows, or on one where the
// fold stage was never assembled into the pipeline the binary runs.
//
// **Measured, at the closing audit (T21):** a pipeline whose path folder
// recognises the doubled slash and then folds nothing left the whole of this
// package green, and failed only tests in internal/httpapi. Nothing in
// conformance/ sent this server any spelling but the canonical one.
// [measurement: mutation of internal/httpapi.PathFolder.Wrap, 2026-09-03]
//
// So this is the four served rows, spelled as spec 3.6's table spells them, at
// the only level where "the same bytes" means the response.
func TestEveryAcceptedSpellingOfAServedRouteAnswersTheSameBytes(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, route := range []struct {
		method    string
		canonical string
		spellings []string
	}{
		{
			http.MethodGet, publicSystemInfoPath,
			[]string{"/system/info/public", "/SYSTEM/INFO/PUBLIC", "/System/info/Public", "/System/Info/Public/"},
		},
		{
			http.MethodGet, systemInfoPath,
			[]string{"/system/info", "/SYSTEM/INFO", "/System/Info/", "/system/info/"},
		},
		{
			http.MethodGet, pingPath,
			[]string{"/system/ping", "/SYSTEM/PING", "/System/Ping/", "/system/PING/"},
		},
		{
			http.MethodPost, pingPath,
			[]string{"/system/ping", "/System/Ping/"},
		},
	} {
		t.Run(route.method+" "+route.canonical, func(t *testing.T) {
			// The same Host on every request of the group: LocalAddress is
			// derived from it (spec 3.4 tier 2, the tier every installation
			// this binary can be started as answers on), so two requests with
			// two Hosts would differ in one field for a reason that has
			// nothing to do with the spelling.
			want := server.do(t, route.method, route.canonical, goldenHost, nil)
			if want.status != http.StatusOK {
				t.Fatalf("%s %s: status %d, want 200\n%s", route.method, route.canonical, want.status, want.body)
			}

			for _, spelling := range route.spellings {
				got := server.do(t, route.method, spelling, goldenHost, nil)

				if got.status != http.StatusOK {
					t.Errorf("%s %s: status %d, want 200 — the same route as %s\n%s",
						route.method, spelling, got.status, route.canonical, got.body)
					continue
				}
				if string(got.body) != string(want.body) {
					t.Errorf("%s %s: body\n got %s\nwant %s — the same bytes as %s",
						route.method, spelling, got.body, want.body, route.canonical)
				}
				if got.header.Get("Content-Type") != want.header.Get("Content-Type") {
					t.Errorf("%s %s: Content-Type: got %q, want %q — the same as %s",
						route.method, spelling, got.header.Get("Content-Type"),
						want.header.Get("Content-Type"), route.canonical)
				}
			}
		})
	}
}

// spec 3.6: one trailing slash is the same route and the same bytes, and
// **not** a redirect.
//
// The test above cannot see the difference. Go's http.Client follows a 301 on
// its own, and the followed response is the canonical route's — so a server
// that answered every trailing slash with a redirect would pass every byte
// comparison in this file. A client that does not follow is the only instrument
// that can tell the two apart, and it is the reason this assertion is separate
// rather than another column above.
func TestATrailingSlashIsNotAnsweredWithARedirect(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	// http.ErrUseLastResponse hands back the 3xx instead of following it.
	// Everything else about this client matches the harness's.
	client := &http.Client{
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	defer client.CloseIdleConnections()

	for _, path := range []string{
		publicSystemInfoPath + "/",
		systemInfoPath + "/",
		pingPath + "/",
		"/system/info/public/",
	} {
		request, err := http.NewRequest(http.MethodGet, server.baseURL+path, nil)
		if err != nil {
			t.Fatalf("building a request for %s: %v", path, err)
		}
		request.Host = goldenHost

		got, err := client.Do(request)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		got.Body.Close()

		if got.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200 — a trailing slash is the route itself, not a redirect to it (Location: %q)",
				path, got.StatusCode, got.Header.Get("Location"))
		}
	}
}

// spec 3.6: two trailing slashes are a 404, and behaviours 1.11's refusal shape
// — empty body, no Content-Type.
//
// It is the same shape the router sends for a path matching no route, which is
// deliberate (refusal.go writes both), and this is the request that proves the
// binary really refuses rather than folding the second slash away too.
func TestTwoTrailingSlashesOnAServedRouteAreRefused(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, path := range []string{
		publicSystemInfoPath + "//",
		systemInfoPath + "//",
		pingPath + "//",
	} {
		got := server.get(t, path, goldenHost, nil)

		if got.status != http.StatusNotFound {
			t.Errorf("GET %s: status %d, want %d\n%s", path, got.status, http.StatusNotFound, got.body)
		}
		if len(got.body) != 0 {
			t.Errorf("GET %s: body: got %s, want it empty (behaviours 1.11)", path, got.body)
		}
		if contentType := got.header.Get("Content-Type"); contentType != "" {
			t.Errorf("GET %s: Content-Type: got %q, want none (behaviours 1.11)", path, contentType)
		}
	}
}
