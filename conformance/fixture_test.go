package conformance_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The installation 002's wire criteria are asserted against, and the one way
// this package authenticates anybody.
//
// # The fixture is built by the command an operator runs
//
// 002 plan 8 names six accounts, and every one of them is created by
// `atrium user add` through withProvisionedAccount. That is the point rather
// than a limitation: this package may import nothing of ours
// (tools/check_conformance_imports enforces it), so there is no back door into
// the store and could not be one. What these tests can put an installation
// into is exactly what an operator can put it into.
//
// # A bare `user add` is HIDDEN, so the flags read backwards
//
// The reference grants a freshly created account PermissionKind.IsHidden
// **true** [source: Jellyfin.Data/UserEntityExtensions.cs:173 @ v10.11.11] and
// users.DefaultPolicy() transcribes it, so `--hidden` alone is a no-op and
// `--hidden=false` is what puts an account on a login screen (002 plan 6.9's
// amendment). The consequence for this file: the *hidden* account is the bare
// one, and "every user is hidden" needs no flags at all — which is why the
// second fixture below is one line.

// The six accounts of 002 plan 8, named for the seat each one is.
//
// A name rather than a person's, because these strings appear in a failing
// test's output and in the golden body of AC-1, where "administrator" says
// what the row is for and "Ada" would not. The account identifier is derived
// from the name (Principle VII), so naming a seat also pins its identifier.
const (
	administratorAccount = "administrator"
	restrictedAccount    = "restricted"
	hiddenAccount        = "hidden"
	disabledAccount      = "disabled"
	passwordlessAccount  = "no-password"
	lockedOutAccount     = "locked-out"
)

// The credential every account in the fixture holds, and the lockout threshold
// the last of them is provisioned with.
//
// The password is a literal in a public repository and that is deliberate, for
// the reason sweep_test.go and provisioning_test.go both give: it authenticates
// against an installation these tests create in a temporary directory and
// destroy when they end, and a value that looked like a real secret would be
// worse rather than better.
const (
	fixturePassword = "hunter2"

	// Two, so that AC-10's fixture (T20) locks out after two failures rather
	// than after the reference's default. It is provisioned here because the
	// account is one of plan 8's six and the fixture is built in one place.
	fixtureLockoutThreshold = "2"
)

// fixtureAccounts is 002 plan 8's fixture as a list of provisioning options.
//
// Ordered, and the order is load-bearing in exactly one way: the first
// `user add` completes setup (002 plan 6.8), so every server built on this
// list answers StartupWizardCompleted true and GET /System/Info requires a
// credential. Which account is first does not matter; that there is one does.
func fixtureAccounts() []serverOption {
	return []serverOption{
		// An administrator with a password. Visible, because a login screen is
		// where a client finds it.
		withProvisionedAccount(administratorAccount, fixturePassword+"\n",
			"--administrator", "--hidden=false"),

		// A restricted non-administrator with a password: the seat that proves
		// a refusal is about who is asking rather than about whether anybody
		// is.
		withProvisionedAccount(restrictedAccount, fixturePassword+"\n", "--hidden=false"),

		// A hidden user — bare, because bare is hidden.
		withProvisionedAccount(hiddenAccount, fixturePassword+"\n"),

		// A disabled user, which is AC-2's 403 and is otherwise an ordinary
		// account with a correct password.
		withProvisionedAccount(disabledAccount, fixturePassword+"\n",
			"--disabled", "--hidden=false"),

		// An account with no password. Nothing is read from standard input for
		// this one, which is what --no-password means.
		withProvisionedAccount(passwordlessAccount, "", "--no-password", "--hidden=false"),

		// An account that locks out after two failures rather than after the
		// default.
		withProvisionedAccount(lockedOutAccount, fixturePassword+"\n",
			"--login-attempts-before-lockout", fixtureLockoutThreshold, "--hidden=false"),
	}
}

// newInstallation starts a server on 002 plan 8's fixture.
//
// Further options are appended, so a caller that needs a stated installation
// identity passes withInstallationIdentity and gets it written into the same
// data directory before anything else runs.
func newInstallation(t *testing.T, options ...serverOption) *server {
	t.Helper()
	return startServer(t, append(fixtureAccounts(), options...)...)
}

// newAllHiddenInstallation starts a server on which every account is hidden.
//
// This is 002 plan 8's second fixture and it is one line because a bare
// `user add` is hidden: an installation holding nothing but bare accounts is
// an installation with nobody on its login screen. AC-6's assertion over it
// belongs to T19; what T18 uses it for is AC-3, where /Users/Public answers
// the same bytes to every credential including on a body that is empty.
func newAllHiddenInstallation(t *testing.T, options ...serverOption) *server {
	t.Helper()
	return startServer(t, append([]serverOption{
		withProvisionedAccount(hiddenAccount, fixturePassword+"\n"),
	}, options...)...)
}

// loginBody is the request body of 002 spec 3.3, written as a client writes it.
//
// Concatenated rather than marshalled, because Principle VIII: what a test
// states is what goes on the wire, and a marshaller is a second opinion about
// that. Every name and password in this package is plain ASCII with nothing to
// escape, and a caller that needs one that is not should state the bytes.
func loginBody(username, password string) []byte {
	return []byte(`{"Username":"` + username + `","Pw":"` + password + `"}`)
}

// authenticate posts one POST /Users/AuthenticateByName and hands back whatever
// answered, refusal included.
//
// It is deliberately not an assertion: AC-2's three requests are refusals and
// they go through here too. logIn below is the half that insists on success.
func authenticate(t *testing.T, s *server, device, username, password string) *response {
	t.Helper()
	return s.send(t, http.MethodPost, authenticateByNamePath, goldenHost,
		http.Header{"Authorization": {clientIdentification(device, "")}},
		loginBody(username, password))
}

// credential is what one successful authentication handed back.
//
// The three identifiers are read off the response rather than derived here.
// This package cannot derive one — it may import nothing of ours — and
// transcribing what the server computed would be a test agreeing with itself.
type credential struct {
	token     string
	userID    string
	sessionID string

	// device is the DeviceId this credential was minted from. It is carried
	// because a second authentication from the same device revokes this token
	// (002 plan 6.5), so a caller that needs a *second* live credential has to
	// know which device not to reuse.
	device string
}

// logIn authenticates and refuses to continue if it did not work.
//
// **There is one of these in this package on purpose.** T17's fixture had the
// login inline; two ways of authenticating in one package are two answers to
// what a token is, and the day they disagree the failure is in whichever test
// happens to run first. newSweepFixture calls this too.
func logIn(t *testing.T, s *server, device, username, password string) credential {
	t.Helper()

	got := authenticate(t, s, device, username, password)
	if got.status != http.StatusOK {
		t.Fatalf("authenticating %s from %s: status %d, want %d\nbody: %s",
			username, device, got.status, http.StatusOK, got.body)
	}

	session := json.RawMessage(rawField(t, got.body, "SessionInfo"))
	held := credential{
		token:     unquote(t, rawField(t, got.body, "AccessToken")),
		userID:    unquote(t, rawField(t, []byte(rawField(t, got.body, "User")), "Id")),
		sessionID: unquote(t, rawField(t, session, "Id")),
		device:    device,
	}
	if held.token == "" || held.userID == "" || held.sessionID == "" {
		t.Fatalf("the login answered an empty identifier or token:\n%s", got.body)
	}
	return held
}

// bearing is the header a client sends once it holds a token: its own
// identification and the token together, which is what 002 spec 3.1 says most
// requests carry.
func (c credential) bearing() http.Header {
	return http.Header{"Authorization": {clientIdentification(c.device, c.token)}}
}
