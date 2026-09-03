package sessions_test

import (
	"slices"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
)

// The fixture visible_test.go builds is reused rather than rebuilt: four
// sessions over two accounts, with three different idle times, which is exactly
// what a rule that picks the least recently used needs. ada holds
// `ada-living-room` (idle 0) and `ada-phone` (idle an hour); bob holds
// `bob-living-room` (10 s) and `bob-kitchen` (90 s).
//
// The idle times are what makes "least recently used" different from "oldest":
// every row in that fixture was created at the same instant, so a rule reading
// CreatedAt would have nothing to order by and would fall through to the
// tie-break. AC-13 says least recently used and this is the fixture that can
// tell the two apart.

// The table is the whole of 002 plan 6.7's ceiling, read as a rule about the
// session that is *about to* exist.
//
// The `opening` column is the identifier the login is about to write. It is
// excluded from the count, which is what makes a second authentication from a
// device the account is already logged in on evict nothing: the session is
// replaced rather than added (002 plan 6.5), so nothing is exceeded. The
// reference counts it and refuses — see Evictions' own comment and register
// row U-13.
func TestTheSessionCeilingEvictsTheLeastRecentlyUsed(t *testing.T) {
	for _, row := range []struct {
		name    string
		user    string
		opening string
		ceiling int
		want    []string
	}{
		{
			// The default every account the reference provisions carries
			// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
			// 2026-08-28]. Nothing is capped, so nothing is evicted, however
			// many sessions the account holds.
			name: "a ceiling of zero is unlimited", user: ada, opening: "ada-new", ceiling: 0,
		},
		{
			// The same guard, which the reference spells `>= 1`. An operator
			// who typed a negative number gets "unlimited" rather than an
			// account that can never open a session.
			name: "a negative ceiling is unlimited too", user: ada, opening: "ada-new", ceiling: -1,
		},
		{
			// Two sessions already, one place allowed, one arriving: two have
			// to go, least recently used first. ada-phone has been idle an
			// hour and ada-living-room not at all.
			name: "a ceiling of one evicts everything the account already holds",
			user: ada, opening: "ada-new", ceiling: 1,
			want: []string{"ada-phone", "ada-living-room"},
		},
		{
			// One place spare, so the account's least recently used session
			// goes and the other stays. This is the row that fails on a build
			// evicting the *most* recently used, which every other row here is
			// satisfied by.
			name: "a ceiling of two evicts one, and it is the idle one",
			user: ada, opening: "ada-new", ceiling: 2,
			want: []string{"ada-phone"},
		},
		{
			// Room to spare: two sessions, three allowed, one arriving.
			name: "an account below its ceiling evicts nothing",
			user: ada, opening: "ada-new", ceiling: 3,
		},
		{
			// The re-authentication. ada holds two sessions and the ceiling is
			// two, but one of them is the session this login replaces, so the
			// account ends with the two it is allowed. A build that counted
			// the replaced session would evict ada's other device on every
			// login from this one — a device logged out with nothing in any
			// response reporting it.
			name: "the session being replaced does not count against the ceiling",
			user: ada, opening: "ada-phone", ceiling: 2,
		},
		{
			// The same request one place tighter, which is where the exclusion
			// stops being free: ada-living-room still has to go.
			name: "a replacement still evicts when the rest are over the ceiling",
			user: ada, opening: "ada-phone", ceiling: 1,
			want: []string{"ada-living-room"},
		},
		{
			// The count is per account and not per installation. bob holds two
			// sessions and ada's ceiling of one says nothing about them.
			name: "another account's sessions are not counted",
			user: "nobody-has-this-identifier", opening: "new", ceiling: 1,
		},
		{
			// bob's own, for the same reason the row above is here: a rule
			// that ignored the user would evict ada-phone, which is the oldest
			// session on the installation and belongs to somebody else.
			name: "the least recently used of the other account is that account's",
			user: bob, opening: "bob-new", ceiling: 1,
			want: []string{"bob-kitchen", "bob-living-room"},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := sessions.Evictions(fixture(), row.user, row.opening, row.ceiling)
			if !slices.Equal(got, row.want) {
				t.Errorf("Evictions answered %v, want %v", got, row.want)
			}
		})
	}
}

// Two sessions idle since the same instant are evicted in an order that does
// not derive from the order the store returned them (Principle VII).
//
// The two lists below are the same three sessions in opposite orders, and an
// implementation that sorted only on the activity date would answer whichever
// one it was handed first — deterministically in a test that always builds the
// list the same way, and by storage order in production. The identifier breaks
// the tie, and it is derived from the client and the device, so the answer
// survives a rebuild of the store.
func TestSessionsIdleSinceTheSameInstantAreEvictedInAStatedOrder(t *testing.T) {
	forward := []ports.Session{
		session("ada-alpha", ada, "Embeat", "alpha", 0, nil),
		session("ada-beta", ada, "Embeat", "beta", 0, nil),
		session("ada-gamma", ada, "Embeat", "gamma", 0, nil),
	}
	backward := []ports.Session{forward[2], forward[1], forward[0]}

	want := []string{"ada-alpha", "ada-beta"}
	for _, row := range []struct {
		name string
		all  []ports.Session
	}{{"as the store returned them", forward}, {"reversed", backward}} {
		got := sessions.Evictions(row.all, ada, "ada-new", 2)
		if !slices.Equal(got, want) {
			t.Errorf("%s: Evictions answered %v, want %v", row.name, got, want)
		}
	}
}

// The count that matters is the one *after* the login, which is the off-by-one
// this rule is easiest to get wrong by.
//
// An account at exactly its ceiling has to lose a session before it gains one,
// or the login ends one over the cap it was asked to enforce — and a build
// evicting only when the account is already *over* the ceiling would answer
// nothing here and would be caught by no other row, because every request that
// exceeds a ceiling passes through this state first.
func TestAnAccountExactlyAtItsCeilingLosesASessionBeforeItGainsOne(t *testing.T) {
	held := []ports.Session{
		session("ada-alpha", ada, "Embeat", "alpha", 0, nil),
		session("ada-beta", ada, "Embeat", "beta", time.Minute, nil),
	}

	got := sessions.Evictions(held, ada, "ada-new", 2)
	if !slices.Equal(got, []string{"ada-beta"}) {
		t.Fatalf("an account holding two sessions with a ceiling of two evicted %v, want [ada-beta]", got)
	}

	// And the answer is one and not both: the login takes the place the
	// eviction made and no more.
	if len(got) != 1 {
		t.Errorf("Evictions answered %d sessions, want 1", len(got))
	}
}
