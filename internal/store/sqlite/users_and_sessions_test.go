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

// insertUser writes one account directly, without going through a store method.
//
// T1 wrote it because T1 shipped no store method. It survives T4 because T4
// still ships none that creates an account: 002 plan 5's UserStore reads
// accounts, replaces their credentials and configurations and records what a
// login did to them, and nothing in it makes one — provisioning is T7's, and
// the port method it needs is T7's to add. So this remains the only way a test
// builds a row, which is what the handoff asked for: not a second way, the
// only one.
//
// It is now a thin call onto insertUserWithPolicy, so that the empty document
// is one choice made in one place rather than a literal repeated in two files.
func insertUser(t *testing.T, db *sql.DB, id, username, folded string) error {
	t.Helper()
	return insertUserWithPolicy(t, db, id, username, folded, []byte("{}"), 0)
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
//
// Amended 2026-09-05 by 003 T10, which files the third precious migration. The
// body was three literals — "the precious lineage has 3 migrations, want 2" is
// what this test said the morning 0003_libraries.sql landed — and it is the
// same mistake this test's own 2026-09-03 amendment corrected in 001's: a
// number that every later migration invalidates, standing in for a rule that
// never mentioned one. The body is now filedUnderThePreciousLineage in
// migrate_test.go, given this migration's file name, and 003's test calls the
// same helper. Nothing about what is asserted changed.
func TestTheMigrationIsFiledUnderThePreciousLineage(t *testing.T) {
	filedUnderThePreciousLineage(t, "0002_users_and_sessions.sql")
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
// four tables, the migration that creates them among the ones a first start
// applied, and the derived half at the generation this build declares.
//
// Amended 2026-09-05 by 003 T10, for the reason above
// TestTheMigrationIsFiledUnderThePreciousLineage. "want [1 2]" and "want 2"
// were literals about how many migrations existed rather than about this
// feature's, and 0003_libraries.sql turned both red. What this test is for is
// the four tables and the version this migration carries, so it now names that
// version and lets the lineage be as long as it is.
//
// Amended again 2026-09-05 by 003 T11, which gives the derived half a schema.
// The last literal here was "the derived half is at version 0", which meant
// *this feature owns no derived table* and spelled it as the number that
// produced. It is theDerivedHalfIsAtItsGeneration now — the same rule, and the
// same helper the other three callers use.
func TestOpenAppliesTheUsersAndSessionsMigration(t *testing.T) {
	ctx := context.Background()
	store := openForTest(t)

	lineage, err := loadLineage(migrationFiles, Precious)
	if err != nil {
		t.Fatalf("loading the precious lineage returned %v", err)
	}
	version := 1 + slices.IndexFunc(lineage, func(m migration) bool {
		return m.name == "0002_users_and_sessions.sql"
	})
	if version == 0 {
		t.Fatalf("0002_users_and_sessions.sql is not in the precious lineage")
	}

	if applied := store.AppliedMigrations(Precious); !slices.Contains(applied, version) {
		t.Errorf("a first start applied %v precious migrations, want %d among them",
			applied, version)
	}
	if applied := store.AppliedMigrations(Derived); len(applied) != 0 {
		t.Errorf("a first start applied %v to the derived half, want nothing", applied)
	}

	precious, err := store.SchemaVersion(ctx, Precious)
	if err != nil {
		t.Fatalf("SchemaVersion(precious) returned %v", err)
	}
	if precious < version {
		t.Errorf("the precious half is at version %d after a first start, want at least %d — "+
			"the version this migration carries", precious, version)
	}
	theDerivedHalfIsAtItsGeneration(t, store, "a first start")

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
