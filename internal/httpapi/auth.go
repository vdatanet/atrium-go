package httpapi

import (
	"net/http"

	"github.com/vdatanet/atrium-go/internal/users"
)

// Access is what an authenticated route learns about the credential a request
// presents.
//
// It is an enumeration rather than a boolean because it is going to grow: the
// reference answers an authenticated request that lacks a route's permission
// with 403 rather than 401, and clients branch on the difference — they
// re-authenticate on a 401 and show an error on a 403, so a permission refusal
// answered 401 loops a client through a login it can never complete
// (002 spec 3.1). ~~That third value is deliberately **not declared here**. 001
// routes nothing that can reach it, behaviours 1.11 gives the shape a body
// this feature has no measurement for, and a constant no caller can produce
// and no test can exercise is exactly the plausible-looking stub Principle VI
// refuses. 002 adds it, with the shape, when it writes both.~~
//
// **002 T10 added it, and with a caller that can produce it**: a live token
// whose user has since been disabled. behaviours 1.11 has the shape now — the
// empty policy 403, no body and no content type, measured at 009 T2 — and
// 002 plan 7 is the table that maps each value to it.
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

	// AccessForbidden is a credential this server recognises, held by somebody
	// this server will not serve: 002 plan 7's row for a live token whose user
	// was disabled after it was issued. It answers 403 in behaviours 1.11's
	// *policy* shape — empty, and with no content type at all, which is a
	// different shape from the controller's 403 on the same status.
	//
	// The distinction from AccessUnauthenticated is what clients branch on.
	// They re-authenticate on a 401 and stop on a 403, so a disabled account
	// answered 401 loops a client through a login it can never complete, with
	// the user's password correct every time (002 spec 3.1, measured
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]).
	//
	// Its position in this block carries no meaning beyond keeping
	// AccessUnauthenticated the zero: nothing serialises an Access, no request
	// or response ever carries one, and 002 plan 7 is what maps a value to a
	// status. What does matter is that every switch over this type is
	// exhaustive — a value a handler does not recognise is answered 500 rather
	// than admitted or refused by a fall-through, which is what made adding
	// this one a change to 001's handler as well as to this list.
	AccessForbidden
)

// Caller is who a request authenticated as, carried out of the authenticator
// with the access it was granted.
//
// # Why it travels with the access rather than through a second method
//
// A second interface answering "who is this" would read 002 spec 3.1's five
// mechanisms a second time, and two reads of one credential can disagree — a
// token revoked between them, or a second reader that never learned about
// X-Emby-Authorization. One question, one answer (002 plan 5).
//
// # Why the policy is here rather than fetched by the handler
//
// GET /Sessions branches on IsAdministrator and the delivery routes 008 owns
// branch on EnableMediaPlayback. A handler that had to fetch a policy is a
// handler that could forget to, and one that fetched it would be reading the
// account a second time inside a request that already read it. The value is a
// copy and it is the domain type rather than a map, so a flag that does not
// exist does not compile.
//
// internal/sessions has a Caller of its own and the two are deliberately not
// one type: that package may import neither this one nor internal/users, so the
// handler reduces this to sessions.Caller in one visible line at the call site
// (002 plan 5, amended at T9).
type Caller struct {
	// UserID is the account the *token* was issued to, which is not always the
	// account the session names. A session is keyed on (Client, DeviceId) and
	// names whoever authenticated there last, while a token is keyed on
	// (user, device), so two people sharing one client on one device hold two
	// live tokens against one session row (002 plan 6.5). Resolving this off
	// the session would hand the request to whoever logged in most recently,
	// on somebody else's account, with no error anywhere.
	UserID string

	// SessionID is the session the presented token opened. It is what
	// POST /Sessions/Capabilities/Full writes to and what /Sessions reads the
	// caller's own row by.
	SessionID string

	// Policy is the account's stored policy, decoded onto the reference's
	// defaults with InvalidLoginAttemptCount overlaid from its own column
	// (users.PolicyOf) — never onto Go's zero value, which would answer a
	// permissive default for every flag the stored document happens not to
	// carry.
	Policy users.Policy
}

// Authentication is what the port answers: an access, and the caller it was
// granted to.
//
// # The zero value is the invariant 001 built on, and widening the return did
// not move it
//
// 001 plan 6.10 relies on AccessUnauthenticated being the zero Access so that a
// nil Authenticator — and any future failure to wire one — admits nobody.
// Authentication{} still means *unauthenticated with no caller*: the safe
// direction is what a caller gets for free, including a caller that ignored an
// error, because every error path here returns this value beside its error.
//
// # Caller is a pointer, and that is the type carrying the rule
//
// It is nil unless Access is AccessGranted. A struct value would have made
// "there is no caller" indistinguishable from "the caller with the empty
// identifier and a zero policy" — which is a caller whose UserID matches no
// account and whose every permission flag is false, and which a handler would
// read without noticing. The pointer makes the refused case unreadable rather
// than quietly wrong: a handler that reads through it on a refusal fails on the
// first request instead of answering somebody else's data.
type Authentication struct {
	// Access is what the credential entitles the request to.
	Access Access

	// Caller is who the request is, and it is nil unless Access is
	// AccessGranted.
	Caller *Caller
}

// Authenticator decides what the credential a request carries entitles it to.
//
// ~~**001 declares it and fills it with nothing.**~~ **002 T10 fills it, with
// TokenAuthenticator, and widened the return to carry the caller.** 001's
// paragraph is kept because it is the record of what a nil Authenticator means
// and why that is not a stub: five mechanisms — three header spellings and two
// query names — with a measured precedence between them, tokens that name a
// session, and users that hold permissions (002 spec 3.1). A server built
// without one passes no Authenticator and every credential is therefore
// unrecognised, which is the true answer for a server that has issued no token
// and knows no user, and it is what makes the 401 of 001 spec 3.2 reachable and
// testable with nothing wired.
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
	//
	// An implementation returns Authentication{} beside any error it returns,
	// so that a caller which read the value and ignored the error refuses
	// rather than admits.
	Authenticate(r *http.Request) (Authentication, error)
}
