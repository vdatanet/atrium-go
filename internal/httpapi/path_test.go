package httpapi_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// newFolder builds the folder over the real v1 table, which is the only table
// the server ever runs on.
func newFolder(t *testing.T) *httpapi.PathFolder {
	t.Helper()
	folder, err := httpapi.NewPathFolder(surface.V1())
	if err != nil {
		t.Fatalf("building the fold map over the v1 table: %v", err)
	}
	return folder
}

// TestTheV1TableFoldsWithoutAmbiguity is the check that the shipped table can
// be canonicalised at all: no path that fails to compile, and no two paths
// that accept the same requests.
func TestTheV1TableFoldsWithoutAmbiguity(t *testing.T) {
	folder := newFolder(t)

	for _, path := range surface.V1().Paths() {
		canonical, ok := folder.Canonicalise(path)
		if !ok {
			t.Errorf("Canonicalise(%q) refused the table's own spelling", path)
			continue
		}
		if canonical != path {
			t.Errorf("Canonicalise(%q) = %q, want the path unchanged: a canonical path is its own canonical form", path, canonical)
		}
	}
}

// TestEveryAcceptedSpellingFoldsToTheRoutesOwnSpelling is spec 3.6's table,
// row by row. behaviours 1.14 names three of the casings itself
// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-26].
func TestEveryAcceptedSpellingFoldsToTheRoutesOwnSpelling(t *testing.T) {
	folder := newFolder(t)

	rows := []struct {
		name string
		sent string
		want string
	}{
		{"the canonical spelling", "/System/Info/Public", "/System/Info/Public"},
		{"lower case throughout", "/system/info/public", "/System/Info/Public"},
		{"upper case throughout", "/SYSTEM/INFO/PUBLIC", "/System/Info/Public"},
		{"mixed case, the spelling behaviours 1.14 names", "/System/info/Public", "/System/Info/Public"},
		{"one trailing slash", "/System/Info/Public/", "/System/Info/Public"},
		{"one trailing slash and the wrong case", "/system/info/public/", "/System/Info/Public"},
		{"a shorter path that is a prefix of a longer one", "/system/info", "/System/Info"},
		{"a shorter path with a trailing slash", "/SYSTEM/info/", "/System/Info"},
		{"a path served by two methods", "/system/ping", "/System/Ping"},
		{"a two-word segment", "/users/authenticatebyname", "/Users/AuthenticateByName"},

		// A literal path is looked up before any parametrised one, so
		// /Items/Filters is itself rather than an item called Filters.
		{"a literal path that a parameter would also accept", "/items/filters", "/Items/Filters"},
		{"a second one", "/ITEMS/LATEST", "/Items/Latest"},

		// Parameters are values, not spellings (spec 3.6).
		{"a parameter keeps its case", "/items/AbC-123", "/Items/AbC-123"},
		{"a literal between two parameters", "/ITEMS/AbC/images/PriMary", "/Items/AbC/Images/PriMary"},
		{"two parameters, one after another", "/items/AbC/IMAGES/PriMary/2", "/Items/AbC/Images/PriMary/2"},
		{"a literal after a parameter", "/shows/SeRiEs/episodes", "/Shows/SeRiEs/Episodes"},
		{"a segment that is half literal and half parameter", "/audio/AbC/STREAM.MP4", "/Audio/AbC/stream.MP4"},
		{"the same route without the extension", "/audio/AbC/STREAM", "/Audio/AbC/stream"},
		{"a parameter that is the whole path but for one segment", "/userfavoriteitems/AbCdEf", "/UserFavoriteItems/AbCdEf"},
		{"a literal with a digit in it", "/videos/AbC/HLS1/PlAyList/SeG.TS", "/Videos/AbC/hls1/PlAyList/SeG.TS"},
		{"a fully literal last segment after four parameters", "/videos/A/B/subtitles/C/SUBTITLES.M3U8", "/Videos/A/B/Subtitles/C/subtitles.m3u8"},
		{"a half-literal last segment on the same shape", "/videos/A/B/subtitles/C/stream.SRT", "/Videos/A/B/Subtitles/C/Stream.SRT"},
		{"six segments, three of them parameters", "/playlists/P/items/I/move/3", "/Playlists/P/Items/I/Move/3"},

		// A path the table does not have is left alone for the router to
		// refuse; the trailing slash still goes.
		{"a path matching no route", "/Nope", "/Nope"},
		{"a path matching no route, with a trailing slash", "/nope/", "/nope"},
		{"the root", "/", "/"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, ok := folder.Canonicalise(row.sent)
			if !ok {
				t.Fatalf("Canonicalise(%q) refused the path; want %q", row.sent, row.want)
			}
			if got != row.want {
				t.Errorf("Canonicalise(%q) = %q, want %q", row.sent, got, row.want)
			}
		})
	}
}

// TestTwoOrMoreTrailingSlashesAreRefused is the other half of spec 3.6's
// trailing-slash row: one is trimmed, two are one too many.
func TestTwoOrMoreTrailingSlashesAreRefused(t *testing.T) {
	folder := newFolder(t)

	for _, path := range []string{
		"/System/Info/Public//",
		"/system/info/public//",
		"/System/Info/Public///",
		"/system/ping//",
		"/Nope//",
		"//",
	} {
		if got, ok := folder.Canonicalise(path); ok {
			t.Errorf("Canonicalise(%q) = %q, true; want a refusal", path, got)
		}
	}
}

// routeEcho is the handler the routers below serve: it writes the route
// pattern chi matched, so that two requests answering the same bytes is the
// same statement as two requests reaching the same route.
func routeEcho(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, chi.RouteContext(r.Context()).RoutePattern())
}

// paramEcho writes the path parameters chi bound, in the order the route
// declares them, so that a test can assert on the bytes a handler received.
func paramEcho(w http.ResponseWriter, r *http.Request) {
	params := chi.RouteContext(r.Context()).URLParams
	pairs := make([]string, 0, len(params.Keys))
	for i, key := range params.Keys {
		pairs = append(pairs, key+"="+params.Values[i])
	}
	_, _ = io.WriteString(w, strings.Join(pairs, " "))
}

// testRoutes are the paths the routers below serve: 001's own four, and the
// parametrised shapes the table carries, which are what prove a parameter is
// not respelled.
var testRoutes = []struct {
	method string
	path   string
}{
	{http.MethodGet, "/System/Info/Public"},
	{http.MethodGet, "/System/Info"},
	{http.MethodGet, "/System/Ping"},
	{http.MethodPost, "/System/Ping"},
	{http.MethodGet, "/Items/Filters"},
	{http.MethodGet, "/Items/{itemId}"},
	{http.MethodGet, "/Items/{itemId}/Images/{imageType}"},
	{http.MethodGet, "/Audio/{itemId}/stream.{container}"},
	{http.MethodGet, "/Videos/{itemId}/hls1/{playlistId}/{segmentId}.{container}"},
}

// newRouter builds a router over testRoutes. When folder is nil the router is
// bare, which is what the control below needs.
func newRouter(folder *httpapi.PathFolder, handler http.HandlerFunc) http.Handler {
	router := chi.NewRouter()
	if folder != nil {
		router.Use(folder.Wrap)
	}
	for _, route := range testRoutes {
		router.Method(route.method, route.path, handler)
	}
	return router
}

func get(t *testing.T, handler http.Handler, target string) (*http.Response, []byte) {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, target, nil))
	response := recorder.Result()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading the body of %s: %v", target, err)
	}
	response.Body.Close()
	return response, body
}

// TestEveryAcceptedSpellingReachesTheSameRouteWithTheSameBytes is spec 3.6's
// promise at the HTTP boundary rather than over a string: the router is the
// one this server runs, and the bytes compared are the response's.
func TestEveryAcceptedSpellingReachesTheSameRouteWithTheSameBytes(t *testing.T) {
	handler := newRouter(newFolder(t), routeEcho)

	groups := map[string][]string{
		"/System/Info/Public": {"/system/info/public", "/SYSTEM/INFO/PUBLIC", "/System/info/Public", "/System/Info/Public/"},
		"/System/Info":        {"/system/info", "/SYSTEM/INFO/", "/System/Info/"},
		"/System/Ping":        {"/system/ping", "/System/PING/"},
		"/Items/Filters":      {"/items/filters", "/ITEMS/FILTERS/"},
	}

	for canonical, spellings := range groups {
		response, want := get(t, handler, canonical)
		if response.StatusCode != http.StatusOK {
			t.Fatalf("GET %s: status = %d, want %d", canonical, response.StatusCode, http.StatusOK)
		}

		for _, spelling := range spellings {
			response, got := get(t, handler, spelling)
			if response.StatusCode != http.StatusOK {
				t.Errorf("GET %s: status = %d, want %d — the same route as %s", spelling, response.StatusCode, http.StatusOK, canonical)
				continue
			}
			if string(got) != string(want) {
				t.Errorf("GET %s: body = %q, want %q — the same bytes as %s", spelling, got, want, canonical)
			}
		}
	}
}

// TestTheRouterAloneRefusesWhatTheFolderAccepts is the control. Every check
// above would pass on a router that folded case by itself, and chi does not
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03] — so
// this asserts that the middleware is what makes the difference, and would
// start failing if the router ever grew the behaviour instead.
func TestTheRouterAloneRefusesWhatTheFolderAccepts(t *testing.T) {
	bare := newRouter(nil, routeEcho)

	for _, spelling := range []string{"/system/info/public", "/SYSTEM/INFO/PUBLIC", "/System/Info/Public/"} {
		response, _ := get(t, bare, spelling)
		if response.StatusCode == http.StatusOK {
			t.Errorf("GET %s against the bare router: status = 200; the folder is not what makes this spelling work", spelling)
		}
	}
}

// TestAPathParameterReachesTheHandlerByteForByte is spec 3.6's last paragraph:
// only the segments a route declares literally are matched without regard to
// case, and whatever occupies a parameter arrives as the client wrote it.
func TestAPathParameterReachesTheHandlerByteForByte(t *testing.T) {
	handler := newRouter(newFolder(t), paramEcho)

	rows := []struct {
		sent string
		want string
	}{
		{"/items/AbCdEf012", "itemId=AbCdEf012"},
		{"/ITEMS/AbCdEf012", "itemId=AbCdEf012"},
		{"/Items/AbCdEf012/images/PriMary", "itemId=AbCdEf012 imageType=PriMary"},
		{"/audio/AbCdEf012/STREAM.Mp4", "itemId=AbCdEf012 container=Mp4"},
		{"/VIDEOS/AbC/HLS1/PlAy/SeG.Ts", "itemId=AbC playlistId=PlAy segmentId=SeG container=Ts"},
	}

	for _, row := range rows {
		response, got := get(t, handler, row.sent)
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s: status = %d, want %d", row.sent, response.StatusCode, http.StatusOK)
			continue
		}
		if string(got) != row.want {
			t.Errorf("GET %s: parameters = %q, want %q", row.sent, got, row.want)
		}
	}
}

// TestTwoTrailingSlashesAnswerAnEmpty404WithNoContentType is the refusal
// shape of behaviours 1.11 for the one refusal canonicalisation owns. The
// other 404 — a path matching no route — is the router's, and T11's.
func TestTwoTrailingSlashesAnswerAnEmpty404WithNoContentType(t *testing.T) {
	reached := false
	handler := newFolder(t).Wrap(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))

	for _, target := range []string{"/System/Info/Public//", "/system/ping//", "/Nope//"} {
		response, body := get(t, handler, target)

		if response.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: status = %d, want %d", target, response.StatusCode, http.StatusNotFound)
		}
		if len(body) != 0 {
			t.Errorf("GET %s: body = %q, want empty", target, body)
		}
		if got, present := response.Header["Content-Type"]; present {
			t.Errorf("GET %s: Content-Type = %q, want the header absent", target, got)
		}
		if reached {
			t.Errorf("GET %s: the next handler ran; a doubled slash is refused before routing", target)
			reached = false
		}
	}
}

// TestNoRedirectIsEverIssued is behaviours 1.14's warning made into a check.
// The framework default answers an unmatched trailing slash with a 307 and the
// doubled slash with a 307 to a URL that works — two divergences in opposite
// directions, where the reference makes neither round trip.
func TestNoRedirectIsEverIssued(t *testing.T) {
	handler := newRouter(newFolder(t), routeEcho)

	for _, target := range []string{
		"/System/Info/Public/",
		"/system/info/public/",
		"/System/Info/Public//",
		"/System/Ping/",
		"/Nope/",
		"/Nope//",
	} {
		response, _ := get(t, handler, target)
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			t.Errorf("GET %s: status = %d, want no redirect", target, response.StatusCode)
		}
		if location := response.Header.Get("Location"); location != "" {
			t.Errorf("GET %s: Location = %q, want no redirect", target, location)
		}
	}
}

// TestTheIncomingRequestIsNotMutated guards the copy in Wrap. A middleware
// that rewrote the caller's own URL would change what every earlier stage sees
// after it returns — the response-time stamp and the access log among them.
func TestTheIncomingRequestIsNotMutated(t *testing.T) {
	handler := newFolder(t).Wrap(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/System/Info/Public" {
			t.Errorf("the routed request's path = %q, want %q", r.URL.Path, "/System/Info/Public")
		}
	}))

	request := httptest.NewRequest(http.MethodGet, "/system/info/public/", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if request.URL.Path != "/system/info/public/" {
		t.Errorf("the caller's request path = %q, want it untouched at %q", request.URL.Path, "/system/info/public/")
	}
}
