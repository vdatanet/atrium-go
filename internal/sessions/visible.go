package sessions

import (
	"math"
	"strings"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

// Caller is who is asking, reduced to the two things the visibility rule reads.
//
// 002 plan 5 wrote Visible's second parameter as "Caller" inside the
// internal/sessions block while declaring a Caller in the internal/httpapi
// block, and the two cannot be the same type: httpapi.Caller carries a
// users.Policy, and this package importing either the edge or internal/users
// would invert architecture 2's arrow — plan 3 says in terms that sessions
// "knows a user by identifier and never the reverse". So the edge reduces its
// own Caller to this one at the call site, which is one line and is visible,
// rather than the domain reaching up for a type it must not see.
//
// IsAdministrator is a bool and not a policy for the same reason: this rule
// reads one flag [source: Emby.Server.Implementations/Session/SessionManager.cs:1967 @ v10.11.11],
// and a parameter carrying twenty-eight would invite a second branch to grow
// here that belongs at the edge. The zero value is the safe direction — a
// caller nobody filled in is a non-administrator with no sessions of their own,
// which answers an empty list rather than everybody's.
type Caller struct {
	// UserID is the account the request authenticated as.
	UserID string

	// IsAdministrator is users.Policy's flag of the same name, carried across
	// by the handler.
	IsAdministrator bool
}

// Selection is spec 3.8's three request parameters, already bound.
//
// The three are not three filters — spec 3.8 is emphatic about it — and the
// order they apply in is Visible's, not the caller's.
type Selection struct {
	// DeviceID is deviceId, and it is "" when the parameter was absent *or*
	// empty. Spec 3.8 measures the two as the same request: "deviceId=" is
	// ignored rather than read as a device nothing is named after
	// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29].
	DeviceID string

	// ControllableByUser is controllableByUserId, "" when absent or empty.
	// It is not a filter; see Visible.
	ControllableByUser string

	// ActiveWithinSeconds is activeWithinSeconds, and it is an int rather than
	// a pointer because spec 3.8 makes 0 and absent the same request: the zero
	// value already means what the wire means. A negative value means it too.
	ActiveWithinSeconds int
}

// Visible is the session list the caller may see, given the request's
// selection, as of now.
//
// It is one function rather than a filter the handler composes with a
// visibility rule, and 002 plan 5 says why: the three parts apply in a fixed
// order, and two exported halves are two chances to compose them the wrong way
// round in a package whose tests would still pass because each half is right.
//
// The order is the reference's
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1947-2034 @ v10.11.11]:
//
//  1. DeviceID narrows the whole list, matched without regard to case.
//  2. Then the visibility rule — the caller's own sessions, or every session
//     for an administrator — or the replacement ControllableByUser makes of it.
//  3. Then ActiveWithinSeconds, and only when it is greater than zero.
//
// # What the order buys, and what it does not
//
// All three are predicates over one list, so they commute: no request tells one
// sequence from another, and spec 3.8 records that writing AC-15 is what
// discovered it. ActiveWithinSeconds running last is the same kind of
// no-difference as DeviceID running first. What a client *can* see is the
// combination in step 2's first case — DeviceID still narrows a request that
// also carries ControllableByUser — and that is what the test asserts, under
// that name rather than under "the order". The sequence is written the
// reference's way regardless, because being right is not conditional on being
// observable.
//
// # Why a non-empty ControllableByUser answers nothing
//
// The reference's first clause for that parameter keeps only the sessions that
// support remote control
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1977 @ v10.11.11],
// and v1 attaches no control channel, so spec 3.8 reports SupportsRemoteControl
// false on every session — measured to be what the reference reports for a
// request-response client too (behaviours 2.14). The rule's three later clauses
// therefore decide nothing observable, and spec 3.8 does not state them.
//
// This is the one branch in this feature whose correctness is an argument
// rather than a comparison, so it is worth being exact about what it is *not*:
// it does not read the session's declared capabilities. A client that posted
// SupportsMediaControl: true is still absent from this answer, because the
// declaration is the client's and the flag is the server's judgement about it
// (spec 3.8) [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
// The list stays empty for every caller, an administrator included; the 403 for
// a non-administrator naming somebody else is the handler's and never reaches
// here (002 plan 6.10).
//
// The result is never nil, so an empty answer is [] on the wire and not null.
func Visible(all []ports.Session, caller Caller, sel Selection, now units.Time) []ports.Session {
	result := make([]ports.Session, 0, len(all))
	result = append(result, all...)

	if sel.DeviceID != "" {
		result = keep(result, func(session ports.Session) bool {
			return strings.EqualFold(session.DeviceID, sel.DeviceID)
		})
	}

	switch {
	case sel.ControllableByUser != "":
		result = result[:0]
	case caller.IsAdministrator:
		// Every session, which is the whole of what being an administrator
		// buys on this route.
	default:
		result = keep(result, func(session ports.Session) bool {
			return session.UserID == caller.UserID
		})
	}

	if sel.ActiveWithinSeconds > 0 {
		window := windowTicks(sel.ActiveWithinSeconds)
		result = keep(result, func(session ports.Session) bool {
			return now.Ticks()-session.LastActivityAt.Ticks() <= window
		})
	}

	return result
}

// keep filters in place, which is safe because Visible owns the slice it built.
func keep(sessions []ports.Session, wanted func(ports.Session) bool) []ports.Session {
	kept := sessions[:0]
	for _, session := range sessions {
		if wanted(session) {
			kept = append(kept, session)
		}
	}
	return kept
}

// maxWindowSeconds is the largest window that fits in a tick count. Anything
// above it saturates rather than wrapping: activeWithinSeconds arrives off a
// query string, so the value is the client's, and a multiplication that
// overflowed would turn "every session ever" into a window in the past and
// answer an empty list for a request that asked for everything.
const (
	maxTicks         = units.Ticks(math.MaxInt64)
	maxWindowSeconds = int(maxTicks / units.TicksPerSecond)
)

func windowTicks(seconds int) units.Ticks {
	if seconds >= maxWindowSeconds {
		return maxTicks
	}
	return units.Ticks(seconds) * units.TicksPerSecond
}
