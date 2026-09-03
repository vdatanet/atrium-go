package httpapi

import "net/http"

// Access is what an authenticated route learns about the credential a request
// presents.
//
// It is an enumeration rather than a boolean because it is going to grow: the
// reference answers an authenticated request that lacks a route's permission
// with 403 rather than 401, and clients branch on the difference — they
// re-authenticate on a 401 and show an error on a 403, so a permission refusal
// answered 401 loops a client through a login it can never complete
// (002 spec 3.1). That third value is deliberately **not declared here**. 001
// routes nothing that can reach it, behaviours 1.11 gives the shape a body
// this feature has no measurement for, and a constant no caller can produce
// and no test can exercise is exactly the plausible-looking stub Principle VI
// refuses. 002 adds it, with the shape, when it writes both.
type Access int

const (
	// AccessUnauthenticated is a request that presented no credential, or one
	// that names nothing. spec 3.2 answers 401, in behaviours 1.11's empty
	// shape: no body, Content-Length: 0, and no WWW-Authenticate.
	//
	// It is the zero value on purpose. An Authenticator that answers nothing —
	// and, until 002, the absence of an Authenticator at all — admits nobody,
	// which is the only safe direction for a value that decides admission.
	AccessUnauthenticated Access = iota

	// AccessGranted is a credential that admits this request to this route.
	AccessGranted
)

// Authenticator decides what the credential a request carries entitles it to.
//
// **001 declares it and fills it with nothing.** Feature 002 owns
// authentication: five mechanisms — three header spellings and two query
// names — with a measured precedence between them, tokens that name a session,
// and users that hold permissions (002 spec 3.1). None of that exists yet, so
// a server built today passes no Authenticator and every credential is
// therefore unrecognised. That is not a stub standing in for the real thing:
// it is the true answer for a server that has issued no token and knows no
// user, and it is what makes the 401 of spec 3.2 reachable and testable now.
//
// # Why it takes the request
//
// Everything else the domain needs of a request reaches it through
// system.RequestFacts, which is the seam that keeps HTTP out of the domain
// (plan 5). This does not, and the reason is that the credential is *in* the
// HTTP: three header names and two query names, each read with its own
// grammar, with a measured order between them when two disagree. Reducing that
// to a value before this interface would mean implementing 002's reader here —
// which is the one thing this task must not do — and getting it wrong would
// silently unauthenticate a client that presents a form this reduction had not
// heard of.
//
// So the interface lives in the edge package, where HTTP belongs, rather than
// in internal/ports beside the store. The domain half of authentication —
// a token to a session, a session to a user, a user to a permission — is 002's
// to declare there, and it is not this interface.
type Authenticator interface {
	// Authenticate reports what the request's credential entitles it to. It
	// takes the whole request because the credential can be in either the
	// header or the query.
	//
	// An error is a failure to decide — an unreadable store, not an
	// unrecognised token — and a caller answers 500 rather than 401: a client
	// told 401 discards its credential and re-authenticates, which is the
	// wrong thing to make it do because a database was briefly unavailable.
	Authenticate(r *http.Request) (Access, error)
}
