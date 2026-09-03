package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/units"
)

var _ ports.SessionStore = (*Store)(nil)

// sessionColumns is the SELECT list every read of a session shares, in the
// order scanSession reads them, and prefixed so that it can be used in the join
// SessionByTokenDigest performs — where user_id and device_id exist on both
// tables and an unqualified name would be ambiguous in one direction and wrong
// in the other.
const sessionColumns = `sessions.id, sessions.user_id, sessions.client, sessions.device_id,
                        sessions.device_name, sessions.application_version, sessions.remote_endpoint,
                        sessions.capabilities_document, sessions.created_at,
                        sessions.last_activity_at, sessions.last_playback_check_in_at`

// scanSession reads one row of sessionColumns, followed by extra destinations
// for whatever else the query selected after them.
//
// The tail is what lets the join below add the token's own user column without
// a second scanning function that could drift from this one.
func scanSession(row interface{ Scan(...any) error }, extra ...any) (ports.Session, error) {
	var (
		session      ports.Session
		capabilities sql.Null[string]
		created      int64
		activity     int64
		checkIn      int64
	)
	destinations := []any{
		&session.ID, &session.UserID, &session.Client, &session.DeviceID,
		&session.DeviceName, &session.ApplicationVersion, &session.RemoteEndpoint,
		&capabilities, &created, &activity, &checkIn,
	}
	if err := row.Scan(append(destinations, extra...)...); err != nil {
		return ports.Session{}, err
	}
	// NULL is "no client has posted a declaration", which is an absence and not
	// an empty declaration: 002 plan 6.10 stores the posted document whole, so
	// the difference between "nothing posted" and "an empty object posted" is
	// one a client can create and this column has to keep.
	if capabilities.Valid {
		session.CapabilitiesDocument = []byte(capabilities.V)
	}
	session.CreatedAt = units.TimeFromTicks(units.Ticks(created))
	session.LastActivityAt = units.TimeFromTicks(units.Ticks(activity))
	session.LastPlaybackCheckInAt = units.TimeFromTicks(units.Ticks(checkIn))
	return session, nil
}

// OpenSession writes the session and the token that opens it, as one statement.
//
// The transaction is the contract and not an optimisation. The token names its
// session by a foreign key, so the session has to be written first — and a
// build that wrote the two without a transaction would, on any failure of the
// second write, leave a session nothing can reach and, worse in the other
// direction, would make "a token exists" stop implying "its session exists" the
// moment the order ever changed. Either both rows are there afterwards or
// neither is.
//
// The session is inserted when it is new and updated when the same client
// authenticates again on the same device, which is spec 3.3's "authenticating
// again from the same DeviceId replaces that session rather than accumulating
// one per login". The update carries exactly what 002 plan 6.5 step 4 names —
// this user, this device name, this version and this remote endpoint — plus the
// activity date, because the login is itself activity. created_at,
// capabilities_document and last_playback_check_in_at are deliberately not
// touched: the session is the same session, and clearing a client's posted
// declaration on every login is a behaviour nothing has measured.
//
// Revoking the tokens the replacement invalidates is RevokeTokensFor's and
// happens before this, because 002 plan 6.5 makes it step three and this step
// four, and because a revocation inside this call would delete the token this
// call is about on the second login from one device.
func (s *Store) OpenSession(ctx context.Context, session ports.Session, tokenDigest string) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: opening session %s: %w", s.path, session.ID, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx,
		`INSERT INTO sessions (id, user_id, client, device_id, device_name, application_version,
		                       remote_endpoint, capabilities_document,
		                       created_at, last_activity_at, last_playback_check_in_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		     user_id             = excluded.user_id,
		     client              = excluded.client,
		     device_id           = excluded.device_id,
		     device_name         = excluded.device_name,
		     application_version = excluded.application_version,
		     remote_endpoint     = excluded.remote_endpoint,
		     last_activity_at    = excluded.last_activity_at`,
		session.ID, session.UserID, session.Client, session.DeviceID,
		session.DeviceName, session.ApplicationVersion, session.RemoteEndpoint,
		nullableDocument(session.CapabilitiesDocument),
		int64(session.CreatedAt.Ticks()), int64(session.LastActivityAt.Ticks()),
		int64(session.LastPlaybackCheckInAt.Ticks()),
	); err != nil {
		return failed(err)
	}

	// The token's created_at is the session's activity date rather than a
	// clock read here. OpenSession takes no instant of its own (002 plan 5),
	// and it does not need one: the authentication that calls this sets
	// LastActivityAt to the instant of the login, which is the instant the
	// token was minted. A store that read a clock would also be a store that
	// could not be held still by a test, which architecture 2 forbids.
	if _, err := transaction.ExecContext(ctx,
		`INSERT INTO access_tokens (token_digest, user_id, session_id, device_id, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		tokenDigest, session.UserID, session.ID, session.DeviceID,
		int64(session.LastActivityAt.Ticks()),
	); err != nil {
		return failed(err)
	}

	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// nullableDocument binds a document column, writing NULL for an absent one.
//
// It returns any rather than a string because the column distinguishes NULL
// from the empty string, and binding []byte directly would be worse than
// useless: these are STRICT tables, and a []byte binds as a BLOB, which a TEXT
// column refuses.
func nullableDocument(document []byte) any {
	if document == nil {
		return nil
	}
	return string(document)
}

// SessionByTokenDigest resolves a presented token to its session and to the
// user holding the token.
//
// The join is what makes a per-request check one indexed read of each table:
// the digest is access_tokens' primary key and the session identifier is
// sessions'.
//
// The user comes from access_tokens and not from sessions, and that is the
// whole reason this returns two values. A session names whoever authenticated
// on that client and device last; a token belongs to the user it was issued to
// (002 plan 6.5). Reading the caller off the session would hand a request to
// whoever logged in most recently on the same device, which is a different
// person's account and no error anywhere.
func (s *Store) SessionByTokenDigest(ctx context.Context, digest string) (ports.Session, string, bool, error) {
	var tokenUserID string
	row := s.reader.QueryRowContext(ctx,
		`SELECT `+sessionColumns+`, access_tokens.user_id
		 FROM access_tokens JOIN sessions ON sessions.id = access_tokens.session_id
		 WHERE access_tokens.token_digest = ?`, digest)

	session, err := scanSession(row, &tokenUserID)
	if errors.Is(err, sql.ErrNoRows) {
		// An unknown or revoked token is an absence, not a failure. 002 plan 7
		// makes it indistinguishable from no credential at all, which is
		// measured — and a store error here must never take that path, because
		// a client told 401 discards a credential that was fine.
		return ports.Session{}, "", false, nil
	}
	if err != nil {
		return ports.Session{}, "", false, fmt.Errorf("%s: resolving a token: %w", s.path, err)
	}
	return session, tokenUserID, true, nil
}

// Sessions returns every session.
//
// Ordered by creation and then by identifier, for the reason Users gives: an
// unordered SELECT is storage order, and /Sessions answers an array whose rows
// L3 compares by position. Creation order is the one order a client could
// reasonably expect; the identifier breaks a tie between two sessions opened
// inside the same tick, which is a hundred nanoseconds and therefore only ever
// a test.
func (s *Store) Sessions(ctx context.Context) ([]ports.Session, error) {
	rows, err := s.reader.QueryContext(ctx,
		`SELECT `+sessionColumns+` FROM sessions ORDER BY sessions.created_at, sessions.id`)
	if err != nil {
		return nil, fmt.Errorf("%s: reading the sessions: %w", s.path, err)
	}
	defer rows.Close()

	var open []ports.Session
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, fmt.Errorf("%s: reading the sessions: %w", s.path, err)
		}
		open = append(open, session)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: reading the sessions: %w", s.path, err)
	}
	return open, nil
}

// ReplaceCapabilities stores the declaration a client posted, whole.
//
// It replaces rather than merges (spec 3.8) and the document is not decoded on
// the way in, which is what makes behaviours 5.9's divergence — an unknown
// capabilities property surviving into /Sessions — the stated one rather than
// an accident. That is the opposite of what a configuration document does, and
// the two are opposite because the reference is.
func (s *Store) ReplaceCapabilities(ctx context.Context, sessionID string, document []byte) error {
	return s.updateOneSession(ctx, "storing a capabilities declaration", sessionID,
		`UPDATE sessions SET capabilities_document = ? WHERE id = ?`,
		nullableDocument(document), sessionID)
}

// TouchSession advances the session's LastActivityDate.
func (s *Store) TouchSession(ctx context.Context, sessionID string, at units.Time) error {
	return s.updateOneSession(ctx, "recording session activity", sessionID,
		`UPDATE sessions SET last_activity_at = ? WHERE id = ?`, int64(at.Ticks()), sessionID)
}

// CloseSession removes the session and every token that opens it.
//
// The transaction is OpenSession's contract read backwards. A token names its
// session by a foreign key, so the tokens go first and the session second — the
// other order is refused by the database rather than by this function, which is
// the schema doing the arguing. Either both are gone afterwards or neither is,
// and a half-applied eviction would leave a live credential resolving to a
// session that has been evicted.
//
// The token deletion requires no rows. A session nothing is logged in against
// is an ordinary state — every token on a device can be revoked without closing
// the session (RevokeTokensFor) — and a caller that had to treat that as a
// failure would refuse to evict exactly the session that has been idle longest,
// which is the one the ceiling picks.
//
// The session deletion requires exactly one, for updateOneSession's reason: a
// DELETE that matched nothing succeeds, and an eviction that evicted nothing
// would let the login it was clearing room for exceed the ceiling silently.
func (s *Store) CloseSession(ctx context.Context, sessionID string) error {
	failed := func(err error) error {
		return fmt.Errorf("%s: closing session %s: %w", s.path, sessionID, err)
	}

	transaction, err := s.writer.BeginTx(ctx, nil)
	if err != nil {
		return failed(err)
	}
	defer transaction.Rollback()

	if _, err := transaction.ExecContext(ctx,
		`DELETE FROM access_tokens WHERE session_id = ?`, sessionID); err != nil {
		return failed(err)
	}

	result, err := transaction.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return failed(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return failed(err)
	}
	if affected != 1 {
		return failed(fmt.Errorf("the deletion removed %d rows, want 1", affected))
	}

	if err := transaction.Commit(); err != nil {
		return failed(err)
	}
	return nil
}

// updateOneSession is updateOneUser for the other table, and carries the same
// guard for the same reason: an UPDATE that matched nothing succeeds.
func (s *Store) updateOneSession(ctx context.Context, what, sessionID, statement string, arguments ...any) error {
	result, err := s.writer.ExecContext(ctx, statement, arguments...)
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: %s: %w", s.path, what, err)
	}
	if affected != 1 {
		return fmt.Errorf("%s: %s changed %d rows for session %s, want 1", s.path, what, affected, sessionID)
	}
	return nil
}

// RevokeTokensFor invalidates every token this user holds on this device.
//
// Both columns are in the WHERE clause, and the index the migration declares is
// on exactly that pair. The narrower half is the one that matters: a DELETE by
// user alone would revoke the same person's token on every other device on
// every login, and nothing in any response would report it — the client that
// was logged out simply gets a 401 on its next request, hours later, and
// re-authenticates. A test asserting only that the replaced token is gone
// passes on that build.
//
// It does not report how many rows it removed, and must not require any. The
// first login from a device revokes nothing, and a caller that had to treat
// that as a failure would be a caller that refused every first login.
func (s *Store) RevokeTokensFor(ctx context.Context, userID, deviceID string) error {
	if _, err := s.writer.ExecContext(ctx,
		`DELETE FROM access_tokens WHERE user_id = ? AND device_id = ?`, userID, deviceID,
	); err != nil {
		return fmt.Errorf("%s: revoking the tokens of account %s on device %s: %w",
			s.path, userID, deviceID, err)
	}
	return nil
}
