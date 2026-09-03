package sqlite

import (
	"bytes"
	"context"
	"database/sql"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// insertUserWithPolicy writes one account with a chosen policy document and a
// chosen failed-attempt counter, which is what the transition tests need and
// what T1's insertUser could not express.
//
// The policy document is deliberately a caller's choice rather than
// DefaultPolicy().Document(): the lockout test seeds a *partial* document, and
// a helper that always wrote a complete one would have hidden the rule the
// lockout depends on — that a stored document decodes onto the reference's
// defaults and never onto Go's zero value.
func insertUserWithPolicy(t *testing.T, db *sql.DB, id, username, folded string, policy []byte, attempts int) error {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO users (id, username, username_folded, policy_document, configuration_document,
		                    invalid_login_attempt_count, last_login_at, last_activity_at)
		 VALUES (?, ?, ?, ?, '{}', ?, NULL, NULL)`,
		id, username, folded, string(policy), attempts)
	return err
}

// aLoginInstant is a fixed instant, so that a stamped date is compared with a
// value the test chose rather than with whatever a clock returned.
var aLoginInstant = units.TimeFromTicks(638_000_000_000_000_000)

// mustReadUser reads an account back through the port, which is where the
// transition tests do their asserting: a test that read the columns with its
// own SQL would prove that a statement ran and not that the account changed.
func mustReadUser(t *testing.T, store *Store, id string) ports.User {
	t.Helper()
	user, found, err := store.UserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("UserByID returned %v", err)
	}
	if !found {
		t.Fatalf("UserByID did not find account %s", id)
	}
	return user
}

// TestRecordLoginOutcomeIsOneTransition is T4's first clause, and the three
// cases are one assertion each about a transition rather than about a field.
//
// The third case is what makes the other two worth anything. 002 plan 5 makes
// this one method rather than "increment the counter", "reset the counter",
// "set the disabled flag" and "stamp the login date" precisely because four
// callers would be four chances to perform three quarters of the transition —
// and a build that performed three quarters passes any test written per field.
// So the lockout case asserts the counter *and* the flag *and* the properties
// the document already carried, together.
//
// There is no is_disabled column, deliberately (002 plan 4): the flag lives in
// policy_document, so the lockout is a read, a decode, a change and an encode
// inside the same transaction as the counter.
func TestRecordLoginOutcomeIsOneTransition(t *testing.T) {
	ctx := context.Background()

	// A partial document, holding two properties whose stored values are not
	// the reference's defaults — IsAdministrator defaults to false and
	// LoginAttemptsBeforeLockout to -1. Everything the document omits has to
	// come back as the reference's default and not as Go's zero value, which is
	// what makes this a test of the decode rule and not only of the flag.
	storedPolicy := []byte(`{"IsAdministrator":true,"LoginAttemptsBeforeLockout":2}`)

	t.Run("a failure increments the counter and moves nothing else", func(t *testing.T) {
		store := openForTest(t)
		if err := insertUserWithPolicy(t, store.writer, testUserID, "Alice", "alice", storedPolicy, 0); err != nil {
			t.Fatalf("inserting the account returned %v", err)
		}
		before := mustReadUser(t, store, testUserID)

		if err := store.RecordLoginOutcome(ctx, testUserID, ports.LoginFailed, aLoginInstant); err != nil {
			t.Fatalf("RecordLoginOutcome(failed) returned %v", err)
		}

		after := mustReadUser(t, store, testUserID)
		if after.InvalidLoginAttemptCount != 1 {
			t.Errorf("the counter is %d after one failure, want 1", after.InvalidLoginAttemptCount)
		}
		// "Moves nothing else" is the half a per-field test would not have
		// written. A failed login is not a login: spec 3.5 makes LastLoginDate
		// the date of one that worked, so it is still absent here.
		if after.LastLoginAt != nil {
			t.Errorf("a failed login stamped last_login_at as %v, want it still absent", after.LastLoginAt)
		}
		if after.LastActivityAt != nil {
			t.Errorf("a failed login stamped last_activity_at as %v, want it still absent", after.LastActivityAt)
		}
		if !bytes.Equal(after.PolicyDocument, before.PolicyDocument) {
			t.Errorf("a failed login rewrote the policy document as %s, want %s unchanged",
				after.PolicyDocument, before.PolicyDocument)
		}
		if !bytes.Equal(after.ConfigurationDocument, before.ConfigurationDocument) {
			t.Errorf("a failed login rewrote the configuration document as %s, want %s unchanged",
				after.ConfigurationDocument, before.ConfigurationDocument)
		}
	})

	t.Run("a success resets the counter and stamps the login date", func(t *testing.T) {
		store := openForTest(t)
		// Three failures already recorded, so that a reset is observable as a
		// change rather than as a value that was already zero.
		if err := insertUserWithPolicy(t, store.writer, testUserID, "Alice", "alice", storedPolicy, 3); err != nil {
			t.Fatalf("inserting the account returned %v", err)
		}

		if err := store.RecordLoginOutcome(ctx, testUserID, ports.LoginSucceeded, aLoginInstant); err != nil {
			t.Fatalf("RecordLoginOutcome(succeeded) returned %v", err)
		}

		after := mustReadUser(t, store, testUserID)
		if after.InvalidLoginAttemptCount != 0 {
			t.Errorf("the counter is %d after a success, want 0", after.InvalidLoginAttemptCount)
		}
		if after.LastLoginAt == nil {
			t.Fatal("a successful login left last_login_at absent, want it stamped")
		}
		if !after.LastLoginAt.Equal(aLoginInstant) {
			t.Errorf("last_login_at is %s, want %s", after.LastLoginAt, aLoginInstant)
		}
		if !bytes.Equal(after.PolicyDocument, storedPolicy) {
			t.Errorf("a successful login rewrote the policy document as %s, want %s unchanged",
				after.PolicyDocument, storedPolicy)
		}
	})

	t.Run("reaching the threshold sets IsDisabled in the stored policy document", func(t *testing.T) {
		store := openForTest(t)
		if err := insertUserWithPolicy(t, store.writer, testUserID, "Alice", "alice", storedPolicy, 2); err != nil {
			t.Fatalf("inserting the account returned %v", err)
		}

		if err := store.RecordLoginOutcome(ctx, testUserID, ports.LoginLockedOut, aLoginInstant); err != nil {
			t.Fatalf("RecordLoginOutcome(locked out) returned %v", err)
		}

		after := mustReadUser(t, store, testUserID)

		// One transition, asserted as a conjunction. Each of the four
		// assertions below is a quarter a wrong build could skip, and every
		// one of them passes on a build that skipped a different quarter.
		if after.InvalidLoginAttemptCount != 3 {
			t.Errorf("the counter is %d after the failure that locked the account, want 3: "+
				"the lockout is still a failed attempt", after.InvalidLoginAttemptCount)
		}
		if after.LastLoginAt != nil {
			t.Errorf("the lockout stamped last_login_at as %v, want it still absent", after.LastLoginAt)
		}

		policy, err := users.DecodePolicy(after.PolicyDocument)
		if err != nil {
			t.Fatalf("decoding the stored policy returned %v", err)
		}
		if !policy.IsDisabled {
			t.Error("IsDisabled is false in the stored policy document after a lockout, want true: " +
				"there is no is_disabled column, so this document is where the lockout lives")
		}
		// The document was partial, and the properties it did and did not
		// carry both have to survive. A build that wrote DefaultPolicy() with
		// IsDisabled set would pass the assertion above and silently answer the
		// reference's defaults for these two — an administrator demoted, and a
		// threshold of 2 turned into the -1 that never locks, by the very
		// lockout that was supposed to enforce it.
		if !policy.IsAdministrator {
			t.Error("IsAdministrator is false after a lockout, want the stored document's own value: " +
				"the lockout is a decode-modify-encode, not an overwrite")
		}
		if policy.LoginAttemptsBeforeLockout != 2 {
			t.Errorf("LoginAttemptsBeforeLockout is %d after a lockout, want the stored 2",
				policy.LoginAttemptsBeforeLockout)
		}
		// And the defaults rule, which the partial document depends on. A
		// decode that started from a zero Policy would answer false here, and
		// the account would come back out of its lockout unable to play
		// anything.
		if !policy.EnableMediaPlayback {
			t.Error("EnableMediaPlayback is false after a lockout, want true: " +
				"a stored document decodes onto the reference's defaults (002 plan 4)")
		}
	})
}

// TestAnUnrecordableLoginOutcomeIsRefused covers the zero value, which is
// deliberately not an outcome.
//
// There is no safe default: counting an attempt nobody made is wrong in one
// direction and clearing a lockout is wrong in the other, so a caller that
// forgot to say is told rather than given whichever reading looked harmless.
func TestAnUnrecordableLoginOutcomeIsRefused(t *testing.T) {
	store := openForTest(t)
	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	if err := store.RecordLoginOutcome(context.Background(), testUserID, ports.LoginOutcome(0), aLoginInstant); err == nil {
		t.Fatal("the zero outcome was accepted, want a refusal")
	}

	after := mustReadUser(t, store, testUserID)
	if after.InvalidLoginAttemptCount != 0 || after.LastLoginAt != nil {
		t.Errorf("the refused outcome still changed the account: counter %d, last login %v",
			after.InvalidLoginAttemptCount, after.LastLoginAt)
	}
}

// TestRecordingALoginOutcomeForNobodyIsAnError covers the other end of every
// write in this file: an identifier nobody has.
//
// An UPDATE that matched no row succeeds in SQL, so without the guard each of
// these writes against a missing account would look exactly like one that
// worked. The lockout takes a different path from the other two — it reads the
// document first — so both are asserted.
func TestRecordingALoginOutcomeForNobodyIsAnError(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()
	const nobody = "ffffffffffffffffffffffffffffffff"

	for _, outcome := range []ports.LoginOutcome{ports.LoginFailed, ports.LoginSucceeded, ports.LoginLockedOut} {
		if err := store.RecordLoginOutcome(ctx, nobody, outcome, aLoginInstant); err == nil {
			t.Errorf("recording %s against an account that does not exist succeeded, want an error", outcome)
		}
	}
}

// TestAnAccountIsFoundByItsFoldedNameAndAnAbsenceIsNotAnError is the read an
// authentication performs.
//
// The absence half is the one worth a test: it is what a login against a
// username nobody has looks like, it happens on this feature's most exposed
// path, and 002 plan 6.4 answers it by verifying the decoy rather than by
// reporting a failure. A port that returned an error there would have made the
// most ordinary refusal in the feature indistinguishable from a broken store.
func TestAnAccountIsFoundByItsFoldedNameAndAnAbsenceIsNotAnError(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	user, found, err := store.UserByFoldedName(ctx, "alice")
	if err != nil || !found {
		t.Fatalf("UserByFoldedName(alice) returned (%v, %t, %v), want the account", user.ID, found, err)
	}
	if user.Username != "Alice" {
		t.Errorf("the account is named %q, want %q: the stored spelling is what Name returns", user.Username, "Alice")
	}
	if user.LastLoginAt != nil {
		t.Errorf("a fresh account has last_login_at %v, want it absent, which is what makes "+
			"LastLoginDate absent on the wire (spec 3.5)", user.LastLoginAt)
	}

	// The folded column is matched as it is stored. Folding is the caller's,
	// which is why this is a miss and not a hit.
	if _, found, err := store.UserByFoldedName(ctx, "ALICE"); err != nil || found {
		t.Errorf("UserByFoldedName(ALICE) returned (%t, %v), want no account and no error: "+
			"the column holds the folded spelling and this method does not fold", found, err)
	}

	if _, found, err := store.UserByFoldedName(ctx, "nobody"); err != nil || found {
		t.Errorf("UserByFoldedName(nobody) returned (%t, %v), want (false, nil)", found, err)
	}
	if _, found, err := store.UserByID(ctx, "ffffffffffffffffffffffffffffffff"); err != nil || found {
		t.Errorf("UserByID for an identifier nobody has returned (%t, %v), want (false, nil)", found, err)
	}
}

// TestUsersAreReturnedInAStatedOrder is architecture 2's rule about order
// applied to the list /Users/Public answers.
//
// The rows are inserted in an order that is neither the answer nor its reverse,
// so a build that returned storage order would have to be lucky twice.
func TestUsersAreReturnedInAStatedOrder(t *testing.T) {
	store := openForTest(t)

	seed := []struct{ id, username, folded string }{
		{"3c59dc048e8850243be8079a5c74d079", "Carol", "carol"},
		{"b6d767d2f8ed5d21a44b0e5886680cb9", "alice", "alice"},
		{"37693cfc748049e45d87b8c7d8b9aacd", "Bob", "bob"},
	}
	for _, row := range seed {
		if err := insertUser(t, store.writer, row.id, row.username, row.folded); err != nil {
			t.Fatalf("inserting %s returned %v", row.username, err)
		}
	}

	accounts, err := store.Users(context.Background())
	if err != nil {
		t.Fatalf("Users returned %v", err)
	}
	var got []string
	for _, account := range accounts {
		got = append(got, account.UsernameFolded)
	}
	want := []string{"alice", "bob", "carol"}
	if len(got) != len(want) {
		t.Fatalf("Users returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Users returned %v, want %v", got, want)
		}
	}
}

// TestACredentialIsWrittenOnceAndReplacedInPlace covers the upsert a
// rehash-on-successful-login depends on, and the absence that is an account
// with no password.
func TestACredentialIsWrittenOnceAndReplacedInPlace(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	// An account with no password is a state an operator can provision, and
	// spec 3.5 reports it as HasPassword false. It is an answer, not an error.
	if _, found, err := store.Credential(ctx, testUserID); err != nil || found {
		t.Errorf("Credential for an account with no password returned (%t, %v), want (false, nil)", found, err)
	}

	const first = "$argon2id$v=19$m=65536,t=3,p=2$c2FsdHNhbHRzYWx0c2E$a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2V5a2U"
	const second = "$argon2id$v=19$m=131072,t=4,p=2$c2FsdHNhbHRzYWx0c2E$b3RoZXJvdGhlcm90aGVyb3RoZXJvdGhlcm90aGVy"

	if err := store.ReplaceCredential(ctx, testUserID, first, aLoginInstant); err != nil {
		t.Fatalf("ReplaceCredential returned %v", err)
	}
	credential, found, err := store.Credential(ctx, testUserID)
	if err != nil || !found {
		t.Fatalf("Credential returned (%t, %v), want the record", found, err)
	}
	if credential.PHC != first {
		t.Errorf("the stored record is %q, want %q", credential.PHC, first)
	}
	if !credential.WrittenAt.Equal(aLoginInstant) {
		t.Errorf("written_at is %s, want %s", credential.WrittenAt, aLoginInstant)
	}

	// The rehash. It is one call and not a branch on whether the account
	// already had a password, which is what the upsert buys.
	rehashedAt := units.TimeFromTicks(aLoginInstant.Ticks() + units.TicksPerSecond)
	if err := store.ReplaceCredential(ctx, testUserID, second, rehashedAt); err != nil {
		t.Fatalf("replacing the record returned %v", err)
	}
	credential, _, err = store.Credential(ctx, testUserID)
	if err != nil {
		t.Fatalf("Credential returned %v", err)
	}
	if credential.PHC != second {
		t.Errorf("the record after a rehash is %q, want %q", credential.PHC, second)
	}
	if !credential.WrittenAt.Equal(rehashedAt) {
		t.Errorf("written_at after a rehash is %s, want %s: it is the only way to see one happened",
			credential.WrittenAt, rehashedAt)
	}

	// And a record for an account nobody has is refused by the foreign key
	// rather than written where nothing can reach it.
	if err := store.ReplaceCredential(ctx, "ffffffffffffffffffffffffffffffff", first, aLoginInstant); err == nil {
		t.Error("a credential for an account that does not exist was accepted, want a foreign-key refusal")
	}
}

// TestAConfigurationIsReplacedWholeAndActivityIsStamped covers the two
// remaining writers, and the guard that an identifier nobody has is an error
// rather than a write that quietly matched nothing.
func TestAConfigurationIsReplacedWholeAndActivityIsStamped(t *testing.T) {
	store := openForTest(t)
	ctx := context.Background()

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}

	document, err := users.DefaultConfiguration().Document()
	if err != nil {
		t.Fatalf("encoding a configuration returned %v", err)
	}
	if err := store.ReplaceConfiguration(ctx, testUserID, document); err != nil {
		t.Fatalf("ReplaceConfiguration returned %v", err)
	}
	if err := store.TouchActivity(ctx, testUserID, aLoginInstant); err != nil {
		t.Fatalf("TouchActivity returned %v", err)
	}

	after := mustReadUser(t, store, testUserID)
	if !bytes.Equal(after.ConfigurationDocument, document) {
		t.Errorf("the stored configuration is %s, want the document it was given, byte for byte",
			after.ConfigurationDocument)
	}
	if after.LastActivityAt == nil || !after.LastActivityAt.Equal(aLoginInstant) {
		t.Errorf("last_activity_at is %v, want %s", after.LastActivityAt, aLoginInstant)
	}
	// Stamping activity is not logging in.
	if after.LastLoginAt != nil {
		t.Errorf("TouchActivity stamped last_login_at as %v, want it still absent", after.LastLoginAt)
	}

	const nobody = "ffffffffffffffffffffffffffffffff"
	if err := store.ReplaceConfiguration(ctx, nobody, document); err == nil {
		t.Error("a configuration written against an account that does not exist succeeded, want an error")
	}
	if err := store.TouchActivity(ctx, nobody, aLoginInstant); err == nil {
		t.Error("activity stamped against an account that does not exist succeeded, want an error")
	}
}
