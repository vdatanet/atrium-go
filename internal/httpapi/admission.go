package httpapi

import "net/http"

// admitted asks an authenticator what a request's credential entitles it to,
// writes the refusal itself when the answer is not "carry on", and reports the
// caller to a handler that may carry on.
//
// # One home for the mapping, because a rule enforced twice is a rule no
// mutation of either half can reach
//
// 001 wrote this switch inside SystemHandler.admits, which was the only route
// that had one. 002 T14 is where the second, third and fourth routes needing it
// arrive (spec 3.7's two, then T15's and T16's), and a second copy of the
// mapping would be a rule with two homes: a change to one half is invisible to
// every test of the other, and it reads as defensive depth rather than as the
// duplicate it is. That is 002 T8's finding, applied before it could happen a
// second time — so SystemHandler.admits calls this and discards the caller it
// has no use for, and the four responses below are decided in one place.
//
// # What each answer is, and why the default is a 500
//
// 002 plan 7 is the table. AccessUnauthenticated is behaviours 1.11's empty
// 401 — no body, Content-Length: 0, no Content-Type and no WWW-Authenticate,
// three of which are invisible to a test that reads the status alone.
// AccessForbidden is the *policy* 403 — empty, and with no content type at all
// — which is a different shape from the controller's 403 on the same status
// (refusal.go). An Access this package does not recognise is a 500 rather than
// a fall-through, for internal/wire's reason: the two directions a
// fall-through could take are "admit everybody" and "refuse everybody", and
// both are wrong silently.
//
// # An error from the port is a 500 and never a 401
//
// A client answered 401 discards its credential and logs in again, which is
// the wrong thing to make every client in the house do because a database was
// briefly unreadable (002 plan 7's last row).
//
// # A nil authenticator admits nobody
//
// It is not a special case worth a branch: Authentication{} is
// AccessUnauthenticated, so a server built without one recognises no
// credential, which is the true answer for a server that has issued none. 001
// ships that way and 001 plan 6.10 relies on it.
func admitted(w http.ResponseWriter, r *http.Request, authenticator Authenticator) (*Caller, bool) {
	var authentication Authentication
	if authenticator != nil {
		decided, err := authenticator.Authenticate(r)
		if err != nil {
			WriteInternalServerError(w)
			return nil, false
		}
		authentication = decided
	}

	switch authentication.Access {
	case AccessGranted:
		// The caller is handed back exactly as the port answered it, nil
		// included. auth.go's Caller pointer exists so that reading a caller
		// off a refusal is a nil dereference rather than a silently empty
		// account, and substituting a zero Caller here would restore the bug
		// that pointer removes.
		return authentication.Caller, true
	case AccessForbidden:
		WriteForbidden(w)
		return nil, false
	case AccessUnauthenticated:
		WriteUnauthorized(w)
		return nil, false
	default:
		WriteInternalServerError(w)
		return nil, false
	}
}
