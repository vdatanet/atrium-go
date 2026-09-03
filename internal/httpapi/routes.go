package httpapi

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// Handlers is what a server serves: one field per handler a feature has.
//
// It is a struct rather than a list of arguments so that a feature adding its
// handlers does not change the signature every caller of Routes has already
// written.
//
// Every field is required to be non-nil by the L0 check's own guard
// (registration_test.go): Routes registers what this value contains, so a field
// filled in cmd/atrium's wiring and left empty by the check would register
// routes nothing walks.
type Handlers struct {
	// System answers the four /System routes of feature 001.
	System *SystemHandler

	// Users answers five of feature 002's seven rows: the login, the two
	// readings of a user object, the public listing and the configuration
	// write.
	Users *UsersHandler

	// Sessions answers the other two: the session listing and the capabilities
	// declaration.
	Sessions *SessionsHandler
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
			registration{operationGetSystemInfo, handlers.System.Info()},
			registration{operationGetPingSystem, ping},
			registration{operationPostPingSystem, ping},
		)
	}

	// Feature 002's seven rows arrive together, and that is a decision rather
	// than a convenience. Both halves of the L0 check derive *implemented*
	// rather than reading a list — a feature the server serves any row of must
	// serve every row of it — so the first of these registrations makes all
	// seven required at once (002 plan 8.4, 002 tasks T17). Registering them
	// one at a time is a sequence of builds that cannot go green.
	if handlers.Users != nil {
		registrations = append(registrations,
			registration{operationGetPublicUsers, handlers.Users.PublicUsers()},
			registration{operationAuthenticateUserByName, handlers.Users.AuthenticateByName()},
			registration{operationGetCurrentUser, handlers.Users.CurrentUser()},
			registration{operationGetUserByID, handlers.Users.UserByID()},
			registration{operationUpdateUserConfiguration, handlers.Users.UpdateConfiguration()},
		)
	}
	if handlers.Sessions != nil {
		registrations = append(registrations,
			registration{operationPostFullCapabilities, handlers.Sessions.PostFullCapabilities()},
			registration{operationGetSessions, handlers.Sessions.Sessions()},
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

	// The authenticated superset of the row above. Two operations, two paths,
	// one body built from the other (systeminfo.go).
	operationGetSystemInfo = "GetSystemInfo"

	// The two rows of /System/Ping. They are two operations on one path, which
	// is why the table is indexed by operation here and not by path: a lookup
	// by path would have to choose between them, and the method each is
	// registered on is exactly what distinguishes them.
	operationGetPingSystem  = "GetPingSystem"
	operationPostPingSystem = "PostPingSystem"
)

// The seven rows of feature 002.
//
// /Users/Configuration and /Users/{userId} are one path each in the table and
// two patterns chi has to tell apart, and it tells them apart by their methods
// alone: a GET whose path is literally /Users/Configuration matches the
// parametrised row, with `Configuration` as the identifier
// [measurement: github.com/go-chi/chi/v5 v5.3.2, Go 1.27.0, 2026-09-03]. That
// is a reading of the router rather than a measurement of the reference, and it
// is recorded at the registration because this is where the two rows meet.
const (
	// The login screen's listing, which reads no credential at all (spec 3.4).
	operationGetPublicUsers = "GetPublicUsers"

	// The project's first L3 row (spec 3.3).
	operationAuthenticateUserByName = "AuthenticateUserByName"

	// The two readings of one user object, built by one filler (spec 3.5,
	// spec 3.7, plan 6.6).
	operationGetCurrentUser = "GetCurrentUser"
	operationGetUserByID    = "GetUserById"

	// The configuration write (spec 3.6). Its `userId` query parameter is
	// declared by the reference and ignored here, which is register row U-14
	// rather than an omission — so it is deliberately absent from
	// V1QuerySpellings.
	operationUpdateUserConfiguration = "UpdateUserConfiguration"

	// The two /Sessions rows (spec 3.8).
	operationPostFullCapabilities = "PostFullCapabilities"
	operationGetSessions          = "GetSessions"
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
