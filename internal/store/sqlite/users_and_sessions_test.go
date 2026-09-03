package sqlite

import (
	"context"
	"database/sql"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/units"
)

// The identifiers the tests below share. They are the shape 002 plan 6.5
// derives — 32 lowercase hex — rather than convenient short strings, so that a
// column too narrow for one would fail here and not in the feature that first
// writes a real account.
const (
	testUserID    = "5d41402abc4b2a76b9719d911017c592"
	testSessionID = "7d793037a0760186574b0282f2f435e7"
)

// insertUser writes one account directly, without going through a store method,
// because T1 ships no store method: this task owns the schema and the lineage
// it is filed under, and T4 owns the readers and writers. What is under test is
// what the database refuses, so the SQL is the subject and not the detour.
func insertUser(t *testing.T, db *sql.DB, id, username, folded string) error {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO users (id, username, username_folded, policy_document, configuration_document,
		                    invalid_login_attempt_count, last_login_at, last_activity_at)
		 VALUES (?, ?, ?, '{}', '{}', 0, NULL, NULL)`, id, username, folded)
	return err
}

// insertSession writes one session with every date at the zero tick.
func insertSession(t *testing.T, db *sql.DB, id, userID, client, deviceID string) error {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, client, device_id, device_name, application_version,
		                       remote_endpoint, capabilities_document,
		                       created_at, last_activity_at, last_playback_check_in_at)
		 VALUES (?, ?, ?, ?, 'a device', '1.0.0', '127.0.0.1', NULL, 0, 0, 0)`,
		id, userID, client, deviceID)
	return err
}

// TestTheMigrationIsFiledUnderThePreciousLineage is T1's first clause, and it
// is the one mistake here that no test of the SQL itself would catch.
//
// A file that created these four tables out of the derived directory would
// create exactly the same tables, so every other test in this file would pass
// over it. What it would have changed is the policy: the derived half is the
// one a rescan is entitled to drop and rebuild (ADR-0003), so the accounts and
// the credentials would be deleted by a library scan, and the first symptom
// would be an installation that had logged everybody out after a scan.
//
// The assertion is therefore about the two version numbers rather than about
// the schema: applying the lineage this build ships to a database already at
// the precious version 001 left it at moves the precious half by exactly one
// and does not move the derived half at all.
func TestTheMigrationIsFiledUnderThePreciousLineage(t *testing.T) {
	ctx := context.Background()
	db := newDatabase(t)

	precious, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}
	if len(precious) != 2 {
		t.Fatalf("the precious lineage has %d migrations, want 2", len(precious))
	}
	if precious[1].name != "0002_users_and_sessions.sql" {
		t.Errorf("the second precious migration is %s, want 0002_users_and_sessions.sql", precious[1].name)
	}

	derived, err := loadLineage(migrationFiles, Derived)
	if err != nil {
		t.Fatalf("loading the derived lineage returned %v", err)
	}
	if len(derived) != 0 {
		t.Fatalf("the derived lineage has %d migrations, want none: 002 scans nothing", len(derived))
	}

	// The state 001 left behind: the precious half at 1, the derived half at 0.
	if _, err := migrate(ctx, db, Precious, precious[:1]); err != nil {
		t.Fatalf("applying 001's lineage returned %v", err)
	}
	before := versions(t, db)
	if before != (halfVersions{precious: 1, derived: 0}) {
		t.Fatalf("after 001's lineage the versions are %+v, want precious 1 and derived 0", before)
	}

	applied, err := migrate(ctx, db, Precious, precious)
	if err != nil {
		t.Fatalf("applying this feature's lineage returned %v", err)
	}
	if !slices.Equal(applied, []int{2}) {
		t.Errorf("applying the lineage applied %v, want exactly [2]", applied)
	}

	after := versions(t, db)
	if after.precious != before.precious+1 {
		t.Errorf("the precious half moved from %d to %d, want an advance of exactly one",
			before.precious, after.precious)
	}
	if after.derived != before.derived {
		t.Errorf("the derived half moved from %d to %d: this migration is filed under the wrong lineage, "+
			"and a rescan would drop the accounts it creates",
			before.derived, after.derived)
	}
}

type halfVersions struct{ precious, derived int }

func versions(t *testing.T, db *sql.DB) halfVersions {
	t.Helper()
	ctx := context.Background()

	precious, err := schemaVersion(ctx, db, Precious)
	if err != nil {
		t.Fatalf("SchemaVersion(precious) returned %v", err)
	}
	derived, err := schemaVersion(ctx, db, Derived)
	if err != nil {
		t.Fatalf("SchemaVersion(derived) returned %v", err)
	}
	return halfVersions{precious: precious, derived: derived}
}

// TestOpenAppliesTheUsersAndSessionsMigration is the same clause seen from a
// start rather than from the runner: an empty data directory comes up with the
// four tables and with the precious half at 2 while the derived half stays at
// 0.
func TestOpenAppliesTheUsersAndSessionsMigration(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	if applied := store.AppliedMigrations(Precious); !slices.Equal(applied, []int{1, 2}) {
		t.Errorf("a first start applied %v precious migrations, want [1 2]", applied)
	}
	if applied := store.AppliedMigrations(Derived); len(applied) != 0 {
		t.Errorf("a first start applied %v to the derived half, want nothing", applied)
	}

	precious, err := store.SchemaVersion(ctx, Precious)
	if err != nil {
		t.Fatalf("SchemaVersion(precious) returned %v", err)
	}
	if precious != 2 {
		t.Errorf("the precious half is at version %d after a first start, want 2", precious)
	}
	derived, err := store.SchemaVersion(ctx, Derived)
	if err != nil {
		t.Fatalf("SchemaVersion(derived) returned %v", err)
	}
	if derived != 0 {
		t.Errorf("the derived half is at version %d, want 0: this feature owns no derived table", derived)
	}

	for _, table := range []string{"users", "user_credentials", "sessions", "access_tokens"} {
		var name string
		if err := store.reader.QueryRow(
			`SELECT name FROM sqlite_schema WHERE type = 'table' AND name = ?`, table,
		).Scan(&name); err != nil {
			t.Errorf("the %s table is not there after a first start: %v", table, err)
		}
	}
}

// TestASecondStartAppliesNothingToTheNewTables is T1's second clause. It is
// worth its own test beside TestASecondStartAppliesNothing, which asserts the
// same rule over 001's lineage: what would fail here and not there is a 0002
// that re-ran, which would either fail the start on a CREATE TABLE that already
// exists or — with the wrong defensive clause bolted on — quietly recreate an
// empty users table on every restart.
func TestASecondStartAppliesNothingToTheNewTables(t *testing.T) {
	directory := t.TempDir()

	first, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("the first Open returned %v", err)
	}
	if err := insertUser(t, first.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting an account returned %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}

	second := openIn(t, directory)
	for _, half := range halves {
		if applied := second.AppliedMigrations(half); len(applied) != 0 {
			t.Errorf("a second start applied %v to the %s half, want nothing", applied, half)
		}
	}

	var username string
	if err := second.reader.QueryRow(
		`SELECT username FROM users WHERE id = ?`, testUserID,
	).Scan(&username); err != nil {
		t.Fatalf("reading the account back after a restart: %v", err)
	}
	if username != "Alice" {
		t.Errorf("the account is named %q after a restart, want %q", username, "Alice")
	}
}

// TestUsernamesDifferingOnlyInCaseAreRefused is T1's third clause.
//
// username_folded exists so that the assumption the login makes is the
// database's rule rather than a convention: spec 3.3 matches a username
// case-insensitively, so two accounts that fold to one name would leave the
// credential check choosing between two credentials with no defined answer.
// The column carries the uniqueness; this is the test that it does.
func TestUsernamesDifferingOnlyInCaseAreRefused(t *testing.T) {
	store := openForTest(t)

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the first account returned %v", err)
	}

	err := insertUser(t, store.writer, "0cc175b9c0f1b6a831c399e269772661", "ALICE", "alice")
	if err == nil {
		t.Fatal("a second account folding to the same username was accepted, want a refusal from username_folded")
	}
	if !strings.Contains(err.Error(), "users.username_folded") {
		t.Errorf("the refusal is %v, want one naming users.username_folded", err)
	}

	// The spelling is not what is unique. Two genuinely different names that
	// differ in case are two accounts, and a UNIQUE on username would have
	// passed the assertion above while refusing this.
	if err := insertUser(t, store.writer, "0cc175b9c0f1b6a831c399e269772661", "ALICE", "alice2"); err != nil {
		t.Errorf("a second account with a different folded name returned %v, want it accepted", err)
	}
}

// TestASecondSessionForOneClientAndDeviceIsRefused covers the constraint this
// task added beyond the columns 002 plan 4 lists, and it is the reason the
// constraint is defensible rather than decoration.
//
// The plan states the key in prose — "one row per (Client, DeviceId)" — and
// leaves it enforced by the primary key only because the identifier is derived
// from exactly that pair (plan 6.5). That holds until the derivation changes,
// and on the day it does, the symptom is two live sessions for one client on
// one device and an authentication that replaces neither. The UNIQUE is what
// turns that into a failed write.
func TestASecondSessionForOneClientAndDeviceIsRefused(t *testing.T) {
	store := openForTest(t)

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}
	if err := insertSession(t, store.writer, testSessionID, testUserID, "music-client", "device-1"); err != nil {
		t.Fatalf("inserting the session returned %v", err)
	}

	// A different identifier, so the primary key does not answer this: what
	// refuses it has to be the key the plan states.
	err := insertSession(t, store.writer, "e358efa489f58062f10dd7316b65649e", testUserID, "music-client", "device-1")
	if err == nil {
		t.Fatal("a second session for one client and device was accepted, want a refusal from the key")
	}
	if !strings.Contains(err.Error(), "sessions.client") {
		t.Errorf("the refusal is %v, want one naming the (client, device_id) key", err)
	}

	// The same device under a different client is a different session, which
	// is the half a UNIQUE on device_id alone would have refused.
	if err := insertSession(t, store.writer, "e358efa489f58062f10dd7316b65649e", testUserID, "video-client", "device-1"); err != nil {
		t.Errorf("a session for another client on the same device returned %v, want it accepted", err)
	}
}

// TestATokenNamingAMissingSessionIsRefused is T1's fourth clause.
//
// It is a test of the foreign key and, underneath it, of the pragma: SQLite
// leaves foreign keys off by default, so a schema that declares one on a
// connection that has not enabled them has a comment where it wanted a
// constraint. Both halves are asserted — the row that names a live session is
// accepted, so that a refusal caused by something else could not pass for this.
func TestATokenNamingAMissingSessionIsRefused(t *testing.T) {
	store := openForTest(t)

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}
	if err := insertSession(t, store.writer, testSessionID, testUserID, "music-client", "device-1"); err != nil {
		t.Fatalf("inserting the session returned %v", err)
	}

	insertToken := func(sessionID string) error {
		_, err := store.writer.Exec(
			`INSERT INTO access_tokens (token_digest, user_id, session_id, device_id, created_at)
			 VALUES (?, ?, ?, 'device-1', 0)`,
			"digest-for-"+sessionID, testUserID, sessionID)
		return err
	}

	if err := insertToken(testSessionID); err != nil {
		t.Fatalf("a token naming a live session returned %v, want it accepted", err)
	}
	if err := insertToken("2f8a1bd4a94d4f5f9f0a5a1c8e3b7d60"); err == nil {
		t.Error("a token naming a session that does not exist was accepted, want a foreign-key refusal")
	} else if !strings.Contains(strings.ToLower(err.Error()), "foreign key") {
		t.Errorf("the refusal is %v, want a foreign-key constraint failure", err)
	}
}

// TestTheZeroTickIsAWritableLastPlaybackCheckIn is T1's fifth clause.
//
// Spec 3.3 measures LastPlaybackCheckIn as 0001-01-01T00:00:00.0000000Z for a
// session that has never played anything — .NET's minimum date, "not null and
// not absent" [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
// 2026-08-28]. So the column is NOT NULL and the zero tick is a value it holds,
// not a value it rejects: a nullable column would let a writer answer an
// absence where the reference answers a date, and the only honest thing to
// serialise for that absence would be the date this column can already hold.
func TestTheZeroTickIsAWritableLastPlaybackCheckIn(t *testing.T) {
	store := openForTest(t)

	if err := insertUser(t, store.writer, testUserID, "Alice", "alice"); err != nil {
		t.Fatalf("inserting the account returned %v", err)
	}
	if err := insertSession(t, store.writer, testSessionID, testUserID, "music-client", "device-1"); err != nil {
		t.Fatalf("a session with the zero tick returned %v, want it accepted", err)
	}

	var ticks int64
	if err := store.reader.QueryRow(
		`SELECT last_playback_check_in_at FROM sessions WHERE id = ?`, testSessionID,
	).Scan(&ticks); err != nil {
		t.Fatalf("reading last_playback_check_in_at returned %v", err)
	}
	if ticks != 0 {
		t.Errorf("last_playback_check_in_at read back as %d, want 0", ticks)
	}

	// The tie between the stored zero and the measured date. Without it this
	// test asserts that a column accepts 0, which is true of any integer
	// column and proves nothing about the value the wire carries.
	if spelling := units.TimeFromTicks(units.Ticks(ticks)).String(); spelling != "0001-01-01T00:00:00.0000000Z" {
		t.Errorf("the zero tick spells %q, want %q (spec 3.3)", spelling, "0001-01-01T00:00:00.0000000Z")
	}

	// And the other half of NOT NULL: an absence is refused, so the column
	// cannot come to mean "never played" by way of a NULL somebody wrote.
	if _, err := store.writer.Exec(
		`UPDATE sessions SET last_playback_check_in_at = NULL WHERE id = ?`, testSessionID,
	); err == nil {
		t.Error("last_playback_check_in_at accepted NULL, want a NOT NULL refusal")
	}
}
