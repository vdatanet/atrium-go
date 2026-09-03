package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
)

// These tests run against the real store rather than a fake one, and that is a
// decision rather than convenience.
//
// The rule under test in the last of them is *what the store holds afterwards*
// — 002 plan 6.10's "at most once per session per second" — and the throttle's
// state is the stored date itself. A fake that answered whatever it was last
// handed would agree with any implementation that kept its own copy of the
// last-written instant, which is exactly the build the rule forbids. The one
// thing a fake is used for below is the failure a real store will not produce
// on demand: an unreadable store.
//
// internal/httpapi is the edge and may not *import* the store (architecture 2);
// this is its external test package, which is a different thing — the same
// shape internal/app's tests already take.

// aTestInstant is when these tests happen. A fixed instant rather than
// time.Now, because the whole of the activity rule is about how far apart two
// instants are.
var aTestInstant = units.At(time.Date(2026, 9, 3, 10, 30, 0, 0, time.UTC))

// settableClock is ports.Clock with the hands where a test put them.
type settableClock struct{ at units.Time }

func (c *settableClock) Now() units.Time { return c.at }

// advance moves the clock by a duration, rounded to the tick.
func (c *settableClock) advance(d time.Duration) {
	c.at = units.TimeFromTicks(c.at.Ticks() + units.TicksFromDuration(d))
}

// openStore opens a real store on a temporary data directory.
func openStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("opening a store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("closing the store: %v", err)
		}
	})
	return store
}

// createAccount writes one account whose policy is the reference's defaults
// with whatever the caller changed, and returns its identifier.
//
// The policy travels as the bytes of its document because that is what the
// port takes (002 plan 5), and it is built from users.DefaultPolicy rather than
// from a struct literal so that a test never asserts against Go's zero value
// for a flag it did not name.
func createAccount(t *testing.T, store *sqlite.Store, username string, shape func(*users.Policy)) string {
	t.Helper()

	policy := users.DefaultPolicy()
	if shape != nil {
		shape(&policy)
	}
	policyDocument, err := policy.Document()
	if err != nil {
		t.Fatalf("encoding the policy of %q: %v", username, err)
	}
	configurationDocument, err := users.DefaultConfiguration().Document()
	if err != nil {
		t.Fatalf("encoding the configuration of %q: %v", username, err)
	}

	user := ports.User{
		ID:                    users.DeriveID(username),
		Username:              username,
		UsernameFolded:        users.Fold(username),
		PolicyDocument:        policyDocument,
		ConfigurationDocument: configurationDocument,
	}
	if err := store.CreateUser(context.Background(), user); err != nil {
		t.Fatalf("creating the account %q: %v", username, err)
	}
	return user.ID
}

// openSessionFor opens a session for this account on this client and device,
// holding token, as of at. It returns the session identifier.
func openSessionFor(t *testing.T, store *sqlite.Store, userID, client, deviceID, token string, at units.Time) string {
	t.Helper()

	session := ports.Session{
		ID:                 sessions.DeriveID(client, deviceID),
		UserID:             userID,
		Client:             client,
		DeviceID:           deviceID,
		DeviceName:         "A Device",
		ApplicationVersion: "1.0.0",
		RemoteEndpoint:     "192.168.1.44",
		CreatedAt:          at,
		LastActivityAt:     at,
	}
	if err := store.OpenSession(context.Background(), session, sessions.TokenDigest(token)); err != nil {
		t.Fatalf("opening a session for %s: %v", userID, err)
	}
	return session.ID
}

// newAuthenticator builds the authenticator over a store and a clock.
func newAuthenticator(t *testing.T, store *sqlite.Store, clock ports.Clock) *httpapi.TokenAuthenticator {
	t.Helper()
	authenticator, err := httpapi.NewTokenAuthenticator(httpapi.TokenAuthenticatorConfig{
		Sessions: store,
		Accounts: store,
		Clock:    clock,
	})
	if err != nil {
		t.Fatalf("building the authenticator: %v", err)
	}
	return authenticator
}

// requestWithToken is a request presenting a token in the header every client
// sends it in.
func requestWithToken(token string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/Sessions", nil)
	r.Header.Set(httpapi.AuthorizationHeader, `MediaBrowser Client="Atrium Test", Device="A Device", DeviceId="device-1", Version="1.0.0", Token="`+token+`"`)
	return r
}

// sessionActivity reads one session's LastActivityDate back out of the store.
func sessionActivity(t *testing.T, store *sqlite.Store, sessionID string) units.Time {
	t.Helper()

	open, err := store.Sessions(context.Background())
	if err != nil {
		t.Fatalf("reading the sessions: %v", err)
	}
	for _, session := range open {
		if session.ID == sessionID {
			return session.LastActivityAt
		}
	}
	t.Fatalf("session %s is not in the store", sessionID)
	return units.Time{}
}

// The invariant 001 built on, asserted as the zero value rather than as a
// sentence in a comment.
//
// 001 plan 6.10 relies on it: a nil Authenticator — and any future failure to
// wire one — admits nobody, because the value a caller gets for free is the one
// that refuses. Widening the return from Access to Authentication is only safe
// while this holds, which is why it is asserted on the widened type and not
// only on Access (that assertion is TestTheZeroAccessAdmitsNobody, and it
// stays).
func TestTheZeroAuthenticationIsUnauthenticatedWithNoCaller(t *testing.T) {
	t.Parallel()

	var zero httpapi.Authentication
	if zero.Access != httpapi.AccessUnauthenticated {
		t.Errorf("the zero Authentication has Access %d, and it must be AccessUnauthenticated (%d)",
			zero.Access, httpapi.AccessUnauthenticated)
	}
	if zero.Access == httpapi.AccessGranted {
		t.Error("the zero Authentication admits the request, which is the wrong direction for a value that decides admission")
	}
	if zero.Caller != nil {
		t.Errorf("the zero Authentication carries a caller (%+v), and unauthenticated means there is nobody", *zero.Caller)
	}
}

// A credential nobody presented and a credential nobody recognises are the same
// answer at this port: the zero Authentication and no error.
//
// 002 plan 7's first two rows make them the same answer at the wire too, which
// is measured
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26], and
// the assertion below is comparison rather than two lists of expectations —
// two answers that are one value cannot drift apart.
func TestNoCredentialAndAnUnknownTokenAreTheSameAnswer(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	userID := createAccount(t, store, "Ada", nil)
	openSessionFor(t, store, userID, "Atrium Test", "device-1", "0123456789abcdef0123456789abcdef", aTestInstant)

	authenticator := newAuthenticator(t, store, &settableClock{at: aTestInstant})

	absent, err := authenticator.Authenticate(httptest.NewRequest(http.MethodGet, "/Sessions", nil))
	if err != nil {
		t.Fatalf("a request with no credential: %v", err)
	}
	unknown, err := authenticator.Authenticate(requestWithToken("ffffffffffffffffffffffffffffffff"))
	if err != nil {
		t.Fatalf("a request with an unknown token: %v", err)
	}

	if absent != (httpapi.Authentication{}) {
		t.Errorf("no credential answered %+v, want the zero Authentication", absent)
	}
	if unknown != absent {
		t.Errorf("an unknown token answered %+v and no credential answered %+v; 002 plan 7 makes them indistinguishable", unknown, absent)
	}
}

// The same pair, at the wire, where the thing being asserted is that nothing
// about the two responses differs — not the status, not a header, not a byte of
// the body.
//
// It goes over a real connection because three of the four things behaviours
// 1.11 says about this shape are invisible to httptest.ResponseRecorder: the
// declared length, the absent content type and the absent WWW-Authenticate
// (001 T11 measured it). A recorder comparison would pass on a build that told
// the two cases apart in a header.
func TestNoCredentialAndAnUnknownTokenAnswerTheSameBytes(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	userID := createAccount(t, store, "Ada", nil)
	openSessionFor(t, store, userID, "Atrium Test", "device-1", "0123456789abcdef0123456789abcdef", aTestInstant)

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: newAuthenticator(t, store, &settableClock{at: aTestInstant}),
	})

	absent := send(t, handler.Info(), http.MethodGet, "/System/Info")
	unknown := send(t, handler.Info(), http.MethodGet, "/System/Info",
		`X-Emby-Token: ffffffffffffffffffffffffffffffff`)

	assertEmptyRefusal(t, absent, http.StatusUnauthorized, "no credential")
	assertEmptyRefusal(t, unknown, http.StatusUnauthorized, "an unknown token")

	if absent.statusLine != unknown.statusLine {
		t.Errorf("status lines differ: no credential %q, unknown token %q", absent.statusLine, unknown.statusLine)
	}
	if absent.body != unknown.body {
		t.Errorf("bodies differ: no credential %q, unknown token %q", absent.body, unknown.body)
	}
	// Date is the one field line net/http writes that legitimately differs
	// between two responses, and it is not this server's (001 T14).
	for name := range unknown.header {
		if name == "Date" {
			continue
		}
		got, want := unknown.values(name), absent.values(name)
		if len(got) != len(want) {
			t.Errorf("%s: unknown token sent %v, no credential sent %v", name, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: unknown token sent %v, no credential sent %v", name, got, want)
				break
			}
		}
	}
	for name := range absent.header {
		if !unknown.has(name) {
			t.Errorf("%s is sent to a request with no credential and not to one with an unknown token", name)
		}
	}
}

// A token this server recognises, held by an account it will not serve.
//
// The caller is nil, and that is the assertion worth having: 002 plan 7 refuses
// this request, and a handler reading a caller off a refusal would be reading
// somebody the server just declined to serve. The pointer is what makes that
// unreadable rather than quietly wrong — a Caller value would have answered the
// empty identifier and a policy of all-false flags, which is a caller-shaped
// thing a handler would use without noticing.
func TestADisabledAccountIsForbiddenAndCarriesNoCaller(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", func(p *users.Policy) { p.IsDisabled = true })
	sessionID := openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	clock := &settableClock{at: units.TimeFromTicks(aTestInstant.Ticks() + 10*units.TicksPerSecond)}
	authentication, err := newAuthenticator(t, store, clock).Authenticate(requestWithToken(token))
	if err != nil {
		t.Fatalf("authenticating a disabled account: %v", err)
	}

	// A refused request is not activity. The clock is ten seconds past the
	// session's date, so a build that stamped before it refused would move it.
	if got := sessionActivity(t, store, sessionID); !got.Equal(aTestInstant) {
		t.Errorf("a refused request recorded activity at %s, leaving the session's date past its %s", got, aTestInstant)
	}

	if authentication.Access != httpapi.AccessForbidden {
		t.Errorf("Access = %d, want AccessForbidden (%d) — a disabled account is 403 and not the 401 an unknown token gets, because a client re-authenticates on a 401 and would loop through a login it can never complete",
			authentication.Access, httpapi.AccessForbidden)
	}
	if authentication.Caller != nil {
		t.Errorf("a refused request carries the caller %+v", *authentication.Caller)
	}
}

// The same refusal at the wire, on the one route this project serves today.
//
// 001's handler switches on the Access it is handed and answers an unknown one
// 500, which is what a value added without teaching that switch would get: a
// disabled account told 500 is a server saying it is broken, where the
// reference says "stop asking". So the third value and the branch that answers
// it land together, and this is the assertion that they did.
//
// The *shape* — empty, no content type — is 002 plan 7's row for this refusal
// and behaviours 1.11's policy refusal measured at 009 T2. T11 gives it a
// writer of its own (WriteForbidden) and owns the assertions that it differs
// from the controller's 403 on the same status; what is asserted here is the
// status this request gets and that it carries nothing.
func TestADisabledAccountIsRefusedAtTheWireWithTheEmptyShape(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", func(p *users.Policy) { p.IsDisabled = true })
	openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: newAuthenticator(t, store, &settableClock{at: aTestInstant}),
	})

	response := send(t, handler.Info(), http.MethodGet, "/System/Info", "X-Emby-Token: "+token)
	assertEmptyRefusal(t, response, http.StatusForbidden, "a live token whose account was disabled")
}

// A store that cannot be read is a failure to decide, and it must never take
// the path an unknown credential takes.
//
// Asserted twice: at the port, where the answer is an error beside the zero
// Authentication, and at 001's /System/Info handler, where the answer is 500
// and not 401. The second is the one that matters to a client — a client told
// 401 discards a credential that was fine and logs in again, so a database that
// was briefly unreadable would make every client in the house re-authenticate.
func TestAStoreFailureIsAnErrorAndNotARefusal(t *testing.T) {
	t.Parallel()

	unreadable := errors.New("the session store is unreadable")
	authenticator, err := httpapi.NewTokenAuthenticator(httpapi.TokenAuthenticatorConfig{
		Sessions: failingSessions{err: unreadable},
		Accounts: failingAccounts{},
		Clock:    &settableClock{at: aTestInstant},
	})
	if err != nil {
		t.Fatalf("building the authenticator: %v", err)
	}

	authentication, err := authenticator.Authenticate(requestWithToken("0123456789abcdef0123456789abcdef"))
	if err == nil {
		t.Fatal("a store that could not be read answered no error, which a handler would read as a decision")
	}
	if !errors.Is(err, unreadable) {
		t.Errorf("the error does not wrap the store's own: %v", err)
	}
	if authentication != (httpapi.Authentication{}) {
		t.Errorf("the failure answered %+v beside its error; it must be the zero Authentication, so that a caller which drops the error refuses rather than admits", authentication)
	}
	if authentication.Access == httpapi.AccessUnauthenticated && err == nil {
		t.Error("a store failure became an ordinary refusal")
	}

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: authenticator,
	})
	w := systemInfo(t, handler, func(r *http.Request) {
		r.Header.Set(httpapi.EmbyTokenHeader, "0123456789abcdef0123456789abcdef")
	})
	if w.Code == http.StatusUnauthorized {
		t.Error("/System/Info answered 401 to a request whose credential could not be read, which tells a client to throw away a credential that was fine")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// The caller is the *token's* user, and the store returns it separately from
// the session's for exactly this reason (002 plan 6.5).
//
// A session is keyed on (Client, DeviceId) and names whoever authenticated
// there last; a token is keyed on (user, device). So two people sharing one
// client on one device hold two live tokens against one session row, and
// resolving the caller off the session hands the request to whoever logged in
// most recently — somebody else's account, with no error anywhere. Reading
// session.UserID here compiles and passes every other test in this file.
func TestTheCallerIsTheTokensUserAndNotTheSessionsCurrentOne(t *testing.T) {
	t.Parallel()

	const (
		client     = "Atrium Test"
		deviceID   = "device-1"
		adaToken   = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		graceToken = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	)

	store := openStore(t)
	ada := createAccount(t, store, "Ada", nil)
	grace := createAccount(t, store, "Grace", nil)

	// Ada logs in on the shared device, then Grace does. The second login
	// updates the one session row's user and leaves Ada's token alive, which
	// is the state the reference reaches too.
	sessionID := openSessionFor(t, store, ada, client, deviceID, adaToken, aTestInstant)
	if second := openSessionFor(t, store, grace, client, deviceID, graceToken, aTestInstant); second != sessionID {
		t.Fatalf("the two logins opened two sessions (%s, %s), and the fixture needs one", sessionID, second)
	}

	authentication, err := newAuthenticator(t, store, &settableClock{at: aTestInstant}).
		Authenticate(requestWithToken(adaToken))
	if err != nil {
		t.Fatalf("authenticating Ada's token: %v", err)
	}
	if authentication.Access != httpapi.AccessGranted || authentication.Caller == nil {
		t.Fatalf("Ada's token was not admitted: %+v", authentication)
	}
	if authentication.Caller.UserID != ada {
		t.Errorf("the caller is %s, want Ada (%s) — the session names Grace (%s), and the token is Ada's",
			authentication.Caller.UserID, ada, grace)
	}
	if authentication.Caller.SessionID != sessionID {
		t.Errorf("SessionID = %s, want %s", authentication.Caller.SessionID, sessionID)
	}
}

// What an admitted caller carries, and where its policy came from.
//
// The policy is decoded onto the reference's defaults and never onto Go's zero
// value: EnableAllFolders below is true on a default account, and a build that
// decoded onto the zero value would answer false for every flag the stored
// document happened not to carry — a caller who may see nothing, or, on a flag
// whose safe direction is the other way, one who may do anything.
func TestAnAdmittedCallerCarriesItsAccountsOwnPolicy(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", func(p *users.Policy) { p.IsAdministrator = true })
	openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	authentication, err := newAuthenticator(t, store, &settableClock{at: aTestInstant}).
		Authenticate(requestWithToken(token))
	if err != nil {
		t.Fatalf("authenticating: %v", err)
	}
	if authentication.Caller == nil {
		t.Fatalf("an admitted request carries no caller: %+v", authentication)
	}
	if !authentication.Caller.Policy.IsAdministrator {
		t.Error("Policy.IsAdministrator is false for an administrator, which is the flag GET /Sessions branches on")
	}
	if !authentication.Caller.Policy.EnableAllFolders {
		t.Error("Policy.EnableAllFolders is false, which is Go's zero value and not the reference's default — the stored document decodes onto DefaultPolicy")
	}
}

// The token is read over 002 spec 3.1's five mechanisms, not off one header.
//
// The query forms are the half a header-only reader would lose, and losing them
// breaks playback and artwork while leaving browsing intact — a failure that
// looks like a bug in the client (spec 3.1). T8 tests the reader itself; this
// asserts that the authenticator goes through it.
func TestATokenInTheQueryAuthenticates(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", nil)
	openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	r := httptest.NewRequest(http.MethodGet, "/Items/abc/Images/Primary?api_key="+token, nil)
	authentication, err := newAuthenticator(t, store, &settableClock{at: aTestInstant}).Authenticate(r)
	if err != nil {
		t.Fatalf("authenticating a query token: %v", err)
	}
	if authentication.Access != httpapi.AccessGranted {
		t.Errorf("Access = %d, want AccessGranted — the query forms are two of the five mechanisms", authentication.Access)
	}
}

// LastActivityDate is written at most once per session per second, and this is
// a decision about frequency rather than about the value.
//
// A test asserting only that the date advanced passes on a build that writes on
// every request, which is what 002 plan 6.10 forbids: the date is on the wire at
// one-second resolution for any client that reads it, and a busy client would
// otherwise turn every request it makes into a write.
//
// Read back from the store, because that is where the rule lives and because
// the throttle's own state is the stored value — there is no counter to
// interrogate and deliberately none.
func TestActivityIsWrittenAtMostOncePerSessionPerSecond(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", nil)
	sessionID := openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	clock := &settableClock{at: aTestInstant}
	authenticator := newAuthenticator(t, store, clock)

	authenticate := func(what string) {
		t.Helper()
		authentication, err := authenticator.Authenticate(requestWithToken(token))
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
		if authentication.Access != httpapi.AccessGranted {
			t.Fatalf("%s: Access = %d, want AccessGranted", what, authentication.Access)
		}
	}

	// Two requests inside one second. The first is 200 ms after the session
	// was opened, the second 200 ms after that: both are inside the interval,
	// so neither may write.
	clock.advance(200 * time.Millisecond)
	authenticate("the first request")
	clock.advance(200 * time.Millisecond)
	authenticate("the second request")

	if got := sessionActivity(t, store, sessionID); !got.Equal(aTestInstant) {
		t.Errorf("after two requests inside one second the stored activity is %s, want the session's own %s — the rule is at most one write per session per second, and a build that writes on every request passes any assertion that the date merely advanced",
			got, aTestInstant)
	}

	// A second apart. The bound is inclusive, so exactly one second after the
	// last written value writes.
	clock.advance(600 * time.Millisecond)
	third := clock.at
	authenticate("the third request")
	if got := sessionActivity(t, store, sessionID); !got.Equal(third) {
		t.Errorf("a request one second after the last write left the activity at %s, want %s", got, third)
	}

	clock.advance(time.Second)
	fourth := clock.at
	authenticate("the fourth request")
	if got := sessionActivity(t, store, sessionID); !got.Equal(fourth) {
		t.Errorf("a request a second after the previous one left the activity at %s, want %s", got, fourth)
	}
}

// A clock that has gone backwards writes nothing, which is the safe direction:
// the alternative moves a session's last activity into the past, where
// activeWithinSeconds would then hide the session that is being used right now.
func TestActivityIsNeverMovedBackwards(t *testing.T) {
	t.Parallel()

	const token = "0123456789abcdef0123456789abcdef"

	store := openStore(t)
	userID := createAccount(t, store, "Ada", nil)
	sessionID := openSessionFor(t, store, userID, "Atrium Test", "device-1", token, aTestInstant)

	clock := &settableClock{at: units.TimeFromTicks(aTestInstant.Ticks() - 10*units.TicksPerSecond)}
	if _, err := newAuthenticator(t, store, clock).Authenticate(requestWithToken(token)); err != nil {
		t.Fatalf("authenticating: %v", err)
	}
	if got := sessionActivity(t, store, sessionID); !got.Equal(aTestInstant) {
		t.Errorf("a clock ten seconds behind the stored date moved the activity to %s, want it left at %s", got, aTestInstant)
	}
}

// Three ports and none of them optional. A defaulted clock in particular would
// make architecture 2's port an option, and the wall clock is not something a
// test can hold still.
func TestTheAuthenticatorRefusesToBeBuiltWithoutItsPorts(t *testing.T) {
	t.Parallel()

	store := openStore(t)
	clock := &settableClock{at: aTestInstant}

	for _, row := range []struct {
		name string
		cfg  httpapi.TokenAuthenticatorConfig
	}{
		{"no session store", httpapi.TokenAuthenticatorConfig{Accounts: store, Clock: clock}},
		{"no account store", httpapi.TokenAuthenticatorConfig{Sessions: store, Clock: clock}},
		{"no clock", httpapi.TokenAuthenticatorConfig{Sessions: store, Accounts: store}},
	} {
		if _, err := httpapi.NewTokenAuthenticator(row.cfg); err == nil {
			t.Errorf("%s: built an authenticator anyway", row.name)
		}
	}
}

// failingSessions is a session store that cannot be read. It is the one thing
// a real store will not do on demand.
type failingSessions struct{ err error }

func (f failingSessions) OpenSession(context.Context, ports.Session, string) error { return f.err }

func (f failingSessions) SessionByTokenDigest(context.Context, string) (ports.Session, string, bool, error) {
	return ports.Session{}, "", false, f.err
}

func (f failingSessions) Sessions(context.Context) ([]ports.Session, error) { return nil, f.err }

func (f failingSessions) ReplaceCapabilities(context.Context, string, []byte) error { return f.err }

func (f failingSessions) TouchSession(context.Context, string, units.Time) error { return f.err }

func (f failingSessions) RevokeTokensFor(context.Context, string, string) error { return f.err }

func (f failingSessions) CloseSession(context.Context, string) error { return f.err }

// failingAccounts is an account store nothing above reaches: the session store
// fails first. It exists because the authenticator refuses to be built without
// one, which is the point of that refusal.
type failingAccounts struct{}

var errNotReached = errors.New("the account store was reached, and the session store should have failed first")

func (failingAccounts) CreateUser(context.Context, ports.User) error { return errNotReached }

func (failingAccounts) UserByFoldedName(context.Context, string) (ports.User, bool, error) {
	return ports.User{}, false, errNotReached
}

func (failingAccounts) UserByID(context.Context, string) (ports.User, bool, error) {
	return ports.User{}, false, errNotReached
}

func (failingAccounts) Users(context.Context) ([]ports.User, error) { return nil, errNotReached }

func (failingAccounts) Credential(context.Context, string) (ports.Credential, bool, error) {
	return ports.Credential{}, false, errNotReached
}

func (failingAccounts) ReplaceCredential(context.Context, string, string, units.Time) error {
	return errNotReached
}

func (failingAccounts) ReplaceConfiguration(context.Context, string, []byte) error {
	return errNotReached
}

func (failingAccounts) RecordLoginOutcome(context.Context, string, ports.LoginOutcome, units.Time) error {
	return errNotReached
}

func (failingAccounts) TouchActivity(context.Context, string, units.Time) error { return errNotReached }
