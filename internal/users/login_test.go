package users_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// The whole of this file asserts one thing: the *order* the login path tests
// its refusals in.
//
// The order is not visible in a status code — an unknown username and a wrong
// password answer the same 401, and a disabled account answers 403 whether the
// password was right or wrong. What makes it visible is the number of Argon2id
// derivations the attempt spent, which internal/users counts for exactly this
// reason (DerivationStats). So every assertion here is a delta over
// users.Derivations().Completed, and that turns ADR-0006's sentences — "a
// username that matches no account is verified against a decoy record anyway",
// "no equalisation is owed there either" — into properties of the code.
//
// The counter is process-wide and is never zero when a test starts: the decoy
// is derived in the package's init, and every earlier test in this package adds
// to it. Take a baseline; never assert an absolute.

// aFixedInstant is what the clock below answers, so that a stamped date is a
// value the test chose rather than whatever time.Now returned.
var aFixedInstant = units.TimeFromTicks(638_000_000_000_000_000)

type fixedClock struct{}

func (fixedClock) Now() units.Time { return aFixedInstant }

// account is one row of the fake store, kept as the store keeps it: the policy
// as a document, and the failed-attempt counter in its own field.
//
// It is deliberately not a users.Policy. The overlay rule (plan 6.6) is one of
// the things this file tests, and a fake that held a decoded policy would have
// performed it for the code under test.
type account struct {
	user       ports.User
	credential *ports.Credential
}

// fakeAccounts is ports.UserStore for the login path.
//
// It re-implements RecordLoginOutcome's three transitions, which is the one
// piece of duplication in here and is worth naming: the real transition is
// asserted against real SQL in T4's store tests, and what these tests need is
// an account whose state moves the way the store moves it across several
// attempts. A fake that recorded the calls without applying them could not
// answer "does this account lock on the third failure".
type fakeAccounts struct {
	byID     map[string]*account
	byFolded map[string]*account

	// outcomes is every RecordLoginOutcome this store was handed, in order.
	// The login path must call it exactly once per attempt that reaches the
	// verifier: two calls would count one attempt twice.
	outcomes []ports.LoginOutcome

	// replacedCredentials counts rehashes.
	replacedCredentials int

	// lookupErr, when set, makes UserByFoldedName fail. It is how the
	// "a store error is not a 401" case is reached.
	lookupErr error
}

func newFakeAccounts() *fakeAccounts {
	return &fakeAccounts{byID: map[string]*account{}, byFolded: map[string]*account{}}
}

// add writes one account with a chosen policy and a chosen password.
//
// A nil password means an account with no password at all, which is ADR-0006
// rule 4's account and a state an operator can provision.
func (f *fakeAccounts) add(t *testing.T, id, username string, policy users.Policy, password *string) *account {
	t.Helper()
	document, err := policy.Document()
	if err != nil {
		t.Fatalf("encoding the policy of %s: %v", username, err)
	}
	entry := &account{user: ports.User{
		ID:                       id,
		Username:                 username,
		UsernameFolded:           users.Fold(username),
		PolicyDocument:           document,
		ConfigurationDocument:    []byte(`{}`),
		InvalidLoginAttemptCount: policy.InvalidLoginAttemptCount,
	}}
	if password != nil {
		record, err := users.Derive(users.NewPlaintext(*password))
		if err != nil {
			t.Fatalf("deriving the credential of %s: %v", username, err)
		}
		entry.credential = &ports.Credential{UserID: id, PHC: record, WrittenAt: aFixedInstant}
	}
	f.byID[id] = entry
	f.byFolded[entry.user.UsernameFolded] = entry
	return entry
}

// isDisabled reads the flag back out of the stored document, which is where a
// lockout writes it. Reading a decoded copy would not see the write.
func (f *fakeAccounts) isDisabled(t *testing.T, id string) bool {
	t.Helper()
	policy, err := users.DecodePolicy(f.byID[id].user.PolicyDocument)
	if err != nil {
		t.Fatalf("decoding the stored policy of %s: %v", id, err)
	}
	return policy.IsDisabled
}

func (f *fakeAccounts) UserByFoldedName(_ context.Context, folded string) (ports.User, bool, error) {
	if f.lookupErr != nil {
		return ports.User{}, false, f.lookupErr
	}
	entry, found := f.byFolded[folded]
	if !found {
		return ports.User{}, false, nil
	}
	return entry.user, true, nil
}

func (f *fakeAccounts) UserByID(_ context.Context, id string) (ports.User, bool, error) {
	entry, found := f.byID[id]
	if !found {
		return ports.User{}, false, nil
	}
	return entry.user, true, nil
}

func (f *fakeAccounts) Users(context.Context) ([]ports.User, error) {
	return nil, errors.New("Users is not part of the login path")
}

func (f *fakeAccounts) CreateUser(context.Context, ports.User) error {
	return errors.New("CreateUser is not part of the login path")
}

func (f *fakeAccounts) Credential(_ context.Context, userID string) (ports.Credential, bool, error) {
	entry, found := f.byID[userID]
	if !found || entry.credential == nil {
		return ports.Credential{}, false, nil
	}
	return *entry.credential, true, nil
}

func (f *fakeAccounts) ReplaceCredential(_ context.Context, userID, phc string, at units.Time) error {
	entry, found := f.byID[userID]
	if !found {
		return fmt.Errorf("no account %s", userID)
	}
	entry.credential = &ports.Credential{UserID: userID, PHC: phc, WrittenAt: at}
	f.replacedCredentials++
	return nil
}

func (f *fakeAccounts) ReplaceConfiguration(context.Context, string, []byte) error {
	return errors.New("ReplaceConfiguration is not part of the login path")
}

func (f *fakeAccounts) TouchActivity(context.Context, string, units.Time) error {
	return errors.New("TouchActivity is not part of the login path")
}

// RecordLoginOutcome applies the three transitions the real store applies.
func (f *fakeAccounts) RecordLoginOutcome(_ context.Context, userID string, outcome ports.LoginOutcome, at units.Time) error {
	entry, found := f.byID[userID]
	if !found {
		return fmt.Errorf("no account %s", userID)
	}
	f.outcomes = append(f.outcomes, outcome)
	switch outcome {
	case ports.LoginFailed:
		entry.user.InvalidLoginAttemptCount++
	case ports.LoginSucceeded:
		entry.user.InvalidLoginAttemptCount = 0
		stamped := at
		entry.user.LastLoginAt = &stamped
	case ports.LoginLockedOut:
		entry.user.InvalidLoginAttemptCount++
		var policy map[string]any
		if err := json.Unmarshal(entry.user.PolicyDocument, &policy); err != nil {
			return err
		}
		policy["IsDisabled"] = true
		document, err := json.Marshal(policy)
		if err != nil {
			return err
		}
		entry.user.PolicyDocument = document
	default:
		return fmt.Errorf("the outcome is %s", outcome)
	}
	return nil
}

// policyWithThreshold is DefaultPolicy with one property changed, which is how
// every account below is built: a hand-written document would decode onto the
// defaults anyway and would only hide which property the test is about.
func policyWithThreshold(attempts int) users.Policy {
	policy := users.DefaultPolicy()
	policy.LoginAttemptsBeforeLockout = attempts
	return policy
}

// derivationsDuring runs body and reports how many Argon2id derivations it
// spent.
//
// The baseline is taken inside, immediately before the call, because the
// counter carries every derivation this package has performed since the process
// started — the decoy in init, and every account these tests provision.
func derivationsDuring(body func()) uint64 {
	before := users.Derivations().Completed
	body()
	return users.Derivations().Completed - before
}

// TestAUsernameMatchingNoAccountPerformsExactlyOneDerivationAndRefuses is
// ADR-0006 rule 1 as a property of the code.
//
// Exactly one, and not "at least one": the decoy is there so that this refusal
// costs the same as a wrong password, and a path that verified the decoy twice
// would be as distinguishable as one that skipped it, in the other direction.
func TestAUsernameMatchingNoAccountPerformsExactlyOneDerivationAndRefuses(t *testing.T) {
	accounts := newFakeAccounts()
	login := users.NewLogin(accounts, fixedClock{})

	var err error
	spent := derivationsDuring(func() {
		_, err = login.Authenticate(context.Background(), "nobody", users.NewPlaintext(thePassword))
	})

	if !errors.Is(err, users.ErrCredentialsRefused) {
		t.Errorf("authenticating a username nobody has returned %v, want ErrCredentialsRefused", err)
	}
	if spent != 1 {
		t.Errorf("a username matching no account spent %d derivations, want exactly 1 — "+
			"the decoy is what makes this refusal cost what a wrong password costs (ADR-0006 rule 1)", spent)
	}
	if len(accounts.outcomes) != 0 {
		t.Errorf("a username matching no account recorded %v, want no login outcome at all: "+
			"there is no account to record one against", accounts.outcomes)
	}
}

// TestAWrongPasswordSpendsExactlyOneDerivationToo is the other half of the pair
// above. Without it, "exactly one" is a number with nothing to equal.
//
// This is a *count* and not a clock. The wall-clock half — nine of each, medians
// compared — is T6's, and is the whole of ADR-0006's evidence; this assertion
// cannot see a derivation that ran at the wrong parameters, and does not claim
// to.
func TestAWrongPasswordSpendsExactlyOneDerivationToo(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	accounts.add(t, "u1", "Alice", users.DefaultPolicy(), &password)
	login := users.NewLogin(accounts, fixedClock{})

	var err error
	spent := derivationsDuring(func() {
		_, err = login.Authenticate(context.Background(), "alice", users.NewPlaintext("not the password"))
	})

	if !errors.Is(err, users.ErrCredentialsRefused) {
		t.Errorf("a wrong password returned %v, want ErrCredentialsRefused", err)
	}
	if spent != 1 {
		t.Errorf("a wrong password spent %d derivations, want exactly 1", spent)
	}
}

// TestADisabledAccountRefusesWithNoDerivationAtAll is ADR-0006's "no
// equalisation is owed there either", written as a test.
//
// The refusal already discloses itself by answering 403 where the credential
// refusals answer 401, so spending 52 ms and a 64 MiB reservation to hide a
// fact the status announces would be work that protects nothing — and the
// reservation is one an unauthenticated caller can pull.
//
// Both passwords are tried, because spec 3.3 makes a disabled account 403
// "whether the password is right or wrong": a build that verified first and
// checked the flag afterwards would answer the same status and spend a
// derivation, and only the count can see it.
func TestADisabledAccountRefusesWithNoDerivationAtAll(t *testing.T) {
	disabled := users.DefaultPolicy()
	disabled.IsDisabled = true

	for _, presented := range []struct {
		name     string
		password string
	}{
		{"the right password", thePassword},
		{"a wrong password", "not the password"},
	} {
		t.Run(presented.name, func(t *testing.T) {
			accounts := newFakeAccounts()
			password := thePassword
			accounts.add(t, "u1", "Alice", disabled, &password)
			login := users.NewLogin(accounts, fixedClock{})

			var err error
			spent := derivationsDuring(func() {
				_, err = login.Authenticate(context.Background(), "alice", users.NewPlaintext(presented.password))
			})

			if !errors.Is(err, users.ErrAccountDisabled) {
				t.Errorf("a disabled account returned %v, want ErrAccountDisabled", err)
			}
			if spent != 0 {
				t.Errorf("a disabled account spent %d derivations, want none at all", spent)
			}
			if len(accounts.outcomes) != 0 {
				t.Errorf("a disabled account recorded %v, want no login outcome: no attempt was decided",
					accounts.outcomes)
			}
		})
	}
}

// TestALockedOutAccountRefusesWithNoDerivationAtAll drives an account into
// lockout through the path and then measures the attempt afterwards.
//
// "Locked out" is not seeded here on purpose. It is not a column and not a
// state an operator sets: it is what the lockout transition leaves behind, and
// a test that hand-built it would be asserting over a state this server cannot
// reach.
func TestALockedOutAccountRefusesWithNoDerivationAtAll(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	accounts.add(t, "u1", "Alice", policyWithThreshold(2), &password)
	login := users.NewLogin(accounts, fixedClock{})
	ctx := context.Background()

	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
			t.Fatalf("failure %d returned %v, want ErrCredentialsRefused", attempt, err)
		}
	}
	if !accounts.isDisabled(t, "u1") {
		t.Fatalf("two failures against a threshold of 2 did not lock the account")
	}

	var err error
	spent := derivationsDuring(func() {
		_, err = login.Authenticate(ctx, "alice", users.NewPlaintext(thePassword))
	})

	if !errors.Is(err, users.ErrAccountDisabled) {
		t.Errorf("a locked-out account returned %v, want ErrAccountDisabled", err)
	}
	if spent != 0 {
		t.Errorf("a locked-out account spent %d derivations, want none at all", spent)
	}
}

// TestTheAttemptAfterALockoutIsRefusedAsDisabled is the design consequence
// worth preserving rather than smoothing over.
//
// plan 6.7's table has a row for a disabled account and a row for a locked-out
// one, and they are the same state on the second try: the lockout *is* the
// disabled flag being set. So there is no distinct "locked" answer, and this
// test asserts the absence — a build that produced one would be answering
// something the design does not produce, and a test that expected one would
// have to invent it.
//
// The correct password is presented, which is the part that matters: it is
// refused, and it is refused as disabled rather than as a credential.
func TestTheAttemptAfterALockoutIsRefusedAsDisabled(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	accounts.add(t, "u1", "Alice", policyWithThreshold(1), &password)
	login := users.NewLogin(accounts, fixedClock{})
	ctx := context.Background()

	if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
		t.Fatalf("the locking attempt returned %v, want ErrCredentialsRefused — "+
			"the attempt that reaches the threshold is still a wrong password", err)
	}
	if got := accounts.outcomes; len(got) != 1 || got[0] != ports.LoginLockedOut {
		t.Fatalf("the locking attempt recorded %v, want exactly one LoginLockedOut", got)
	}

	_, err := login.Authenticate(ctx, "alice", users.NewPlaintext(thePassword))
	if !errors.Is(err, users.ErrAccountDisabled) {
		t.Errorf("the attempt after a lockout returned %v, want ErrAccountDisabled", err)
	}
	if errors.Is(err, users.ErrCredentialsRefused) {
		t.Errorf("the attempt after a lockout is a credential refusal, so a client would be told to ask again")
	}
	if len(accounts.outcomes) != 1 {
		t.Errorf("the attempt after a lockout recorded %v, want the lockout and nothing since",
			accounts.outcomes)
	}
}

// TestAThresholdOfMinusOneNeverLocks is the first of the two readings a plain
// threshold comparison gets exactly backwards.
//
// -1 is what the reference sends for a default account
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], so it
// is what almost every account carries, and it means *never lock*
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @
// v10.11.11]. A build comparing a failure count against the number directly
// would lock every stock account on its first typo, permanently, and would be
// found by a user rather than by a test.
//
// The account has no password, so each of the fifty failures costs no
// derivation — which is ADR-0006 rule 4's deliberate non-equalisation, asserted
// here in passing, and which keeps fifty attempts from costing fifty times
// 52 ms. The failure transition is the same one either way: the store is handed
// an outcome and never asked how the attempt failed.
func TestAThresholdOfMinusOneNeverLocks(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.add(t, "u1", "Alice", policyWithThreshold(-1), nil)
	login := users.NewLogin(accounts, fixedClock{})
	ctx := context.Background()

	spent := derivationsDuring(func() {
		for attempt := 1; attempt <= 50; attempt++ {
			if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("a password this account does not have")); !errors.Is(err, users.ErrCredentialsRefused) {
				t.Fatalf("failure %d returned %v, want ErrCredentialsRefused", attempt, err)
			}
		}
	})

	if spent != 0 {
		t.Errorf("fifty failures against an account with no password spent %d derivations, want none "+
			"(ADR-0006 rule 4)", spent)
	}
	if accounts.isDisabled(t, "u1") {
		t.Fatalf("an account whose LoginAttemptsBeforeLockout is -1 locked out; -1 is the sentinel for never")
	}
	for i, outcome := range accounts.outcomes {
		if outcome != ports.LoginFailed {
			t.Fatalf("failure %d recorded %s, want LoginFailed for every one of the fifty", i+1, outcome)
		}
	}
	if got := accounts.byID["u1"].user.InvalidLoginAttemptCount; got != 50 {
		t.Errorf("the counter is %d after fifty failures, want 50 — it still counts, it just never trips", got)
	}

	// And the account still works, which is the half a lockout would have taken
	// away.
	if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("")); err != nil {
		t.Errorf("the account is still refused after fifty failures: %v", err)
	}
}

// TestAThresholdOfZeroLocksAfterThree is the second reading a plain comparison
// gets backwards, and it is backwards in the opposite direction: 0 does not
// mean "no attempts allowed", it means three.
//
// The reference's own comment on the line is the whole explanation — a client
// sends the default as a zero
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @
// v10.11.11].
func TestAThresholdOfZeroLocksAfterThree(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	accounts.add(t, "u1", "Alice", policyWithThreshold(0), &password)
	login := users.NewLogin(accounts, fixedClock{})
	ctx := context.Background()

	for attempt := 1; attempt <= 2; attempt++ {
		if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
			t.Fatalf("failure %d returned %v, want ErrCredentialsRefused", attempt, err)
		}
		if accounts.isDisabled(t, "u1") {
			t.Fatalf("the account locked after %d failures, want three", attempt)
		}
	}

	if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
		t.Fatalf("the third failure returned %v, want ErrCredentialsRefused", err)
	}
	if !accounts.isDisabled(t, "u1") {
		t.Errorf("three failures against a threshold of 0 did not lock the account; 0 means three")
	}
	if got := accounts.outcomes; len(got) != 3 || got[2] != ports.LoginLockedOut {
		t.Errorf("the three attempts recorded %v, want the third to be LoginLockedOut", got)
	}
}

// TestAThresholdOfTwoLocksAfterTwoFailuresAndASuccessResetsTheCount is the
// ordinary reading, and the reset is the half that a counter which only ever
// went up would pass without.
func TestAThresholdOfTwoLocksAfterTwoFailuresAndASuccessResetsTheCount(t *testing.T) {
	password := thePassword

	t.Run("two failures lock", func(t *testing.T) {
		accounts := newFakeAccounts()
		accounts.add(t, "u1", "Alice", policyWithThreshold(2), &password)
		login := users.NewLogin(accounts, fixedClock{})
		ctx := context.Background()

		for attempt := 1; attempt <= 2; attempt++ {
			if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
				t.Fatalf("failure %d returned %v, want ErrCredentialsRefused", attempt, err)
			}
		}
		if !accounts.isDisabled(t, "u1") {
			t.Errorf("two failures against a threshold of 2 did not lock the account")
		}
	})

	t.Run("a success between them resets the count", func(t *testing.T) {
		accounts := newFakeAccounts()
		accounts.add(t, "u1", "Alice", policyWithThreshold(2), &password)
		login := users.NewLogin(accounts, fixedClock{})
		ctx := context.Background()

		if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
			t.Fatalf("the first failure returned %v, want ErrCredentialsRefused", err)
		}
		if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext(password)); err != nil {
			t.Fatalf("the correct password returned %v", err)
		}
		if got := accounts.byID["u1"].user.InvalidLoginAttemptCount; got != 0 {
			t.Fatalf("the counter is %d after a success, want 0", got)
		}
		if _, err := login.Authenticate(ctx, "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
			t.Fatalf("the failure after the success returned %v, want ErrCredentialsRefused", err)
		}
		if accounts.isDisabled(t, "u1") {
			t.Errorf("one failure either side of a success locked the account: the success did not reset the count")
		}
	})
}

// TestASuccessfulLoginAnswersTheAccountAsTheStoreHoldsItAfterwards.
//
// The response carries InvalidLoginAttemptCount and LastLoginDate, and both
// move during the login that returns them. Answering the copy read before the
// transition would report a counter the store no longer holds.
func TestASuccessfulLoginAnswersTheAccountAsTheStoreHoldsItAfterwards(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	policy := users.DefaultPolicy()
	policy.InvalidLoginAttemptCount = 2
	entry := accounts.add(t, "u1", "Alice", policy, &password)
	entry.user.InvalidLoginAttemptCount = 2
	login := users.NewLogin(accounts, fixedClock{})

	user, err := login.Authenticate(context.Background(), "ALICE", users.NewPlaintext(password))
	if err != nil {
		t.Fatalf("the correct password returned %v", err)
	}
	if user.Username != "Alice" {
		t.Errorf("the login answered %q, want the spelling the operator chose — "+
			"the fold is a store key and never the wire value", user.Username)
	}
	if user.InvalidLoginAttemptCount != 0 {
		t.Errorf("the answered account carries %d failed attempts, want 0", user.InvalidLoginAttemptCount)
	}
	if user.LastLoginAt == nil || !user.LastLoginAt.Equal(aFixedInstant) {
		t.Errorf("the answered account carries LastLoginAt %v, want the clock's instant", user.LastLoginAt)
	}
	if got := accounts.outcomes; len(got) != 1 || got[0] != ports.LoginSucceeded {
		t.Errorf("a successful login recorded %v, want exactly one LoginSucceeded", got)
	}
}

// TestARecordBelowTheCurrentConstantsIsRehashedInsideTheSameCall is plan 6.4
// rule 5.
//
// Inside the call, because ADR-0006 makes the successful login the only moment
// the plaintext exists: a rehash queued behind the response has nothing to
// derive from. Verify only *reports* needsRehash — the re-derivation is the
// login path's, and nothing in T3's suite would notice if it were skipped.
func TestARecordBelowTheCurrentConstantsIsRehashedInsideTheSameCall(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.add(t, "u1", "Alice", users.DefaultPolicy(), nil)
	weak := recordAt(8*1024, 1, 1, 16, 32).String()
	accounts.byID["u1"].credential = &ports.Credential{UserID: "u1", PHC: weak, WrittenAt: aFixedInstant}
	login := users.NewLogin(accounts, fixedClock{})

	if _, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext(thePassword)); err != nil {
		t.Fatalf("the correct password against an old record returned %v", err)
	}
	if accounts.replacedCredentials != 1 {
		t.Fatalf("the login replaced %d credentials, want exactly 1", accounts.replacedCredentials)
	}

	rehashed := accounts.byID["u1"].credential.PHC
	if rehashed == weak {
		t.Fatalf("the stored record did not change")
	}
	assertParametersAreTheCurrentConstants(t, rehashed, "the rehashed record")
	if ok, needsRehash, err := users.Verify(rehashed, users.NewPlaintext(thePassword)); err != nil || !ok || needsRehash {
		t.Errorf("the rehashed record verified as (%t, %t, %v), want (true, false, nil)", ok, needsRehash, err)
	}

	// And a record already at the constants is left alone, which is what makes
	// the assertion above about a rehash rather than about a write on every
	// login.
	before := accounts.replacedCredentials
	if _, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext(thePassword)); err != nil {
		t.Fatalf("the second login returned %v", err)
	}
	if accounts.replacedCredentials != before {
		t.Errorf("a record already at the current constants was rewritten anyway")
	}
}

// TestAnAccountWithNoPasswordAdmitsAnEmptyPasswordAndNothingElse is ADR-0006
// rule 4's account.
func TestAnAccountWithNoPasswordAdmitsAnEmptyPasswordAndNothingElse(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.add(t, "u1", "Alice", users.DefaultPolicy(), nil)
	login := users.NewLogin(accounts, fixedClock{})

	var err error
	spent := derivationsDuring(func() {
		_, err = login.Authenticate(context.Background(), "alice", users.NewPlaintext(""))
	})
	if err != nil {
		t.Errorf("an empty password against an account with no password returned %v, want a success", err)
	}
	if spent != 0 {
		t.Errorf("an account with no password spent %d derivations, want none: it is deliberately not equalised", spent)
	}

	if _, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext("anything")); !errors.Is(err, users.ErrCredentialsRefused) {
		t.Errorf("a non-empty password against an account with no password returned %v, want ErrCredentialsRefused", err)
	}
}

// TestAnUnreadableStoredCredentialIsNotAWrongPassword.
//
// Verify reports a record this package could not have written as an error, and
// answering that with the credential refusal would tell a client to throw away
// a password that was fine, over a fault in the store. plan 7 answers it 500 —
// which is only reachable if the login path passes the error out instead of
// folding it into a refusal.
func TestAnUnreadableStoredCredentialIsNotAWrongPassword(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.add(t, "u1", "Alice", users.DefaultPolicy(), nil)
	accounts.byID["u1"].credential = &ports.Credential{UserID: "u1", PHC: "not a PHC record", WrittenAt: aFixedInstant}
	login := users.NewLogin(accounts, fixedClock{})

	_, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext(thePassword))
	if err == nil {
		t.Fatalf("an unreadable stored credential authenticated")
	}
	if errors.Is(err, users.ErrCredentialsRefused) || errors.Is(err, users.ErrAccountDisabled) {
		t.Errorf("an unreadable stored credential returned %v, want neither refusal", err)
	}
	if !errors.Is(err, users.ErrMalformedRecord) {
		t.Errorf("the error is %v, want it to carry ErrMalformedRecord so a reader can tell what broke", err)
	}
	if len(accounts.outcomes) != 0 {
		t.Errorf("an unreadable stored credential recorded %v, want no login outcome: "+
			"no attempt was decided, and counting one would lock an account out over a corrupt row",
			accounts.outcomes)
	}
}

// TestAStoreFailureWhileAuthenticatingIsNotARefusal is plan 7's rule: a client
// told its credential is wrong discards one that was fine.
func TestAStoreFailureWhileAuthenticatingIsNotARefusal(t *testing.T) {
	accounts := newFakeAccounts()
	accounts.lookupErr = errors.New("the store is unreadable")
	login := users.NewLogin(accounts, fixedClock{})

	_, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext(thePassword))
	if errors.Is(err, users.ErrCredentialsRefused) || errors.Is(err, users.ErrAccountDisabled) {
		t.Errorf("a store failure returned %v, want neither refusal", err)
	}
	if !errors.Is(err, accounts.lookupErr) {
		t.Errorf("the store's own error did not survive: %v", err)
	}
}

// TestTheLockoutThresholdIsReadFromTheColumnAndNotFromTheDocument is the
// overlay rule (plan 4, 6.6) at the one place where getting it wrong locks
// somebody out.
//
// InvalidLoginAttemptCount lives in its own column because it moves on every
// failure; whatever the stored document carries for it is stale by
// construction. A path that read the count off the document would compare a
// fresh attempt against whatever number the document was written with — here,
// zero — and would never lock at all.
func TestTheLockoutThresholdIsReadFromTheColumnAndNotFromTheDocument(t *testing.T) {
	accounts := newFakeAccounts()
	password := thePassword
	policy := policyWithThreshold(3)
	// The document says no failures; the column says two. The column wins.
	policy.InvalidLoginAttemptCount = 0
	accounts.add(t, "u1", "Alice", policy, &password)
	accounts.byID["u1"].user.InvalidLoginAttemptCount = 2
	login := users.NewLogin(accounts, fixedClock{})

	if _, err := login.Authenticate(context.Background(), "alice", users.NewPlaintext("wrong")); !errors.Is(err, users.ErrCredentialsRefused) {
		t.Fatalf("the third failure returned %v, want ErrCredentialsRefused", err)
	}
	if !accounts.isDisabled(t, "u1") {
		t.Errorf("the third failure against a threshold of 3 did not lock the account: " +
			"the count was read from the stored document rather than from its column")
	}
}
