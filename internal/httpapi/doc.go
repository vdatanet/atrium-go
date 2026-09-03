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
// plan 6.7 fixes the order of the stages:
//
//	readiness gate → response-time stamp → Server header → path canonicalisation
//	  → query canonicalisation → routing → refusal shapes → handler → wire
//
// Canonicalisation precedes routing because it rewrites what the router
// matches: the router only ever sees a canonical path, which is why chi's own
// case sensitivity is never exercised. Assembling the chain, and asserting the
// order with checks only the order can satisfy, is T14's.
package httpapi
