package httpapi

import (
	"net/http"

	"github.com/vdatanet/atrium-go/internal/wire"
)

// PublicUsers answers GET /Users/Public (spec 3.4).
//
// # It reads no credential, and that is the whole design
//
// plan 6.2 puts this route in the "not required, not read" row, and the reason
// is not economy: the route is measured to answer the *same bytes* to an
// authenticated and to an unauthenticated caller
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], and a
// handler that never looks at the credential cannot answer two ways. There is
// no branch to get wrong, so the equality is a property of the code rather than
// a thing a test keeps checking.
//
// The same argument decides the body. spec 3.4 measured `/Users/Public` to be
// byte-identical to the same users read through an authenticated route, and
// this handler guarantees that by calling `userObject` — the one filler of
// plan 6.6, which landed with POST /Users/AuthenticateByName because that route
// returns a user object first. A second filler here would be the exact bug
// plan 6.6 exists to prevent.
//
// # What is excluded, and what the reference excludes that this does not
//
// spec 3.4 excludes users flagged hidden from login screens and says nothing
// about any other exclusion, so that is what this does. The reference's own
// source excludes more, and the difference is recorded rather than acted on
// because none of it is measured — see the amendment in plan 6.2 and
// TestADisabledUserIsListedHereAndTheReferencesSourceExcludesIt, which asserts
// the divergence so that the day a probe lands is a failing test naming the
// behaviour that moved rather than a rediscovery.
//
// # The order
//
// The order is the store's, which is `username_folded, id` — deterministic
// because architecture 2 requires it. The reference orders by the *unfolded*
// username
// [source: Jellyfin.Api/Controllers/UserController.cs:653-655 @ v10.11.11],
// which is evidence and not a measurement: what a running Jellyfin answers here
// has never been asked, and the two orderings can differ. plan 6.2 carries the
// item and the register owes it a row.
//
// # An empty array is a valid 200
//
// An installation where every user is hidden answers `[]` (spec 3.4), and the
// slice is built non-nil for that reason: internal/wire serialises a nil slice
// as `null`, which is a different document from an empty array and one no
// client expects here.
func (h *UsersHandler) PublicUsers() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accounts, err := h.accounts.Users(r.Context())
		if err != nil {
			WriteInternalServerError(w)
			return
		}

		public := make([]UserObject, 0, len(accounts))
		for _, account := range accounts {
			object, err := userObject(r.Context(), h.accounts, h.installationID, account)
			if err != nil {
				WriteInternalServerError(w)
				return
			}
			// The exclusion reads the flag off the object that would have
			// travelled, not off a second decode of the stored document. The
			// two can disagree: a document that does not carry `IsHidden` at
			// all decodes onto the reference's default, which is **true**
			// [source: Jellyfin.Data/UserEntityExtensions.cs:173 @ v10.11.11],
			// and a raw look at the document would read an absent property as
			// Go's false and put the account on every login screen. Filtering
			// on the built object makes the flag that decides the exclusion the
			// same value the body carries.
			if object.Policy.IsHidden {
				continue
			}
			public = append(public, object)
		}

		if err := wire.Write(w, http.StatusOK, public, NegotiateProfile(r)); err != nil {
			WriteInternalServerError(w)
		}
	}
}
