package app

import (
	"context"
	"database/sql"
	"flag"
	"io"
	"path/filepath"
	"strings"
	"testing"

	// The driver, blank-imported so that the read-back below opens the store
	// file directly rather than through internal/store/sqlite. Reading the
	// column in SQL is the point of it: the store exposes setup completion as
	// a boolean, because that is what the wire carries, so the *instant* is
	// only observable here.
	_ "modernc.org/sqlite"

	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/users"
)

// provision runs one `atrium user <subcommand>` the way the binary runs it, and
// returns what it wrote to standard output.
func provision(t *testing.T, password string, args ...string) (string, error) {
	t.Helper()
	stdout := &strings.Builder{}
	err := RunUser(context.Background(), args, noEnvironment,
		strings.NewReader(password), stdout, io.Discard)
	return stdout.String(), err
}

// mustProvision runs one subcommand and fails the test if it did not work.
func mustProvision(t *testing.T, password string, args ...string) string {
	t.Helper()
	stdout, err := provision(t, password, args...)
	if err != nil {
		t.Fatalf("atrium user %s: %v", strings.Join(args, " "), err)
	}
	return stdout
}

// openStoreFile opens the installation's database for reading, directly.
//
// Every assertion below that says "read back in SQL" goes through this. It is
// deliberately not the store's own API: what the store answers about setup
// completion is ports.Installation.SetupCompleted, a boolean, and a test that
// could only see the boolean would pass on a build that rewrote the recorded
// instant on every call — which is exactly the build 002 T7's definition of
// done names.
func openStoreFile(t *testing.T, dataDirectory string) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", filepath.Join(dataDirectory, sqlite.DatabaseFile))
	if err != nil {
		t.Fatalf("opening the store file: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	return database
}

// setupCompletedAt reads the installation's recorded instant, in ticks, as the
// column holds it — NULL included, because NULL is what "setup is outstanding"
// is.
func setupCompletedAt(t *testing.T, dataDirectory string) sql.Null[int64] {
	t.Helper()
	var completedAt sql.Null[int64]
	err := openStoreFile(t, dataDirectory).QueryRow(
		`SELECT setup_completed_at FROM installation WHERE id = 1`).Scan(&completedAt)
	if err != nil {
		t.Fatalf("reading setup_completed_at: %v", err)
	}
	return completedAt
}

// storedPolicy reads one account's policy document out of the column and
// decodes it the way every reader of that column has to.
func storedPolicy(t *testing.T, dataDirectory, username string) users.Policy {
	t.Helper()
	var document []byte
	err := openStoreFile(t, dataDirectory).QueryRow(
		`SELECT policy_document FROM users WHERE username_folded = ?`,
		users.Fold(username)).Scan(&document)
	if err != nil {
		t.Fatalf("reading the policy of %q: %v", username, err)
	}
	policy, err := users.DecodePolicy(document)
	if err != nil {
		t.Fatalf("decoding the policy of %q: %v", username, err)
	}
	return policy
}

// storedIdentifier reads one account's identifier out of the column.
func storedIdentifier(t *testing.T, dataDirectory, username string) string {
	t.Helper()
	var id string
	err := openStoreFile(t, dataDirectory).QueryRow(
		`SELECT id FROM users WHERE username_folded = ?`, users.Fold(username)).Scan(&id)
	if err != nil {
		t.Fatalf("reading the identifier of %q: %v", username, err)
	}
	return id
}

// hasCredential reports whether the account has a password record at all.
func hasCredential(t *testing.T, dataDirectory, username string) bool {
	t.Helper()
	var count int
	err := openStoreFile(t, dataDirectory).QueryRow(
		`SELECT COUNT(*) FROM user_credentials
		 WHERE user_id = (SELECT id FROM users WHERE username_folded = ?)`,
		users.Fold(username)).Scan(&count)
	if err != nil {
		t.Fatalf("counting the credentials of %q: %v", username, err)
	}
	return count > 0
}

// `user add` on an empty data directory creates the account and completes
// setup, which is 002 spec 3.9: an installation is set up from the moment it
// holds one account, and nothing else in v1 makes that happen.
func TestUserAddOnAnEmptyDataDirectoryCreatesTheAccountAndCompletesSetup(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	stdout := mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")

	if want := users.DeriveID("Ada"); !strings.Contains(stdout, want) {
		t.Errorf("the command reported %q, want it to name the identifier %s", stdout, want)
	}
	if !hasCredential(t, directory, "Ada") {
		t.Error("the account has no password record, and one was supplied")
	}

	// Through the port, because that is what /System/Info/Public reads and
	// StartupWizardCompleted is the observable (002 spec 4).
	store, err := sqlite.Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("reopening the store: %v", err)
	}
	defer store.Close()
	installation, err := store.Installation(context.Background())
	if err != nil {
		t.Fatalf("reading the installation: %v", err)
	}
	if !installation.SetupCompleted {
		t.Error("SetupCompleted is false after the first account was created")
	}
}

// The idempotence 002 plan 6.8 puts at the caller, asserted where it is
// observable.
//
// **The instant is read back in SQL and not through the port**, because the
// port answers a boolean: a test that checked SetupCompleted after the second
// account would pass on a build that called MarkSetupComplete every time and
// rewrote the recorded instant. The recorded instant means "when setup was
// first completed", and a second write is a lie about it.
func TestASecondUserAddDoesNotMoveTheRecordedInstant(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")

	first := setupCompletedAt(t, directory)
	if !first.Valid {
		t.Fatal("setup_completed_at is NULL after the first account was created")
	}

	mustProvision(t, "hunter3\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Bob")

	second := setupCompletedAt(t, directory)
	if !second.Valid {
		t.Fatal("setup_completed_at became NULL after a second account was created")
	}
	if second.V != first.V {
		t.Errorf("setup_completed_at moved from %d to %d: the second account rewrote the instant "+
			"the first one recorded", first.V, second.V)
	}
}

// There is no --password, and this asserts it over the flag set rather than
// over this repository's source.
//
// The reason the flag does not exist is not tidiness: an argument vector is
// readable by every process on the host — ps on macOS, /proc/<pid>/cmdline on
// Linux — and it reaches the operator's shell history besides, so a flag would
// put the one value ADR-0006 works to keep ephemeral somewhere that outlives
// the command. A test that grepped this file for the string would pass on a
// binary that took one under another name.
//
// --no-password contains the word and is a *boolean*: it carries no secret, it
// says only that there is none. That is the distinction the second clause
// makes, and it is what stops this test from being satisfied by renaming the
// flag that leaks.
func TestTheFlagsOfUserAddCarryNoPassword(t *testing.T) {
	t.Parallel()

	fs, _ := newUserAddFlags(io.Discard)

	if fs.Lookup("password") != nil {
		t.Error("user add declares a --password flag")
	}

	fs.VisitAll(func(f *flag.Flag) {
		if !strings.Contains(f.Name, "password") {
			return
		}
		boolean, ok := f.Value.(interface{ IsBoolFlag() bool })
		if !ok || !boolean.IsBoolFlag() {
			t.Errorf("--%s names a password and takes a value: a password may not travel in an "+
				"argument vector", f.Name)
		}
	})
}

// Principle VII, and the thing it buys: two installations provisioned with the
// same names hold the same identifiers, byte for byte, so a golden body that
// names a user is not a golden that names one particular run.
//
// The identifiers are compared as strings out of the store rather than as the
// output of users.DeriveID, so a command that derived an identifier correctly
// and then stored something else would fail here.
func TestTwoDirectoriesProvisionedWithTheSameNamesHoldIdenticalIdentifiers(t *testing.T) {
	t.Parallel()

	names := []string{"Ada", "Bob", "Cyd"}

	provisionAll := func() string {
		directory := filepath.Join(t.TempDir(), "installation")
		for _, name := range names {
			mustProvision(t, "hunter2\n", userAdd,
				"--"+flagDataDirectory, directory, "--"+flagName, name)
		}
		return directory
	}

	first, second := provisionAll(), provisionAll()
	for _, name := range names {
		one, other := storedIdentifier(t, first, name), storedIdentifier(t, second, name)
		if one != other {
			t.Errorf("%s: the two installations hold %q and %q", name, one, other)
		}
	}
}

// The identifier is derived from the *folded* name, so two installations that
// spell one account differently in case still agree on its identifier.
//
// This is the clause users.DeriveID's own folding exists for. A command that
// hashed the name as typed would pass the test above — both runs spell it the
// same way — and would fail here.
func TestTheIdentifierIsDerivedFromTheFoldedName(t *testing.T) {
	t.Parallel()

	lower := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, lower, "--"+flagName, "ada")

	upper := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, upper, "--"+flagName, "ADA")

	if one, other := storedIdentifier(t, lower, "ada"), storedIdentifier(t, upper, "ADA"); one != other {
		t.Errorf("ada and ADA hold %q and %q", one, other)
	}
}

// The three seats docs/compatibility/request-cases.yaml names — administrator,
// restricted and playback-denied — are producible, and the assertion is the
// stored policy rather than the command's exit status.
//
// A command that accepted every flag and wrote DefaultPolicy() three times
// exits zero three times. What separates the seats is what the store holds, and
// twelve of the twenty-three reads of this surface answer differently to the
// second of them, so a run that cannot stand one up answers them all with the
// wrong seat.
func TestTheThreeSeatsAreDistinguishedByTheirStoredPolicy(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "administrator", "--"+flagAdministrator)
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "restricted")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "playback-denied",
		"--"+flagEnableMediaPlayback+"=false")

	for _, seat := range []struct {
		name                string
		isAdministrator     bool
		enableMediaPlayback bool
	}{
		{"administrator", true, true},
		{"restricted", false, true},
		{"playback-denied", false, false},
	} {
		policy := storedPolicy(t, directory, seat.name)
		if policy.IsAdministrator != seat.isAdministrator {
			t.Errorf("%s: IsAdministrator = %v, want %v",
				seat.name, policy.IsAdministrator, seat.isAdministrator)
		}
		if policy.EnableMediaPlayback != seat.enableMediaPlayback {
			t.Errorf("%s: EnableMediaPlayback = %v, want %v",
				seat.name, policy.EnableMediaPlayback, seat.enableMediaPlayback)
		}
	}
}

// A bare `user add` writes exactly the policy the reference gives a fresh
// account, which is what makes every flag's default above the reference's own
// default rather than this command's opinion.
//
// IsHidden is the row that makes this worth asserting: it is *true* on a fresh
// reference account [source: Jellyfin.Data/UserEntityExtensions.cs:173 @
// v10.11.11], so a bare account does not appear on a login screen. A reader who
// assumed --hidden turns something on would have got it backwards, and so would
// a fixture that expected a new account in /Users/Public.
func TestABareUserAddStoresTheReferencesOwnDefaultPolicy(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")

	got, want := storedPolicy(t, directory, "Ada"), users.DefaultPolicy()
	if gotDocument, wantDocument := document(t, got), document(t, want); gotDocument != wantDocument {
		t.Errorf("the stored policy is\n%s\nwant\n%s", gotDocument, wantDocument)
	}
	if !want.IsHidden {
		t.Error("DefaultPolicy no longer hides a fresh account; this test's premise has moved")
	}
}

func document(t *testing.T, policy users.Policy) string {
	t.Helper()
	encoded, err := policy.Document()
	if err != nil {
		t.Fatalf("encoding a policy: %v", err)
	}
	return string(encoded)
}

// --no-password creates the account with no password record at all, which
// spec 3.5 reports as HasPassword false and ADR-0006 rule 4 excludes from the
// timing equalisation.
func TestUserAddWithNoPasswordCreatesNoCredential(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada", "--"+flagNoPassword)

	if hasCredential(t, directory, "Ada") {
		t.Error("the account has a password record, and --no-password was given")
	}
}

// An empty standard input is refused rather than read as an empty password,
// because the two are different things and only one of them is asked for by
// name.
func TestUserAddWithNothingOnStandardInputIsRefused(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	_, err := provision(t, "", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")
	if err == nil {
		t.Fatal("an empty standard input created an account")
	}
	if !strings.Contains(err.Error(), flagNoPassword) {
		t.Errorf("the refusal is %q, want it to name --%s", err, flagNoPassword)
	}
}

// One trailing line ending is removed and nothing else, so an operator typing
// the password and pressing return and a script piping it with printf produce
// the same record.
func TestThePasswordLosesOneTrailingLineEndingAndNoMore(t *testing.T) {
	t.Parallel()

	for _, supplied := range []string{"hunter2", "hunter2\n", "hunter2\r\n"} {
		password, err := readPassword(strings.NewReader(supplied))
		if err != nil {
			t.Fatalf("%q: %v", supplied, err)
		}
		if password.IsEmpty() {
			t.Fatalf("%q read as no password at all", supplied)
		}
		// The plaintext will not print itself, so what is compared is whether
		// the same record verifies — which is the only question that matters.
		record, err := users.Derive(password)
		if err != nil {
			t.Fatalf("%q: %v", supplied, err)
		}
		ok, _, err := users.Verify(record, users.NewPlaintext("hunter2"))
		if err != nil || !ok {
			t.Errorf("%q did not read as the password hunter2 (ok=%v, err=%v)", supplied, ok, err)
		}
	}
}

// A second account whose name folds to one that is taken is refused, and the
// refusal names the fold rather than reporting a constraint violation.
//
// The unique index is what actually forbids it (002 plan 4); this is the
// message, and the message is the difference between an operator who can fix
// their invocation and one reading about SQLite.
func TestASecondAccountWithTheSameFoldedNameIsRefused(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")

	_, err := provision(t, "hunter3\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "ADA")
	if err == nil {
		t.Fatal("two accounts whose names differ only in case were both created")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("the refusal is %q, want it to say the account already exists", err)
	}
}

// `user set-password` replaces the record, and the new password is the one that
// verifies afterwards.
func TestUserSetPasswordReplacesTheRecord(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")

	before := credentialRecord(t, directory, "Ada")
	mustProvision(t, "hunter3\n", userSetPassword,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada")
	after := credentialRecord(t, directory, "Ada")

	if before == after {
		t.Fatal("the stored record did not change")
	}
	if ok, _, err := users.Verify(after, users.NewPlaintext("hunter3")); err != nil || !ok {
		t.Errorf("the new password does not verify against the stored record (ok=%v, err=%v)", ok, err)
	}
	if ok, _, _ := users.Verify(after, users.NewPlaintext("hunter2")); ok {
		t.Error("the old password still verifies")
	}
}

func credentialRecord(t *testing.T, dataDirectory, username string) string {
	t.Helper()
	var record string
	err := openStoreFile(t, dataDirectory).QueryRow(
		`SELECT phc FROM user_credentials
		 WHERE user_id = (SELECT id FROM users WHERE username_folded = ?)`,
		users.Fold(username)).Scan(&record)
	if err != nil {
		t.Fatalf("reading the credential of %q: %v", username, err)
	}
	return record
}

// `user list` reports the accounts it was given, and reports whether each has a
// password.
func TestUserListReportsTheAccounts(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	mustProvision(t, "hunter2\n", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Ada", "--"+flagAdministrator)
	mustProvision(t, "", userAdd,
		"--"+flagDataDirectory, directory, "--"+flagName, "Bob", "--"+flagNoPassword)

	listing := mustProvision(t, "", userList, "--"+flagDataDirectory, directory)
	for _, wanted := range []string{"Ada", "Bob", users.DeriveID("Ada"), users.DeriveID("Bob")} {
		if !strings.Contains(listing, wanted) {
			t.Errorf("the listing does not mention %q:\n%s", wanted, listing)
		}
	}
}

// An unknown subcommand is refused by name rather than being run as one of the
// three or silently ignored.
func TestAnUnknownSubcommandIsRefused(t *testing.T) {
	t.Parallel()

	if _, err := provision(t, "", "delete"); err == nil {
		t.Fatal("an unknown subcommand was accepted")
	}
	if _, err := provision(t, ""); err == nil {
		t.Fatal("no subcommand at all was accepted")
	}
}

// --data-dir falls back to the same environment variable the server's does, so
// an operator who exports it once does not create an account in the wrong
// installation.
func TestTheDataDirectoryFallsBackToTheSameEnvironmentVariableTheServerUses(t *testing.T) {
	t.Parallel()

	directory := filepath.Join(t.TempDir(), "installation")
	environment := func(name string) string {
		if name == EnvDataDirectory {
			return directory
		}
		return ""
	}

	stdout := &strings.Builder{}
	err := RunUser(context.Background(),
		[]string{userAdd, "--" + flagName, "Ada"},
		environment, strings.NewReader("hunter2\n"), stdout, io.Discard)
	if err != nil {
		t.Fatalf("atrium user add with %s set: %v", EnvDataDirectory, err)
	}
	if got := storedIdentifier(t, directory, "Ada"); got != users.DeriveID("Ada") {
		t.Errorf("the account was created with identifier %q", got)
	}
}
