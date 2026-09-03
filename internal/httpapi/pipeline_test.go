package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// standIns registers 001's four rows with a handler that answers 200 and
// nothing else.
//
// The bodies are deliberately not the real ones — T16-T18 write those. What is
// under test here is which stage answers, and a stand-in that answers 200 is
// the clearest possible "nothing above me refused".
func standIns(t *testing.T) func(chi.Router) {
	t.Helper()

	served := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return func(router chi.Router) {
		for _, endpoint := range surface.V1().ForFeature("001") {
			router.Method(endpoint.Method, endpoint.Path, served)
		}
	}
}

// newPipeline builds the pipeline the binary serves, over the real v1 table.
// It comes back shut, which is how it is built (plan 6.8).
func newPipeline(t *testing.T) *httpapi.Pipeline {
	t.Helper()

	pipeline, err := httpapi.NewPipeline(surface.V1(), httpapi.V1QuerySpellings(), standIns(t))
	if err != nil {
		t.Fatalf("assembling the pipeline over the v1 table: %v", err)
	}
	return pipeline
}

// readyPipeline is the same pipeline with the gate open, which is the state
// the server spends its life in.
func readyPipeline(t *testing.T) *httpapi.Pipeline {
	t.Helper()

	pipeline := newPipeline(t)
	pipeline.Gate().MarkReady()
	return pipeline
}

// The three assertions of T14's Verified by line follow. Each is chosen so
// that only plan 6.7's order satisfies it, and
// TestEachOrderAssertionFailsOnAPipelineAssembledInTheWrongOrder below builds
// the chain that gets each one wrong and asserts it answers differently. A
// check that has never failed has proved nothing.

// TestA503FromTheGateStillCarriesTheStampAndServer is the first.
//
// The gate answers without calling the next handler, so nothing below it ever
// runs on this request. Both headers can therefore only be on this response if
// the two stages that write them are *outside* the gate — which is what plan
// 6.7's amendment at T13 settled, against an order that had the gate
// outermost.
func TestA503FromTheGateStillCarriesTheStampAndServer(t *testing.T) {
	pipeline := newPipeline(t)

	for _, route := range routesOf001(t) {
		response := send(t, pipeline, route.Method, route.Path)
		what := route.Method + " " + route.Path + " while starting"
		assertStartingRefusal(t, response, "Jellyfin Server is loading. Please try again shortly.", what)
		assertBothHeaders(t, response, what)
	}
}

// TestA404FromCanonicalisationStillCarriesTheStampAndServer is the second.
//
// Two or more trailing slashes is the one refusal path canonicalisation writes
// itself (plan 6.1), and it is the only 404 in this feature that no router
// produces — so it is the row that says canonicalisation is inside the stamp
// rather than beside it. The gate is opened first, because a shut gate would
// answer this request before canonicalisation saw it and the test would then
// be assertion one again under another name.
func TestA404FromCanonicalisationStillCarriesTheStampAndServer(t *testing.T) {
	pipeline := readyPipeline(t)

	// GET and HEAD both, because a body-less GET cannot tell an explicitly
	// empty refusal from an inherited one: net/http adds Content-Length: 0 to
	// a body-less response but not to a body-less response to a HEAD
	// [measurement: net/http, Go 1.27.0, 2026-09-03].
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		response := send(t, pipeline, method, "/System/Ping//")
		what := method + " /System/Ping// once ready"
		assertEmptyRefusal(t, response, http.StatusNotFound, what)
		assertBothHeaders(t, response, what)
	}
}

// TestTheGateAnswersBeforeRoutingSoAnUnknownPathIs503WhileStarting is the
// third, and it is the one that pins the gate *above* routing rather than
// merely inside the stamp.
//
// spec 3.5 makes the 503 a property of the whole server rather than of an
// endpoint, so a request the router would refuse must be refused by the gate
// first. The assertion is a declared inequality rather than an equality: the
// same request answers 404 once the gate is open, so a pipeline that answered
// 503 to everything for ever would fail the second half.
func TestTheGateAnswersBeforeRoutingSoAnUnknownPathIs503WhileStarting(t *testing.T) {
	pipeline := newPipeline(t)

	const unknown = "/Nowhere"
	starting := send(t, pipeline, http.MethodGet, unknown)
	assertStartingRefusal(t, starting, "Jellyfin Server is loading. Please try again shortly.", "GET "+unknown+" while starting")
	assertBothHeaders(t, starting, "GET "+unknown+" while starting")

	pipeline.Gate().MarkReady()

	ready := send(t, pipeline, http.MethodGet, unknown)
	assertEmptyRefusal(t, ready, http.StatusNotFound, "GET "+unknown+" once ready")
}

// TestEachOrderAssertionFailsOnAPipelineAssembledInTheWrongOrder is the proof
// that the three above are about the order and not about the stages.
//
// Each chain here is the real pipeline with exactly one stage moved, and each
// answers the same request differently. Without this, "only the correct order
// satisfies them" would be a claim about code nobody has ever seen fail.
func TestEachOrderAssertionFailsOnAPipelineAssembledInTheWrongOrder(t *testing.T) {
	t.Run("the gate outside the stamp answers a 503 carrying neither header", func(t *testing.T) {
		gate := httpapi.NewReadinessGate()
		misordered := gate.Wrap(httpapi.NewResponseTimeStamp().Wrap(
			httpapi.NewServerHeader().Wrap(routerOf001(t))))

		response := send(t, misordered, http.MethodGet, "/System/Ping")
		if !strings.HasPrefix(response.statusLine, "HTTP/1.1 503 ") {
			t.Fatalf("status line = %q, want a 503 from the gate", response.statusLine)
		}
		if response.has(httpapi.ResponseTimeHeader) {
			t.Errorf("%s = %v on a 503 from a gate above the stamp, want it absent — assertion one cannot fail if this passes",
				httpapi.ResponseTimeHeader, response.values(httpapi.ResponseTimeHeader))
		}
		if response.has(httpapi.ServerHeaderName) {
			t.Errorf("%s = %v on a 503 from a gate above the stamp, want it absent",
				httpapi.ServerHeaderName, response.values(httpapi.ServerHeaderName))
		}
	})

	t.Run("canonicalisation outside the stamp answers a 404 carrying neither header", func(t *testing.T) {
		folder, err := httpapi.NewPathFolder(surface.V1())
		if err != nil {
			t.Fatalf("building the path folder: %v", err)
		}
		misordered := folder.Wrap(httpapi.NewResponseTimeStamp().Wrap(
			httpapi.NewServerHeader().Wrap(routerOf001(t))))

		response := send(t, misordered, http.MethodGet, "/System/Ping//")
		if !strings.HasPrefix(response.statusLine, "HTTP/1.1 404 ") {
			t.Fatalf("status line = %q, want the doubled-slash 404", response.statusLine)
		}
		if response.has(httpapi.ResponseTimeHeader) || response.has(httpapi.ServerHeaderName) {
			t.Errorf("a 404 from canonicalisation above the stamp carries %v and %v, want both absent — assertion two cannot fail if this passes",
				response.values(httpapi.ResponseTimeHeader), response.values(httpapi.ServerHeaderName))
		}
	})

	t.Run("a gate below routing answers 404 to an unknown path while starting", func(t *testing.T) {
		gate := httpapi.NewReadinessGate()
		refusals := newRefusals(t)
		router := chi.NewRouter()
		router.NotFound(refusals.NotFoundHandler())
		router.MethodNotAllowed(refusals.MethodNotAllowedHandler())
		served := gate.Wrap(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		for _, endpoint := range surface.V1().ForFeature("001") {
			router.Method(endpoint.Method, endpoint.Path, served)
		}
		misordered := httpapi.NewResponseTimeStamp().Wrap(httpapi.NewServerHeader().Wrap(router))

		response := send(t, misordered, http.MethodGet, "/Nowhere")
		if !strings.HasPrefix(response.statusLine, "HTTP/1.1 404 ") {
			t.Errorf("status line = %q, want the 404 a gate below routing cannot prevent — assertion three cannot fail if this passes", response.statusLine)
		}
	})
}

// routerOf001 is the four rows on a chi router with this project's refusal
// handlers, for the misordered chains above. It is pingRouter under a name
// that says what it is being used for here.
func routerOf001(t *testing.T) chi.Router {
	t.Helper()
	return pingRouter(t, newRefusals(t))
}

// TestTheAssembledPipelineFoldsBeforeItRoutes is the property architecture 4
// calls load-bearing for a real client: the router only ever sees a canonical
// path, so a request spelled the way a reference's own document spells it
// reaches the handler rather than a 404.
//
// The request carries every fold at once — lower case, one trailing slash —
// which is what makes it a test of the assembly rather than of the folder.
func TestTheAssembledPipelineFoldsBeforeItRoutes(t *testing.T) {
	pipeline := readyPipeline(t)

	response := send(t, pipeline, http.MethodGet, "/system/ping/")
	if !strings.HasPrefix(response.statusLine, "HTTP/1.1 200 ") {
		t.Errorf("GET /system/ping/: status line = %q, want a 200 — canonicalisation runs before routing", response.statusLine)
	}
	assertBothHeaders(t, response, "GET /system/ping/")
}

// TestTheAssembledPipelineRefusesAMethodThePathDoesNotHave asserts the refusal
// handlers reached the router: chi's own 405 names one arbitrary method of the
// path and this project's names every one
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03].
//
// It is here as well as in refusal_test.go because installing them is part of
// assembling the pipeline: a NewPipeline that built the stage and forgot to
// hand it to the router would pass every other test in this file.
func TestTheAssembledPipelineRefusesAMethodThePathDoesNotHave(t *testing.T) {
	pipeline := readyPipeline(t)

	response := send(t, pipeline, http.MethodPut, "/System/Ping")
	assertEmptyRefusal(t, response, http.StatusMethodNotAllowed, "PUT /System/Ping")
	if got := response.values("Allow"); len(got) != 1 || got[0] != "GET, POST" {
		t.Errorf("PUT /System/Ping: Allow = %v, want exactly [\"GET, POST\"]", got)
	}
}

// TestTheAssembledPipelineFoldsAQueryNameBeforeTheHandlerReadsIt is the stage
// no request the server actually serves can see.
//
// ~~V1QuerySpellings is empty — none of 001's four routes takes a query
// parameter (T10) — so removing query canonicalisation from the chain
// altogether breaks no other test in this repository.~~ **002 T16 declared the
// first real names, on GET /Sessions, and removing the stage now fails
// sessions_test.go too.** The declaration here stays this test's own: it is
// about a route of 001's, which still takes no parameter, so this remains the
// assertion that the stage is in the assembly independently of any feature's
// declarations.
func TestTheAssembledPipelineFoldsAQueryNameBeforeTheHandlerReadsIt(t *testing.T) {
	const route = "/System/Info/Public"
	spellings := httpapi.QuerySpellings{
		{Method: http.MethodGet, Path: route}: {"limit"},
	}

	seen := make(chan string, 1)
	pipeline, err := httpapi.NewPipeline(surface.V1(), spellings, func(router chi.Router) {
		router.Get(route, func(w http.ResponseWriter, r *http.Request) {
			seen <- r.URL.RawQuery
			w.WriteHeader(http.StatusOK)
		})
	})
	if err != nil {
		t.Fatalf("assembling the pipeline: %v", err)
	}
	pipeline.Gate().MarkReady()

	if response := send(t, pipeline, http.MethodGet, route+"?LIMIT=1"); !strings.HasPrefix(response.statusLine, "HTTP/1.1 200 ") {
		t.Fatalf("status line = %q, want a 200 from the handler", response.statusLine)
	}
	if got := <-seen; got != "limit=1" {
		t.Errorf("the handler read the query %q, want %q — query canonicalisation is not in the chain", got, "limit=1")
	}
}

// TestNegotiateProfileReadsEveryAcceptFieldLine is the one thing T7 could not
// do for the pipeline: a request may carry more than one Accept field line,
// and RFC 9110 5.3 makes those one header whose value is the lines joined with
// a comma.
//
// r.Header.Get answers only the first, so the second row here is the whole
// point: written with Get, a client that asks for camelCase on a second line
// is answered in PascalCase — behaviours 1.13's failure mode, an empty object
// out of the client's decoder.
func TestNegotiateProfileReadsEveryAcceptFieldLine(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		lines []string
		want  wire.Profile
	}{
		{"no Accept at all", nil, wire.ProfilePlain},
		{"one field line", []string{`application/json; profile="CamelCase"`}, wire.ProfileCamel},
		{"the profile on the second field line", []string{"text/plain", `application/json; profile="CamelCase"`}, wire.ProfileCamel},
		{"the profile on the first of two", []string{`application/json; profile="PascalCase"`, "text/plain"}, wire.ProfilePascal},
		{"quality across two field lines", []string{`application/json; profile="CamelCase";q=0.2`, `application/json; profile="PascalCase";q=0.8`}, wire.ProfilePascal},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
			for _, line := range testCase.lines {
				request.Header.Add("Accept", line)
			}
			if got := httpapi.NegotiateProfile(request); got != testCase.want {
				t.Errorf("NegotiateProfile over %v = %v, want %v", testCase.lines, got, testCase.want)
			}
		})
	}
}
