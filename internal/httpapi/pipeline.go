package httpapi

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// Pipeline is every stage of plan 6.7, assembled in that order and in no
// other, together with the router the handlers of a feature register on.
//
// # Why the assembly is a value rather than a function
//
// The gate has to be reachable after the pipeline is built. It is shut when it
// is constructed and something has to open it once the start has finished
// (plan 6.8), and an operator withdraws the server through the same value
// while the process keeps running (spec 3.5). A function returning a bare
// http.Handler would leave the entry layer holding the gate separately and
// trusting that the handler it was given is the one that gate is in.
//
// # Order is contract
//
// plan 6.7, as amended at T13:
//
//	response-time stamp → Server header → readiness gate → path canonicalisation
//	  → query canonicalisation → routing → refusal shapes → handler → wire
//
// Written outermost first, which is also the order Wrap is applied in below.
// Three properties of this pipeline follow from the order alone and from
// nothing else, and pipeline_test.go asserts each of them against a chain that
// gets the order wrong:
//
//  1. a 503 from the gate carries X-Response-Time-ms and Server, because the
//     gate answers without calling the next handler and is therefore reached
//     by nothing below it — so it has to be *inside* the two stages that stamp
//     every response;
//  2. the 404 canonicalisation answers for a doubled trailing slash carries
//     them too, for the same reason and one stage further in;
//  3. an unknown path answers 503 rather than 404 while the server is
//     starting, because the gate is above routing. spec 3.5 is a property of
//     the server rather than of an endpoint, so a request the router would
//     refuse must be refused by the gate first.
type Pipeline struct {
	gate    *ReadinessGate
	router  chi.Router
	handler http.Handler
}

// NewPipeline assembles the pipeline over a route table and the query
// spellings its routes declare, and calls routes to register the handlers.
//
// routes may be nil, which is a server that routes nothing: every path the
// table names then answers the router's own refusal rather than a handler.
// That is what this feature serves until T16-T18 write the four handlers, and
// it is what a test that is about a stage rather than about a body wants.
//
// It fails the way its stages fail — the three that read the route table
// return an error rather than panicking, because a bad table is a failure to
// start and the entry layer is where that is reported (plan 7).
func NewPipeline(table *surface.Table, spellings QuerySpellings, routes func(chi.Router)) (*Pipeline, error) {
	paths, err := NewPathFolder(table)
	if err != nil {
		return nil, err
	}
	queries, err := NewQueryFolder(table, spellings)
	if err != nil {
		return nil, err
	}
	refusals, err := NewRefusals(table)
	if err != nil {
		return nil, err
	}

	router := chi.NewRouter()
	// Installed on the router rather than chained, because they are what the
	// router calls when it cannot reach a handler. Both compute their shape
	// from the same table the folders were built from (T11).
	router.NotFound(refusals.NotFoundHandler())
	router.MethodNotAllowed(refusals.MethodNotAllowedHandler())
	if routes != nil {
		routes(router)
	}

	gate := NewReadinessGate()

	// Applied innermost first, so this reads as plan 6.7's order backwards.
	// chi's own Use is deliberately not used for any of it: Use panics once a
	// route is registered, which would make the order depend on when routes
	// were added, and the order is the contract.
	handler := NewResponseTimeStamp().Wrap(
		NewServerHeader().Wrap(
			gate.Wrap(
				paths.Wrap(
					queries.Wrap(router)))))

	return &Pipeline{gate: gate, router: router, handler: handler}, nil
}

// ServeHTTP hands the request to the outermost stage.
func (p *Pipeline) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.handler.ServeHTTP(w, r)
}

// Gate is the readiness gate this pipeline refuses through.
//
// The entry layer opens it once the start has finished, and an operator
// withdraws the server through it without stopping the process (spec 3.5).
// Nothing else in the pipeline is reachable from outside it: the folders and
// the refusals are decided at construction and have nothing to say afterwards.
func (p *Pipeline) Gate() *ReadinessGate { return p.gate }

// Router is the router the handlers were registered on, for the L0 check that
// the server exposes exactly the implemented rows of surface.yaml and nothing
// else (Principle VI, plan 8.5). It is not a place to register a route after
// the fact — chi would accept one, and the pipeline would then be serving
// something its construction never saw.
func (p *Pipeline) Router() chi.Router { return p.router }

// NegotiateProfile is how a handler learns which content profile to write
// under: the outcome of negotiating this request's Accept header (spec 3.0.2,
// plan 6.3).
//
// # Why the join, and why not Header.Get
//
// A request may carry more than one Accept field line, and RFC 9110 5.3 makes
// those one header whose value is the field lines joined with a comma.
// r.Header.Get answers only the *first* — so a client sending
//
//	Accept: text/plain
//	Accept: application/json; profile="CamelCase"
//
// would be answered in PascalCase, which is behaviours 1.13's failure mode
// exactly: an empty object out of the client's decoder. Values returns every
// field line, and joining them is what wire.Negotiate is waiting for; plan 6.3
// says in as many words that the join is this layer's to do.
//
// # Why a function here rather than a stage in the pipeline
//
// It writes nothing, refuses nothing and answers nothing, so it is not a stage
// — it is the "handler → wire" step plan 6.7 already names. Two things follow
// from that and are the reason it is not a middleware carrying a Profile in
// the request context. behaviours 1.10 and plan 5 put the content type on
// whatever produced the body, and a Profile that arrived through a context
// would be ProfilePlain in any handler tested without the stage installed —
// silently, and with a correct-looking body. Taking the request means the
// answer cannot be wrong for want of a stage.
//
// A request that asks for nothing this server can write gets ProfilePlain,
// which is what wire.Negotiate lands every fallback on.
func NegotiateProfile(r *http.Request) wire.Profile {
	return wire.Negotiate(strings.Join(r.Header.Values("Accept"), ","))
}
