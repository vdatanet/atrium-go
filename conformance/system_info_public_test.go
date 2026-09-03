package conformance_test

import (
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

// publicSystemInfoPath is the route under test, spelled as a client spells it.
const publicSystemInfoPath = "/System/Info/Public"

// goldenInstallationIdentity is the identity the golden body carries.
//
// A first start generates 16 cryptographically random bytes, so a body carrying
// the identity cannot be recorded unless the identity is stated. This is the
// value spec 3.1's own sample body uses.
const goldenInstallationIdentity = "3f9c1a7e5b2d4e8091a6c3f70d5e2b14"

// goldenHost is the Host header the golden request carries.
//
// spec 3.4's tiers answer LocalAddress per requester, and an installation with
// no published URL, no derivation and no bound address answers from the request
// itself (plan 6.6) — which is every installation this binary can be started
// as, because 001 gives an operator no way to configure the other two. So the
// address in the body is the address the client asked for, and stating it in
// the request is what makes the recorded body reproducible on a server whose
// port the operating system chose.
const goldenHost = "192.168.1.20:8096"

// The byte-compared golden of AC-1, AC-2 and AC-3, on an installation that has
// nothing but a data directory and a stated identity.
//
// It asserts on bytes because that is the only level at which the property
// casing, the absence of any eighth field, the key order and the JSON type of
// StartupWizardCompleted are all visible at once. Every one of those is
// invisible to a test that decodes the body first.
//
// It is deliberately a poor diagnostic and that is why the test below exists
// beside it: a golden says the response changed, and the field-by-field
// assertions say which field moved.
func TestPublicSystemInfoMatchesItsGolden(t *testing.T) {
	t.Parallel()

	server := startServer(t, withInstallationIdentity(goldenInstallationIdentity))
	got := server.get(t, publicSystemInfoPath, goldenHost, nil)

	if got.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}
	if contentType := got.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "application/json; charset=utf-8")
	}

	assertGolden(t, "system_info_public.json", got.body)
}

// installationIdentity is the shape AC-4 requires of Id: 32 lowercase hex.
var installationIdentity = regexp.MustCompile(`^[0-9a-f]{32}$`)

// The field-by-field half of the same criteria, on a genuinely fresh
// installation — nothing on disk but the data directory, so no user exists and
// no library is configured (AC-2, AC-3).
//
// This is what a golden cannot do. A golden diff says the response changed; an
// assertion per field says which field moved and to what, and the three values
// spec 3.1 fixes as literals — ProductName, OperatingSystem, Version — each
// fail on their own line rather than as one wall of bytes.
//
// The values are compared as **raw JSON**, not as decoded Go values, so that
// `"StartupWizardCompleted": "false"` fails here rather than passing as a
// truthy string and so that an empty OperatingSystem is told apart from an
// absent one.
func TestPublicSystemInfoAnswersFieldByFieldOnAnEmptyInstallation(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	// AC-2 and AC-3 in as many words: whatever this answers, it answers with
	// nothing configured. If a later change ever seeds this installation, this
	// is the line that says the criterion stopped being tested.
	if len(server.seeded) != 0 {
		t.Fatalf("this installation was not empty when it started: %v", server.seeded)
	}

	got := server.get(t, publicSystemInfoPath, goldenHost, nil)
	if got.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}

	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(got.body, &fields); err != nil {
		t.Fatalf("the body is not a JSON object: %v\n%s", err, got.body)
	}

	for _, row := range []struct{ field, want string }{
		// spec 3.4 tier 2's shape, from the Host header this request sent.
		{"LocalAddress", `"http://192.168.1.20:8096"`},
		// plan 4's default, which is what an installation nobody has renamed
		// holds.
		{"ServerName", `"atrium"`},
		// The pinned reference version, not this binary's own
		// (reference-target.md 4).
		{"Version", `"10.11.11"`},
		// The discriminator a multi-server client reads (behaviours 4.1).
		{"ProductName", `"Jellyfin Server"`},
		// Present and empty, because the reference marks the property obsolete
		// and never assigns it
		// [source: MediaBrowser.Model/System/PublicSystemInfo.cs:37-38 @ v10.11.11].
		{"OperatingSystem", `""`},
		// False, because setup has not been completed on an installation with
		// nothing on disk. spec 3.1's sample body shows true, and that is a
		// *configured* server.
		{"StartupWizardCompleted", "false"},
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

	// Id is the one field whose value a fresh installation chooses, so the
	// assertion is on its shape (AC-4's first half).
	var id string
	if err := json.Unmarshal(fields["Id"], &id); err != nil {
		t.Errorf("Id: not a JSON string: %s", fields["Id"])
	} else if !installationIdentity.MatchString(id) {
		t.Errorf("Id: got %q, want 32 lowercase hex characters", id)
	}
}

// Exactly seven fields, in spec 3.1's order.
//
// Principle I is the count: an eighth field is a delta whether or not a client
// reads it, and a missing seventh is a client on the unknown-server path. The
// order is asserted because L3 compares bytes, so key order is contract — and
// because a struct field reordered while refactoring is exactly the change
// nobody thinks of as a change.
func TestPublicSystemInfoCarriesSevenFieldsInSpecOrder(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.get(t, publicSystemInfoPath, goldenHost, nil)

	want := []string{
		"LocalAddress",
		"ServerName",
		"Version",
		"ProductName",
		"OperatingSystem",
		"Id",
		"StartupWizardCompleted",
	}
	if names := propertyNames(t, got.body); !equalStrings(names, want) {
		t.Errorf("property names:\n got %v\nwant %v", names, want)
	}
}

// propertyNames reads the keys of a JSON object in the order the bytes carry
// them, which is a thing no map can tell you.
func propertyNames(t *testing.T, body []byte) []string {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	open, err := decoder.Token()
	if err != nil {
		t.Fatalf("reading the body: %v\n%s", err, body)
	}
	if delimiter, ok := open.(json.Delim); !ok || delimiter != '{' {
		t.Fatalf("the body is not a JSON object: %s", body)
	}

	var names []string
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading a property name: %v\n%s", err, body)
		}
		text, ok := name.(string)
		if !ok {
			t.Fatalf("a property name is not a string: %v", name)
		}
		names = append(names, text)

		// Skip the value whole, whatever shape it is.
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil && err != io.EOF {
			t.Fatalf("reading the value of %s: %v", text, err)
		}
	}
	return names
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
