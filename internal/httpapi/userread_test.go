package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/users"
)

// userRoutes mounts the two routes of spec 3.7 on a router of this test's own,
// with spec 3.6's beside them.
//
// They are mounted rather than called directly because GET /Users/{userId}
// reads a path parameter, and a handler invoked without a route context is a
// handler answering a request no client can send. The patterns are
// surface.yaml's own spellings; T17 is what registers them on the server's
// router, and this is deliberately not that check — it is the handlers,
// reachable.
//
// POST /Users/Configuration was added here by T15 rather than mounted a second
// time in its own file, because every assertion that route makes is read back
// through GET /Users/Me: a second router would be a second answer to the
// question of which handler a request reaches, and the round trip is only a
// round trip while both halves are on one.
func userRoutes(handler *httpapi.UsersHandler) http.Handler {
	router := chi.NewRouter()
	router.Get("/Users/Me", handler.CurrentUser())
	router.Get("/Users/{userId}", handler.UserByID())
	router.Post("/Users/Configuration", handler.UpdateConfiguration())
	return router
}

// getUser sends one GET to the routes above and reads the whole answer.
//
// Over a real server for the reason post and getPublic are: an absent
// Content-Type and a declared Content-Length are part of a refusal's shape and
// httptest.ResponseRecorder can express neither, and every refusal in spec 3.7
// is told from the others by exactly those.
func getUser(t *testing.T, handler *httpapi.UsersHandler, path string, headers ...string) response {
	t.Helper()

	if len(headers)%2 != 0 {
		t.Fatalf("getUser was given %d header parts, which is not a whole number of name/value pairs", len(headers))
	}

	server := httptest.NewServer(userRoutes(handler))
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+path, nil)
	if err != nil {
		t.Fatalf("building the request for %s: %v", path, err)
	}
	for i := 0; i < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}

	answer, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("sending the request for %s: %v", path, err)
	}
	defer answer.Body.Close()

	read := make([]byte, 0, 8192)
	buffer := make([]byte, 512)
	for {
		n, err := answer.Body.Read(buffer)
		read = append(read, buffer[:n]...)
		if err != nil {
			break
		}
	}
	return response{
		status:      answer.StatusCode,
		contentType: answer.Header.Get("Content-Type"),
		length:      answer.ContentLength,
		body:        string(read),
	}
}

// readUser is getUser plus the insistence every matrix row makes: a 200, and the
// bytes as the server wrote them.
func readUser(t *testing.T, handler *httpapi.UsersHandler, path, token string) string {
	t.Helper()

	answer := getUser(t, handler, path, httpapi.EmbyTokenHeader, token)
	if answer.status != http.StatusOK {
		t.Fatalf("GET %s answered %d and the body %q, want 200 — spec 3.7 refuses no authenticated caller",
			path, answer.status, answer.body)
	}
	return answer.body
}

// occupant is one account in the caller matrix: who it is, and the token that
// reads as it.
//
// A seat and a subject are the same kind of thing here, which is spec 3.7's
// point: there is no permission on this route to tabulate, so what varies
// across the matrix is only which pair of accounts a request names.
type occupant struct {
	name  string
	id    string
	token string
}

// seat provisions an account with a password, logs it in through the real
// login route, and hands back the token that came back.
//
// The login is the route rather than a session written straight into the
// store, for two reasons the fixture depends on. It stamps `last_login_at`, so
// every subject in the matrix carries a `LastLoginDate` — a member that is
// **absent** on an account that has never logged in, and therefore the member
// most likely to differ between two readings built from two reads of one row.
// And the token is one this feature really minted, which is what makes the
// caller matrix a matrix of credentials rather than of rows.
//
// Each seat logs in from its own device, because a session is keyed on
// (Client, DeviceId) and a second login from one device revokes the first
// one's token (plan 6.5) — three seats sharing a device would leave one live
// token and two dead ones.
func seat(t *testing.T, store *sqlite.Store, handler *httpapi.UsersHandler, name, password, device string, shape func(*users.Policy)) occupant {
	t.Helper()

	id := createAccountWithPassword(t, store, name, password, shape)

	body, err := json.Marshal(map[string]string{"Username": name, "Pw": password})
	if err != nil {
		t.Fatalf("encoding the login body of %q: %v", name, err)
	}
	header := `MediaBrowser Client="Atrium Test", Device="A Device", DeviceId="` + device + `", Version="1.0.0"`
	answer := post(t, handler, string(body), httpapi.AuthorizationHeader, header)
	if answer.status != http.StatusOK {
		t.Fatalf("logging %q in answered %d and the body %q, want 200", name, answer.status, answer.body)
	}

	var result struct{ AccessToken string }
	if err := json.Unmarshal([]byte(answer.body), &result); err != nil {
		t.Fatalf("reading the authentication result of %q: %v", name, err)
	}
	if result.AccessToken == "" {
		t.Fatalf("logging %q in returned no access token", name)
	}
	return occupant{name: name, id: id, token: result.AccessToken}
}

// administrator is the policy of the account the central assertion reads.
//
// It is deliberately not the default with one flag flipped. The mistake this
// task exists to catch is a handler that answers 200 with a **redacted** body,
// and a redaction is invisible over a subject whose object is the same as
// everybody else's — which is 002 T13's finding, one route over: an assertion
// that the same bytes go to everybody proves nothing over data with only one
// possible answer. So the administrator carries the flags a redacting handler
// would most plausibly withhold from a stranger, each with a value no other
// account in the fixture has.
func administrator(policy *users.Policy) {
	policy.IsHidden = false
	policy.IsAdministrator = true
	policy.EnableContentDeletion = true
	policy.EnableCollectionManagement = true
	policy.EnableUserPreferenceAccess = true
	policy.MaxActiveSessions = 3
	policy.LoginAttemptsBeforeLockout = 5
	policy.EnabledDevices = []string{"a-console"}
	policy.EnableContentDeletionFromFolders = []string{"a-library"}
	policy.RemoteClientBitrateLimit = 8000000
}

// restricted is the seat that reads the administrator: an account that may do
// as little as this feature can express.
func restricted(policy *users.Policy) {
	policy.IsHidden = false
	policy.IsAdministrator = false
	policy.EnableAllFolders = false
	policy.EnabledFolders = []string{"one-library"}
	policy.EnableMediaPlayback = false
	policy.EnableAudioPlaybackTranscoding = false
	policy.EnableVideoPlaybackTranscoding = false
	policy.EnablePlaybackRemuxing = false
	policy.EnableContentDeletion = false
	policy.EnableRemoteAccess = false
	policy.EnableUserPreferenceAccess = false
	policy.MaxActiveSessions = 1
}

// ordinary is the third seat: a non-administrator with nothing taken away.
func ordinary(policy *users.Policy) {
	policy.IsHidden = false
	policy.IsAdministrator = false
}

// matrixFixture is the installation spec 3.7's matrix is measured over: three
// seats that can ask, and a fourth account that can only be named.
type matrixFixture struct {
	store    *sqlite.Store
	handler  *httpapi.UsersHandler
	admin    occupant
	ordinary occupant

	// stranger is the restricted non-administrator — the seat whose reading of
	// the administrator is the assertion this whole task turns on.
	stranger occupant

	// hidden is a subject that is not a seat: hidden from login screens, with
	// no password and no login behind it, so its object carries
	// `HasPassword: false` and **no** `LastLoginDate` at all. It is here
	// because a subject nobody can log in as is still a subject any caller may
	// read, and because it is the one account in the fixture whose object
	// differs from the others by an absent member rather than by a value.
	hidden occupant
}

func newMatrixFixture(t *testing.T) matrixFixture {
	t.Helper()

	store := openStore(t)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	fixture := matrixFixture{
		store:    store,
		handler:  handler,
		admin:    seat(t, store, handler, "Ada", "correct horse", "device-ada", administrator),
		ordinary: seat(t, store, handler, "Bob", "battery staple", "device-bob", ordinary),
		stranger: seat(t, store, handler, "Cleo", "a third password", "device-cleo", restricted),
	}
	fixture.hidden = occupant{name: "Dora", id: createAccount(t, store, "Dora", nil)}
	return fixture
}

func (f matrixFixture) seats() []occupant {
	return []occupant{f.admin, f.ordinary, f.stranger}
}

func (f matrixFixture) subjects() []occupant {
	return []occupant{f.admin, f.ordinary, f.stranger, f.hidden}
}

// TestTheCallerMatrixAnswersOneObjectPerSubjectWhoeverAsked is the whole of
// spec 3.7's matrix, asserted as bytes.
//
// Every pair of seat and subject: a non-administrator naming another
// non-administrator, a restricted non-administrator naming an administrator,
// an administrator naming anybody, and each seat naming themselves. Twelve
// requests through GET /Users/{userId}, plus three through GET /Users/Me.
//
// # Why the assertion is an equality of bytes and not a count of statuses
//
// spec 3.7 said this route answered 403 to a non-administrator naming anybody
// else, with no provenance, from the day 002 was written until the matrix was
// measured on 2026-09-01 and found no refusal anywhere in it
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]. The
// **successor** mistake is the one this shape catches: a handler that answers
// 200 with a body it trimmed for a caller it does not trust passes every
// assertion about a status, and passes a shape check over each response taken
// on its own. What it cannot pass is one reading being byte-identical to
// another.
//
// Two comparisons are made per subject, and they catch different wrong
// handlers. All three seats reading one subject must agree, which is what fails
// on a redaction; and a subject that is itself a seat must read the same
// through GET /Users/Me as every seat reads it through GET /Users/{userId},
// which is what fails on a handler that ignores the path and answers the
// caller's own account.
func TestTheCallerMatrixAnswersOneObjectPerSubjectWhoeverAsked(t *testing.T) {
	fixture := newMatrixFixture(t)

	for _, subject := range fixture.subjects() {
		readings := make(map[string]string, len(fixture.seats()))
		for _, asking := range fixture.seats() {
			readings[asking.name] = readUser(t, fixture.handler, "/Users/"+subject.id, asking.token)
		}

		first := fixture.admin
		for _, asking := range fixture.seats()[1:] {
			if readings[asking.name] != readings[first.name] {
				t.Errorf("%s read by %s is different bytes from %s read by %s, and spec 3.7 measured one answer whoever asked:\n%s: %s\n%s: %s",
					subject.name, asking.name, subject.name, first.name,
					asking.name, readings[asking.name], first.name, readings[first.name])
			}
		}

		if subject.token == "" {
			continue
		}
		own := readUser(t, fixture.handler, "/Users/Me", subject.token)
		if own != readings[first.name] {
			t.Errorf("%s reading themselves through /Users/Me is different bytes from %s reading them through /Users/{userId}:\n  me: %s\nthem: %s",
				subject.name, first.name, own, readings[first.name])
		}
	}
}

// TestARestrictedNonAdministratorReadsAnAdministratorsWholeObject is the one
// cell of the matrix the whole task turns on, named for it.
//
// The equality above would pass over a fixture where every account's object is
// the same — 002 T13's finding, which is that "the same bytes to everybody"
// proves nothing over data with only one possible answer. So this asserts the
// two halves that make the equality mean something: the administrator's object
// carries things a redacting handler would take away, and the restricted
// stranger's reading of it is byte-identical to the administrator's own.
func TestARestrictedNonAdministratorReadsAnAdministratorsWholeObject(t *testing.T) {
	fixture := newMatrixFixture(t)

	own := readUser(t, fixture.handler, "/Users/Me", fixture.admin.token)
	stranger := readUser(t, fixture.handler, "/Users/"+fixture.admin.id, fixture.stranger.token)

	if stranger != own {
		t.Fatalf("a restricted non-administrator reads the administrator as different bytes from the administrator's own reading:\nstranger: %s\n    self: %s",
			stranger, own)
	}

	names, values := members(t, []byte(stranger))
	if got := len(memberNamesInOrder(t, values["Policy"])); got != 42 {
		t.Errorf("the administrator's object travels to a stranger with %d policy properties, want 42 —"+
			" spec 3.7 answers the *whole* object and a trimmed Policy is the redaction this test exists for", got)
	}
	if got := len(memberNamesInOrder(t, values["Configuration"])); got != 16 {
		t.Errorf("the administrator's object travels to a stranger with %d configuration properties, want 16", got)
	}

	var isAdministrator bool
	if err := json.Unmarshal(policyMember(t, values["Policy"], "IsAdministrator"), &isAdministrator); err != nil {
		t.Fatalf("reading IsAdministrator: %v", err)
	}
	if !isAdministrator {
		t.Errorf("the administrator's object reads IsAdministrator false to a stranger — the flag a redacting handler" +
			" would blank first is the flag this subject was given so that blanking it would show")
	}

	var hasPassword bool
	if err := json.Unmarshal(values["HasPassword"], &hasPassword); err != nil {
		t.Fatalf("reading HasPassword: %v", err)
	}
	if !hasPassword {
		t.Errorf("the administrator's object reads HasPassword false to a stranger, over an account that logged in with one")
	}

	if !slices.Contains(names, "LastLoginDate") {
		t.Errorf("the administrator's object carries no LastLoginDate to a stranger, over an account that has logged in:\n%v", names)
	}
}

// policyMember reads one member out of the encoded policy.
func policyMember(t *testing.T, policy json.RawMessage, name string) json.RawMessage {
	t.Helper()
	_, values := members(t, policy)
	value, ok := values[name]
	if !ok {
		t.Fatalf("the policy carries no %s", name)
	}
	return value
}

// TestTheThreeSeatsReadAsThreeDifferentObjects is what keeps every equality in
// this file from being satisfied by a constant.
//
// It is 002 T7's and T9's lesson at the wire: a derivation that returns a
// constant passes a test that asserts two runs agree, and only distinctness
// catches it. A handler answering one account's object to every request would
// pass the matrix above in full — every seat would read the same bytes for
// every subject, and each subject's /Users/Me would agree with it — so the
// matrix is an assertion only while the objects it compares are different to
// begin with.
func TestTheThreeSeatsReadAsThreeDifferentObjects(t *testing.T) {
	fixture := newMatrixFixture(t)

	readings := make(map[string]string)
	for _, asking := range fixture.seats() {
		readings[asking.name] = readUser(t, fixture.handler, "/Users/Me", asking.token)
	}

	seats := fixture.seats()
	for i := range seats {
		for j := i + 1; j < len(seats); j++ {
			if readings[seats[i].name] == readings[seats[j].name] {
				t.Errorf("%s and %s read as the same bytes through /Users/Me, so every equality in this file is vacuous:\n%s",
					seats[i].name, seats[j].name, readings[seats[i].name])
			}
		}
	}
}

// TestAnIdentifierBelongingToNobodyIsTheSameSixteenBytesToEverySeat is
// spec 3.7's second row.
//
// The reference answers a well-formed identifier no account has with the
// JSON-encoded bare string "User not found" — behaviours 1.11's fourth error
// shape rather than the problem details every other handler-raised 404 in this
// project answers — and **the same body to an administrator and to a
// non-administrator**
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]. That last
// clause is the one worth a test: a server that concealed which identifiers
// exist would be answering one caller 404 and another 403, and the equality
// below is what says it does not.
//
// The sixteen bytes are asserted as bytes. Fourteen characters and two quotes
// is a length no assertion about a parsed string can see, and the quotes are
// exactly what distinguishes this shape from a plain-text body.
func TestAnIdentifierBelongingToNobodyIsTheSameSixteenBytesToEverySeat(t *testing.T) {
	fixture := newMatrixFixture(t)

	// Well formed and nobody's: the identifier of a username no account in the
	// fixture has. Derived rather than typed, so that it stays an identifier
	// this server could have issued.
	nobody := users.DeriveID("Nobody At All")

	answers := make(map[string]response)
	for _, asking := range fixture.seats() {
		answer := getUser(t, fixture.handler, "/Users/"+nobody, httpapi.EmbyTokenHeader, asking.token)
		answers[asking.name] = answer

		if answer.status != http.StatusNotFound {
			t.Errorf("%s naming an identifier no account has was answered %d and the body %q, want 404",
				asking.name, answer.status, answer.body)
		}
		if answer.body != `"User not found"` {
			t.Errorf("%s was answered the body %q, want the JSON-encoded bare string \"User not found\"", asking.name, answer.body)
		}
		if answer.length != 16 {
			t.Errorf("%s was answered a declared length of %d, want 16", asking.name, answer.length)
		}
		if answer.contentType != "application/json; charset=utf-8" {
			t.Errorf("%s was answered the content type %q, want application/json; charset=utf-8", asking.name, answer.contentType)
		}
	}

	if answers[fixture.admin.name].body != answers[fixture.stranger.name].body {
		t.Errorf("the administrator and the restricted non-administrator are answered different bodies for an identifier nobody has:\nadmin: %q\nother: %q",
			answers[fixture.admin.name].body, answers[fixture.stranger.name].body)
	}
	if answers[fixture.admin.name].status != answers[fixture.stranger.name].status {
		t.Errorf("the administrator and the restricted non-administrator are answered different statuses for an identifier nobody has: %d and %d",
			answers[fixture.admin.name].status, answers[fixture.stranger.name].status)
	}
}

// TestAnIdentifierThatIsNotOneIsTheValidationProblemKeyedOnUserId is
// spec 3.7's third row.
//
// Three things are asserted that a status check cannot see. The refusal is the
// problem-details **model** and not the bare string the 404 carries, which is
// two shapes on two statuses one route away from each other. Its errors map is
// keyed on the parameter's **declared** spelling — `userId`, the reference's
// own, never anything the client wrote — and carries that one key and no
// second one, which is what tells this refusal from the login route's, whose
// map names an action parameter beside `"$"`. And the value is quoted back
// with its apostrophes **escaped**: behaviours 1.16 writes `'` as \u0027 and
// the reference's own encoder does the same, so a message assembled with fmt
// would carry two bytes the reference never sends.
func TestAnIdentifierThatIsNotOneIsTheValidationProblemKeyedOnUserId(t *testing.T) {
	fixture := newMatrixFixture(t)

	answer := getUser(t, fixture.handler, "/Users/not-an-identifier", httpapi.EmbyTokenHeader, fixture.admin.token)
	if answer.status != http.StatusBadRequest {
		t.Fatalf("a userId that is not an identifier was answered %d and the body %q, want 400", answer.status, answer.body)
	}
	if answer.contentType != "application/json; charset=utf-8" {
		t.Errorf("the validation refusal carried the content type %q, want application/json; charset=utf-8 —"+
			" behaviours 1.11 records that application/problem+json is what a framework does by default and not what the reference sends",
			answer.contentType)
	}

	var problem struct {
		Type   string              `json:"type"`
		Title  string              `json:"title"`
		Status int                 `json:"status"`
		Errors map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(answer.body), &problem); err != nil {
		t.Fatalf("reading the problem document %q: %v", answer.body, err)
	}
	if problem.Status != http.StatusBadRequest {
		t.Errorf("the problem document declares status %d, want 400", problem.Status)
	}
	if len(problem.Errors) != 1 {
		t.Errorf("the errors map carries %d keys, want exactly one — this route names a path parameter and no action parameter:\n%v",
			len(problem.Errors), problem.Errors)
	}
	messages, keyed := problem.Errors["userId"]
	if !keyed {
		t.Fatalf("the errors map is not keyed on userId, which is the parameter's declared spelling:\n%v", problem.Errors)
	}
	if len(messages) != 1 || messages[0] != "The value 'not-an-identifier' is not valid." {
		t.Errorf("the message under userId is %v, want [The value 'not-an-identifier' is not valid.]", messages)
	}

	if !strings.Contains(answer.body, `\u0027not-an-identifier\u0027`) {
		t.Errorf("the quoted value travels with literal apostrophes; behaviours 1.16 escapes them to \\u0027:\n%s", answer.body)
	}
}

// TestNoCredentialIsTheEmptyUnauthorizedOnBothRoutes is spec 3.7's fourth row,
// on both of this task's routes.
//
// The empty shape of behaviours 1.11 is four things and three of them are
// invisible to a test that reads a status: no body, `Content-Length: 0`, no
// `Content-Type` and no `WWW-Authenticate`. It is asserted over a real
// connection for that reason.
func TestNoCredentialIsTheEmptyUnauthorizedOnBothRoutes(t *testing.T) {
	fixture := newMatrixFixture(t)

	for _, path := range []string{"/Users/Me", "/Users/" + fixture.admin.id} {
		answer := getUser(t, fixture.handler, path)
		if answer.status != http.StatusUnauthorized {
			t.Errorf("GET %s with no credential answered %d and the body %q, want 401", path, answer.status, answer.body)
		}
		if answer.body != "" {
			t.Errorf("GET %s with no credential carried the body %q, and this shape has none", path, answer.body)
		}
		if answer.contentType != "" {
			t.Errorf("GET %s with no credential declared the content type %q, and this shape declares none", path, answer.contentType)
		}
		if answer.length != 0 {
			t.Errorf("GET %s with no credential declared a length of %d, want 0", path, answer.length)
		}
	}
}

// TestTheCredentialIsReadBeforeTheIdentifierIsBound asserts an order that is
// observable in exactly one request: no credential *and* a segment that is not
// an identifier.
//
// It answers the 401 rather than the 400. That is the reference's own order —
// its authorization filter runs ahead of the model binder, measured on another
// route where a caller who may not act meets the policy refusal for a segment
// that is not an identifier at all (009 spec 3.8, 2026-09-01) — and it is a
// **reading** applied to this route rather than a measurement of it, which the
// register at T23 is owed a row for.
//
// A handler that bound first would tell an unauthenticated caller which of its
// path segments this server dislikes, which is the disclosure the order exists
// to prevent, and no assertion about either refusal on its own can see it.
func TestTheCredentialIsReadBeforeTheIdentifierIsBound(t *testing.T) {
	fixture := newMatrixFixture(t)

	answer := getUser(t, fixture.handler, "/Users/not-an-identifier")
	if answer.status != http.StatusUnauthorized {
		t.Errorf("a request with no credential and a malformed identifier answered %d and the body %q,"+
			" want the 401 — the credential is read before the segment is bound", answer.status, answer.body)
	}
	if answer.body != "" {
		t.Errorf("the refusal carried the body %q, so the binder answered it and not the authenticator", answer.body)
	}
}

// TestAnUpperCaseIdentifierAddressesTheSameAccount is the one alternative
// spelling this server accepts.
//
// The reference parses the segment as a Guid, and an upper-case spelling
// addresses the object rather than being refused — measured on another route
// and recorded in 009 spec 3.8's identifier table, 2026-09-01. The bytes are
// compared with the lower-case reading rather than only the status, because a
// handler that accepted the spelling and then looked the *unfolded* string up
// would answer the 404 and a handler that folded the wrong way would answer
// somebody else.
func TestAnUpperCaseIdentifierAddressesTheSameAccount(t *testing.T) {
	fixture := newMatrixFixture(t)

	lower := readUser(t, fixture.handler, "/Users/"+fixture.admin.id, fixture.stranger.token)
	upper := readUser(t, fixture.handler, "/Users/"+strings.ToUpper(fixture.admin.id), fixture.stranger.token)

	if upper != lower {
		t.Errorf("an upper-case identifier answered different bytes from the lower-case spelling of the same account:\nupper: %s\nlower: %s",
			upper, lower)
	}
}

// TestADashedIdentifierIsRefusedHereAndTheReferencesBinderParsesIt is a
// divergence written as a test rather than as a comment, which is the shape
// 002 T13 used for the disabled account on /Users/Public.
//
// 009 spec 3.8's identifier table measured that a path identifier written
// plain, dashed, braced or upper-case all address the same object, because the
// segment is parsed as a Guid (2026-09-01). This server accepts two of those
// four spellings and refuses the dashed one with the validation 400.
// identifier.go carries the argument for keeping the delta; what matters here
// is that it is asserted, so the day somebody sends a dashed identifier to a
// running reference **on this route**, a failing test names the behaviour that
// moved instead of somebody rediscovering it.
func TestADashedIdentifierIsRefusedHereAndTheReferencesBinderParsesIt(t *testing.T) {
	fixture := newMatrixFixture(t)

	id := fixture.admin.id
	dashed := id[0:8] + "-" + id[8:12] + "-" + id[12:16] + "-" + id[16:20] + "-" + id[20:32]

	answer := getUser(t, fixture.handler, "/Users/"+dashed, httpapi.EmbyTokenHeader, fixture.stranger.token)
	if answer.status != http.StatusBadRequest {
		t.Errorf("a dashed identifier answered %d and the body %q; this server refuses the spelling with the validation 400"+
			" and the reference's binder parses it, so a 200 here means the delta closed and identifier.go's argument"+
			" plus the row the register owes both need rewriting", answer.status, answer.body)
	}
}

// TestTheAllZeroIdentifierIsTheControllerRefusalAndNotEitherOtherRefusal is a
// fourth answer on this route that spec 3.7's table does not have.
//
// The all-zero identifier is well formed and belongs to nobody, so the table's
// second row would make it the 404. It is not: the reference's account lookup
// refuses an empty Guid before it queries anything
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:123-133 @ v10.11.11],
// the ArgumentException that raises is mapped to 400 with behaviours 1.11's
// twenty-five bytes under text/plain
// [source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-136 @ v10.11.11],
// and the same request measured on another route that resolves an identifier
// answered exactly that (009 spec 3.8's identifier table, 2026-09-01).
//
// It is asserted against the *same golden file* the login route's refusals are
// compared with, because it is the same twenty-five bytes and a golden
// compared by five responses is a golden of one constant. And it is asserted
// against the other two refusals of this route: three of the four answers here
// share a status or a body with another, and only the pair together tells them
// apart.
func TestTheAllZeroIdentifierIsTheControllerRefusalAndNotEitherOtherRefusal(t *testing.T) {
	fixture := newMatrixFixture(t)

	answer := getUser(t, fixture.handler, "/Users/"+strings.Repeat("0", 32), httpapi.EmbyTokenHeader, fixture.admin.token)
	if answer.status != http.StatusBadRequest {
		t.Fatalf("the all-zero identifier answered %d and the body %q, want 400", answer.status, answer.body)
	}
	if answer.body != goldenRefusal(t) {
		t.Errorf("the all-zero identifier answered the body %q, want the twenty-five bytes %q", answer.body, goldenRefusal(t))
	}
	if answer.contentType != "text/plain" {
		t.Errorf("the all-zero identifier answered the content type %q, want text/plain with no charset parameter", answer.contentType)
	}

	malformed := getUser(t, fixture.handler, "/Users/not-an-identifier", httpapi.EmbyTokenHeader, fixture.admin.token)
	if malformed.status != answer.status {
		t.Fatalf("a malformed identifier answered %d and the all-zero one %d; both are 400 and the body is the whole difference",
			malformed.status, answer.status)
	}
	if malformed.body == answer.body {
		t.Errorf("a malformed identifier and the all-zero identifier answered the same body on the same status,"+
			" so nothing on this route tells the binder's refusal from the lookup's:\n%s", answer.body)
	}
}

// TestALiveTokenWhoseAccountWasDisabledIsTheEmptyForbidden is plan 7's third
// row, reaching these two routes.
//
// It is the *policy* 403 — empty, and with **no content type at all** — and
// not the controller's 403 on the same status, which carries twenty-five
// bytes. One status, two shapes, and reaching for whichever comes to mind
// first is wrong half the time (refusal.go). The account is disabled after the
// token was minted, which is the only way to hold a live token for an account
// this server will not serve.
func TestALiveTokenWhoseAccountWasDisabledIsTheEmptyForbidden(t *testing.T) {
	fixture := newMatrixFixture(t)

	disable(t, fixture.store, fixture.stranger.id)

	for _, path := range []string{"/Users/Me", "/Users/" + fixture.admin.id} {
		answer := getUser(t, fixture.handler, path, httpapi.EmbyTokenHeader, fixture.stranger.token)
		if answer.status != http.StatusForbidden {
			t.Errorf("GET %s from a disabled account's live token answered %d and the body %q, want 403",
				path, answer.status, answer.body)
		}
		if answer.body != "" {
			t.Errorf("GET %s answered the body %q; the policy refusal carries none, and the controller's twenty-five bytes"+
				" are the other 403 on this status", path, answer.body)
		}
		if answer.contentType != "" {
			t.Errorf("GET %s answered the content type %q; the policy refusal declares none at all", path, answer.contentType)
		}
	}
}

// disable puts one account into the state a live token can outlive.
//
// Through the lockout transition rather than through a written document,
// because it is the only writer of `IsDisabled` this store declares (plan 5)
// and because it is how an account really becomes disabled while somebody
// holds a token for it: the lockout **is** the flag, which is why plan 6.7's
// two rows are one state on the second attempt and why there is no
// ErrLockedOut anywhere in this feature.
func disable(t *testing.T, store *sqlite.Store, id string) {
	t.Helper()

	if err := store.RecordLoginOutcome(context.Background(), id, ports.LoginLockedOut, aTestInstant); err != nil {
		t.Fatalf("locking the account %s out: %v", id, err)
	}
}
