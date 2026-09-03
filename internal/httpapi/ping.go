package httpapi

import (
	"net/http"

	"github.com/vdatanet/atrium-go/internal/system"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// Ping answers both GET /System/Ping and POST /System/Ping (spec 3.3).
//
// The body is a bare JSON string — not an object with one field, not an empty
// body — and its value is the **product name**, `"Jellyfin Server"`.
//
// # The name it returns is not the operator's
//
// This is the whole of the endpoint and the only way to get it wrong. The
// reference's own documentation comment on the operation says it returns "the
// server name", and the code returns the application's product name
// [source: Jellyfin.Api/Controllers/SystemController.cs:102-106 @ v10.11.11,
// returning _appHost.Name; ApplicationHost.cs:260 defines
// Name => ApplicationProductName]. spec 3.3 settles which of the two this
// project follows in four words: **the code is the specification.**
//
// Following the comment instead would return whatever the operator renamed
// this installation to — "atrium" on a fresh one — and a health check or a
// reverse proxy that matches on the product name would then read this server
// as something other than Jellyfin while /System/Info/Public still read as
// Jellyfin. That is a delta on a route whose entire purpose is to be probed
// before anything else is.
//
// So this method reads **nothing** off the handler. The friendly name is one
// field access away — the same value has already been fetched for
// /System/Info/Public — and the distance between the two is a constant from
// internal/system rather than a comment asking the next reader not to reach
// for it.
//
// # Why it is a method on SystemHandler at all, given that
//
// Because Handlers names one field per feature, not one per route, and the
// four rows of 001 are one feature's /System responses. A second handler type
// holding no state would be a type whose only content is the fact that it
// holds no state.
//
// # One handler value, two rows
//
// The two operations differ only in the method token they are registered on:
// spec 3.3 gives them one request shape (no authentication, no parameters) and
// one response. Registering the same value twice is what makes "both methods
// answer identically" true by construction rather than by a second literal
// somebody has to keep in step.
//
// # It still negotiates
//
// spec 3.0 applies to every response in this specification, so the content
// type echoes the profile that matched (AC-9). A bare string carries no
// property names, so the three profiles produce one body over three content
// types — which is a thing worth asserting rather than a reason to skip the
// negotiation.
func (h *SystemHandler) Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := wire.Write(w, http.StatusOK, system.ProductName, NegotiateProfile(r)); err != nil {
			// Unreachable for a constant string, and handled anyway for the
			// reason PublicInfo handles it: wire.Write writes nothing to w
			// unless the whole body serialised, so there is still a refusal to
			// send.
			WriteInternalServerError(w)
		}
	}
}
