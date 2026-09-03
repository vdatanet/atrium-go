package sessions

import (
	"cmp"
	"slices"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// Evictions names the sessions that have to be closed before `opening` may be
// opened for `userID`, least recently used first.
//
// It is spec 3.8's lifecycle row — *"`MaxActiveSessions` exceeded: oldest
// session is evicted"* — and AC-13's sharper spelling of the same rule, which
// says *least recently used* rather than oldest. The two disagree on a session
// that was created first and used last, and this function follows the
// criterion: `LastActivityAt`, not `CreatedAt`.
//
// # This is where the specification and the reference part company
//
// The reference does not evict. It counts the user's sessions and throws
// `SecurityException("User is at their maximum number of sessions.")`
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11],
// which its exception filter turns into the 403 and the 25 bytes. 002 plan 6.7
// implements the specification as written because
// [AGENTS.md 1.3] makes the running server the tie-breaker, there is none in
// this run, and source evidence does not discharge a specification. The
// divergence is register row U-13 and one request settles it: two logins on an
// account whose `MaxActiveSessions` is 1.
//
// # `opening` is excluded from the count, and the reference's is not
//
// A second authentication from the same client and device replaces that
// session rather than adding one (002 plan 6.5), so nothing is exceeded and
// nothing is evicted. The reference counts it anyway — its check runs before it
// touches the session list, over `Sessions.Count(i => i.UserId == user.Id)`
// with a `>=` — so an account whose ceiling is 1 cannot re-authenticate from
// the device it is already logged in on there, and can here. That is a second
// face of U-13 rather than a second finding, and it is the half a probe would
// otherwise not think to send.
//
// # The guard is `ceiling >= 1`, which is where "unlimited" lives
//
// Spec 3.5 makes `0` unlimited and `0` is what the reference sends for a
// default account
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]; the
// reference spells the guard `maxActiveSessions >= 1`, so a negative value is
// unlimited too rather than a cap nobody can satisfy. Written this way, an
// operator who typed a negative number gets the safe reading rather than an
// account that can never log in.
//
// # The order is total, because an eviction is a deletion
//
// Two sessions idle since the same tick would otherwise be evicted in whatever
// order the store happened to return them, and Principle VII forbids a
// behaviour that derives from storage order. The identifier breaks the tie —
// it is derived from the client and the device (DeriveID), so the tie-break is
// stable across runs and across a rebuild of the store.
//
// The result is nil when nothing has to go, which is the common answer: every
// account the reference provisions carries a ceiling of 0.
func Evictions(all []ports.Session, userID, opening string, ceiling int) []string {
	if ceiling < 1 {
		return nil
	}

	others := make([]ports.Session, 0, len(all))
	for _, session := range all {
		if session.UserID == userID && session.ID != opening {
			others = append(others, session)
		}
	}
	if len(others) < ceiling {
		return nil
	}

	slices.SortFunc(others, func(a, b ports.Session) int {
		if by := cmp.Compare(a.LastActivityAt.Ticks(), b.LastActivityAt.Ticks()); by != 0 {
			return by
		}
		return cmp.Compare(a.ID, b.ID)
	})

	// One more than the surplus, because the session being opened takes a
	// place of its own: leaving exactly `ceiling` behind and then opening one
	// would be a login that ends over the cap it just enforced.
	closing := make([]string, 0, len(others)-ceiling+1)
	for _, session := range others[:len(others)-ceiling+1] {
		closing = append(closing, session.ID)
	}
	return closing
}
