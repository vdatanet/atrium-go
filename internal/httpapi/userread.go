package httpapi

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// userIDParameter is the route's own spelling of its path parameter.
//
// It reaches the wire in exactly one place — the key of the validation
// refusal's errors map, which names the parameter the binder could not fill
// (behaviours 1.11, spec 3.7). The reference keys it on the parameter's
// **declared** spelling and never on anything the client sent
// [source: Jellyfin.Api/Controllers/UserController.cs:127-131 @ v10.11.11], so
// it is transcribed here rather than derived from the request, and it is the
// same string chi matches the segment on.
const userIDParameter = "userId"

// userNotFoundMessage is the sixteen bytes of spec 3.7's 404, quotes included
// once it is encoded.
//
// It is a constant because the reference sends the same body to an
// administrator and to a non-administrator
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]: there is
// no caller-dependent half to build, so there is no branch that could grow
// one.
const userNotFoundMessage = "User not found"

// CurrentUser answers GET /Users/Me (spec 3.7, AC-12).
//
// The caller's own object of spec 3.5, in full — configuration and policy
// included — built by the one filler of plan 6.6 that every route returning a
// user object calls.
//
// # The account is read again rather than carried out of the authenticator
//
// The authenticator answers a Caller: an identifier, a session and a decoded
// policy (auth.go). That is what a route needs to decide *whether* to serve a
// request and it is not the object this route returns — the body carries a
// username, a credential fact and a configuration the caller never asked for.
// Reading the account here is one indexed lookup and it keeps the body a
// description of the stored row, which is what makes /Users/Me byte-comparable
// with the same account read through /Users/{userId} and through
// /Users/Public.
//
// # An account the authenticator resolved and this read cannot find is a 500
//
// The reference answers that state **400 with no body**, from the same route
// [source: Jellyfin.Api/Controllers/UserController.cs:604-616 @ v10.11.11], and
// it is unreachable here: this server admits nobody whose account it did not
// just read the policy off (authenticate.go), and v1 serves no route that
// deletes an account, so the two reads are separated by nothing that can act.
// What would reach it is a store that answered two questions inconsistently,
// which is a server that is broken rather than a request that is wrong — so it
// takes the shape reserved for that (plan 7's last row) rather than a refusal
// nothing in this feature can produce. It is a difference on a state no
// request can enter, stated here rather than left for a differential run to
// raise.
func (h *UsersHandler) CurrentUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := admitted(w, r, h.authenticator)
		if !ok {
			return
		}

		account, found, err := h.accounts.UserByID(r.Context(), caller.UserID)
		if err != nil {
			WriteInternalServerError(w)
			return
		}
		if !found {
			WriteInternalServerError(w)
			return
		}
		h.writeUserObject(w, r, account)
	}
}

// UserByID answers GET /Users/{userId} (spec 3.7, AC-7).
//
// # This route refuses no authenticated caller, and that is the whole of it
//
// Any caller carrying a usable token is answered 200 with the named user's
// whole object: a non-administrator naming another non-administrator, a
// restricted non-administrator naming an **administrator**, an administrator
// naming anybody, and a user naming themselves are one answer, and the bytes
// do not depend on who asked
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01].
//
// That is not the shape this project believed. spec 3.7 stated a 403 for a
// non-administrator reading anybody else, with no provenance, from the day 002
// was written until the matrix was measured on 2026-09-01 and found no refusal
// anywhere in it. The decision to replicate is behaviours 3.22, class B: the
// disclosure /Users/Public already makes is the same disclosure reached by a
// second road, and refusing on one road while publishing on the other is the
// inconsistency rather than the protection.
//
// **The successor mistake is a handler that answers 200 with a redacted body**,
// which every assertion about a status passes. There is nothing here to redact
// with — the object is built by the same filler for every caller, and the
// caller is read for admission and then not consulted — and the assertion that
// holds it that way is a byte comparison
// (TestARestrictedNonAdministratorReadsAnAdministratorsWholeObject).
//
// # Three shapes on one path segment, in this order
//
//  1. **No credential is the empty 401**, and it comes first: the reference's
//     authorization filter runs ahead of the model binder, which was measured
//     on another route where a caller who may not act meets the policy's
//     refusal for a segment that is not an identifier at all
//     (009 spec 3.8, 2026-09-01). What is read from that measurement is the
//     *order*; this route's own ordering has never been asked of a running
//     reference, and the register at T23 is owed the row.
//  2. **A segment that is not an identifier is the validation 400**, keyed on
//     the parameter's declared spelling and quoting the value back
//     (behaviours 1.11's fourth shape is not this one — this is problem
//     details). Note that the value reaches the wire through internal/wire, so
//     behaviours 1.16's escape pass applies to it: the apostrophes around the
//     quoted value travel as ' and the reference's do too.
//  3. **A well-formed identifier belonging to nobody is the 404**, carrying the
//     JSON-encoded bare string "User not found" — behaviours 1.11's fourth
//     error shape rather than the problem details every other handler-raised
//     404 in this project answers — and **the same body to an administrator and
//     to a non-administrator**. The reference conceals which identifiers exist
//     from nobody, which is the same decision as the paragraph above.
//
// The all-zero identifier is a fourth answer and is neither of the two 400s
// above: see emptyIdentifier.
func (h *UsersHandler) UserByID() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := admitted(w, r, h.authenticator); !ok {
			return
		}

		presented := chi.URLParam(r, userIDParameter)
		id, wellFormed := canonicalIdentifier(presented)
		if !wellFormed {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				userIDParameter: {invalidValueMessage(presented)},
			})
			return
		}
		if id == emptyIdentifier {
			WriteControllerRefusal(w, http.StatusBadRequest)
			return
		}

		account, found, err := h.accounts.UserByID(r.Context(), id)
		if err != nil {
			WriteInternalServerError(w)
			return
		}
		if !found {
			WriteJSONMessage(w, http.StatusNotFound, userNotFoundMessage)
			return
		}
		h.writeUserObject(w, r, account)
	}
}

// writeUserObject is the tail both routes share: build spec 3.5's object for
// one account and send it.
//
// It is shared because the two routes are measured to answer the *same bytes*
// for the same account — the administrator's own reading of themselves and a
// stranger's reading of them are byte-identical (spec 3.7) — and one tail is
// how that becomes a property of the code rather than a thing two handlers
// keep agreeing on by hand. It is plan 6.6's argument one level up from the
// filler.
func (h *UsersHandler) writeUserObject(w http.ResponseWriter, r *http.Request, account ports.User) {
	object, err := userObject(r.Context(), h.accounts, h.installationID, account)
	if err != nil {
		WriteInternalServerError(w)
		return
	}
	if err := wire.Write(w, http.StatusOK, object, NegotiateProfile(r)); err != nil {
		WriteInternalServerError(w)
	}
}
