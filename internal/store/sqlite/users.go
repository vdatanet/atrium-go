package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// The store implements what the account domain asked for. The assertion is here
// rather than in a test for the reason installation.go's is: it is what makes
// the interface load-bearing while the handlers that will call it are still
// unwritten.
var _ ports.UserStore = (*Store)(nil)

// userColumns is the SELECT list every read of an account shares, in the order
// scanUser reads them.
//
// It is one constant rather than three spellings because the three reads
// differ only in their WHERE clause, and a column list that drifted between
// them would be a field silently filled from the wrong position — which
// compiles, and which no reader notices until a username turns up in a policy.
//
// The credential is not in it, and cannot be: 002 plan 4 puts the verifier in a
// table of its own so that no read of a user object holds a password record.
const userColumns = `id, username, username_folded, policy_document, configuration_document,
                     invalid_login_attempt_count, last_login_at, last_activity_at`

// scanUser reads one row of userColumns.
func scanUser(row interface{ Scan(...any) error }) (ports.User, error) {
	var (
		user           ports.User
		lastLogin      sql.Null[int64]
		lastActivity   sql.Null[int64]
		policy         []byte
		configuration  []byte
		attemptCounter int64
	)
	if err := row.Scan(
		&user.ID, &user.Username, &user.UsernameFolded, &policy, &configuration,
		&attemptCounter, &lastLogin, &lastActivity,
	); err != nil {
		return ports.User{}, err
	}
	user.PolicyDocument = policy
	user.ConfigurationDocument = configuration
	user.InvalidLoginAttemptCount = int(attemptCounter)
	user.LastLoginAt = nullableTime(lastLogin)
	user.LastActivityAt = nullableTime(lastActivity)
	return user, nil
}

// nullableTime turns a nullable tick column into the pointer ports.User
// carries.
//
// NULL is an absence rather than the minimum date, and the distinction is
// observable: spec 3.5 makes LastLoginDate *absent* until the first login, and
// a column read as the zero tick would answer 0001-01-01T00:00:00.0000000Z
// instead — a date, where the reference sends no member at all.
func nullableTime(column sql.Null[int64]) *units.Time {
	if !column.Valid {
		return nil
	}
	at := units.TimeFromTicks(units.Ticks(column.V))
	return &at
}

// CreateUser writes a new account.
//
// It writes every column of the record it is handed, including the two
// nullable dates: a fresh account has never logged in and has never been seen,
// so both are NULL, and NULL there is what makes LastLoginDate *absent* rather
// than the minimum date (spec 3.5). nullableTicks is what turns a nil pointer
// into that NULL, so an account created with a date already on it — a restore,
// one day — writes the date rather than losing it.
//
// The documents are bound as strings and not as []byte, because the table is
// STRICT and a []byte bound to a TEXT column is refused as a BLOB.
//
// There is no ON CONFLICT clause and there deliberately is none. A second
// account whose folded username is already taken is a mistake, not an update:
// the identifier is derived from that name (002 plan 6.9), so an upsert would
// silently overwrite somebody else's policy, configuration and login history
// with a new account's defaults. The unique index refuses it instead.
func (s *Store) CreateUser(ctx context.Context, user ports.User) error {
	_, err := s.writer.ExecContext(ctx,
		`INSERT INTO users (id, username, username_folded, policy_document,
		                    configuration_document, invalid_login_attempt_count,
		                    last_login_at, last_activity_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.UsernameFolded,
		string(user.PolicyDocument), string(user.ConfigurationDocument),
		int64(user.InvalidLoginAttemptCount),
		nullableTicks(user.LastLoginAt), nullableTicks(user.LastActivityAt))
	if err != nil {
		return fmt.Errorf("%s: creating the account %q: %w", s.path, user.Username, err)
	}
	return nil
}

// nullableTicks is nullableTime's inverse: a nil date is the NULL the column
// holds, and a date is its tick count.
func nullableTicks(at *units.Time) any {
	if at == nil {
		return nil
	}
	return int64(at.Ticks())
}

// UserByFoldedName finds the account whose folded username is folded.
//
// It reads through the reading pool. This is the first thing an authentication
// does, and 002 plan 6.4 puts an Argon2id derivation immediately after it: a
// lookup queued behind whatever is writing would add its wait to a path that is
// already the slowest request this server serves.
func (s *Store) UserByFoldedName(ctx context.Context, folded string) (ports.User, bool, error) {
	return s.userBy(ctx, `username_folded = ?`, folded)
}

// UserByID finds one account by its identifier.
func (s *Store) UserByID(ctx context.Context, id string) (ports.User, bool, error) {
	return s.userBy(ctx, `id = ?`, id)
}

func (s *Store) userBy(ctx context.Context, where string, argument string) (ports.User, bool, error) {
	user, err := scanUser(s.reader.QueryRowContext(ctx,
		`SELECT `+userColumns+` FROM users WHERE `+where, argument))
	if errors.Is(err, sql.ErrNoRows) {
		// An absence, not a failure. It is what an authentication against a
		// username nobody has looks like, and 002 plan 6.4 answers it by
		// verifying the decoy rather than by reporting an error.
		return ports.User{}, false, nil
	}
	if err != nil {
		return ports.User{}, false, fmt.Errorf("%s: reading an account: %w", s.path, err)
	}
	return user, true, nil
}

// Users returns every account.
//
// The order is username_folded and then id, stated here because architecture 2
// forbids an order that derives from anything but stable input: /Users/Public
// answers an array, L3 compares list rows by position, and an unordered SELECT
// is SQLite's storage order, which is stable until a row is rewritten and then
// is not.
//
// The tie-break on id is not decoration. username_folded is unique today, so
// the pair can never tie — and if the uniqueness ever moved, the order would
// silently become an arbitrary one, which is exactly the failure this ordering
// exists to prevent.
//
// Whether this is the order the *reference* answers /Users/Public in is
// unmeasured and is the route's question, not the store's.
func (s *Store) Users(ctx context.Context) ([]ports.User, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT `+userColumns+` FROM users ORDER BY username_folded, id`)
	if err != nil {
		return nil, fmt.Errorf("%s: reading the accounts: %w", s.path, err)
	}
	defer rows.Close()

	var accounts []ports.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: reading the accounts: %w", s.path, err)
		}
		accounts = append(accounts, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: reading the accounts: %w", s.path, err)
	}
	return accounts, nil
}

// Credential returns the account's password record, if it has one.
func (s *Store) Credential(ctx context.Context, userID string) (ports.Credential, bool, error) {
	credential := ports.Credential{UserID: userID}
	var writtenAt int64
	err := s.reader.QueryRowContext(ctx,
		`SELECT phc, written_at FROM user_credentials WHERE user_id = ?`, userID,
	).Scan(&credential.PHC, &writtenAt)
	if errors.Is(err, sql.ErrNoRows) {
		// An account with no password, which spec 3.5 reports as HasPassword
		// false. It is a state an operator can provision, so it is an answer
		// and not an error.
		return ports.Credential{}, false, nil
	}
	if err != nil {
		return ports.Credential{}, false, fmt.Errorf("%s: reading a credential: %w", s.path, err)
	}
	credential.WrittenAt = units.TimeFromTicks(units.Ticks(writtenAt))
	return credential, true, nil
}

// ReplaceCredential writes the account's password record, replacing any it
// already had.
//
// The upsert is what makes a rehash-on-successful-login one call rather than a
// branch on whether the account had a password: 002 plan 6.4 rule 5 re-derives
// and replaces inside the login, and a caller that had to know which statement
// to run would be a caller that could pick the wrong one on the account whose
// password was just set.
func (s *Store) ReplaceCredential(ctx context.Context, userID string, phc string, at units.Time) error {
	_, err := s.writer.ExecContext(ctx,
		`INSERT INTO user_credentials (user_id, phc, written_at) VALUES (?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET phc = excluded.phc, written_at = excluded.written_at`,
		userID, phc, int64(at.Ticks()))
	if err != nil {
		// The account's own row is what the foreign key protects, so an
		// identifier nobody has fails here rather than writing a credential
		// nothing can reach.
		return fmt.Errorf("%s: writing a credential: %w", s.path, err)
	}
	return nil
}

// ReplaceConfiguration writes the account's configuration document whole.
func (s *Store) ReplaceConfiguration(ctx context.Context, userID string, document []byte) error {
	return s.updateOneUser(ctx, "writing a configuration", userID,
		`UPDATE users SET configuration_document = ? WHERE id = ?`, string(document), userID)
}

// TouchActivity records that the account was seen at at.
func (s *Store) TouchActivity(ctx context.Context, userID string, at units.Time) error {
	return s.updateOneUser(ctx, "recording activity", userID,
		`UPDATE users SET last_activity_at = ? WHERE id = ?`, int64(at.Ticks()), userID)
}

// updateOneUser runs a statement that must change exactly one account, and
// reports it when it did not.
//
// The guard is 001's, for the reason installation.go gives: an UPDATE that
// matched nothing succeeds, so without it every write against an identifier
// nobody has looks exactly like a write that worked.
func (s *Store) updateOneUser(ctx context.Context, what, userID, statement string, arguments ...any) error {
	result, err := s.writer.ExecContext(ctx, statement, arguments...)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: %s changed %d rows for account %s, want 1", s.path, what, affected, userID)
	}
	return nil
}

// RecordLoginOutcome applies one authentication attempt's whole effect on the
// account.
//
// It is one method and one transaction for the reason 002 plan 5 gives, and the
// third outcome is what makes that worth anything: a lockout increments the
// counter *and* sets IsDisabled in the stored policy document, and a build that
// did one of the two would pass any test written per field. There is no
// is_disabled column — 002 plan 4 keeps the flag in the document, so the
// lockout is a read, a decode, a change and an encode, and all four have to
// happen inside the same transaction as the counter or a concurrent failed
// login can write the document back without the flag.
//
// The decode goes through users.DecodePolicy rather than through
// json.Unmarshal, which is the one place this package reaches into the domain.
// It has to: a document written by an older build must decode onto the
// reference's defaults and never onto Go's zero value (002 plan 4), and
// unmarshalling into a users.Policy{} here would silently rewrite every
// property the stored document does not carry — turning EnableMediaPlayback
// false and LoginAttemptsBeforeLockout into the sentinel that means "lock after
// three attempts". A round-trip test over a complete document would not see it.
func (s *Store) RecordLoginOutcome(ctx context.Context, userID string, outcome ports.LoginOutcome, at units.Time) error {
	switch outcome {
	case ports.LoginSucceeded:
		// A success is the whole transition in one statement: the counter goes
		// back to zero and the login date is stamped. The policy is untouched,
		// because a success never disables an account.
		return s.updateOneUser(ctx, "recording a successful login", userID,
			`UPDATE users SET invalid_login_attempt_count = 0, last_login_at = ? WHERE id = ?`,
			int64(at.Ticks()), userID)

	case ports.LoginFailed:
		// A failure moves the counter and nothing else — not the login date,
		// which spec 3.5 makes the date of a login that worked, and not the
		// policy.
		return s.updateOneUser(ctx, "recording a failed login", userID,
			`UPDATE users SET invalid_login_attempt_count = invalid_login_attempt_count + 1 WHERE id = ?`,
			userID)

	case ports.LoginLockedOut:
		return s.recordLockout(ctx, userID)

	default:
		// The zero value lands here, which is the point of it being invalid:
		// there is no reading of "no outcome" that is safe to guess, and a
		// caller that forgot to say gets told so rather than getting whichever
		// transition looked harmless.
		return fmt.Errorf("%s: recording a login outcome for account %s: the outcome is %s",
			s.path, userID, outcome)
	}
}

// recordLockout is the third outcome: the failed attempt that reached the
// threshold.
func (s *Store) recordLockout(ctx context.Context, userID string) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: recording a lockout for account %s: %w", s.path, userID, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	// Rollback after a commit is a no-op, so this is the whole of the "or
	// none of it" half of the contract: every return between here and the
	// commit leaves the account exactly as it was.
	defer transaction.Rollback()

	var document []byte
	err = transaction.QueryRowContext(ctx,
		`SELECT policy_document FROM users WHERE id = ?`, userID).Scan(&document)
	if errors.Is(err, sql.ErrNoRows) {
		return failed(errors.New("there is no such account"))
	}
	if err != nil {
		return failed(err)
	}

	policy, err := users.DecodePolicy(document)
	if err != nil {
		return failed(err)
	}
	policy.IsDisabled = true

	// InvalidLoginAttemptCount is deliberately left as the decode produced it.
	// The column is the value, the document's copy is stale by construction,
	// and the user object overlays the column over whatever this writes
	// (002 plan 4, 002 plan 6.6). Writing the counter into the document here
	// would create a second answer to a question that already has one.
	updated, err := policy.Document()
	if err != nil {
		return failed(err)
	}

	// One statement for both halves of the transition. Splitting it would put
	// the counter and the flag in two statements that a future edit could
	// separate, inside a transaction whose whole purpose is that they cannot
	// be.
	result, err := transaction.ExecContext(ctx,
		`UPDATE users SET invalid_login_attempt_count = invalid_login_attempt_count + 1,
		                  policy_document = ?
		 WHERE id = ?`, string(updated), userID)
	if err != nil {
		return failed(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return failed(err)
	}
	if affected != 1 {
		return failed(fmt.Errorf("the update changed %d rows, want 1", affected))
	}

	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}
