package httpapi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// Handlers is what a server serves: one field per feature that has handlers.
// 001 is the only one so far.
//
// It is a struct rather than a list of arguments so that a feature adding its
// handlers does not change the signature every caller of Routes has already
// written.
type Handlers struct {
	// System answers the four /System routes of feature 001.
	System *SystemHandler
}

// Routes builds the registration callback NewPipeline takes, over the route
// table.
//
// # Why registration goes through the table
//
// The pattern a handler is registered on is surface.yaml's spelling of it, read
// out of the table rather than typed here a second time. A row renamed in the
// document then fails to start, naming the operation, instead of leaving the
// router serving a pattern the surface file no longer contains — which is a
// state T20's registration check would eventually catch and which a start
// should never reach in the first place.
//
// It is also what makes the L0 check meaningful. The check asks that the router
// expose exactly the implemented rows of surface.yaml (Principle VI, plan 8.5),
// and a router built from the same table it is checked against would agree with
// it by construction on the *spelling*. What the table cannot supply, and what
// the check is therefore still worth running for, is which rows are served at
// all: this function names them one at a time, and a row nobody names is a row
// nothing answers.
//
// # Why it returns an error
//
// chi's routes callback cannot fail, so a missing row has to be discovered
// before the callback is built. plan 7 makes a table this server cannot serve
// a failure to start, in the same class as a table the folders cannot fold.
func Routes(table *surface.Table, handlers Handlers) (func(chi.Router), error) {
	type registration struct {
		operation string
		handler   http.Handler
	}

	var registrations []registration
	if handlers.System != nil {
		// Both spellings of /System/Ping take the same handler value. spec 3.3
		// gives them one request shape and one response, so two handlers would
		// be two things to keep identical (ping.go).
		ping := handlers.System.Ping()
		registrations = append(registrations,
			registration{operationPublicSystemInfo, handlers.System.PublicInfo()},
			registration{operationGetPingSystem, ping},
			registration{operationPostPingSystem, ping},
		)
	}

	// Resolved before the callback is returned, so that a row the document does
	// not have is an error the entry layer reports rather than a panic inside
	// chi at registration time.
	type route struct {
		method  string
		pattern string
		handler http.Handler
	}
	routes := make([]route, 0, len(registrations))
	for _, r := range registrations {
		endpoint, ok := endpointForOperation(table, r.operation)
		if !ok {
			return nil, fmt.Errorf("httpapi: the route table has no row for operation %s, so there is no pattern to register its handler on", r.operation)
		}
		routes = append(routes, route{endpoint.Method, endpoint.Path, r.handler})
	}

	return func(router chi.Router) {
		for _, r := range routes {
			router.Method(r.method, r.pattern, r.handler)
		}
	}, nil
}

// The operationIds of the rows this package registers. They are the names a
// probe, an allowlist entry and a request case all use, and they are the one
// spelling of a route that does not change when a path does.
const (
	operationPublicSystemInfo = "GetPublicSystemInfo"

	// The two rows of /System/Ping. They are two operations on one path, which
	// is why the table is indexed by operation here and not by path: a lookup
	// by path would have to choose between them, and the method each is
	// registered on is exactly what distinguishes them.
	operationGetPingSystem  = "GetPingSystem"
	operationPostPingSystem = "PostPingSystem"
)

// endpointForOperation finds one row by its operationId.
//
// The table indexes by method and path, which is the wrong key here: the point
// of naming the operation is that this file does not repeat the path. The scan
// is over 59 rows, once, at start.
func endpointForOperation(table *surface.Table, operation string) (surface.Endpoint, bool) {
	for _, endpoint := range table.Endpoints() {
		if endpoint.Operation == operation {
			return endpoint, true
		}
	}
	return surface.Endpoint{}, false
}
