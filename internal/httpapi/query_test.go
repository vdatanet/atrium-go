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

// declared is the declaration set the tests below run on.
//
// It is a fixture over the *real* table, not a fixture table: the routes are
// rows of surface.yaml and the route lookup is the shipped one, so a test that
// passes here is a test about the code that runs. The declarations themselves
// have to be invented, because the set this server runs on —
// httpapi.V1QuerySpellings — is empty: none of 001's four routes takes a query
// parameter. TestTheSetTheServerRunsOnFoldsNothing is that state's own check.
//
// The names are spelled camelCase, which is how the pinned OpenAPI document
// spells every parameter (behaviours 1.15).
var declared = httpapi.QuerySpellings{
	{Method: http.MethodGet, Path: "/Items/Filters"}:  {"limit", "startIndex", "sortBy"},
	{Method: http.MethodGet, Path: "/Items/{itemId}"}: {"userId"},

	// One method of a two-method path declares a name and the other does not,
	// which is what makes the declarations per route rather than per path.
	{Method: http.MethodGet, Path: "/System/Ping"}: {"limit"},
}

// queryEcho writes the raw query string the handler received. It is the bytes
// at the boundary rather than a parsed map: a test that compared parsed values
// could not see the order of the fragments, a repeated name, or a fragment
// that is not a name=value pair at all.
func queryEcho(w http.ResponseWriter, r *http.Request) {
	_, _ = io.WriteString(w, r.URL.RawQuery)
}

// boundEcho writes what a handler binding these names would read, which is the
// question behaviours 1.15 is actually about.
func boundEcho(names ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values := r.URL.Query()
		pairs := make([]string, 0, len(names))
		for _, name := range names {
			pairs = append(pairs, name+"="+strings.Join(values[name], ","))
		}
		_, _ = io.WriteString(w, strings.Join(pairs, " "))
	}
}

// newQueryRouter builds the two canonicalisation stages in the order plan 6.7
// fixes — path, then query — over the routes T9's tests already declare. When
// spellings is nil the query stage is left out, which is what the control
// below needs.
func newQueryRouter(t *testing.T, spellings httpapi.QuerySpellings, handler http.HandlerFunc) http.Handler {
	t.Helper()

	router := chi.NewRouter()
	router.Use(newFolder(t).Wrap)
	if spellings != nil {
		folder, err := httpapi.NewQueryFolder(surface.V1(), spellings)
		if err != nil {
			t.Fatalf("building the query folder over the v1 table: %v", err)
		}
		router.Use(folder.Wrap)
	}
	for _, route := range testRoutes {
		router.Method(route.method, route.path, handler)
	}
	return router
}

func do(t *testing.T, handler http.Handler, method, target string) (*http.Response, []byte) {
	t.Helper()

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(method, target, nil))
	response := recorder.Result()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading the body of %s %s: %v", method, target, err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("%s %s: status = %d, want %d", method, target, response.StatusCode, http.StatusOK)
	}
	return response, body
}

// TestEverySpellingOfADeclaredNameReachesTheHandlerAsTheDeclaredOne is
// behaviours 1.15's own three spellings — Limit=1, limit=1 and LIMIT=1 are the
// same parameter
// [probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28] — at
// the HTTP boundary, asserted on the bytes the handler received.
func TestEverySpellingOfADeclaredNameReachesTheHandlerAsTheDeclaredOne(t *testing.T) {
	handler := newQueryRouter(t, declared, queryEcho)

	for _, sent := range []string{
		"/Items/Filters?Limit=1",
		"/Items/Filters?limit=1",
		"/Items/Filters?LIMIT=1",
		"/Items/Filters?LiMiT=1",

		// The path stage runs first, so a folded path finds its route here
		// too.
		"/items/filters?LIMIT=1",
		"/ITEMS/FILTERS/?Limit=1",
	} {
		_, got := do(t, handler, http.MethodGet, sent)
		if string(got) != "limit=1" {
			t.Errorf("GET %s: the handler saw %q, want %q", sent, got, "limit=1")
		}
	}
}

// TestADeclaredNameIsBoundWhateverTheClientSpelled is the same statement one
// level up: what a handler reading the declared name actually gets.
func TestADeclaredNameIsBoundWhateverTheClientSpelled(t *testing.T) {
	handler := newQueryRouter(t, declared, boundEcho("limit", "startIndex", "sortBy"))

	rows := []struct {
		sent string
		want string
	}{
		{"/Items/Filters?limit=25&startIndex=50&sortBy=SortName", "limit=25 startIndex=50 sortBy=SortName"},
		{"/Items/Filters?Limit=25&StartIndex=50&SortBy=SortName", "limit=25 startIndex=50 sortBy=SortName"},
		{"/Items/Filters?LIMIT=25&STARTINDEX=50&SORTBY=SortName", "limit=25 startIndex=50 sortBy=SortName"},
		{"/Items/Filters?sortby=SortName&limit=25&startindex=50", "limit=25 startIndex=50 sortBy=SortName"},
	}

	for _, row := range rows {
		_, got := do(t, handler, http.MethodGet, row.sent)
		if string(got) != row.want {
			t.Errorf("GET %s: the handler bound %q, want %q", row.sent, got, row.want)
		}
	}
}

// TestTheFrameworkAloneIgnoresTheOtherSpelling is the control, and it is the
// failure behaviours 1.15 describes rather than a hypothetical one: an
// unrecognised spelling is not rejected but ignored, "which for StartIndex
// against a camelCase route means every page is page one". Every check above
// would pass on a stack that folded query names by itself, and this asserts
// that nothing here does.
func TestTheFrameworkAloneIgnoresTheOtherSpelling(t *testing.T) {
	bare := newQueryRouter(t, nil, boundEcho("startIndex"))
	folded := newQueryRouter(t, declared, boundEcho("startIndex"))

	const sent = "/Items/Filters?StartIndex=25"

	if _, got := do(t, bare, http.MethodGet, sent); string(got) != "startIndex=" {
		t.Errorf("GET %s without the stage: the handler bound %q, want %q — page one, silently", sent, got, "startIndex=")
	}
	if _, got := do(t, folded, http.MethodGet, sent); string(got) != "startIndex=25" {
		t.Errorf("GET %s with the stage: the handler bound %q, want %q", sent, got, "startIndex=25")
	}
}

// TestAValueIsNeverTouched is the rule in plan 6.2 and architecture 4: only
// names fold, and everything after the first `=` is spliced back unread.
func TestAValueIsNeverTouched(t *testing.T) {
	handler := newQueryRouter(t, declared, queryEcho)

	rows := []struct {
		name string
		sent string
		want string
	}{
		{"a value differing from its name only in case", "/Items/Filters?Limit=LIMIT", "limit=LIMIT"},
		{"a value that is the declared spelling", "/Items/Filters?LIMIT=limit", "limit=limit"},
		{"a mixed-case value", "/Items/Filters?limit=AbCdEf", "limit=AbCdEf"},
		{"a percent-escaped value, kept in the client's own escaping", "/Items/Filters?LIMIT=a%2Fb%2Dc", "limit=a%2Fb%2Dc"},
		{"a value carrying a plus", "/Items/Filters?Limit=a+b", "limit=a+b"},
		{"a value carrying an equals sign", "/Items/Filters?Limit=a=b", "limit=a=b"},
		{"an empty value", "/Items/Filters?LIMIT=", "limit="},
		{"a name with no value at all", "/Items/Filters?LIMIT", "limit"},
		{"a repeated name, each spelled differently", "/Items/Filters?Limit=1&LIMIT=2", "limit=1&limit=2"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, got := do(t, handler, http.MethodGet, row.sent)
			if string(got) != row.want {
				t.Errorf("GET %s: the handler saw %q, want %q", row.sent, got, row.want)
			}
		})
	}
}

// TestAnUnrecognisedNameIsLeftInPlace is behaviours 1.12's half of this stage.
// An unrecognised parameter is ignored rather than rejected, and 010 3.6
// counts what was ignored — so a name this stage does not know must still be
// there, in its own position, for something later to count.
func TestAnUnrecognisedNameIsLeftInPlace(t *testing.T) {
	handler := newQueryRouter(t, declared, queryEcho)

	rows := []struct {
		name string
		sent string
		want string
	}{
		{"an unknown name, alone", "/Items/Filters?NotAParameter=1", "NotAParameter=1"},
		{"an unknown name beside a declared one", "/Items/Filters?NotAParameter=1&Limit=2", "NotAParameter=1&limit=2"},
		{"the declared one first", "/Items/Filters?LIMIT=2&NotAParameter=1", "limit=2&NotAParameter=1"},
		{"an unknown name in the middle", "/Items/Filters?Limit=1&Nope=x&SortBy=y", "limit=1&Nope=x&sortBy=y"},
		{"an empty fragment", "/Items/Filters?Limit=1&&SortBy=y", "limit=1&&sortBy=y"},
		{"a fragment that is only a value", "/Items/Filters?=1&Limit=2", "=1&limit=2"},

		// `&` is the only separator this stage splits on, so a semicolon is
		// an ordinary byte of a name and the fragment is left where it is.
		// What binds it afterwards is a separate question and not this
		// stage's: Go's own query parser discards a pair carrying a
		// semicolon, answering `invalid semicolon separator in query`
		// [measurement: net/url, Go 1.27.0, 2026-09-03]. This stage neither
		// creates that nor hides it.
		{"a semicolon inside a name", "/Items/Filters?a;Limit=1&Limit=2", "a;Limit=1&limit=2"},

		// The fold is on the query string's own bytes, undecoded — the same
		// choice the path stage made (plan 6.1). A percent-encoded name is
		// therefore not the name it decodes to, and what the reference does
		// with one has not been measured; plan 6.2 records the owed probe.
		{"a percent-encoded name", "/Items/Filters?%4Cimit=1", "%4Cimit=1"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, got := do(t, handler, http.MethodGet, row.sent)
			if string(got) != row.want {
				t.Errorf("GET %s: the handler saw %q, want %q", row.sent, got, row.want)
			}
		})
	}
}

// TestTheDeclarationsAreThoseOfTheMatchedRoute covers the three ways a request
// finds — or fails to find — a declaration: the wrong method on a path that
// has one, a route that declares nothing, and a path the table does not have.
func TestTheDeclarationsAreThoseOfTheMatchedRoute(t *testing.T) {
	handler := newQueryRouter(t, declared, queryEcho)

	rows := []struct {
		name   string
		method string
		sent   string
		want   string
	}{
		{"the method that declares it", http.MethodGet, "/System/Ping?LIMIT=1", "limit=1"},
		{"the method on the same path that does not", http.MethodPost, "/System/Ping?LIMIT=1", "LIMIT=1"},
		{"a route that declares nothing", http.MethodGet, "/System/Info?LIMIT=1", "LIMIT=1"},
		{"another route's name", http.MethodGet, "/Items/Filters?UserId=1", "UserId=1"},
		{"its own route's name", http.MethodGet, "/Items/AbC?UserId=1", "userId=1"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, got := do(t, handler, row.method, row.sent)
			if string(got) != row.want {
				t.Errorf("%s %s: the handler saw %q, want %q", row.method, row.sent, got, row.want)
			}
		})
	}
}

// TestAParametrisedRouteFoldsItsQueryAndNotItsPath puts the two stages
// together on the shape that needs both: the route is found by its pattern,
// the query name folds, and the path parameter arrives byte for byte.
func TestAParametrisedRouteFoldsItsQueryAndNotItsPath(t *testing.T) {
	handler := newQueryRouter(t, declared, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, chi.URLParam(r, "itemId")+" "+r.URL.RawQuery)
	})

	rows := []struct {
		sent string
		want string
	}{
		{"/Items/AbCdEf?UserId=XyZ", "AbCdEf userId=XyZ"},
		{"/items/AbCdEf?USERID=XyZ", "AbCdEf userId=XyZ"},
		{"/ITEMS/AbCdEf/?userid=XyZ", "AbCdEf userId=XyZ"},
	}

	for _, row := range rows {
		_, got := do(t, handler, http.MethodGet, row.sent)
		if string(got) != row.want {
			t.Errorf("GET %s: the handler saw %q, want %q", row.sent, got, row.want)
		}
	}
}

// TestTheSetTheServerRunsOnFoldsNothing is 001's own state, made into a check
// rather than left as a claim in a comment: the declaration set is empty, so
// every query on every route this server serves arrives exactly as it was
// sent. The day a feature declares its first name, this test still passes and
// the routes below still take no query parameter.
func TestTheSetTheServerRunsOnFoldsNothing(t *testing.T) {
	folder, err := httpapi.NewQueryFolder(surface.V1(), httpapi.V1QuerySpellings())
	if err != nil {
		t.Fatalf("building the query folder the server runs on: %v", err)
	}

	for _, endpoint := range surface.V1().ForFeature("001") {
		for _, rawQuery := range []string{"", "Limit=1", "limit=1&StartIndex=2", "NotAParameter=x"} {
			got := folder.Canonicalise(endpoint.Method, endpoint.Path, rawQuery)
			if got != rawQuery {
				t.Errorf("Canonicalise(%s, %q, %q) = %q, want the query unchanged: 001 declares no parameter", endpoint.Method, endpoint.Path, rawQuery, got)
			}
		}
	}
}

// TestTheIncomingRequestIsNotMutatedByTheQueryStage guards the copy in Wrap,
// the way T9's test guards the path stage's. A middleware that rewrote the
// caller's own URL would change what every earlier stage sees after it
// returns — the response-time stamp and the access log among them.
func TestTheIncomingRequestIsNotMutatedByTheQueryStage(t *testing.T) {
	folder, err := httpapi.NewQueryFolder(surface.V1(), declared)
	if err != nil {
		t.Fatalf("building the query folder: %v", err)
	}

	handler := folder.Wrap(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "limit=1" {
			t.Errorf("the routed request's query = %q, want %q", r.URL.RawQuery, "limit=1")
		}
	}))

	request := httptest.NewRequest(http.MethodGet, "/Items/Filters?LIMIT=1", nil)
	handler.ServeHTTP(httptest.NewRecorder(), request)

	if request.URL.RawQuery != "LIMIT=1" {
		t.Errorf("the caller's query = %q, want it untouched at %q", request.URL.RawQuery, "LIMIT=1")
	}
}

// TestADeclarationForARouteTheTableDoesNotHaveIsRefused is the constructor's
// load-bearing refusal, from outside the package because that is where a
// declaration is written.
func TestADeclarationForARouteTheTableDoesNotHaveIsRefused(t *testing.T) {
	rows := []struct {
		name  string
		route httpapi.Route
	}{
		{"a path nothing serves", httpapi.Route{Method: http.MethodGet, Path: "/Nope"}},
		{"a method that path does not have", httpapi.Route{Method: http.MethodDelete, Path: "/System/Ping"}},
		{"the path in the wrong case", httpapi.Route{Method: http.MethodGet, Path: "/system/ping"}},
		{"a pattern spelled as one request that matches it", httpapi.Route{Method: http.MethodGet, Path: "/Items/AbC"}},
		{"a parameter under a different name", httpapi.Route{Method: http.MethodGet, Path: "/Items/{id}"}},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, err := httpapi.NewQueryFolder(surface.V1(), httpapi.QuerySpellings{row.route: {"limit"}})
			if err == nil {
				t.Fatalf("NewQueryFolder accepted a declaration for %s, which the table does not have", row.route)
			}
			if !strings.Contains(err.Error(), "the route table does not have") {
				t.Errorf("NewQueryFolder error = %q, want it to say the table does not have the route", err)
			}
		})
	}
}
