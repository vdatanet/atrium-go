package users

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// The two refusals this path can produce.
//
// They are two and not four because the disclosure is the point. spec 3.3
// measures four refusal conditions and answers them with two statuses: an
// unknown username and a wrong password are both 401, and a disabled account is
// 403 whether the password was right or wrong. Collapsing the first pair into
// one sentinel is not tidiness — it is what makes it impossible for a later
// handler to answer them differently, which is the disclosure ADR-0006's decoy
// spends 52 ms per request to close on the clock. A path that returned
// "no such account" and "wrong password" as distinct values would hand the
// caller the very distinction the derivation exists to hide.
//
// Neither names a status. internal/users is Domain (architecture 2), so it does
// not know what an HTTP status is; the handler maps these two, and plan 7's
// failure table is where the mapping is written down.
var (
	// ErrCredentialsRefused is "no account has this username, or this is not
	// its password". spec 3.3 answers it 401, and the two cases are
	// deliberately indistinguishable from here outwards.
	ErrCredentialsRefused = errors.New("users: the username or the password is wrong")

	// ErrAccountDisabled is "this account cannot log in at all", which spec
	// 3.3 answers 403 — "stop asking" rather than "ask again".
	//
	// There is no ErrLockedOut beside it, and that absence is the design
	// rather than an omission. A lockout is *stored* as the disabled flag
	// (plan 6.7, and the reference does the same
	// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @
	// v10.11.11]), so "locked out" is not a state an account is ever in on a
	// later attempt: it is disabled, and it answers as disabled. The two rows
	// of plan 6.7's table are one state on the second try, and a second
	// sentinel would be a distinction nothing in this server can produce.
	ErrAccountDisabled = errors.New("users: the account is disabled")
)

// defaultLockoutThreshold is what LoginAttemptsBeforeLockout = 0 means.
//
// Three, and not "no attempts allowed", which is the reading the number looks
// like it has. The reference's own comment on the line explains it as a client
// sending the default as a zero
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @
// v10.11.11].
const defaultLockoutThreshold = 3

// Fold reduces a username to the spelling two accounts may not share, and the
// only spelling an authentication looks a row up by (spec 3.3, plan 4).
//
// spec 3.3 matches a username case-insensitively. The fold is stored in its own
// unique column rather than computed per query, so the uniqueness the login
// depends on is the database's rule; this function is what fills that column
// (T7's provisioning) and what a login reduces the presented name with, and it
// has to be the same function on both sides or an account becomes
// unauthenticatable without any error anywhere.
//
// It folds *down* where the reference folds up — the reference normalises with
// ToUpperInvariant
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:155-166 @
// v10.11.11]. The direction is not observable: the fold is a store key, and the
// spelling that reaches the wire is Username, which is what the operator typed.
// What *is* observable is whether two names collide, and the two directions can
// disagree there on a handful of characters. No client can produce that
// disagreement in v1, because v1 has no rename and provisioning refuses a name
// whose fold is already taken; if one ever can, this comment is where the
// question starts.
func Fold(username string) string { return strings.ToLower(username) }

// Login is the login path: everything spec 3.3 does before a session exists.
//
// It is a value with two ports and no state of its own. The order it tests its
// refusals in is the behaviour and not an implementation detail — plan 6.4
// rule 3 fixes it, and the assertions that hold it are counts of Argon2id
// derivations rather than statements about statuses, because a count is the
// only thing that can see the difference between "refused after verifying" and
// "refused without verifying".
type Login struct {
	accounts ports.UserStore
	clock    ports.Clock
}

// NewLogin builds the login path over the account store and a clock.
func NewLogin(accounts ports.UserStore, clock ports.Clock) *Login {
	return &Login{accounts: accounts, clock: clock}
}

// Authenticate is spec 3.3's credential check.
//
// It returns the account as the store holds it *after* the attempt, or one of
// the two sentinels above, or a store error — which is neither refusal and
// which plan 7 answers 500, never 401, because a client told 401 discards a
// credential that was fine.
//
// # The order, which is the behaviour
//
//  1. Find the account by its folded name.
//  2. No account: verify the decoy record and throw the result away. ADR-0006
//     rule 1 — without it the unknown-username refusal returns in microseconds
//     and the wrong-password refusal returns in 52 ms, and two refusals spec
//     3.3 made byte-identical are told apart with a stopwatch.
//  3. A disabled account is refused *without* verifying. That is not a
//     shortcut: ADR-0006 says "no equalisation is owed there either", because
//     the refusal already discloses itself by answering 403 where the others
//     answer 401. Equalising it would buy nothing and cost 52 ms of a memory
//     reservation an unauthenticated caller can pull.
//  4. Otherwise verify, and on success re-derive if the stored record is below
//     the current constants — inside this call, because ADR-0006 makes the
//     successful login the only moment the plaintext exists.
//
// The reference's order is not this one: it verifies first and looks at the
// disabled flag afterwards
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:524-593 @
// v10.11.11]. Both send the same bytes; the difference is in the clock, and it
// is the direction that discloses less.
func (l *Login) Authenticate(ctx context.Context, username string, password Plaintext) (ports.User, error) {
	user, found, err := l.accounts.UserByFoldedName(ctx, Fold(username))
	if err != nil {
		return ports.User{}, fmt.Errorf("users: authenticating %q: %w", Fold(username), err)
	}
	if !found {
		// The decoy, and the result is dropped on purpose: it can only be
		// false, and reading it would invite an optimiser-minded reader to
		// delete the call.
		_, _, _ = Verify(DecoyRecord(), password)
		return ports.User{}, ErrCredentialsRefused
	}

	policy, err := PolicyOf(user)
	if err != nil {
		return ports.User{}, fmt.Errorf("users: authenticating %s: %w", user.ID, err)
	}
	if policy.IsDisabled {
		return ports.User{}, ErrAccountDisabled
	}

	verified, needsRehash, err := l.verify(ctx, user, password)
	if err != nil {
		return ports.User{}, err
	}
	if !verified {
		if err := l.accounts.RecordLoginOutcome(ctx, user.ID, failureOutcome(policy), l.clock.Now()); err != nil {
			return ports.User{}, err
		}
		return ports.User{}, ErrCredentialsRefused
	}

	if needsRehash {
		// plan 6.4 rule 5: inside the request and not behind it. A background
		// rewrite would be a write nothing waits for and nothing observes, and
		// the plaintext it needs does not survive the response.
		record, err := Derive(password)
		if err != nil {
			return ports.User{}, fmt.Errorf("users: rehashing the credential of %s: %w", user.ID, err)
		}
		if err := l.accounts.ReplaceCredential(ctx, user.ID, record, l.clock.Now()); err != nil {
			return ports.User{}, err
		}
	}

	if err := l.accounts.RecordLoginOutcome(ctx, user.ID, ports.LoginSucceeded, l.clock.Now()); err != nil {
		return ports.User{}, err
	}

	// Read the account back rather than patching the copy in hand. The
	// response carries InvalidLoginAttemptCount and LastLoginDate, and both
	// have just moved; a patched copy would be a second spelling of the
	// transition plan 5 deliberately made one store method, and the two
	// spellings would drift.
	after, found, err := l.accounts.UserByID(ctx, user.ID)
	if err != nil {
		return ports.User{}, fmt.Errorf("users: rereading %s after a successful login: %w", user.ID, err)
	}
	if !found {
		return ports.User{}, fmt.Errorf("users: account %s vanished during its own login", user.ID)
	}
	return after, nil
}

// verify answers whether this password opens this account, and whether the
// record it opened is below the current constants.
//
// The account with no password is the branch ADR-0006 rule 4 declines to
// equalise: Pw may be empty when the account has no password, and skipping the
// derivation is observable in time — but HasPassword is already published for
// every account on the unauthenticated GET /Users/Public (spec 3.4), so
// equalising it would protect a fact the reference gives away on an open route.
// The reference behaves the same way, admitting an empty password and refusing
// a non-empty one without hashing anything
// [source: Jellyfin.Server.Implementations/Users/DefaultAuthenticationProvider.cs:62-73 @ v10.11.11].
func (l *Login) verify(ctx context.Context, user ports.User, password Plaintext) (verified, needsRehash bool, err error) {
	credential, hasPassword, err := l.accounts.Credential(ctx, user.ID)
	if err != nil {
		return false, false, fmt.Errorf("users: reading the credential of %s: %w", user.ID, err)
	}
	if !hasPassword {
		return password.IsEmpty(), false, nil
	}

	verified, needsRehash, err = Verify(credential.PHC, password)
	if err != nil {
		// A record this package could not have written. It is not a wrong
		// password and must not be answered as one: 401 tells a client to
		// throw away a credential that was fine, and the fault is in the
		// store. plan 7 answers this 500, with the store-unreadable row.
		return false, false, fmt.Errorf("users: the stored credential of %s is unreadable: %w", user.ID, err)
	}
	return verified, needsRehash, nil
}

// failureOutcome decides which of the two failing transitions this attempt is:
// an ordinary failure, or the one that reaches the threshold and locks the
// account.
//
// It is here and not in the store because LoginAttemptsBeforeLockout is a
// sentinel and the store is not allowed to know that (ports.LoginOutcome says
// so). It is called exactly once per attempt, because RecordLoginOutcome is one
// transition and two calls would count one attempt twice.
func failureOutcome(policy Policy) ports.LoginOutcome {
	threshold, locks := lockoutThreshold(policy.LoginAttemptsBeforeLockout)
	if !locks {
		return ports.LoginFailed
	}
	// The counter as it will be once this attempt is recorded. The reference
	// increments first and compares afterwards
	// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @
	// v10.11.11], so the attempt that reaches the threshold is the one that
	// locks, not the one after it.
	if policy.InvalidLoginAttemptCount+1 >= threshold {
		return ports.LoginLockedOut
	}
	return ports.LoginFailed
}

// lockoutThreshold reads LoginAttemptsBeforeLockout as the three-way switch the
// reference makes it
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @
// v10.11.11]:
//
//	-1 → never lock
//	 0 → lock after three
//	 n → lock after n
//
// **It is not a count, and both edge readings are the opposite of the obvious
// one.** A build that compared a failure count against this number directly
// would read -1 as "lock immediately" and 0 as "lock immediately" — and -1 is
// what the reference sends for a default account
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], so
// the mistake would lock every stock account out on its first typo, and would
// be found by a user rather than by a test. spec 3.5 and spec 7's OQ-6 carry
// the same warning; this is where it is enforced.
//
// A negative value that is not -1 falls through to itself and therefore locks
// on the first failure, which is what the reference's own switch does with it.
// No probe has sent one and none of this project's writers can produce one; it
// is replicated rather than special-cased because inventing a third reading
// here would be a divergence with no measurement behind it.
func lockoutThreshold(loginAttemptsBeforeLockout int) (threshold int, locks bool) {
	switch loginAttemptsBeforeLockout {
	case -1:
		return 0, false
	case 0:
		return defaultLockoutThreshold, true
	default:
		return loginAttemptsBeforeLockout, true
	}
}

// PolicyOf decodes an account's stored policy document and overlays the
// failed-attempt counter from its own column.
//
// Two rules in one function, because performing one without the other is the
// bug and neither is visible in a round-trip test:
//
//   - The document decodes onto the reference's defaults and never onto Go's
//     zero value (the package comment argues what that costs).
//   - InvalidLoginAttemptCount is state and lives in a column, so whatever the
//     stored document carries for it is stale by construction (plan 4, 6.6).
//     Reading the lockout rule off the document alone would compare a fresh
//     count against a number the document happened to be written with.
func PolicyOf(user ports.User) (Policy, error) {
	policy, err := DecodePolicy(user.PolicyDocument)
	if err != nil {
		return Policy{}, err
	}
	policy.InvalidLoginAttemptCount = user.InvalidLoginAttemptCount
	return policy, nil
}
