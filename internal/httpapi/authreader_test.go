package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// The rows here are the ones that need a *http.Request rather than a string:
// where the token is read *from* is the whole of what they assert.

// TestATokenIsReadOffARequestThatMatchedNoRoute is plan 6.1's decisive
// argument for reading the query names here rather than declaring them to
// query canonicalisation.
//
// 001's canonicalisation stage keys a declared spelling by route, so it folds
// nothing on a path the route table does not describe. A credential reader
// built on it would therefore stop reading APIKEY= on exactly the requests a
// credential reader is most needed for — an unrouted path, which is every
// request that is about to be refused, and every route a later feature has not
// registered yet.
//
// The request below travels through the two real canonicalisation stages, so
// the assertion is about the pipeline this server runs and not about a bare
// request value.
func TestATokenIsReadOffARequestThatMatchedNoRoute(t *testing.T) {
	for _, target := range []string{
		"/Nowhere?APIKEY=tok",
		"/Nowhere?ApiKey=tok",
		"/Nowhere?api_key=tok",
		"/Nowhere/At/All?apikey=tok",
		"/System/Ping?APIKEY=tok",
	} {
		t.Run(target, func(t *testing.T) {
			var read string
			seen := false
			handler := canonicalising(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				read, seen = httpapi.PresentedToken(r), true
			}))

			handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, target, nil))

			if !seen {
				t.Fatalf("the request never reached the handler")
			}
			if read != "tok" {
				t.Errorf("PresentedToken over %q = %q, want %q", target, read, "tok")
			}
		})
	}
}

// canonicalising wraps a handler in the two stages plan 6.7 puts in front of
// routing, built over the table and the declarations the server runs on.
func canonicalising(t *testing.T, next http.Handler) http.Handler {
	t.Helper()

	paths, err := httpapi.NewPathFolder(surface.V1())
	if err != nil {
		t.Fatalf("building the path fold over the v1 table: %v", err)
	}
	queries, err := httpapi.NewQueryFolder(surface.V1(), httpapi.V1QuerySpellings())
	if err != nil {
		t.Fatalf("building the query fold over the v1 table: %v", err)
	}
	return paths.Wrap(queries.Wrap(next))
}

// TestPresentedTokenReadsTheFieldNamesItDeclares is the wiring assertion the
// pure core cannot make: that the reader looks in the three header names 002
// spec 3.1 lists and in the request's raw query, and not in some other name
// that happens to hold a token in a test.
func TestPresentedTokenReadsTheFieldNamesItDeclares(t *testing.T) {
	for _, row := range []struct {
		name   string
		set    func(*http.Request)
		want   string
		target string
	}{
		{
			name:   httpapi.AuthorizationHeader,
			set:    func(r *http.Request) { r.Header.Set(httpapi.AuthorizationHeader, `MediaBrowser Token="tok"`) },
			want:   "tok",
			target: "/System/Ping",
		},
		{
			name:   httpapi.EmbyAuthorizationHeader,
			set:    func(r *http.Request) { r.Header.Set(httpapi.EmbyAuthorizationHeader, `MediaBrowser Token="tok"`) },
			want:   "tok",
			target: "/System/Ping",
		},
		{
			name:   httpapi.EmbyTokenHeader,
			set:    func(r *http.Request) { r.Header.Set(httpapi.EmbyTokenHeader, "tok") },
			want:   "tok",
			target: "/System/Ping",
		},
		{
			name:   "?ApiKey=",
			set:    func(*http.Request) {},
			want:   "tok",
			target: "/System/Ping?ApiKey=tok",
		},
		{
			name:   "?api_key=",
			set:    func(*http.Request) {},
			want:   "tok",
			target: "/System/Ping?api_key=tok",
		},
		{
			name:   "X-MediaBrowser-Token, which is not one of the five",
			set:    func(r *http.Request) { r.Header.Set("X-MediaBrowser-Token", "tok") },
			want:   "",
			target: "/System/Ping",
		},
		{
			name:   "nothing at all",
			set:    func(*http.Request) {},
			want:   "",
			target: "/System/Ping",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, row.target, nil)
			row.set(request)
			if got := httpapi.PresentedToken(request); got != row.want {
				t.Errorf("PresentedToken with %s = %q, want %q", row.name, got, row.want)
			}
		})
	}
}

// TestAuthorizationBearerBesideAGoodApiKeyAuthenticates is the row named in
// this task's own definition of done, sent as a request rather than as four
// strings, because it is the shape a client really produces: a connection-wide
// Authorization header set once for something else, and a credential in the
// URL handed to a media player.
func TestAuthorizationBearerBesideAGoodApiKeyAuthenticates(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/System/Info?api_key=good", nil)
	request.Header.Set(httpapi.AuthorizationHeader, "Bearer x")

	if got := httpapi.PresentedToken(request); got != "good" {
		t.Errorf("PresentedToken = %q, want %q — a header that is present but yields nothing does not stop the search", got, "good")
	}
}

// TestOnlyTheFirstAuthorizationFieldLineIsRead pins which of two field lines
// answers, because Go keeps repeated headers and something has to choose. The
// reference reads the first [source:
// Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:238 @ v10.11.11].
func TestOnlyTheFirstAuthorizationFieldLineIsRead(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/System/Ping", nil)
	request.Header.Add(httpapi.AuthorizationHeader, `MediaBrowser Token="first"`)
	request.Header.Add(httpapi.AuthorizationHeader, `MediaBrowser Token="second"`)

	if got := httpapi.PresentedToken(request); got != "first" {
		t.Errorf("PresentedToken = %q, want %q", got, "first")
	}
}
