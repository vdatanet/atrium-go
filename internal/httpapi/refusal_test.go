package httpapi_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// newRefusals builds the stage over the real v1 table, which is the only table
// the server ever runs on.
func newRefusals(t *testing.T) *httpapi.Refusals {
	t.Helper()
	refusals, err := httpapi.NewRefusals(surface.V1())
	if err != nil {
		t.Fatalf("building the refusal stage over the v1 table: %v", err)
	}
	return refusals
}

// pingRouter registers 001's four rows on a chi router — the two methods of
// /System/Ping among them — and installs this project's refusal handlers.
//
// The handler bodies are deliberately not the real ones. What is under test is
// what happens when the router cannot reach a handler at all.
func pingRouter(t *testing.T, refusals *httpapi.Refusals) chi.Router {
	t.Helper()
	router := chi.NewRouter()
	if refusals != nil {
		router.NotFound(refusals.NotFoundHandler())
		router.MethodNotAllowed(refusals.MethodNotAllowedHandler())
	}
	served := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	for _, endpoint := range surface.V1().ForFeature("001") {
		router.Method(endpoint.Method, endpoint.Path, served)
	}
	return router
}

// rawResponse is one response read off the wire, headers unparsed by anything
// that might add a default.
//
// Principle VIII is why this exists: the refusal shapes of behaviours 1.11 are
// defined by headers that are *absent*, and an absence is invisible to a test
// that asks a parsed response for a value and gets an empty string. The
// header block is read as bytes and every field line is kept.
type rawResponse struct {
	statusLine string
	header     textproto.MIMEHeader
	body       string
}

// has reports whether the response carries a field line with this name at all.
func (r rawResponse) has(name string) bool {
	_, present := r.header[textproto.CanonicalMIMEHeaderKey(name)]
	return present
}

// values returns every field line sent under this name, in order.
func (r rawResponse) values(name string) []string {
	return r.header[textproto.CanonicalMIMEHeaderKey(name)]
}

// send runs one request against a handler over a loopback listener and reads
// the response back as bytes.
//
// It writes the request line itself rather than going through http.Client,
// because two of the methods under test — an unknown token, and a HEAD the
// client would rewrite the response of — do not survive a client that is
// trying to be helpful.
//
// headerLines are extra field lines, written verbatim after the Host, for a
// test that has to send a credential (systeminfo_authenticated_test.go).
func send(t *testing.T, handler http.Handler, method, target string, headerLines ...string) rawResponse {
	t.Helper()

	server := httptest.NewServer(handler)
	defer server.Close()

	address := strings.TrimPrefix(server.URL, "http://")
	connection, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("dialling the test server: %v", err)
	}
	defer connection.Close()

	extra := ""
	for _, line := range headerLines {
		extra += line + "\r\n"
	}
	request := fmt.Sprintf("%s %s HTTP/1.1\r\nHost: %s\r\n%sConnection: close\r\n\r\n", method, target, address, extra)
	if _, err := io.WriteString(connection, request); err != nil {
		t.Fatalf("writing %s %s: %v", method, target, err)
	}

	reader := bufio.NewReader(connection)
	textReader := textproto.NewReader(reader)
	statusLine, err := textReader.ReadLine()
	if err != nil {
		t.Fatalf("reading the status line of %s %s: %v", method, target, err)
	}
	header, err := textReader.ReadMIMEHeader()
	if err != nil {
		t.Fatalf("reading the headers of %s %s: %v", method, target, err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the body of %s %s: %v", method, target, err)
	}
	return rawResponse{statusLine: statusLine, header: header, body: string(body)}
}

// assertEmptyRefusal asserts everything the three shapes of behaviours 1.11
// agree on: the status, an empty body, a declared length of zero, and no
// content type at all.
func assertEmptyRefusal(t *testing.T, response rawResponse, status int, what string) {
	t.Helper()

	wantStatus := fmt.Sprintf("HTTP/1.1 %d ", status)
	if !strings.HasPrefix(response.statusLine, wantStatus) {
		t.Errorf("%s: status line = %q, want it to begin %q", what, response.statusLine, wantStatus)
	}
	if response.body != "" {
		t.Errorf("%s: body = %q, want it empty", what, response.body)
	}
	if got := response.values("Content-Length"); len(got) != 1 || got[0] != "0" {
		t.Errorf("%s: Content-Length = %v, want exactly [\"0\"]", what, got)
	}
	if response.has("Content-Type") {
		t.Errorf("%s: Content-Type = %v, want the header absent — behaviours 1.11 measures no content type on this shape", what, response.values("Content-Type"))
	}
}

// TestTheMeasuredCaseChiGetsWrong is the reason this task exists.
//
// /System/Ping carries GET and POST, and spec 3.6 requires Allow to name every
// method the *path* has. chi names one, and which one varies with the request
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-02] — so
// this is the assertion the router cannot satisfy on its own.
func TestTheMeasuredCaseChiGetsWrong(t *testing.T) {
	router := pingRouter(t, newRefusals(t))

	// The four methods of the measurement, and one token chi does not know.
	for _, method := range []string{"HEAD", "PUT", "OPTIONS", "DELETE", "PATCH", "FOO"} {
		t.Run(method, func(t *testing.T) {
			response := send(t, router, method, "/System/Ping")

			what := method + " /System/Ping"
			assertEmptyRefusal(t, response, http.StatusMethodNotAllowed, what)

			got := response.values("Allow")
			if len(got) != 1 || got[0] != "GET, POST" {
				t.Errorf("%s: Allow = %v, want exactly [\"GET, POST\"] — every method the path has, alphabetically", what, got)
			}
		})
	}
}

// TestChiCannotWriteTheAllowHeaderTheReferenceSends records, against chi
// itself, what the stage above exists to replace.
//
// It is a declared inequality rather than a check of this project's code: chi
// is wrong here, this test says exactly how, and the day chi stops being wrong
// this test fails and somebody rereads plan 1 and plan 10 instead of finding
// dead code. Without it, the only evidence that Refusals is needed at all is a
// sentence in a plan.
//
// Two of chi's three faults are asserted here, because both are deterministic
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]:
//
//	PUT /System/Ping -> 405  Allow: GET\r\nAllow: POST   (two field lines)
//	FOO /System/Ping -> 405  no Allow field line at all
//
// The third is not assertable without flakiness and is recorded instead: chi
// builds the list by ranging over a Go map (tree.go's endpoints), so the order
// of the two lines is map-iteration order. Over 200 identical requests it was
// `GET` then `POST` 171 times and `POST` then `GET` 29. behaviours 1.11
// measured the reference sending one comma-joined field line in alphabetical
// order, and Principle VII wants an order derived from a stable input, so this
// is two divergences in one header.
func TestChiCannotWriteTheAllowHeaderTheReferenceSends(t *testing.T) {
	// No refusal handlers: this is chi answering on its own.
	router := pingRouter(t, nil)

	t.Run("one field line per method rather than one comma-joined line", func(t *testing.T) {
		for _, method := range []string{"HEAD", "PUT", "OPTIONS", "DELETE"} {
			response := send(t, router, method, "/System/Ping")

			if !strings.HasPrefix(response.statusLine, "HTTP/1.1 405 ") {
				t.Fatalf("%s /System/Ping: status line = %q, want a 405", method, response.statusLine)
			}
			got := response.values("Allow")
			if len(got) != 2 {
				t.Errorf("%s /System/Ping: chi sent %d Allow field lines (%v), want the measured 2. If chi now sends one, plan 1's measurement is stale and plan 10's argument needs rereading", method, len(got), got)
			}
			if len(got) == 1 && got[0] == "GET, POST" {
				t.Errorf("%s /System/Ping: chi now writes %q itself, which is what this project computes; plan 10 chose to compute it and that choice is now unnecessary", method, got[0])
			}
		}
	})

	t.Run("no Allow at all for a method chi does not know", func(t *testing.T) {
		response := send(t, router, "FOO", "/System/Ping")

		if !strings.HasPrefix(response.statusLine, "HTTP/1.1 405 ") {
			t.Fatalf("FOO /System/Ping: status line = %q, want a 405", response.statusLine)
		}
		if got := response.values("Allow"); len(got) != 0 {
			t.Errorf("FOO /System/Ping: chi's own Allow = %v, want the measured absence — spec 3.6 requires an Allow on every 405, and chi sends none when the method is not one of the nine it knows", got)
		}
	})
}

// TestAllowIsAPropertyOfThePathNotOfTheRoute walks the whole table.
//
// Three paths in v1 carry three methods and are owned by two features at once,
// which is where a header computed from the row a request nearly matched, or
// from the feature that owns most of a path, goes wrong.
func TestAllowIsAPropertyOfThePathNotOfTheRoute(t *testing.T) {
	refusals := newRefusals(t)
	table := surface.V1()

	for _, path := range table.Paths() {
		methods := table.Methods(path)
		sorted := append([]string(nil), methods...)
		sort.Strings(sorted)
		want := strings.Join(sorted, ", ")

		// A path with parameters is asked about the way a request spells it,
		// not the way the document does.
		request := strings.NewReplacer("{", "", "}", "").Replace(path)

		got, ok := refusals.Allow(request)
		if !ok {
			t.Errorf("Allow(%q) reported the path unknown; the table declares it with %v", request, methods)
			continue
		}
		if got != want {
			t.Errorf("Allow(%q) = %q, want %q", request, got, want)
		}
	}
}

// TestTheThreeMethodPathsAdvertiseAllThree is T8's warning made into an
// assertion: the methods of a path are not the methods of the feature that
// owns most of it. 005 owns a GET on each of these and 009 owns the rest.
func TestTheThreeMethodPathsAdvertiseAllThree(t *testing.T) {
	refusals := newRefusals(t)

	rows := []struct {
		sent string
		want string
	}{
		{"/System/Ping", "GET, POST"},
		{"/Items/abc", "DELETE, GET, POST"},
		{"/Playlists/abc/Items", "DELETE, GET, POST"},
	}
	for _, row := range rows {
		got, ok := refusals.Allow(row.sent)
		if !ok {
			t.Errorf("Allow(%q) reported the path unknown", row.sent)
			continue
		}
		if got != row.want {
			t.Errorf("Allow(%q) = %q, want %q", row.sent, got, row.want)
		}
	}
}

// TestAPathTheTableDoesNotDescribeIsUnknownRatherThanEmpty separates the two
// answers Allow can give. There is no row in the table with a path and no
// method, so false must mean "no such path".
func TestAPathTheTableDoesNotDescribeIsUnknownRatherThanEmpty(t *testing.T) {
	refusals := newRefusals(t)

	for _, path := range []string{"/Nowhere", "/System", "/System/Ping/Extra", "*", ""} {
		if got, ok := refusals.Allow(path); ok {
			t.Errorf("Allow(%q) = %q, true; want the path reported unknown", path, got)
		}
	}
}

// TestAnUnroutablePathIsAnEmpty404 covers the router's NotFound, which is the
// other half of behaviours 1.11's empty pair.
func TestAnUnroutablePathIsAnEmpty404(t *testing.T) {
	router := pingRouter(t, newRefusals(t))

	response := send(t, router, "GET", "/Nowhere")
	assertEmptyRefusal(t, response, http.StatusNotFound, "GET /Nowhere")
	if response.has("Allow") {
		t.Errorf("GET /Nowhere: Allow = %v, want the header absent on a 404", response.values("Allow"))
	}
}

// TestAnUnknownMethodOnAnUnroutablePathIsA404 is the case chi reaches through
// its 405 branch rather than its 404 one, because the method is not one of the
// nine it knows. spec 3.6 keys the 404 on the path, so the path decides.
func TestAnUnknownMethodOnAnUnroutablePathIsA404(t *testing.T) {
	router := pingRouter(t, newRefusals(t))

	response := send(t, router, "FOO", "/Nowhere")
	assertEmptyRefusal(t, response, http.StatusNotFound, "FOO /Nowhere")
	if response.has("Allow") {
		t.Errorf("FOO /Nowhere: Allow = %v, want the header absent — there are no methods to name", response.values("Allow"))
	}
}

// TestTheUnauthorizedRefusalCarriesNoChallenge is the 401 shape, and the
// absent WWW-Authenticate is the whole point of it. RFC 9110 asks for the
// header; the reference sends none
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
func TestTheUnauthorizedRefusalCarriesNoChallenge(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteUnauthorized(w)
	})

	response := send(t, handler, "GET", "/System/Info")
	assertEmptyRefusal(t, response, http.StatusUnauthorized, "GET /System/Info with no token")
	if response.has("WWW-Authenticate") {
		t.Errorf("a 401 carried WWW-Authenticate = %v, want the header absent", response.values("WWW-Authenticate"))
	}
}

// TestARefusalDropsHeadersAStageBeforeItLeftBehind is the reason refuse
// deletes rather than merely declines to set.
//
// The header map is shared by the whole chain, so a stage that has already
// written a content type — or a framework-minded one that added a challenge —
// decides the shape of a refusal taken after it unless the refusal says
// otherwise.
func TestARefusalDropsHeadersAStageBeforeItLeftBehind(t *testing.T) {
	rows := []struct {
		name  string
		write func(http.ResponseWriter)
		want  int
	}{
		{"404", httpapi.WriteNotFound, http.StatusNotFound},
		{"401", httpapi.WriteUnauthorized, http.StatusUnauthorized},
		{"405", func(w http.ResponseWriter) { httpapi.WriteMethodNotAllowed(w, "GET, POST") }, http.StatusMethodNotAllowed},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.Header().Set("WWW-Authenticate", `Basic realm="Jellyfin"`)
				row.write(w)
			})

			response := send(t, handler, "GET", "/System/Info")
			assertEmptyRefusal(t, response, row.want, "a refusal after a stage that set a content type")
			if response.has("WWW-Authenticate") {
				t.Errorf("WWW-Authenticate = %v, want the header absent", response.values("WWW-Authenticate"))
			}
		})
	}
}

// TestAnEarlierAllowIsReplacedRatherThanAppended is why the header is Set.
// chi's own 405 handler adds one field line per method, and an Add here would
// ship both answers.
func TestAnEarlierAllowIsReplacedRatherThanAppended(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Allow", "POST")
		httpapi.WriteMethodNotAllowed(w, "GET, POST")
	})

	response := send(t, handler, "PUT", "/System/Ping")
	got := response.values("Allow")
	if len(got) != 1 || got[0] != "GET, POST" {
		t.Errorf("Allow = %v, want exactly [\"GET, POST\"]", got)
	}
}

// TestTheDoubledSlash404IsTheSameShapeAsTheRouters checks the call site plan
// 6.1 folded into this package: canonicalisation's own refusal and the
// router's are one shape, so they cannot drift.
//
// # Why HEAD is in here, and why the test is worth nothing without it
//
// Written with GET alone this test passed against a canonicalisation stage
// that had gone back to writing its own bare w.WriteHeader(404) — because
// net/http adds `Content-Length: 0` to a body-less GET response and there was
// nothing left to see. It does **not** add one to a body-less HEAD response
// [measurement: net/http, Go 1.27.0, 2026-09-03], so HEAD is the request on
// which the shared shape is the only thing declaring the length, and it is the
// only request that can tell the two spellings apart.
func TestTheDoubledSlash404IsTheSameShapeAsTheRouters(t *testing.T) {
	folder, err := httpapi.NewPathFolder(surface.V1())
	if err != nil {
		t.Fatalf("building the fold map over the v1 table: %v", err)
	}
	pipeline := folder.Wrap(pingRouter(t, newRefusals(t)))

	for _, method := range []string{"GET", "HEAD"} {
		fromCanonicalisation := send(t, pipeline, method, "/System/Ping//")
		assertEmptyRefusal(t, fromCanonicalisation, http.StatusNotFound, method+" /System/Ping//")

		fromRouter := send(t, pipeline, method, "/Nowhere")
		assertEmptyRefusal(t, fromRouter, http.StatusNotFound, method+" /Nowhere")

		if fromCanonicalisation.statusLine != fromRouter.statusLine {
			t.Errorf("%s: status lines differ: %q from canonicalisation, %q from the router", method, fromCanonicalisation.statusLine, fromRouter.statusLine)
		}
	}
}
