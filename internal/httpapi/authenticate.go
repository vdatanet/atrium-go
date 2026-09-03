package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// activityInterval is how often one session's LastActivityDate may be written:
// once a second, at most (002 plan 6.10).
//
// **It is a decision about frequency and not about the value.** The date is on
// the wire, so it has to advance; a busy client would otherwise turn every
// request it makes into a write, and the difference between that and this is
// invisible to a test asserting only that the date moved.
//
// One second is the granularity below which nothing observable changes for a
// client that reads the date, and it is far below the reference's own
// throttles — the reference stamps a *user's* activity only when more than 60
// seconds have passed
// [source: Emby.Server.Implementations/Session/SessionManager.cs:265-271 @ v10.11.11]
// and a token's only after three minutes
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:180-184 @ v10.11.11],
// while keeping the session's own date exact because it holds sessions in
// memory and this server holds them in a table. What the reference does with
// the session date is therefore not a rule this can copy, and 002 plan 6.10's
// once a second is the rule that is implemented.
const activityInterval = units.TicksPerSecond

// TokenAuthenticatorConfig is what the authenticator is built from.
//
// It is a struct rather than a parameter list for NewSystemHandler's reason:
// three ports of which two are stores, told apart at a call site only by their
// order.
type TokenAuthenticatorConfig struct {
	// Sessions resolves a presented token and records the session's activity.
	Sessions ports.SessionStore

	// Accounts is where the token's user and its policy come from.
	Accounts ports.UserStore

	// Clock is the instant an authenticated request is stamped with.
	// architecture 2 makes the clock a port, so nothing here reads the wall
	// clock: a test that could not choose the instant could not tell one write
	// per second from one write per request.
	Clock ports.Clock
}

// TokenAuthenticator is the Authenticator 001 declared: 002 spec 3.1's
// credential, resolved to a session, a user and that user's policy.
//
// # What it decides, and what it deliberately does not
//
// It answers only "what does this credential entitle the request to". It knows
// no route, so it applies no route's policy: 001's first-time-setup exemption
// stays at the handler that owns it, because an exemption moved in here would
// be inherited by every route that is not /System/Info
// [source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11].
// Which routes require a token is 002 plan 6.2's table and is asked by each
// handler.
//
// # No credential and an unknown token are one answer
//
// Both are Authentication{} — AccessUnauthenticated, no caller, and **no
// error**. 002 plan 7's first two rows make them indistinguishable at the wire
// as well, which is measured
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26], and
// the cheapest way to keep two wire answers identical is for there to be one
// value behind them. A token that resolves to a user the store no longer holds
// takes the same path for the same reason: it is a credential naming nobody.
type TokenAuthenticator struct {
	sessions ports.SessionStore
	accounts ports.UserStore
	clock    ports.Clock
}

// Interface satisfaction, asserted here rather than discovered at the one call
// site that wires it.
var _ Authenticator = (*TokenAuthenticator)(nil)

// NewTokenAuthenticator builds the authenticator over the two stores and a
// clock.
//
// It refuses all three absences rather than defaulting any of them, and the
// clock is the one worth saying why about: a nil clock defaulted to the wall
// clock would make the port an option, and architecture 2 made it a port so
// that the instant is chosen by whoever assembles the process.
func NewTokenAuthenticator(cfg TokenAuthenticatorConfig) (*TokenAuthenticator, error) {
	if cfg.Sessions == nil {
		return nil, errors.New("httpapi: the authenticator needs a session store, and was given none")
	}
	if cfg.Accounts == nil {
		return nil, errors.New("httpapi: the authenticator needs an account store, and was given none")
	}
	if cfg.Clock == nil {
		return nil, errors.New("httpapi: the authenticator needs a clock, and was given none")
	}
	return &TokenAuthenticator{sessions: cfg.Sessions, accounts: cfg.Accounts, clock: cfg.Clock}, nil
}

// Authenticate resolves the credential this request presents.
//
// The order is the order the answers are decidable in, and each step's absence
// is the same answer as the step before it:
//
//  1. The token, over 002 spec 3.1's five mechanisms (PresentedToken). An
//     empty string is a request that presented no credential, and it is not an
//     error: nothing in that reader can fail.
//  2. The session the token opened, by the token's digest. Not found is an
//     unknown or revoked token.
//  3. The user the *token* was issued to, which is the store's second return
//     value and not the session's user (002 plan 6.5).
//  4. That user's policy. A disabled account is AccessForbidden with no
//     caller.
//  5. The session's activity date, advanced at most once a second.
//
// # A store error is an error, and never a refusal
//
// Every failure below returns Authentication{} *and* a non-nil error, and
// 002 plan 7's last row is what the handler does with it: 500, never 401. A
// client answered 401 discards a credential that was fine and logs in again,
// so a database that was briefly unreadable would make every client in the
// house re-authenticate. The zero Authentication beside the error is the
// second half of the same rule — a caller that read the value and dropped the
// error refuses rather than admits.
func (a *TokenAuthenticator) Authenticate(r *http.Request) (Authentication, error) {
	token := PresentedToken(r)
	if token == "" {
		return Authentication{}, nil
	}

	ctx := r.Context()
	session, tokenUserID, found, err := a.sessions.SessionByTokenDigest(ctx, sessions.TokenDigest(token))
	if err != nil {
		return Authentication{}, fmt.Errorf("httpapi: resolving a presented token: %w", err)
	}
	if !found {
		return Authentication{}, nil
	}

	user, found, err := a.accounts.UserByID(ctx, tokenUserID)
	if err != nil {
		return Authentication{}, fmt.Errorf("httpapi: reading the account %s a token belongs to: %w", tokenUserID, err)
	}
	if !found {
		// A live token whose account is gone. The account store is the
		// authority on who exists, so this is a credential naming nobody and
		// it answers what an unknown token answers. It is not an error: the
		// foreign key makes it unreachable through this store (002 plan 4),
		// and answering 500 to a state a future store could reach would refuse
		// with the shape reserved for a server that is broken.
		return Authentication{}, nil
	}

	policy, err := users.PolicyOf(user)
	if err != nil {
		return Authentication{}, fmt.Errorf("httpapi: reading the policy of %s: %w", user.ID, err)
	}
	if policy.IsDisabled {
		// 002 plan 7's third row, and the caller is deliberately nil: this
		// request is refused, and a handler that read a caller off a refusal
		// would be reading somebody this server just declined to serve.
		//
		// A lockout arrives here too rather than through a value of its own,
		// because a lockout is *stored* as this flag (002 plan 6.7, and the
		// reference does the same
		// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @ v10.11.11]),
		// so "locked out" is not a state an account is in on a later request.
		return Authentication{Access: AccessForbidden}, nil
	}

	if err := a.recordActivity(ctx, session); err != nil {
		return Authentication{}, err
	}

	return Authentication{
		Access: AccessGranted,
		Caller: &Caller{UserID: tokenUserID, SessionID: session.ID, Policy: policy},
	}, nil
}

// recordActivity advances the session's LastActivityDate, at most once per
// session per second (002 plan 6.10).
//
// # The throttle's state is the stored date itself
//
// There is no per-session timer and no map of last-written instants, which is
// what makes this correct across a restart, across two processes reading one
// store, and in a test that does not have to reach inside anything: the value
// being compared is the value that was written, and it arrived on the session
// this request already read to resolve its token. A cache here would be a
// second copy of a fact the store holds, and the failure mode of the two
// disagreeing is a date that stops advancing.
//
// # Elapsed, not "a different second"
//
// The comparison is against the interval, not against the second boundary a
// truncation would put the two instants either side of. Truncating would make
// two requests 200 ms apart write twice whenever they straddled a second, which
// is a throttle whose behaviour depends on when it started rather than on how
// often it is asked. The bound is inclusive — exactly one second apart writes —
// because that is the frequency the rule names.
//
// A clock that has gone backwards writes nothing: the difference is negative,
// which is below the interval. That is the safe direction, because the
// alternative moves a session's last activity into the past.
func (a *TokenAuthenticator) recordActivity(ctx context.Context, session ports.Session) error {
	now := a.clock.Now()
	if now.Ticks()-session.LastActivityAt.Ticks() < activityInterval {
		return nil
	}
	if err := a.sessions.TouchSession(ctx, session.ID, now); err != nil {
		return fmt.Errorf("httpapi: recording the activity of session %s: %w", session.ID, err)
	}
	return nil
}
