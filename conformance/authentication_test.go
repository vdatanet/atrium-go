package conformance_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
	"time"
)

// 002 AC-1, AC-2 and AC-5 against the running binary.
//
// All three were already asserted at the HTTP boundary inside internal/httpapi,
// over a handler built in the test. They are here as well, and the reason is
// 001's closing audit stated as a policy by 002 plan 8: **a criterion written
// about a request is not met by a test about the mechanism that serves it,
// however good that test is.** What is new here is that the account was made by
// the command an operator runs, the token was minted by the process, and the
// bytes came off a socket.

// accessToken is the shape 002 spec 3.3 measures: 32 lowercase hexadecimal
// characters [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
// 2026-08-28].
var accessToken = regexp.MustCompile(`^[0-9a-f]{32}$`)

// goldenLoginDevice is the DeviceId the recorded authentication was made from.
//
// Stated, because sessions.DeriveID hashes the client and the device into
// SessionInfo.Id — so the identifier in the golden is a consequence of this
// constant and of clientIdentification's own Client, and neither is a record of
// one particular run.
const goldenLoginDevice = "golden-login"

// AC-1 at the wire, as a byte-compared golden.
//
// # What the golden states, and why three members are stated rather than two
//
// 002 plan 8 says the golden holds the body "with the two derived members
// stated". Recording it found **three**, and the third is the interesting one.
// Six members of this body derive from something, and each is stated by
// stating its input, which is 001 T16's rule:
//
//   - ServerId, twice, and User.ServerId — the installation identity, stated by
//     writing the file before the server starts.
//   - User.Id and SessionInfo.UserId — users.DeriveID over the folded account
//     name, stated by naming the account.
//   - SessionInfo.Id — sessions.DeriveID over the client and the device, stated
//     by clientIdentification and goldenLoginDevice.
//
// The three that are left have no input to state. AccessToken is 16 bytes of
// the system's randomness and the two dates are wall clocks, and this binary
// gives an operator no way to fix any of them. They are therefore stated in the
// golden at the positions they occupy, and each value is asserted against a
// rule of its own before it is put there — see assertGoldenWithStatedMembers.
// plan 8's row is amended with the count.
//
// # What the byte comparison is for
//
// Everything a decoded body cannot see: the four top-level members in the order
// the server writes them, the sixty policy and configuration members inside
// User, PascalCase throughout, `LastPlaybackCheckIn` as .NET's minimum date
// rather than null, empty arrays as `[]` rather than `null`, and
// `SupportsMediaControl` as the boolean false rather than the string.
func TestAnAuthenticationMatchesItsGolden(t *testing.T) {
	t.Parallel()

	server := newInstallation(t, withInstallationIdentity(goldenInstallationIdentity))

	// The window the two dates must fall inside. Read around the request and
	// widened by a second at each end, because the server's clock and this
	// process's are the same clock but the response is written between two
	// reads of it rather than at either.
	before := time.Now().UTC().Add(-time.Second)
	got := authenticate(t, server, goldenLoginDevice, administratorAccount, fixturePassword)
	after := time.Now().UTC().Add(time.Second)

	if got.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}
	if contentType := got.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: got %q, want %q", contentType, "application/json; charset=utf-8")
	}

	// AC-1's own three assertions, made before the golden so that a failure
	// says which member is wrong rather than printing two kilobytes twice.
	names := propertyNames(t, got.body)
	if len(names) != 4 {
		t.Fatalf("the result carries %d members, want 4: %v", len(names), names)
	}
	for _, wanted := range []string{"User", "SessionInfo", "AccessToken", "ServerId"} {
		if !slices.Contains(names, wanted) {
			t.Errorf("%s is absent from the authentication result: %v", wanted, names)
		}
	}

	token := unquote(t, rawField(t, got.body, "AccessToken"))
	if !accessToken.MatchString(token) {
		t.Errorf("AccessToken = %q, want 32 lowercase hexadecimal characters", token)
	}

	session := json.RawMessage(rawField(t, got.body, "SessionInfo"))
	user := json.RawMessage(rawField(t, got.body, "User"))
	loginDate := statedDate(t, "LastLoginDate", rawField(t, user, "LastLoginDate"), before, after)
	activityDate := statedDate(t, "LastActivityDate", rawField(t, session, "LastActivityDate"), before, after)

	assertGoldenWithStatedMembers(t, "authenticate_by_name.json", got.body, []statedMember{
		{name: "LastLoginDate", value: loginDate},
		{name: "LastActivityDate", value: activityDate},
		{name: "AccessToken", value: token},
	})
}

// statedDate asserts one wall-clock member before it is allowed into a golden.
//
// Two rules, and both are needed. The shape rule is the cross-cutting L1 sweep
// applied to a value; the window is what stops a build answering a constant.
// **A constant passes a shape assertion** — .NET's minimum date is a perfectly
// well-formed seven-digit wire date, and a server that stamped every login with
// it would satisfy the regular expression, be substituted into the golden and
// report green. That trap has now caught three tasks of this feature in
// internal/, and it is the same trap at the wire.
func statedDate(t *testing.T, member, raw string, before, after time.Time) string {
	t.Helper()

	// wireDate is sweep_test.go's, and it is the rule the cross-cutting L1
	// sweep applies to every date this server sends. Called rather than
	// restated: a second copy of it is a second opinion about what a date is.
	value := unquote(t, raw)
	if !wireDate.MatchString(value) {
		t.Fatalf("%s = %q, want a date with seven fractional digits and a Z", member, value)
	}
	stamped, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("%s = %q, which does not parse: %v", member, value, err)
	}
	if stamped.Before(before) || stamped.After(after) {
		t.Fatalf("%s = %q, which is outside the window this request was made in (%s to %s)",
			member, value, before.Format(time.RFC3339Nano), after.Format(time.RFC3339Nano))
	}
	return value
}

// The three refusals of 002 AC-2, and the one body all three carry.
//
// # One golden compared by three requests
//
// That is what makes "all three carry the same 25 bytes" an assertion rather
// than three assertions written to look alike. Three goldens with identical
// contents would pass on a build where one refusal grew a full stop, because
// each would have been recorded from the response it is compared against.
//
// # Content-Type is asserted as a field value and not by a Contains
//
// The measured field is `text/plain` with **no charset parameter**
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26], and
// the absence of the parameter is the part a client sees. A Contains check for
// "text/plain" passes on `text/plain; charset=utf-8`, which is a different
// field, so the whole value is compared.
func TestTheThreeMeasuredRefusalsCarryOneBodyAndThreeStatuses(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)

	for _, row := range []struct {
		name   string
		status int
		send   func(t *testing.T) *response
	}{
		{
			// 002 spec 3.3: an unknown username is 401
			// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26].
			name:   "an unknown username",
			status: http.StatusUnauthorized,
			send: func(t *testing.T) *response {
				return authenticate(t, server, "refusal-unknown", "nobody-has-this-name", fixturePassword)
			},
		},
		{
			// 002 spec 3.3: a disabled account is 403 **whether the password is
			// right or wrong**, and the password here is right — which is the
			// half that makes the criterion about the account rather than about
			// the credential
			// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26].
			name:   "a disabled account with the correct password",
			status: http.StatusForbidden,
			send: func(t *testing.T) *response {
				return authenticate(t, server, "refusal-disabled", disabledAccount, fixturePassword)
			},
		},
		{
			// 002 spec 3.2: the client identification is mandatory on this
			// route and on no other. Neither header name is sent, because the
			// route reads both (002 plan 6.1) and sending one of them would
			// prove only that the other is unread.
			name:   "no client identification at all",
			status: http.StatusBadRequest,
			send: func(t *testing.T) *response {
				return server.send(t, http.MethodPost, authenticateByNamePath, goldenHost,
					nil, loginBody(administratorAccount, fixturePassword))
			},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := row.send(t)

			if got.status != row.status {
				t.Errorf("status: got %d, want %d\nbody: %s", got.status, row.status, got.body)
			}
			if contentType := got.header.Get("Content-Type"); contentType != "text/plain" {
				t.Errorf("Content-Type: got %q, want %q — the charset parameter is absent and that is the measurement",
					contentType, "text/plain")
			}
			if length := got.header.Get("Content-Length"); length != "25" {
				t.Errorf("Content-Length: got %q, want %q", length, "25")
			}
			assertGolden(t, refusalGolden, got.body)
		})
	}
}

// refusalGolden is this package's copy of the 25 bytes every measured refusal
// of 002 spec 3.3 carries.
const refusalGolden = "authenticate_refusal.txt"

// The copy is a copy, asserted rather than assumed.
//
// internal/httpapi holds the same bytes under its own testdata, because six
// responses are compared against it there. A golden's directory is per package
// and this one may not import that one, so the file exists twice — and two
// files that are meant to be identical and are never compared are two files
// that will one day differ, with the wire's copy being the one nobody edits.
//
// Reading the other package's file is a **file read and not an import**: it
// tells this test nothing about the server it is talking to, and
// tools/check_conformance_imports still passes because nothing here is linked.
// If that file moves or is renamed, this fails loudly and names it.
func TestTheRefusalGoldenIsTheSameBytesAsTheOneInternalHttpapiCompares(t *testing.T) {
	t.Parallel()

	other := filepath.Join("..", "internal", "httpapi", "testdata", "golden", refusalGolden)

	mine, err := os.ReadFile(filepath.Join(goldenDirectory, refusalGolden))
	if err != nil {
		t.Fatalf("reading this package's copy: %v", err)
	}
	theirs, err := os.ReadFile(other)
	if err != nil {
		t.Fatalf("reading %s: %v\nIf it moved, this copy is now the only one and that is worth knowing.", other, err)
	}
	if string(mine) != string(theirs) {
		t.Errorf("the two copies of the refusal body differ.\n here %q\nthere %q", mine, theirs)
	}
}

// 002 AC-5: re-authenticating from the same DeviceId replaces the session and
// invalidates the token the first authentication issued.
//
// # What each assertion catches, and what the row count really asserts
//
// The first token answering 401 is the criterion's own clause, and a build that
// never revokes anything fails it — measured: with the revocation removed, the
// stale token answers 200 with the whole user object
// `[measurement: conformance/, Go 1.27.0, 2026-09-03]`.
//
// The tokens differing is here for the reason T7 and T9 both recorded one layer
// in: **a constant passes a shape assertion.** A server minting one token for
// ever satisfies "32 lowercase hexadecimal characters", is substituted happily
// into AC-1's golden, and makes the first token keep working while looking like
// a replacement. Only this line sees it.
//
// The single row is **not** an assertion about accumulation, and saying so is
// more useful than implying it. `sessions` carries UNIQUE (client, device_id),
// so a build that derived a fresh identifier per login cannot write a second
// row — it answers 500 at the second authentication
// `[measurement: conformance/, a counter added to sessions.DeriveID, Go 1.27.0,
// 2026-09-03]`. What the count and the two identifier comparisons do assert is
// that the surviving row is the *replacement* — the second login's session, on
// the device it was made from — rather than the first login's left behind.
func TestASecondAuthenticationFromOneDeviceReplacesTheFirst(t *testing.T) {
	t.Parallel()

	// A server of its own, and an administrator, because the third assertion
	// is a count over GET /Sessions: an administrator sees every session on the
	// installation (002 spec 3.8), so any other session anybody opened would
	// be counted too. On this server there are none.
	server := newInstallation(t)

	const device = "one-device"
	first := logIn(t, server, device, administratorAccount, fixturePassword)
	second := logIn(t, server, device, administratorAccount, fixturePassword)

	if first.token == second.token {
		t.Fatalf("both authentications answered the same token %q, so nothing was replaced", first.token)
	}
	if first.sessionID != second.sessionID {
		t.Errorf("the two sessions have different identifiers (%q and %q), and a replacement keeps the derived one",
			first.sessionID, second.sessionID)
	}

	stale := server.get(t, currentUserPath, goldenHost, first.bearing())
	if stale.status != http.StatusUnauthorized {
		t.Errorf("the first token answered %d on %s, want %d\nbody: %s",
			stale.status, currentUserPath, http.StatusUnauthorized, stale.body)
	}
	if len(stale.body) != 0 {
		t.Errorf("the refusal carries a body, want the empty 401 shape: %q", stale.body)
	}

	live := server.get(t, currentUserPath, goldenHost, second.bearing())
	if live.status != http.StatusOK {
		t.Fatalf("the second token answered %d on %s, want %d\nbody: %s",
			live.status, currentUserPath, http.StatusOK, live.body)
	}

	sessions := server.get(t, sessionsPath, goldenHost, second.bearing())
	if sessions.status != http.StatusOK {
		t.Fatalf("status: got %d, want %d\nbody: %s", sessions.status, http.StatusOK, sessions.body)
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(sessions.body, &rows); err != nil {
		t.Fatalf("the session list is not a JSON array: %v\n%s", err, sessions.body)
	}
	if len(rows) != 1 {
		t.Fatalf("two authentications from one device left %d sessions, want 1\n%s", len(rows), sessions.body)
	}
	if got := unquote(t, rawField(t, rows[0], "Id")); got != second.sessionID {
		t.Errorf("the surviving session is %q, want the one the second authentication answered (%q)",
			got, second.sessionID)
	}
	if got := unquote(t, rawField(t, rows[0], "DeviceId")); got != device {
		t.Errorf("the surviving session's DeviceId is %q, want %q", got, device)
	}
}
