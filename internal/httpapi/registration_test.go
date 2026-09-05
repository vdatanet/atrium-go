package httpapi_test

import (
	"fmt"
	"maps"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// This file is the **registration** half of the L0 check of plan 8.5: the
// router exposes exactly the surface.yaml rows whose owning feature is
// implemented, and nothing outside surface.yaml at all (spec 3.6, AC-11,
// Principle VI).
//
// Its twin is the **reachability** half in conformance/routes_test.go, which
// asks the same question of a running binary over the wire. plan 8.5 keeps
// both because each covers the other's blind spot: this half sees what the
// router was *told*, and misses a route made unreachable by a stage above it;
// the other sees what a client actually *gets*, and cannot enumerate a route
// nobody asked for.
//
// # Why this half is here and not in conformance/
//
// chi.Walk needs the router, the router is built from internal/httpapi, and
// conformance/ may not import anything under internal/ (architecture 3,
// enforced by tools/check_conformance_imports). The same split the two L1
// sweeps arrived at, for the same reason.
//
// # What "implemented" means, and why it is not a list
//
// A list of implemented features written down here would be correct until 002
// lands and then quietly wrong, and the check would report exactly nothing
// about the rows it had stopped knowing about. So it is derived, from the one
// thing that cannot go stale: **a feature the router serves any row of must
// serve every row of it.** A feature with no row served is a feature this
// build does not implement, which is not a claim about the roadmap — it is a
// reading of the router.
//
// That definition has one blind spot and it is worth naming: a feature that is
// implemented and serves *none* of its rows passes. There is no wording under
// which such a build implements the feature, and no signal available here that
// would say otherwise. What it does catch is the failure that actually
// happens: three of a feature's four rows registered, and the fourth forgotten.

// route is one registration, as chi.Walk reports one.
type route struct {
	method  string
	pattern string
}

func (r route) String() string { return r.method + " " + r.pattern }

// checkRegistration is the L0 rule, as a function over a table and a set of
// registrations, so that it can be run against a router that is deliberately
// wrong. A check that has only ever been run against a correct router has
// proved nothing.
func checkRegistration(table *surface.Table, registered []route) []string {
	var findings []string

	// A router that serves nothing satisfies the completeness rule below
	// vacuously — there is no served feature to be incomplete — so it is
	// refused here rather than passing quietly.
	if len(registered) == 0 {
		return []string{"the router serves no routes at all, which no rule below can fail on"}
	}

	registeredSet := make(map[route]bool, len(registered))
	features := map[string]bool{}
	for _, r := range registered {
		registeredSet[r] = true

		endpoint, ok := table.Lookup(r.method, r.pattern)
		if !ok {
			findings = append(findings, fmt.Sprintf(
				"the router serves %s and surface.yaml has no such row, so this server has a route v1 does not (Principle VI)", r))
			continue
		}
		features[endpoint.Feature] = true
	}

	for _, feature := range slices.Sorted(maps.Keys(features)) {
		for _, endpoint := range table.ForFeature(feature) {
			wanted := route{method: endpoint.Method, pattern: endpoint.Path}
			if !registeredSet[wanted] {
				findings = append(findings, fmt.Sprintf(
					"feature %s is implemented — the router serves rows of it — and %s (%s) is not registered",
					feature, wanted, endpoint.Operation))
			}
		}
	}

	slices.Sort(findings)
	return findings
}

// TestTheRouterServesExactlyTheImplementedRowsOfTheSurfaceDocument is the check
// itself, run against the pipeline this server is really built from.
//
// The router walked is Pipeline.Router(), not a bare chi.Router the test
// assembled: the pipeline is what cmd/atrium serves, and a check run against a
// router the test built would be checking the test's wiring.
func TestTheRouterServesExactlyTheImplementedRowsOfTheSurfaceDocument(t *testing.T) {
	t.Parallel()

	table := surface.V1()
	registered := walkRouter(t, buildRouter(t, table, everyHandler(t)))

	if findings := checkRegistration(table, registered); len(findings) != 0 {
		t.Errorf("the router does not serve exactly the implemented rows of surface.yaml:\n%s",
			strings.Join(findings, "\n"))
	}
}

// TestTheRegistrationCheckIsRunWithEveryHandlerAServerCanBeBuiltWith closes the
// gap the test above cannot see on its own.
//
// Routes registers what the Handlers value it is given contains, and this file
// builds that value itself. A feature that adds a field to Handlers and fills
// it in cmd/atrium's wiring would therefore register a route this check never
// walked — and if that route had no row, nothing would say so. So the struct
// is walked by reflection and every field is required to be set, which makes
// adding a field to Handlers fail this test until everyHandler below fills it.
func TestTheRegistrationCheckIsRunWithEveryHandlerAServerCanBeBuiltWith(t *testing.T) {
	t.Parallel()

	handlers := reflect.ValueOf(everyHandler(t))
	for i := range handlers.NumField() {
		field := handlers.Type().Field(i)
		value := handlers.Field(i)

		switch value.Kind() {
		case reflect.Pointer, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func:
			if value.IsNil() {
				t.Errorf("Handlers.%s is not set, so no route it would register is walked by the L0 check",
					field.Name)
			}
		default:
			t.Errorf("Handlers.%s is a %s, which this test cannot tell set from unset — "+
				"decide what an unfilled one means before adding it", field.Name, value.Kind())
		}
	}
}

// TestARouteWithNoRowFailsTheRegistrationCheck is half of the failure proof.
// A check that has never failed has proved nothing.
func TestARouteWithNoRowFailsTheRegistrationCheck(t *testing.T) {
	t.Parallel()

	table := surface.V1()
	registered := append(walkRouter(t, buildRouter(t, table, everyHandler(t))),
		route{method: http.MethodGet, pattern: "/System/Diagnostics"})

	findings := checkRegistration(table, registered)
	if len(findings) != 1 {
		t.Fatalf("a route with no row produced %d findings, want 1:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], "/System/Diagnostics") {
		t.Errorf("the finding is %q and does not name the route that has no row", findings[0])
	}
}

// TestARowOfAnImplementedFeatureThatIsNotServedFailsTheRegistrationCheck is the
// other half, and it is the one that makes the derived definition of
// "implemented" worth having: the feature is implemented *because* the other
// three rows are served, and the fourth is missing.
func TestARowOfAnImplementedFeatureThatIsNotServedFailsTheRegistrationCheck(t *testing.T) {
	t.Parallel()

	table := surface.V1()
	registered := walkRouter(t, buildRouter(t, table, everyHandler(t)))
	if len(registered) < 2 {
		t.Fatalf("the router serves %d routes; this test needs one it can remove and still have a served feature", len(registered))
	}

	dropped := registered[0]
	findings := checkRegistration(table, registered[1:])
	if len(findings) != 1 {
		t.Fatalf("dropping %s produced %d findings, want 1:\n%s",
			dropped, len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], dropped.pattern) {
		t.Errorf("the finding is %q and does not name the row that is not served", findings[0])
	}
}

// TestARouterThatServesNothingFailsTheRegistrationCheck exercises the guard
// that would otherwise be code no case reaches.
//
// "Every row of every served feature is registered" is true of a router with no
// routes at all, and a check that reported a server serving nothing as correct
// would be the most expensive kind of green.
func TestARouterThatServesNothingFailsTheRegistrationCheck(t *testing.T) {
	t.Parallel()

	if findings := checkRegistration(surface.V1(), nil); len(findings) != 1 {
		t.Fatalf("an empty router produced %d findings, want 1:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
}

// --- no new rows: 003's half of this check -------------------------------------

// theElevenRowsOf001And002 is what this router registered before 003, written
// out as a literal.
//
// 003 registers **no route at all** — plan §3 says *"internal/httpapi is
// untouched, and that is the sentence a reader should check first"*, and plan
// §10 refuses `POST /Library/Refresh` on the ground that a route added to make a
// test possible is a delta added to make a test possible. A feature that adds
// nothing has to be able to prove it, and the check above cannot: it derives
// what is expected from surface.yaml, so a route added **together with a row**
// satisfies it by construction.
//
// So this is the literal the reviewer of such a change has to edit. It is the
// registration half of the pair; conformance/library_configuration_test.go holds
// the reachability half, with its own literal, read through different code — the
// same trade plan §8.5 makes for surface.yaml itself.
//
// That the derived check is silent about it was measured rather than argued:
// registering a POST /Library/Refresh and declaring it in both copies of the
// surface document leaves TestTheRouterServesExactlyTheImplementedRowsOfThe
// SurfaceDocument green and turns this one red
// `[measurement: 003 T18, 18 mutations, 2026-09-05]`.
var theElevenRowsOf001And002 = []route{
	{method: http.MethodGet, pattern: "/System/Info"},
	{method: http.MethodGet, pattern: "/System/Info/Public"},
	{method: http.MethodGet, pattern: "/System/Ping"},
	{method: http.MethodPost, pattern: "/System/Ping"},
	{method: http.MethodGet, pattern: "/Sessions"},
	{method: http.MethodPost, pattern: "/Sessions/Capabilities/Full"},
	{method: http.MethodPost, pattern: "/Users/AuthenticateByName"},
	{method: http.MethodGet, pattern: "/Users/Me"},
	{method: http.MethodGet, pattern: "/Users/Public"},
	{method: http.MethodPost, pattern: "/Users/Configuration"},
	{method: http.MethodGet, pattern: "/Users/{userId}"},
}

// checkNoNewRows compares what the router registered against that literal, in
// both directions, and is a function for checkRegistration's reason: a check
// that has only ever seen a correct router has proved nothing.
func checkNoNewRows(registered, allowed []route) []string {
	inAllowed := map[route]bool{}
	for _, r := range allowed {
		inAllowed[r] = true
	}
	inRegistered := map[route]bool{}
	for _, r := range registered {
		inRegistered[r] = true
	}

	var findings []string
	for _, r := range registered {
		if !inAllowed[r] {
			findings = append(findings, fmt.Sprintf(
				"the router registers %s, which is not one of the eleven rows 001 and 002 registered", r))
		}
	}
	for _, r := range allowed {
		if !inRegistered[r] {
			findings = append(findings, fmt.Sprintf(
				"%s is one of the eleven rows 001 and 002 registered and the router does not register it", r))
		}
	}
	slices.Sort(findings)
	return findings
}

// TestTheRouterRegistersExactlyTheElevenRowsOf001And002 is 003's own assertion
// about this package: that it did not touch it.
func TestTheRouterRegistersExactlyTheElevenRowsOf001And002(t *testing.T) {
	t.Parallel()

	registered := walkRouter(t, buildRouter(t, surface.V1(), everyHandler(t)))
	if findings := checkNoNewRows(registered, theElevenRowsOf001And002); len(findings) != 0 {
		t.Errorf("the router does not register exactly the eleven rows of 001 and 002:\n%s",
			strings.Join(findings, "\n"))
	}
}

// TestATwelfthRouteFailsTheNoNewRowsCheck is half of the failure proof, and it
// is the half the derived check above cannot make: this route has a row in
// surface.yaml, so checkRegistration passes on it.
func TestATwelfthRouteFailsTheNoNewRowsCheck(t *testing.T) {
	t.Parallel()

	grown := append(slices.Clone(theElevenRowsOf001And002),
		route{method: http.MethodGet, pattern: "/Items"})

	findings := checkNoNewRows(grown, theElevenRowsOf001And002)
	if len(findings) != 1 {
		t.Fatalf("a twelfth route produced %d findings, want 1:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], "/Items") {
		t.Errorf("the finding is %q and does not name the route that was added", findings[0])
	}
}

// TestARouteThatWentAwayFailsTheNoNewRowsCheck is the other half. Without it a
// router serving nothing satisfies "no new rows".
func TestARouteThatWentAwayFailsTheNoNewRowsCheck(t *testing.T) {
	t.Parallel()

	dropped := theElevenRowsOf001And002[0]
	findings := checkNoNewRows(theElevenRowsOf001And002[1:], theElevenRowsOf001And002)
	if len(findings) != 1 {
		t.Fatalf("a missing route produced %d findings, want 1:\n%s",
			len(findings), strings.Join(findings, "\n"))
	}
	if !strings.Contains(findings[0], dropped.pattern) {
		t.Errorf("the finding is %q and does not name %s, the route that went away", findings[0], dropped)
	}
}

// everyHandler is one of each handler a server can be built with — the value
// cmd/atrium's wiring passes to Routes, with fakes where a real dependency
// would need a store.
//
// The fakes are irrelevant to this check: what is registered depends on which
// fields are non-nil, not on what any handler answers. 002's two handlers
// refuse to be built without their ports, so they are built over a real store
// on a temporary directory rather than over a stand-in — which costs a
// migration per call and buys the same thing a fake would.
func everyHandler(t *testing.T) httpapi.Handlers {
	t.Helper()

	store := openStore(t)
	clock := &settableClock{at: aTestInstant}

	return httpapi.Handlers{
		System: newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
			Installations: &fakeInstallations{installation: freshInstallation},
		}),
		Users:    newUsersHandler(t, store, clock),
		Sessions: newSessionsHandler(t, store, clock),
	}
}

// buildRouter assembles the real pipeline and hands back the router inside it.
func buildRouter(t *testing.T, table *surface.Table, handlers httpapi.Handlers) chi.Routes {
	t.Helper()

	routes, err := httpapi.Routes(table, handlers)
	if err != nil {
		t.Fatalf("building the routes callback: %v", err)
	}
	pipeline, err := httpapi.NewPipeline(table, httpapi.V1QuerySpellings(), routes)
	if err != nil {
		t.Fatalf("building the pipeline: %v", err)
	}
	return pipeline.Router()
}

// walkRouter enumerates every method and pattern the router carries.
//
// chi.Walk reports the pattern as it was registered, including the
// partial-segment ones a route table can spell
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]. What
// it does not report is NotFound and MethodNotAllowed, which are not routes:
// those are the refusals, and refusing is not serving.
func walkRouter(t *testing.T, router chi.Routes) []route {
	t.Helper()

	var registered []route
	err := chi.Walk(router, func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered = append(registered, route{method: method, pattern: pattern})
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	slices.SortFunc(registered, func(a, b route) int { return strings.Compare(a.String(), b.String()) })
	return registered
}
