package conformance_test

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The account subcommand's own words, spelled here rather than imported.
//
// That is the boundary doing its job: this package may import nothing of ours
// (doc.go), so the command line below is the same string an operator types. If
// a subcommand is ever renamed, these tests fail with "is not a subcommand"
// rather than passing against something else.
const (
	userCommand    = "user"
	userAddCommand = "add"
)

// withProvisionedAccount runs `atrium user add` on the data directory before
// the server starts, with the password on standard input.
//
// It is the fixture 002 plan 8's "one provisioning helper in conformance/,
// calling the subcommand", and it is the reason that subcommand exists at all:
// this package cannot reach into the store, so the only way it can put an
// installation into a state is to run the same command an operator runs. The
// fixture is not a back door.
func withProvisionedAccount(name, password string, extra ...string) serverOption {
	return func(t *testing.T, setup *installationSetup) {
		t.Helper()
		arguments := append([]string{
			userCommand, userAddCommand,
			"--data-dir", setup.dataDirectory,
			"--name", name,
		}, extra...)

		command := exec.Command(atriumBinary, arguments...)
		command.Stdin = strings.NewReader(password)
		command.Env = withoutAtriumEnvironment(os.Environ())

		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("atrium %s: %v\n%s", strings.Join(arguments, " "), err, output)
		}
	}
}

// runBinary runs the built server with the arguments given and returns what it
// wrote and whether it succeeded. Nothing here starts a listener.
func runBinary(t *testing.T, stdin string, arguments ...string) (string, error) {
	t.Helper()
	command := exec.Command(atriumBinary, arguments...)
	command.Stdin = strings.NewReader(stdin)
	command.Env = withoutAtriumEnvironment(os.Environ())
	output, err := command.CombinedOutput()
	return string(output), err
}

// rawField reads one property of a JSON object body without decoding its value.
//
// Raw, because Principle VIII: a StartupWizardCompleted carrying the string
// "true" and one carrying the boolean true are the same thing to a decoded
// bool, and the difference is a client that breaks.
func rawField(t *testing.T, body []byte, name string) string {
	t.Helper()
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("the body is not a JSON object: %v\n%s", err, body)
	}
	value, present := fields[name]
	if !present {
		t.Fatalf("%s is absent from the body\n%s", name, body)
	}
	return string(value)
}

// The wire half of 002 spec 3.9, and the only assertion this feature can make
// at the wire before any of its routes is registered.
//
// **Setup completion is observable and nothing else in v1 makes it happen.** An
// installation that can never finish setting up answers StartupWizardCompleted
// false for ever — and 001 spec 3.2 admits *every* request to GET /System/Info
// while setup is outstanding, so on such a server a criterion about a token is
// met by a request carrying nothing at all. That is the shape 001's closing
// audit caught itself in twice, and it is what makes this the row the later
// tasks stand on.
//
// It is also the first difference on an L3 response that this project could
// close: a reference stood up for a differential run has completed its own
// wizard (ADR-0007), so it answers true, and before this change Atrium could
// only answer false.
//
// The two servers differ in exactly one thing — one data directory was
// provisioned and the other was not — and the field is compared as raw JSON.
func TestAProvisionedInstallationReportsThatSetupIsCompleteAndAFreshOneDoesNot(t *testing.T) {
	t.Parallel()

	fresh := startServer(t)
	if len(fresh.seeded) != 0 {
		t.Fatalf("the unprovisioned installation was not empty when it started: %v", fresh.seeded)
	}
	before := fresh.get(t, publicSystemInfoPath, goldenHost, nil)
	if before.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", before.status, http.StatusOK, before.body)
	}
	if got := rawField(t, before.body, "StartupWizardCompleted"); got != "false" {
		t.Errorf("unprovisioned: StartupWizardCompleted = %s, want false", got)
	}

	provisioned := startServer(t, withProvisionedAccount("Ada", "hunter2\n"))
	after := provisioned.get(t, publicSystemInfoPath, goldenHost, nil)
	if after.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", after.status, http.StatusOK, after.body)
	}
	if got := rawField(t, after.body, "StartupWizardCompleted"); got != "true" {
		t.Errorf("provisioned: StartupWizardCompleted = %s, want true", got)
	}

	// The two bodies must differ in that field and not in the shape around it:
	// a provisioned installation is still an installation, and the response is
	// still the seven fields of 001 spec 3.1 in the same order.
	if names := propertyNames(t, after.body); len(names) != 7 {
		t.Errorf("the provisioned installation answers %d fields, want 7: %v", len(names), names)
	}
}

// The regression a dispatch on the first argument is most likely to introduce.
//
// `atrium --data-dir …` must still serve, and it must serve because the first
// argument was not a subcommand rather than by accident. Every other test in
// this package starts a server that way, so this one exists to say so: if the
// dispatch ever swallowed a leading flag, the failure would be the whole suite
// at once and nobody would know which change caused it.
//
// The second half is the other side of the same branch. A first argument that
// is not a flag and is not `user` is still refused by name — 001's "unexpected
// argument" — rather than being read as a subcommand nobody wrote or, worse,
// silently ignored.
func TestTheBinaryStillServesWhenTheFirstArgumentIsNotASubcommand(t *testing.T) {
	t.Parallel()

	// startServer passes --data-dir first and no subcommand at all. Reaching a
	// 200 means the dispatch let a leading flag through.
	server := startServer(t)
	got := server.get(t, publicSystemInfoPath, goldenHost, nil)
	if got.status != http.StatusOK {
		t.Fatalf("a server started with no subcommand answered %d\nbody: %s", got.status, got.body)
	}

	output, err := runBinary(t, "", "serve", "--data-dir", t.TempDir())
	if err == nil {
		t.Fatalf("a first argument that is not a subcommand was accepted:\n%s", output)
	}
	if !strings.Contains(output, "unexpected argument") {
		t.Errorf("the refusal was %q, want it to name the unexpected argument", output)
	}
}

// The password is read from standard input, and there is no flag that takes
// one — asserted against the binary an operator runs, because that is the
// process whose argument vector `ps` reads.
//
// internal/app asserts the same thing over the parsed flag set. This asserts it
// where it matters: the usage text the binary prints is what an operator goes
// looking for when they want to script this, and a flag that existed would be
// listed there.
func TestTheBinaryOffersNoWayToPutAPasswordInAnArgumentVector(t *testing.T) {
	t.Parallel()

	output, err := runBinary(t, "", userCommand, userAddCommand, "--help")
	if err == nil {
		t.Log("--help exited zero, which is the standard library's contract")
	}
	if strings.Contains(output, "--password ") || strings.Contains(output, "-password string") {
		t.Errorf("the usage text offers a password flag:\n%s", output)
	}

	// The value is deliberately not a plausible password. The flag is refused
	// before anything reads what follows it, so this string is only ever
	// compared against nothing — and a name-and-password pair on one command
	// line is the shape a public repository's secret scanners are built to
	// find. Leaving a realistic one here costs a false positive every time
	// somebody looks, and proves nothing this does not.
	directory := t.TempDir()
	rejected, err := runBinary(t, "", userCommand, userAddCommand,
		"--data-dir", directory, "--name", "Ada", "--password", "refused-before-this-is-read")
	if err == nil {
		t.Fatalf("--password was accepted:\n%s", rejected)
	}
}

// Provisioning twice with the same names produces byte-identical identifiers,
// asserted at the wire on the one response that carries a derived identifier
// today.
//
// This is the installation identity rather than a user's, and it is here as the
// counterexample: the installation's Id is 16 random bytes, so two fresh
// installations differ. The account identifiers this feature derives do not,
// and internal/app asserts that against the stored column — which is where they
// are observable until 002's routes are registered. When they are, the same
// assertion belongs on a user object.
func TestTwoFreshInstallationsDifferInTheirIdentityAndProvisioningDoesNotChangeThat(t *testing.T) {
	t.Parallel()

	one := startServer(t, withProvisionedAccount("Ada", "hunter2\n"))
	other := startServer(t, withProvisionedAccount("Ada", "hunter2\n"))

	first := rawField(t, one.get(t, publicSystemInfoPath, goldenHost, nil).body, "Id")
	second := rawField(t, other.get(t, publicSystemInfoPath, goldenHost, nil).body, "Id")
	if first == second {
		t.Errorf("two installations provisioned the same way answer the same Id %s", first)
	}
}

// The three seats a differential run needs are producible by the command, and
// each one is refused or accepted on its own terms.
//
// What the seats *mean* is the stored policy, which conformance/ cannot read;
// that half is asserted in internal/app. What this asserts is the half that
// belongs here: the binary accepts the three invocations
// docs/compatibility/request-cases.yaml's identities require, so a run that
// needs them can stand them up.
func TestTheBinaryProvisionsTheThreeSeatsARunNeeds(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	for _, seat := range []struct {
		name  string
		extra []string
	}{
		{"administrator", []string{"--administrator"}},
		{"restricted", nil},
		{"playback-denied", []string{"--enable-media-playback=false"}},
	} {
		arguments := append([]string{
			userCommand, userAddCommand, "--data-dir", directory, "--name", seat.name,
		}, seat.extra...)
		output, err := runBinary(t, "hunter2\n", arguments...)
		if err != nil {
			t.Fatalf("%s: %v\n%s", seat.name, err, output)
		}
	}

	listing, err := runBinary(t, "", userCommand, "list", "--data-dir", directory)
	if err != nil {
		t.Fatalf("atrium user list: %v\n%s", err, listing)
	}
	for _, seat := range []string{"administrator", "restricted", "playback-denied"} {
		if !strings.Contains(listing, seat) {
			t.Errorf("the listing does not mention %s:\n%s", seat, listing)
		}
	}
}

// A second account whose name folds to one already taken is refused, and the
// installation is left as it was.
//
// The uniqueness is the database's rule and not the command's (002 plan 4), and
// it is what the login's one-row-per-folded-name assumption stands on.
func TestProvisioningRefusesASecondAccountWhoseNameOnlyDiffersInCase(t *testing.T) {
	t.Parallel()

	directory := t.TempDir()
	if output, err := runBinary(t, "hunter2\n", userCommand, userAddCommand,
		"--data-dir", directory, "--name", "Ada"); err != nil {
		t.Fatalf("the first account: %v\n%s", err, output)
	}

	output, err := runBinary(t, "hunter3\n", userCommand, userAddCommand,
		"--data-dir", directory, "--name", "ADA")
	if err == nil {
		t.Fatalf("two accounts differing only in case were both created:\n%s", output)
	}

	listing, err := runBinary(t, "", userCommand, "list", "--data-dir", directory)
	if err != nil {
		t.Fatalf("atrium user list: %v\n%s", err, listing)
	}
	if count := strings.Count(listing, "\n"); count != 2 {
		t.Errorf("the listing has %d lines (a header and the accounts), want 2:\n%s", count, listing)
	}
}
