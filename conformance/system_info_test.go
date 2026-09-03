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

	publicNames := propertyNames(t, public.body)
	publicFields := rawFields(t, public.body)
	authenticatedFields := rawFields(t, authenticated.body)

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

	want := []string{
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
	if names := propertyNames(t, got.body); !equalStrings(names, want) {
		t.Errorf("property names:\n got %v\nwant %v", names, want)
	}
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
