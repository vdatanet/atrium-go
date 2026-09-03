// Package httpapi is the request pipeline: the readiness gate, the
// response-time stamp, the Server header, path and query canonicalisation,
// routing, the refusal shapes, and the handlers of every implemented feature.
//
// # How a stage in this package is shaped
//
// A stage is a value, constructed once from whatever it reads, whose Wrap
// method is the middleware:
//
//	folder, err := httpapi.NewPathFolder(surface.V1())
//	if err != nil {
//	    return err
//	}
//	router.Use(folder.Wrap)
//
// Two properties come out of that shape and both are deliberate.
//
// The work a stage can do once is done once, at construction. Path
// canonicalisation folds every route's literal segments into a map when it is
// built, not per request. A stage that reads a table therefore has a
// constructor that takes it rather than a package-level default, which is also
// what lets a test build one over a fixture table.
//
// A stage that cannot be built returns an error rather than panicking. The
// inputs are the embedded route table and the process's own configuration,
// so a failure here is a failure to start — plan 7 — and the entry layer is
// where a failure to start is reported. A middleware constructor that panicked
// would move that decision into a package that has no way to report it.
//
// A method value is a func(http.Handler) http.Handler, so folder.Wrap is
// exactly what chi's Use and a hand-assembled chain both want, with no adapter.
//
// # Order is contract
//
// plan 6.7 fixes the order of the stages, as amended at T13:
//
//	response-time stamp → Server header → readiness gate → path canonicalisation
//	  → query canonicalisation → routing → refusal shapes → handler → wire
//
// Canonicalisation precedes routing because it rewrites what the router
// matches: the router only ever sees a canonical path, which is why chi's own
// case sensitivity is never exercised.
//
// NewPipeline is where that chain is assembled, and it is the only place: a
// stage wrapped anywhere else is a second order, and there is no second order.
// pipeline_test.go asserts three properties only this order has, each against
// a chain with exactly one stage moved.
//
// The readiness gate is third rather than outermost, and that is a decision
// rather than an accident. A middleware that answers without calling the next
// handler is never reached by anything below it, so a gate above the stamp
// would answer spec 3.5's 503 carrying neither X-Response-Time-ms nor Server.
// The reference resolves it the same way — its response-time middleware near
// the outside of the main pipeline, its startup gate well inside it
// [source: Jellyfin.Server/Startup.cs:163,217 @ v10.11.11] — and it costs
// spec 3.5's "nothing is exempt" nothing, because neither stage above the gate
// reads a path, matches a route or refuses.
package httpapi
