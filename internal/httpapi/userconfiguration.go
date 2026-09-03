package httpapi

import (
	"errors"
	"io"
	"net/http"

	"github.com/vdatanet/atrium-go/internal/users"
)

// configurationActionParameter is the reference's own name for this route's
// request body.
//
// It reaches the wire in exactly one place — the second key of the validation
// refusal, which names the action parameter the binder could not fill
// (behaviours 1.11) — and it is never anything the client sent, so it is
// transcribed rather than derived
// [source: Jellyfin.Api/Controllers/UserController.cs:492-494 @ v10.11.11].
// The login route's is `request` and this one's is `userConfig`; two routes,
// two names, and neither is guessable from the other.
const configurationActionParameter = "userConfig"

// UpdateConfiguration answers POST /Users/Configuration (spec 3.6, AC-8).
//
// # It replaces the caller's own configuration, and a `userId` naming somebody
// else changes nothing about that
//
// spec 3.6 says "the authenticated user's configuration" and names no
// parameter. The reference declares one — `[FromQuery] Guid? userId`, defaulted
// to the caller's own identifier, with a `404` for an identifier nobody has and
// its own `403` for a caller who may not update that account
// [source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11]. An
// unrecognised query value is ignored rather than rejected (behaviours 1.12),
// so this route reads no query at all and an administrator naming another
// account writes to their **own**.
//
// That is U-14 in docs/compatibility/reference-target.md, and it is the first
// row of the register where what this project does is not a different answer
// but a different **act**: a silent write to the wrong account rather than a
// status a client can branch on. The specification is implemented as written
// (AGENTS.md 1.3 — source evidence does not discharge a specification, and no
// running reference has been asked), the divergence is asserted as a test
// rather than recorded as a comment, and one request settles it.
//
// # Replacement, not merge, and the decode is what makes it one
//
// The posted document is decoded over users.DefaultConfiguration() and **never
// over the stored one**, so a property the client omitted returns to the
// reference's default instead of keeping whatever was there. That is the
// reference's own behaviour rather than a reading of the route's name: its
// binder constructs a fresh UserConfiguration and the update assigns fifteen of
// the sixteen properties from it unconditionally
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:760-799 @ v10.11.11].
//
// The sixteenth is CastReceiverId, and it is the one property this route does
// not round-trip the way the reference does: there, a posted value is kept only
// when the installation declares a cast receiver application with that
// identifier, so an unknown one leaves the stored value alone
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:785-789 @ v10.11.11].
// Here it is stored like the other fifteen. Atrium ships no cast receiver
// applications at all, so replicating the condition would mean discarding every
// value this route is ever sent — and spec 3.6 says the property is stored and
// returned faithfully. The difference is asserted rather than left to be found,
// for U-14's reason, and it belongs in the register beside T2's note that
// CastReceiverId is the one member of this model whose **value** cannot match.
//
// # An unknown property is dropped, which is the opposite of a session's
// capabilities
//
// spec 3.6: unknown properties are ignored, not rejected. users.DecodeConfiguration
// does that by not asking encoding/json to refuse them, and behaviours 5.9
// records the capabilities document doing the **opposite** — an unknown property
// there survives into /Sessions, as a declared divergence. The two look like one
// question and are not.
//
// # The credential is read before the body is bound
//
// The same order GET /Users/{userId} takes, and for the same reason: the
// reference's authorization filter runs ahead of its model binder (009 spec 3.8,
// measured 2026-09-01). So a request with no credential and a body that is not
// JSON is the empty 401 and never the validation 400 — a handler that bound
// first would tell an unauthenticated caller what this server thinks of its
// body.
func (h *UsersHandler) UpdateConfiguration() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := admitted(w, r, h.authenticator)
		if !ok {
			return
		}

		posted, err := io.ReadAll(r.Body)
		if err != nil {
			// The body could not be read off the connection, which is not the
			// request being wrong (users.go's AuthenticateByName takes the
			// same view of the same failure).
			WriteInternalServerError(w)
			return
		}

		configuration, err := users.DecodeConfiguration(posted)
		if err != nil {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				jsonDocumentKey:              {deserialiserMessage(err)},
				configurationActionParameter: {requiredBodyMessage(configurationActionParameter)},
			})
			return
		}

		document, err := configuration.Document()
		if err != nil {
			WriteInternalServerError(w)
			return
		}
		if err := h.accounts.ReplaceConfiguration(r.Context(), caller.UserID, document); err != nil {
			WriteInternalServerError(w)
			return
		}

		writeNoContent(w)
	}
}

// deserialiserMessage is the text a validation refusal carries under `"$"`: the
// deserialiser's own words about a document it could not read.
//
// The domain wraps its decoder's error with its own package name, and that
// prefix must not travel. The login route sends encoding/json's text bare, and
// two refusals of one shape spelled differently would be a difference between
// two of this server's own routes before it was ever a difference from the
// reference. behaviours 1.11 already declares the class — the message under
// `"$"` is this parser's where the reference's is .NET's — so what is kept
// consistent here is the spelling, not the match.
//
// It unwraps rather than reaching past users.DecodeConfiguration with a second
// parse, because the decode over the reference's defaults is the domain's rule
// (plan 6.6) and a handler that unmarshalled the body itself to get a better
// error message would have quietly stopped following it.
func deserialiserMessage(err error) string {
	if unwrapped := errors.Unwrap(err); unwrapped != nil {
		return unwrapped.Error()
	}
	return err.Error()
}

// writeNoContent answers the `204` of spec 3.6: the status, no body and no
// content type.
//
// Unexported, and deliberately not one of refusal.go's shapes. It is a success
// rather than a refusal, so it is not in the walk that proves no two refusal
// writers produce the same response, and both of its callers are in this
// package — this route and the capabilities route of spec 3.8, which answers
// the same three things (behaviours 2.14).
//
// # The Del is the whole of the function, and unlike a declared length it is
// assertable
//
// net/http drops a `Content-Length` from a `204` whether or not a handler set
// one, and **keeps** a `Content-Type` an earlier stage left in the header map
// [measurement: net/http, Go 1.27.0, 2026-09-03]. So setting a length here
// would be a line no request could observe — WriteControllerRefusal records one
// of those already and there is nothing to gain by writing a second — while the
// Del is what keeps the absence of a content type a property of this project's
// code rather than of what happens to have run before it.
func writeNoContent(w http.ResponseWriter) {
	w.Header().Del("Content-Type")
	w.WriteHeader(http.StatusNoContent)
}
