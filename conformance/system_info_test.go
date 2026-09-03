package conformance_test

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// systemInfoPath is the authenticated route of spec 3.2, spelled as a client
// spells it.
const systemInfoPath = "/System/Info"

// AC-5's superset half, against the running binary.
//
// # Why there is no golden beside this one
//
// Seven of this body's fields are the installation's own paths and one is the
// port the operating system chose, so there is nothing to record: a golden
// would either be a file that only ever matches on the machine that wrote it,
// or a comparison softened until it stopped being a byte comparison. What a
// golden buys is bought here instead by the key-order test below and by the
// per-field assertions — and the superset itself is compared as bytes, member
// by member. plan 8 records the reasoning; spec 6 carries the amendment.
//
// # What "agrees" means here
//
// The two bodies are compared as raw JSON, which is the only level at which a
// shared field that had changed type — `true` becoming `"true"` — is a
// difference. Both requests carry the same Host, because LocalAddress is
// derived from it (spec 3.4 tier 2, which is what every installation this
// binary can be started as answers with).
func TestSystemInfoIsASupersetOfThePublicBody(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	public := server.get(t, publicSystemInfoPath, goldenHost, nil)
	if public.status != http.StatusOK {
		t.Fatalf("%s: status %d\n%s", publicSystemInfoPath, public.status, public.body)
	}
	authenticated := server.get(t, systemInfoPath, goldenHost, nil)
	if authenticated.status != http.StatusOK {
		t.Fatalf("%s: status %d\n%s", systemInfoPath, authenticated.status, authenticated.body)
	}

	assertTheSupersetAgreesWithThePublicBody(t, authenticated.body, public.body)
}

// The superset itself, member by member, as raw JSON.
//
// It is a function rather than the body of the test above because 002's AC-14
// asserts the same thing about a body admitted by a **token** rather than by
// first-time setup, and two spellings of one comparison are two answers to what
// "superset" means.
func assertTheSupersetAgreesWithThePublicBody(t *testing.T, authenticated, public []byte) {
	t.Helper()

	publicNames := propertyNames(t, public)
	publicFields := rawFields(t, public)
	authenticatedFields := rawFields(t, authenticated)

	if len(publicNames) != 7 {
		t.Fatalf("the public body carries %d fields and spec 3.1 has seven: %v", len(publicNames), publicNames)
	}

	for _, name := range publicNames {
		got, present := authenticatedFields[name]
		if !present {
			t.Errorf("%s is in the public body and absent from the authenticated one", name)
			continue
		}
		if string(got) != string(publicFields[name]) {
			t.Errorf("%s: %s says %s, %s says %s",
				name, systemInfoPath, got, publicSystemInfoPath, publicFields[name])
		}
	}

	if len(authenticatedFields) <= len(publicNames) {
		t.Errorf("the authenticated body carries %d fields and the public one %d; spec 3.2 makes the first a strict superset",
			len(authenticatedFields), len(publicNames))
	}
}

// Twenty-six fields, in order, over the wire.
//
// This is what the missing golden would have caught, minus the values that
// cannot be held still: the count (Principle I is a count — an extra field is a
// delta whether or not a client reads it), the order, and the absence of
// PackageName, which the reference declares and does not send (behaviours 1.7).
func TestSystemInfoCarriesTheSupersetInSpecOrder(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.get(t, systemInfoPath, goldenHost, nil)

	if contentType := got.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "application/json; charset=utf-8")
	}

	if names := propertyNames(t, got.body); !equalStrings(names, systemInfoNames) {
		t.Errorf("property names:\n got %v\nwant %v", names, systemInfoNames)
	}
}

// The twenty-six names spec 3.2's body carries, in the order it carries them.
//
// A package-level list rather than a literal inside the test above, because
// 002's AC-14 asserts the same order on a body admitted by a token: the count
// and the order are the half of the missing golden that can be held still, and
// a second copy of them is a second answer.
var systemInfoNames = []string{
	// spec 3.1's seven, first, because they are the embedded half.
	"LocalAddress",
	"ServerName",
	"Version",
	"ProductName",
	"OperatingSystem",
	"Id",
	"StartupWizardCompleted",
	// spec 3.2's additions, in the reference model's declaration order
	// [source: MediaBrowser.Model/System/SystemInfo.cs:29-143 @ v10.11.11].
	// PackageName belongs between the first two and is deliberately not
	// sent.
	"OperatingSystemDisplayName",
	"HasPendingRestart",
	"IsShuttingDown",
	"SupportsLibraryMonitor",
	"WebSocketPortNumber",
	"CompletedInstallations",
	"CanSelfRestart",
	"CanLaunchWebBrowser",
	"ProgramDataPath",
	"WebPath",
	"ItemsByNamePath",
	"CachePath",
	"LogPath",
	"InternalMetadataPath",
	"TranscodingTempPath",
	"CastReceiverApplications",
	"HasUpdateAvailable",
	"EncoderLocation",
	"SystemArchitecture",
}

// The values spec 3.2 fixes, field by field, as raw JSON.
//
// Raw, so that `"HasPendingRestart": "false"` fails here rather than passing as
// a truthy string, and so that an empty array is told apart from a null one —
// the difference between [] and null is one keystroke in Go and a decoder
// failure in a strongly typed client.
func TestSystemInfoAnswersTheFlagsAndArraysOfSpecThreeTwo(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	if len(server.seeded) != 0 {
		t.Fatalf("this installation was not empty when it started: %v", server.seeded)
	}

	fields := rawFields(t, server.get(t, systemInfoPath, goldenHost, nil).body)

	for _, row := range []struct{ field, want string }{
		// Atrium has no self-update, no restart and no browser to launch.
		{"HasPendingRestart", "false"},
		{"IsShuttingDown", "false"},
		{"CanSelfRestart", "false"},
		{"CanLaunchWebBrowser", "false"},
		{"HasUpdateAvailable", "false"},
		// v1 does not watch the filesystem. The reference answers true here
		// unconditionally, so this is a difference and it is an honest one.
		{"SupportsLibraryMonitor", "false"},
		// Empty arrays, not nulls.
		{"CompletedInstallations", "[]"},
		{"CastReceiverApplications", "[]"},
		// Present and empty: the reference's own values are stale constants
		// under obsolete markers, and spec 3.2 asks for a real value only
		// where one is meaningful.
		{"OperatingSystemDisplayName", `""`},
		{"EncoderLocation", `""`},
		{"SystemArchitecture", `""`},
	} {
		got, present := fields[row.field]
		if !present {
			t.Errorf("%s: absent, want %s", row.field, row.want)
			continue
		}
		if string(got) != row.want {
			t.Errorf("%s: got %s, want %s", row.field, got, row.want)
		}
	}

	if _, present := fields["PackageName"]; present {
		t.Errorf("PackageName is on the wire as %s; the reference declares it and does not send it (behaviours 1.7)", fields["PackageName"])
	}
}

// The seven paths are this installation's own, derived from the directory this
// server was started with — which the test knows, because it chose it.
//
// This is the assertion the request case in request-cases.yaml is describing
// when it calls these "installation paths that differ on every run": they are
// real, they are this installation's, and a differential run triages them
// rather than allowlisting them.
func TestSystemInfoReportsThisInstallationsPaths(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	fields := rawFields(t, server.get(t, systemInfoPath, goldenHost, nil).body)

	data := server.dataDirectory
	for _, row := range []struct{ field, want string }{
		{"ProgramDataPath", data},
		{"WebPath", filepath.Join(data, "web")},
		{"ItemsByNamePath", filepath.Join(data, "metadata")},
		{"CachePath", filepath.Join(data, "cache")},
		{"LogPath", filepath.Join(data, "log")},
		{"InternalMetadataPath", filepath.Join(data, "metadata")},
		{"TranscodingTempPath", filepath.Join(data, "cache", "transcodes")},
	} {
		want, err := json.Marshal(row.want)
		if err != nil {
			t.Fatalf("encoding the expected %s: %v", row.field, err)
		}
		if got := string(fields[row.field]); got != string(want) {
			t.Errorf("%s: got %s, want %s", row.field, got, want)
		}
	}

	// And none of them was created by this start: 001 creates the data
	// directory and nothing inside it, so a path field is an address rather
	// than a promise that something is at it. If a later feature starts
	// creating one, this is the line that says which.
	created := map[string]bool{}
	for _, name := range directoryContents(t, data) {
		created[name] = true
	}
	for _, name := range []string{"web", "metadata", "cache", "log"} {
		if created[name] {
			t.Errorf("the start created %s/%s; 001 creates the data directory and nothing in it (plan 4)", data, name)
		}
	}
}

// WebSocketPortNumber is the port this server is actually listening on, which
// the test knows because it read the address out of the server's own log.
//
// The port is chosen by the operating system here, so a handler answering a
// constant — 8096, or the port in the request's Host header, or the zero it was
// built with — fails. That is the whole reason the entry layer fills it in
// after it binds rather than reading it off the configuration.
func TestSystemInfoReportsThePortItIsListeningOn(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	fields := rawFields(t, server.get(t, systemInfoPath, goldenHost, nil).body)

	_, port, found := strings.Cut(strings.TrimPrefix(server.baseURL, "http://"), ":")
	if !found {
		t.Fatalf("the server's address carries no port: %s", server.baseURL)
	}
	if got := string(fields["WebSocketPortNumber"]); got != port {
		t.Errorf("WebSocketPortNumber: got %s, want %s — the port this server is listening on", got, port)
	}
	// The request named a different port in its Host header, so the two
	// candidate answers really are different numbers.
	if _, hostPort, _ := strings.Cut(goldenHost, ":"); hostPort == port {
		t.Fatalf("this test is vacuous: the Host header names the port the server bound (%s)", port)
	}
	if _, err := strconv.Atoi(port); err != nil {
		t.Errorf("the port is not a number: %s", port)
	}
}

// spec 3.2: the reference permits this route during first-time setup, before
// any user exists
// [source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11].
// That is the state every installation this binary can be started in is in, and
// it is what makes the route reachable at all today.
//
// A credential changes nothing: an unrecognised token is not a refusal at the
// reference either
// [source: Jellyfin.Api/Auth/CustomAuthenticationHandler.cs:48-51,79-83 @ v10.11.11].
//
// **The other half of AC-5 is not here, and that is recorded rather than
// papered over.** The 401 needs an installation whose setup is complete, and
// 001 has no endpoint that completes setup — 002 owns it. So the refusal is
// asserted at the HTTP boundary in internal/httpapi, over a real connection,
// where the store can be put into that state; and "200 with a valid token"
// needs a token nothing can issue and is a criterion carried into 002
// (tasks.md T18, T21).
func TestSystemInfoIsServedDuringFirstTimeSetup(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, row := range []struct {
		what   string
		header http.Header
	}{
		{"no credential at all", nil},
		{"a token nothing issued", http.Header{"X-Emby-Token": []string{"not-a-token"}}},
		{"an Authorization header", http.Header{"Authorization": []string{`MediaBrowser Token="not-a-token"`}}},
	} {
		got := server.get(t, systemInfoPath, goldenHost, row.header)
		if got.status != http.StatusOK {
			t.Errorf("%s: status %d, want %d\n%s", row.what, got.status, http.StatusOK, got.body)
		}
		fields := rawFields(t, got.body)
		if string(fields["StartupWizardCompleted"]) != "false" {
			t.Fatalf("%s: this installation's setup is not outstanding, so the test is measuring something else", row.what)
		}
	}
}

// rawFields reads an object's members, keeping each value as the bytes it
// arrived as. Principle VIII: a decoded value has already lost the difference
// between 8096 and "8096" and between [] and null.
func rawFields(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()

	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("the body is not a JSON object: %v\n%s", err, body)
	}
	return fields
}

// The devices this file's credentials are minted from.
//
// One each, because a second authentication from one DeviceId revokes the
// first token (002 plan 6.5): the administrator's credential has to survive
// the lockout sequence below, and the failed logins that lock an account must
// not be able to take it away by accident.
const (
	systemInfoDevice        = "system-info-administrator"
	systemInfoLockedDevice  = "system-info-locked-out"
	systemInfoFailureDevice = "system-info-failures"
)

// 002 AC-14, and spec 3.2's 403 row, on one installation.
//
// # This is the debt 001's closing audit recorded in 002's specification
//
// [001 AC-5] is three claims — 401 without a token, 200 with a valid one, and a
// body that is a superset of /System/Info/Public agreeing on every shared
// field. 001 proved the first and the third. The second needs a **valid
// credential**, and a credential is something this feature issues: 001 serves
// no route that authenticates anybody. It became 002 AC-14 rather than a note,
// for the reason 001 gave — a criterion carried in a sentence is one nobody
// closes.
//
// # The second request is what makes the first a proof
//
// On an installation whose setup is outstanding, GET /System/Info admits a
// request carrying nothing at all: the reference's authorisation handler
// succeeds on "!IsStartupWizardCompleted" before it looks at a role
// [source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11].
// Every installation 001 could start was in that state, so a test written
// against one would be green, named for AC-14, and proving nothing about the
// token. Two requests against **one** server settle it: the token answering 200
// with the superset body, and the *identical* request without the token
// answering 401. What this installation has that 001's could not is a completed
// setup, which spec 3.9 makes a consequence of holding one account and
// withProvisionedAccount therefore arranges.
//
// # Why the three criteria share a server
//
// T18's rule: provisioning 002 plan 8's fixture costs six Argon2id derivations
// and installations that do not disturb one another share one. These do not.
// AC-14 uses `administrator`, the 403 row uses `locked-out` and locks it, and
// nothing here reads a list of sessions or of users.
func TestSystemInfoAnswersTheCredentialRatherThanTheExemption(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)

	t.Run("AC-14: the token is answered 200 and the identical request without it is 401", func(t *testing.T) {
		assertTheTokenIsWhatAdmitsSystemInfo(t, server)
	})
	t.Run("spec 3.2's 403: a live token whose account a lockout disabled", func(t *testing.T) {
		assertALockedOutAccountsTokenIsTheEmptyForbidden(t, server)
	})
}

// AC-14 and its companion: one server, two requests, one header apart.
//
// The pair is one function rather than two subtests on purpose. What the
// criterion asserts is a *difference between two requests*, and two subtests
// each holding one of them could drift into asserting two unrelated facts —
// which is the shape 001's audit caught twice.
func assertTheTokenIsWhatAdmitsSystemInfo(t *testing.T, server *server) {
	t.Helper()

	held := logIn(t, server, systemInfoDevice, administratorAccount, fixturePassword)

	// The two headers differ in the Token parameter and in nothing else, so
	// the pair below cannot be told apart by the client identification, by the
	// path, by the method or by the Host.
	bearing := held.bearing()
	unbearing := http.Header{"Authorization": {clientIdentification(held.device, "")}}

	admitted := server.get(t, systemInfoPath, goldenHost, bearing)
	if admitted.status != http.StatusOK {
		t.Fatalf("a token this server issued was answered %d, want %d\nbody: %s",
			admitted.status, http.StatusOK, admitted.body)
	}
	if contentType := admitted.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "application/json; charset=utf-8")
	}

	// The state the criterion has to be asserted in, stated rather than
	// assumed. A build that never recorded setup completion answers false here
	// and admits everything below, and this line is what says so instead of
	// letting the 401 assertion fail with a status nobody can explain.
	fields := rawFields(t, admitted.body)
	if got := string(fields["StartupWizardCompleted"]); got != "true" {
		t.Fatalf("StartupWizardCompleted = %s on a provisioned installation; "+
			"while setup is outstanding this route admits every request, so nothing below would be about the token", got)
	}

	// The superset half of 001 AC-5, now on a body a credential admitted. The
	// public route needs none, which is what lets the same server answer both.
	public := server.get(t, publicSystemInfoPath, goldenHost, nil)
	if public.status != http.StatusOK {
		t.Fatalf("%s: status %d\n%s", publicSystemInfoPath, public.status, public.body)
	}
	assertTheSupersetAgreesWithThePublicBody(t, admitted.body, public.body)
	if names := propertyNames(t, admitted.body); !equalStrings(names, systemInfoNames) {
		t.Errorf("property names:\n got %v\nwant %v", names, systemInfoNames)
	}

	// The companion, and the whole reason this test is a proof.
	assertEmptyRefusal(t, server.get(t, systemInfoPath, goldenHost, unbearing),
		http.StatusUnauthorized, "the identical request with the Token parameter removed")

	// And the same request carrying no credential at all, which is the request
	// 001 could only send to a server that answered it 200.
	assertEmptyRefusal(t, server.get(t, systemInfoPath, goldenHost, nil),
		http.StatusUnauthorized, "no credential at all")
}

// spec 3.2's 403 row, which 001 recorded as **unreachable** and this feature
// can now enter.
//
// 001's tasks.md says of it: *"Untested, and unreachable. 001 issues no
// credential, so no request can be valid and insufficient."* Both halves of
// that are now false — this feature issues credentials, and it has a state in
// which one is valid and its holder is refused.
//
// # How a live token comes to belong to a disabled account
//
// Not by provisioning one disabled: `disabled` is refused at the login route
// (AC-2), so it never holds a token. The state is reached the way an operator
// would reach it by accident — the account authenticates, and is *then*
// locked out. A lockout is stored as the disabled flag (002 plan 6.7, and the
// reference does the same
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @ v10.11.11]),
// so the token that was minted before the failures is a valid credential whose
// holder this server will no longer serve.
//
// # The shape is the empty one, and that is the assertion
//
// behaviours 1.11 has four error shapes, and the two 403s are two of them: a
// controller's own refusal is text/plain with no charset and 25 bytes, while an
// authorization **policy**'s refusal has no body and no content type
// [probe: tools/probe_playlist_visibility.py, Jellyfin 10.11.11, 2026-08-31].
// This route's 403 is the policy one. Nothing at the wire asserted that until
// now — internal/httpapi asserts it over a stubbed store, which is where the
// declared length and the two absent fields are seen beside a handler built in
// the test, and this is the same shape over a real installation.
func assertALockedOutAccountsTokenIsTheEmptyForbidden(t *testing.T, server *server) {
	t.Helper()

	held := logIn(t, server, systemInfoLockedDevice, lockedOutAccount, fixturePassword)

	// The token works first. Without this the 403 below would be the right
	// status for the wrong reason — a token that was never valid is a 401, and
	// a build that answered 403 to every credential would pass a test that only
	// looked afterwards.
	if before := server.get(t, systemInfoPath, goldenHost, held.bearing()); before.status != http.StatusOK {
		t.Fatalf("the token was answered %d before the lockout, want %d — "+
			"the refusal below would not be about the account\nbody: %s",
			before.status, http.StatusOK, before.body)
	}

	// The threshold is fixtureLockoutThreshold, and the reference's comparison
	// increments before it compares
	// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:636-641 @ v10.11.11],
	// so the second failure is the one that locks. The failures are sent from a
	// device of their own: a refusal opens no session, but a device is what a
	// successful login would replace a token on.
	for attempt := 1; attempt <= 2; attempt++ {
		refused := authenticate(t, server, systemInfoFailureDevice, lockedOutAccount, "not-the-password")
		if refused.status != http.StatusUnauthorized {
			t.Fatalf("failure %d: status %d, want %d\nbody: %s",
				attempt, refused.status, http.StatusUnauthorized, refused.body)
		}
	}

	assertEmptyRefusal(t, server.get(t, systemInfoPath, goldenHost, held.bearing()),
		http.StatusForbidden, "a live token whose account a lockout disabled")
}

// assertEmptyRefusal is behaviours 1.11's empty shape at the wire: the status,
// no body, a declared length of zero, no content type and no challenge.
//
// One function for every route that answers it, because the shape is one
// measurement and a second spelling of it is a second answer. The 401 and the
// policy 403 differ only in the status, which is why the status is a parameter
// rather than two near-identical helpers.
func assertEmptyRefusal(t *testing.T, got *response, status int, what string) {
	t.Helper()

	if got.status != status {
		t.Fatalf("%s: status %d, want %d\nbody: %s", what, got.status, status, got.body)
	}
	if len(got.body) != 0 {
		t.Errorf("%s: the refusal carries a body, want the empty shape: %q", what, got.body)
	}
	if length := got.header.Get("Content-Length"); length != "0" {
		t.Errorf("%s: Content-Length: got %q, want %q", what, length, "0")
	}
	if contentType := got.header.Get("Content-Type"); contentType != "" {
		t.Errorf("%s: Content-Type: got %q, want the field to be absent", what, contentType)
	}
	if challenge := got.header.Get("WWW-Authenticate"); challenge != "" {
		t.Errorf("%s: WWW-Authenticate: got %q, want the field to be absent", what, challenge)
	}
}
