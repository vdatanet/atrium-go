package httpapi

import (
	"context"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// UserObject is spec 3.5's user object: the body returned by the
// authentication route's `User` member, by `/Users/Public`, by `/Users/Me` and
// by `/Users/{userId}`.
//
// The type is named for what it is rather than for the wire, because the wire
// never sees the name: it is a member called `User` in one body and an array
// element in another.
//
// # One model and one filler, which is plan 6.6's whole argument
//
// spec 3.4 measured `/Users/Public` to be **byte-identical** to the same users
// read through an authenticated route. The cheapest way to guarantee that is
// for there to be nothing to keep in agreement — one struct, one function that
// fills it — which is 001's "the superset is structural" argument applied to a
// body four routes share. A second filler is the bug this file exists to
// prevent.
//
// # Key order
//
// The members are declared in the reference's own declaration order
// [source: MediaBrowser.Model/Dto/UserDto.cs:26-105 @ v10.11.11], which is the
// order .NET's serialiser writes them in and the order spec 3.5's table records
// for every member that travels — `ServerId` before `Id` in particular, which
// spec 3.5 notes was the other way round in this project's own table until it
// was measured
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
//
// The two orders disagree in one respect and it is unobservable: spec 3.5's
// table puts `ServerName` after `Id` and `PrimaryImageAspectRatio` sixth, while
// the reference declares `ServerName` third and `PrimaryImageAspectRatio` last.
// Both are null on every account this binary can hold, and behaviours 1.7 omits
// a null, so no request can tell the two orders apart — spec 3.5 says as much
// ("nothing can measure where a property that is never sent would sit"). The
// declaration order is followed because it is the only evidence either way.
//
// # What v1 never sends, and why each one is absent rather than empty
//
//   - `ServerName` — the reference never assigns it, so it is null
//     [source: Jellyfin.Server.Implementations/Users/UserManager.cs:415-437 @ v10.11.11].
//   - `PrimaryImageTag` and `PrimaryImageAspectRatio` — v1 gives a user no
//     avatar, so the condition spec 3.5 puts on both is never met (plan 6.6).
//     006 owns the day that changes.
//   - `LastLoginDate` — absent until the first login, which is the NULL column
//     T1 shipped for it. A non-pointer date here would answer
//     `0001-01-01T00:00:00.0000000Z` for an account that has never logged in,
//     which is a value where the reference sends no member at all. It is the
//     exact opposite of `SessionInfo.LastPlaybackCheckIn`, where the zero tick
//     *is* the value.
//   - `LastActivityDate` — absent for a reason that is a known gap rather than
//     a rule: nothing in this feature calls `ports.UserStore.TouchActivity`,
//     so `users.last_activity_at` stays NULL. The reference stamps a user only
//     after sixty seconds
//     [source: Emby.Server.Implementations/Session/SessionManager.cs:265-271 @ v10.11.11]
//     and holds sessions in memory where this server holds them in a table.
//     The throttle is measurable and unmeasured here, and the register owes a
//     row for it.
type UserObject struct {
	// Name is the spelling the operator chose, not the folded one. The fold is
	// a store key and never reaches the wire (internal/users' Fold).
	Name string

	// ServerId is the installation identity of 001 spec 3.1 — the same value
	// `/System/Info/Public` answers as `Id`, and the same value the
	// authentication result carries at its top level.
	ServerId string

	// ServerName is null on every reference response measured, and therefore
	// never sent. Declared so that Principle I is a count as well as a
	// spelling: the reference has the property and so does this model.
	ServerName *string `json:",omitempty"`

	// Id is the account identifier: 32 lowercase hex, derived from the folded
	// username (users.DeriveID, Principle VII).
	Id string

	// PrimaryImageTag is present only when the user has an avatar, which no
	// account in v1 does.
	PrimaryImageTag *string `json:",omitempty"`

	// HasPassword and HasConfiguredPassword are one fact under two names. The
	// reference fills both from the same call
	// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:413,420-421 @ v10.11.11],
	// so they are filled from one read here; whether any client reads them is
	// spec 7's OQ-4 and is not this model's question.
	HasPassword           bool
	HasConfiguredPassword bool

	// HasConfiguredEasyPassword is always false: v1 has no PIN concept
	// (spec 3.5), and the reference never assigns the property either, leaving
	// its declaration's false.
	HasConfiguredEasyPassword bool

	// EnableAutoLogin is nullable in the reference's DTO and filled from a
	// non-nullable column, so it always travels
	// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:422 @ v10.11.11].
	// v1 stores no such column and no route sets it, so it is false for every
	// account.
	EnableAutoLogin bool

	// LastLoginDate is absent until the first login — see the type comment.
	LastLoginDate *units.Time `json:",omitempty"`

	// LastActivityDate is absent for as long as nothing stamps it — see the
	// type comment.
	LastActivityDate *units.Time `json:",omitempty"`

	// Configuration is the sixteen properties of spec 3.6, decoded over the
	// reference's defaults. Sent by `/Users/Public` too, which is measured and
	// is behaviours 3.5's replicated disclosure.
	Configuration users.Configuration

	// Policy is the forty-two properties that travel of the forty-four the
	// reference declares, decoded over the defaults with
	// `InvalidLoginAttemptCount` overlaid from its own column (plan 6.6).
	// Sent by `/Users/Public` too, for the same reason.
	Policy users.Policy

	// PrimaryImageAspectRatio is null whenever PrimaryImageTag is, so it is
	// never sent. Declared for the reason ServerName is.
	PrimaryImageAspectRatio *float64 `json:",omitempty"`
}

// userObject builds spec 3.5's object for one account.
//
// It takes the account as the store holds it and reads exactly one more thing:
// whether the account has a password record. That read is why the function
// takes a context and a store rather than being a pure mapping — `HasPassword`
// is a fact about a row in another table, and the alternative (a `HasPassword`
// column on `users`, or a `bool` threaded in by every caller) would be a second
// place the answer is decided.
//
// The policy and the configuration are decoded here through the domain's own
// decoders, never with a bare unmarshal: a document decodes **over** the
// reference's defaults so that a property a stored document does not carry
// still travels with the value the reference would send, and
// `InvalidLoginAttemptCount` is overlaid from its column afterwards because
// whatever the document holds for it is stale by construction. `users.PolicyOf`
// is that pair in one call, and calling it is why there is no second decode
// here (plan 6.6).
func userObject(ctx context.Context, accounts ports.UserStore, installationID string, user ports.User) (UserObject, error) {
	policy, err := users.PolicyOf(user)
	if err != nil {
		return UserObject{}, fmt.Errorf("httpapi: building the user object of %s: %w", user.ID, err)
	}
	configuration, err := users.DecodeConfiguration(user.ConfigurationDocument)
	if err != nil {
		return UserObject{}, fmt.Errorf("httpapi: building the user object of %s: %w", user.ID, err)
	}

	_, hasPassword, err := accounts.Credential(ctx, user.ID)
	if err != nil {
		return UserObject{}, fmt.Errorf("httpapi: building the user object of %s: %w", user.ID, err)
	}

	return UserObject{
		Name:                      user.Username,
		ServerId:                  installationID,
		Id:                        user.ID,
		HasPassword:               hasPassword,
		HasConfiguredPassword:     hasPassword,
		HasConfiguredEasyPassword: false,
		EnableAutoLogin:           false,
		LastLoginDate:             user.LastLoginAt,
		LastActivityDate:          user.LastActivityAt,
		Configuration:             configuration,
		Policy:                    policy,
	}, nil
}
