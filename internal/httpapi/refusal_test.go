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

// The three shapes 002 T11 added, asserted over a real connection.
//
// Every assertion below goes through send() rather than through
// httptest.ResponseRecorder, and that is not a style preference. 001 T11
// measured that three of the four things behaviours 1.11 states about a
// refusal shape are invisible to a recorder: it synthesises no
// Content-Length, it never sniffs a content type the way net/http does on the
// wire, and it cannot show what a HEAD response leaves out. A recorder-based
// test of a shape whose definition is "text/plain with **no** charset" would
// pass against a handler that sends the charset.

// TestTheControllerRefusalSendsTheMeasuredTwentyFiveBytes is spec 3.3's shape:
// the fixed body, the bare media type, the declared length.
//
// The charset is the assertion with teeth. Go sniffs
// `text/plain; charset=utf-8` for a body like this one, so a writer that let
// net/http decide would put a parameter on the wire that the reference does
// not send [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
// 2026-08-26] — on every refusal of this feature at once.
func TestTheControllerRefusalSendsTheMeasuredTwentyFiveBytes(t *testing.T) {
	// The four statuses spec 3.3 and spec 3.8 send this shape on. The bytes do
	// not vary with the status, which is the property that makes the status
	// the whole of the difference between the login refusals.
	for _, status := range []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound} {
		t.Run(fmt.Sprintf("%d", status), func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				httpapi.WriteControllerRefusal(w, status)
			})

			response := send(t, handler, "GET", "/Users/AuthenticateByName")

			wantStatus := fmt.Sprintf("HTTP/1.1 %d ", status)
			if !strings.HasPrefix(response.statusLine, wantStatus) {
				t.Errorf("status line = %q, want it to begin %q", response.statusLine, wantStatus)
			}
			if response.body != "Error processing request." {
				t.Errorf("body = %q, want %q", response.body, "Error processing request.")
			}
			if len(response.body) != 25 {
				t.Errorf("body is %d bytes, want the measured 25", len(response.body))
			}
			if got := response.values("Content-Type"); len(got) != 1 || got[0] != "text/plain" {
				t.Errorf("Content-Type = %v, want exactly [\"text/plain\"] — behaviours 1.11 measures no charset parameter on this shape, and net/http sniffs one", got)
			}
			if got := response.values("Content-Length"); len(got) != 1 || got[0] != "25" {
				t.Errorf("Content-Length = %v, want exactly [\"25\"]", got)
			}
		})
	}
}

// TestTheControllerRefusalDeclaresItsLengthToAHeadRequest asserts this shape
// on the request that shows the least of it: the body is discarded and the
// headers are all that arrive.
//
// It was written expecting to be the request that tells an explicit
// Content-Length from an inherited one, the way 001's doubled-slash test is.
// **It is not, and the measurement is worth more than the expectation was.**
// 001's finding is about a *body-less* response: net/http adds no length to
// one answering a HEAD. This shape writes a body, and net/http computes the
// length from it and keeps the header when it drops the body for a HEAD
// [measurement: net/http, Go 1.27.0, 2026-09-03] — so a mutation removing the
// Set in WriteControllerRefusal survives this test and every other one here.
// That is recorded at the writer rather than papered over.
//
// What this does assert is real and is not inherited: the bare content type
// survives on a HEAD too. net/http sniffs `text/plain; charset=utf-8` from the
// body it is about to discard, so the charset behaviours 1.11 does not measure
// arrives here as readily as on a GET — a writer that let net/http decide
// fails this test, and it does.
func TestTheControllerRefusalDeclaresItsLengthToAHeadRequest(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteControllerRefusal(w, http.StatusForbidden)
	})

	response := send(t, handler, "HEAD", "/Users/AuthenticateByName")

	if response.body != "" {
		t.Errorf("HEAD body = %q, want it empty — the server discards it", response.body)
	}
	if got := response.values("Content-Length"); len(got) != 1 || got[0] != "25" {
		t.Errorf("HEAD Content-Length = %v, want exactly [\"25\"] — the length of the body a GET would have carried", got)
	}
	if got := response.values("Content-Type"); len(got) != 1 || got[0] != "text/plain" {
		t.Errorf("HEAD Content-Type = %v, want exactly [\"text/plain\"]", got)
	}
}

// TestTheJSONMessageRefusalSendsAQuotedString is spec 3.7's 404 for a userId
// nobody has: sixteen bytes including the quotes, under the JSON content type
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01].
//
// The quotes are the assertion. A JSON-encoded bare string is a document, and
// a writer that sent `User not found` unquoted would answer fourteen bytes a
// client's decoder refuses — which is invisible to any test comparing the
// decoded value.
func TestTheJSONMessageRefusalSendsAQuotedString(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSONMessage(w, http.StatusNotFound, "User not found")
	})

	response := send(t, handler, "GET", "/Users/00000000000000000000000000000000")

	if !strings.HasPrefix(response.statusLine, "HTTP/1.1 404 ") {
		t.Errorf("status line = %q, want a 404", response.statusLine)
	}
	if response.body != `"User not found"` {
		t.Errorf("body = %q, want %q — the quotes are part of it", response.body, `"User not found"`)
	}
	if len(response.body) != 16 {
		t.Errorf("body is %d bytes, want the measured 16", len(response.body))
	}
	if got := response.values("Content-Type"); len(got) != 1 || got[0] != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %v, want exactly [\"application/json; charset=utf-8\"]", got)
	}
}

// TestTheJSONMessageRefusalEscapesItsMessage is why the body goes through
// internal/wire rather than being assembled here.
//
// behaviours 1.16 applies to every string in a JSON document, and the messages
// this shape carries are not all constants: the measured one on another route
// interpolates an item's name. A hand-rolled `"` + message + `"` would send an
// apostrophe raw where the reference sends ', and the difference only
// appears once somebody names a playlist with one.
func TestTheJSONMessageRefusalEscapesItsMessage(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteJSONMessage(w, http.StatusNotFound, "Bob's & Alice's does not have an image of type Box")
	})

	response := send(t, handler, "GET", "/Items/abc/Images/Box")

	// behaviours 1.16's seven characters, upper-cased and four hex digits wide:
	// the apostrophe and the ampersand both go out escaped, which is what
	// wire's pass does and what a `"` + message + `"` would not.
	want := `"Bob\u0027s \u0026 Alice\u0027s does not have an image of type Box"`
	if response.body != want {
		t.Errorf("body = %s, want %s — behaviours 1.16's escape pass applies to a bare string too", response.body, want)
	}
}

// TestThePolicyRefusalCarriesNoContentTypeAtAll is behaviours 1.11's second
// 403, measured at 009 T2: no content type, no body
// [probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31].
//
// It is asserted with 001's own assertEmptyRefusal, which is the point: this
// shape is not a new one, it is the empty shape on a status 001 never sent.
// The Content-Length: 0 that assertion also requires is *not* measured for
// this row — see WriteForbidden's own note, which decides it deliberately and
// hands the register a row.
func TestThePolicyRefusalCarriesNoContentTypeAtAll(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httpapi.WriteForbidden(w)
	})

	for _, method := range []string{"GET", "HEAD"} {
		response := send(t, handler, method, "/Items/abc")
		assertEmptyRefusal(t, response, http.StatusForbidden, method+" refused by policy")
		if response.has("Allow") {
			t.Errorf("%s: Allow = %v, want the header absent on a 403", method, response.values("Allow"))
		}
	}
}

// TestThePolicyRefusalDropsHeadersAStageBeforeItLeftBehind extends 001's own
// test to the new empty shape, for the reason it gives: the header map is
// shared by the whole chain, and a stage that already wrote a content type
// would otherwise decide the shape of a refusal taken after it. The
// authenticator refuses *after* the pipeline has run, which is exactly that
// position.
func TestThePolicyRefusalDropsHeadersAStageBeforeItLeftBehind(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("WWW-Authenticate", `Basic realm="Jellyfin"`)
		httpapi.WriteForbidden(w)
	})

	response := send(t, handler, "GET", "/System/Info")
	assertEmptyRefusal(t, response, http.StatusForbidden, "a policy refusal after a stage that set a content type")
	if response.has("WWW-Authenticate") {
		t.Errorf("WWW-Authenticate = %v, want the header absent", response.values("WWW-Authenticate"))
	}
}

// TestTheTwoBodiedRefusalsDropAChallengeLeftBehind is the same rule for the
// two shapes that do carry a body. Both are sent on 401 by a measured route,
// so a stage above them that set a challenge has to be overruled here too — a
// body is not a licence to inherit the rest of the header map.
func TestTheTwoBodiedRefusalsDropAChallengeLeftBehind(t *testing.T) {
	rows := []struct {
		name  string
		write func(http.ResponseWriter)
	}{
		{"the 25 bytes", func(w http.ResponseWriter) { httpapi.WriteControllerRefusal(w, http.StatusUnauthorized) }},
		{"the bare string", func(w http.ResponseWriter) {
			httpapi.WriteJSONMessage(w, http.StatusUnauthorized, "Unauthorized access")
		}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("WWW-Authenticate", `Basic realm="Jellyfin"`)
				row.write(w)
			})

			response := send(t, handler, "DELETE", "/Items/abc")
			if response.has("WWW-Authenticate") {
				t.Errorf("WWW-Authenticate = %v, want the header absent — behaviours 1.11 measures none on any 401 shape", response.values("WWW-Authenticate"))
			}
			if got := response.values("Content-Type"); len(got) != 1 || got[0] == "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %v, want exactly one field line, and this shape's own rather than the one an earlier stage left", got)
			}
		})
	}
}

// shapeOf renders one refusal writer over a real connection and returns what a
// differential run would compare: the status line, every header field line
// sorted, and the body.
//
// Date is dropped because it moves with the clock and nothing else is: the
// point of the fingerprint is that two shapes differ for a reason somebody
// chose, and a timestamp would make every pair differ for free.
func shapeOf(t *testing.T, write func(http.ResponseWriter)) (statusLine, headers, body string) {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { write(w) })
	response := send(t, handler, "GET", "/System/Info")

	var lines []string
	for name, values := range response.header {
		if name == "Date" {
			continue
		}
		for _, value := range values {
			lines = append(lines, name+": "+value)
		}
	}
	sort.Strings(lines)
	return response.statusLine, strings.Join(lines, "\r\n"), response.body
}

// refusalWriters is every shape this file writes, named as a differential
// report would name it.
//
// A writer taking a status is rendered on 403 rather than on a status of its
// own, so that the comparison below is about the *shape* and not about the
// number in the status line. That is the whole of behaviours 1.11's finding
// about this status: a 403 is two shapes, and the reference sends both.
func refusalWriters() []struct {
	name  string
	write func(http.ResponseWriter)
} {
	return []struct {
		name  string
		write func(http.ResponseWriter)
	}{
		{"WriteNotFound", httpapi.WriteNotFound},
		{"WriteMethodNotAllowed", func(w http.ResponseWriter) { httpapi.WriteMethodNotAllowed(w, "GET, POST") }},
		{"WriteUnauthorized", httpapi.WriteUnauthorized},
		{"WriteInternalServerError", httpapi.WriteInternalServerError},
		{"WriteControllerRefusal", func(w http.ResponseWriter) { httpapi.WriteControllerRefusal(w, http.StatusForbidden) }},
		{"WriteJSONMessage", func(w http.ResponseWriter) {
			httpapi.WriteJSONMessage(w, http.StatusForbidden, "User not found")
		}},
		{"WriteForbidden", httpapi.WriteForbidden},
	}
}

// TestNoTwoRefusalWritersProduceTheSameResponse is the check that catches the
// realistic mistake, and it is one assertion rather than a property of seven
// separate tests.
//
// The mistake is a copy-paste: a new shape written from the shape beside it,
// keeping a header it should not have or losing one it should. Every
// individual test above would still pass — each asserts what its own shape
// carries, and none asserts what another shape carries instead. This one fails
// the moment two writers agree, which is a year before a differential run
// would have said so.
//
// The comparison is over the whole response because that is what a client and
// a differential report see. Four of the seven agree on every header they send
// and are told apart by their status alone, which is not a weakness of the
// check but the measurement: behaviours 1.11's empty shape *is* one shape, and
// 001's argument for writing it in one place is that it stays that way.
func TestNoTwoRefusalWritersProduceTheSameResponse(t *testing.T) {
	type fingerprint struct{ statusLine, headers, body string }

	seen := make(map[fingerprint]string)
	for _, writer := range refusalWriters() {
		statusLine, headers, body := shapeOf(t, writer.write)
		print := fingerprint{statusLine, headers, body}
		if earlier, clash := seen[print]; clash {
			t.Errorf("%s and %s send the same response — %q, %q, body %q. Two refusal shapes that cannot be told apart on the wire are one shape, and behaviours 1.11 measures them as different", earlier, writer.name, statusLine, headers, body)
			continue
		}
		seen[print] = writer.name
	}
}

// TestTheThreeShapesOnOneStatusAreToldApartWithoutTheStatus is the sharper
// half, and it is the one behaviours 1.11 forces.
//
// A `403` is two shapes there, split by how the refusal was expressed rather
// than by which layer expressed it — and this feature sends both: the login
// route refuses a disabled account with the twenty-five bytes, and the
// authenticator refuses a token whose account was disabled afterwards with
// nothing at all. So the status cannot be what tells them apart, and this
// asserts the headers and the body alone.
//
// The JSON message is in here on the same status for the same reason: it is
// measured on 401 and on 404 on two different routes, so nothing stops a third
// route sending it on 403, and a shape that is only distinct because nobody
// has sent it yet is not distinct.
func TestTheThreeShapesOnOneStatusAreToldApartWithoutTheStatus(t *testing.T) {
	type shape struct{ headers, body string }

	rows := []struct {
		name  string
		write func(http.ResponseWriter)
	}{
		{"WriteControllerRefusal(403)", func(w http.ResponseWriter) { httpapi.WriteControllerRefusal(w, http.StatusForbidden) }},
		{"WriteJSONMessage(403)", func(w http.ResponseWriter) {
			httpapi.WriteJSONMessage(w, http.StatusForbidden, "User not found")
		}},
		{"WriteForbidden()", httpapi.WriteForbidden},
	}

	seen := make(map[shape]string)
	for _, row := range rows {
		statusLine, headers, body := shapeOf(t, row.write)
		if !strings.HasPrefix(statusLine, "HTTP/1.1 403 ") {
			t.Fatalf("%s: status line = %q, want a 403 — this test compares three shapes on one status and proves nothing if they are not on one", row.name, statusLine)
		}
		print := shape{headers, body}
		if earlier, clash := seen[print]; clash {
			t.Errorf("%s and %s send the same headers and body on 403 — %q, body %q. behaviours 1.11 measures a 403 as two shapes and one status cannot carry two spellings of one", earlier, row.name, headers, body)
			continue
		}
		seen[print] = row.name
	}
}
