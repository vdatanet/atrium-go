package httpapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/users"
)

// The client header every test here sends unless it is testing the header
// itself. It carries all four components of spec 3.2 and no token: a login
// presents credentials in the body, and a token in the header would be a
// credential this route never reads.
const testClientHeader = `MediaBrowser Client="Atrium Test", Device="A Device", DeviceId="device-1", Version="1.0.0"`

// newUsersHandler builds the /Users handler over a real store and a clock a
// test can hold still.
func newUsersHandler(t *testing.T, store *sqlite.Store, clock ports.Clock) *httpapi.UsersHandler {
	t.Helper()

	handler, err := httpapi.NewUsersHandler(httpapi.UsersHandlerConfig{
		InstallationID: testInstallationID,
		Login:          users.NewLogin(store, clock),
		Accounts:       store,
		Sessions:       store,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("building the users handler: %v", err)
	}
	return handler
}

// createAccountWithPassword writes an account and its password record.
//
// The record goes through users.Derive rather than through a literal, because
// the record carries its own parameters and a transcribed one would be a
// credential nobody could regenerate when the constants move.
func createAccountWithPassword(t *testing.T, store *sqlite.Store, username, password string, shape func(*users.Policy)) string {
	t.Helper()

	id := createAccount(t, store, username, shape)
	record, err := users.Derive(users.NewPlaintext(password))
	if err != nil {
		t.Fatalf("deriving the credential of %q: %v", username, err)
	}
	if err := store.ReplaceCredential(context.Background(), id, record, aTestInstant); err != nil {
		t.Fatalf("writing the credential of %q: %v", username, err)
	}
	return id
}

// response is one answer, read off a real connection rather than a recorder:
// behaviours 1.11 makes an absent Content-Type and a declared Content-Length
// part of a refusal's shape, and httptest.ResponseRecorder can express neither.
type response struct {
	status      int
	contentType string
	length      int64
	body        string
}

// post sends one request to a handler over a real server, and reads the whole
// answer.
//
// headers are extra field lines as name/value pairs. A test that wants *no*
// client header passes none, which is a different request from one carrying an
// empty header value — the second is a header that said nothing and the first
// is no header at all, and spec 3.3 answers them alike only because the parser
// yields the same value for both.
func post(t *testing.T, handler *httpapi.UsersHandler, body string, headers ...string) response {
	t.Helper()

	if len(headers)%2 != 0 {
		t.Fatalf("post was given %d header parts, which is not a whole number of name/value pairs", len(headers))
	}

	server := httptest.NewServer(handler.AuthenticateByName())
	defer server.Close()

	request, err := http.NewRequest(http.MethodPost, server.URL+"/Users/AuthenticateByName", strings.NewReader(body))
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for i := 0; i < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}

	answer, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("sending the request: %v", err)
	}
	defer answer.Body.Close()

	read := make([]byte, 0, 1024)
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

// authenticate is the request a well-formed client sends.
func authenticate(t *testing.T, handler *httpapi.UsersHandler, username, password string) response {
	t.Helper()

	body, err := json.Marshal(map[string]string{"Username": username, "Pw": password})
	if err != nil {
		t.Fatalf("encoding the request body: %v", err)
	}
	return post(t, handler, string(body), httpapi.AuthorizationHeader, testClientHeader)
}

// goldenRefusal is the one body spec 3.3's four measured refusals share.
//
// It is read from a file rather than written as a literal in each test for the
// reason the task list gives: comparing four responses against **one** golden
// is what makes "they carry the same twenty-five bytes" a single assertion
// instead of four written alike, and it is what fails when one of them drifts.
func goldenRefusal(t *testing.T) string {
	t.Helper()

	path := filepath.Join("testdata", "golden", "authenticate_refusal.txt")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(body)
}

// thirtyTwoLowercaseHex is behaviours 1.4's identifier shape, which spec 3.3
// measures for the access token as well.
var thirtyTwoLowercaseHex = regexp.MustCompile(`^[0-9a-f]{32}$`)

// TestAuthenticatingAnswersATokenAndASessionThatHasNeverPlayedAnything is AC-1
// and the two members of spec 3.3 that a shape check would pass and a value
// check would not.
//
// LastPlaybackCheckIn is asserted as **the value**
// `0001-01-01T00:00:00.0000000Z` and not as a present field, because it is a
// value and not an absence: .NET's minimum date is what the reference sends for
// a session that has never played anything
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. 001
// T4 left the zero units.Time able to be mistaken for a missing one, and T1
// shipped the column NOT NULL so that the store could not express the mistake
// either. The assertion is on the encoded bytes for the reason Principle VIII
// gives: `"0001-01-01T00:00:00.0000000Z"`, three fractional digits, and `null`
// are three different documents and one parsed value.
func TestAuthenticatingAnswersATokenAndASessionThatHasNeverPlayedAnything(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := authenticate(t, handler, "Ada", "correct horse")
	if answer.status != http.StatusOK {
		t.Fatalf("authenticating with the right password answered %d and the body %q, want 200", answer.status, answer.body)
	}

	var result struct {
		AccessToken string
		ServerId    string
		SessionInfo struct {
			Id                  string
			DeviceId            string
			LastPlaybackCheckIn json.RawMessage
		}
		User struct {
			Name string
			Id   string
		}
	}
	if err := json.Unmarshal([]byte(answer.body), &result); err != nil {
		t.Fatalf("decoding the authentication result %q: %v", answer.body, err)
	}

	if !thirtyTwoLowercaseHex.MatchString(result.AccessToken) {
		t.Errorf("the access token is %q, want thirty-two lowercase hex characters", result.AccessToken)
	}
	if want := `"0001-01-01T00:00:00.0000000Z"`; string(result.SessionInfo.LastPlaybackCheckIn) != want {
		t.Errorf("SessionInfo.LastPlaybackCheckIn is %s, want %s — the zero tick is the value the reference sends, not an absence",
			result.SessionInfo.LastPlaybackCheckIn, want)
	}
	if want := sessions.DeriveID("Atrium Test", "device-1"); result.SessionInfo.Id != want {
		t.Errorf("the session identifier is %q, want the derivation of (client, device) %q", result.SessionInfo.Id, want)
	}
	if result.ServerId != testInstallationID || result.User.Id != users.DeriveID("Ada") {
		t.Errorf("the result names server %q and user %q, want %q and %q",
			result.ServerId, result.User.Id, testInstallationID, users.DeriveID("Ada"))
	}
}

// TestTheUsernameIsMatchedThroughTheFoldedColumnAndNotByTheHandler is spec
// 3.3's "matched case-insensitively", asserted so that it can only pass through
// the store's own column.
//
// Two halves. Every spelling authenticates, which a handler doing its own
// lowering would also pass — and the answer names the account by the spelling
// **the operator chose**, which it would not: a handler that reduced the
// presented name and carried its own reduction into the response would answer
// `Name: "ADA"`. The name in the body is read back out of the row the fold
// found, which is the only way the two can agree.
func TestTheUsernameIsMatchedThroughTheFoldedColumnAndNotByTheHandler(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	for _, spelling := range []string{"Ada", "ADA", "ada", "aDa"} {
		answer := authenticate(t, handler, spelling, "correct horse")
		if answer.status != http.StatusOK {
			t.Fatalf("authenticating as %q answered %d, want 200", spelling, answer.status)
		}

		var result struct {
			User struct{ Name string }
		}
		if err := json.Unmarshal([]byte(answer.body), &result); err != nil {
			t.Fatalf("decoding the result for %q: %v", spelling, err)
		}
		if result.User.Name != "Ada" {
			t.Errorf("authenticating as %q answered the name %q, want the operator's own spelling %q",
				spelling, result.User.Name, "Ada")
		}
	}
}

// TestTheFourMeasuredRefusalsAreOneBodyAndFourStatuses is AC-2.
//
// The four conditions spec 3.3 answers with the controller's refusal shape are
// compared, byte for byte, against **one** golden file. That is the whole point
// of the case: behaviours 2.11 records that the status is the entire difference
// between them, so a test writing the same literal four times would assert four
// independent facts and pass on a build where one of them had drifted.
//
// Two of the four are measured at the reference and two are v1's own decision
// held open as OQ-5 — an enabled account given a wrong password, and a
// locked-out one. The lockout is not a fifth row: reaching the threshold writes
// IsDisabled, so on the next attempt a locked account *is* the disabled row
// (plan 6.7).
func TestTheFourMeasuredRefusalsAreOneBodyAndFourStatuses(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	createAccountWithPassword(t, store, "Grace", "correct horse", func(p *users.Policy) { p.IsDisabled = true })
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	golden := goldenRefusal(t)

	for _, row := range []struct {
		name     string
		username string
		password string
		header   []string
		want     int
	}{
		{
			name: "an unknown username", username: "Nobody", password: "correct horse",
			header: []string{httpapi.AuthorizationHeader, testClientHeader}, want: http.StatusUnauthorized,
		},
		{
			name: "a wrong password on an enabled account", username: "Ada", password: "wrong horse",
			header: []string{httpapi.AuthorizationHeader, testClientHeader}, want: http.StatusUnauthorized,
		},
		{
			name: "a disabled account", username: "Grace", password: "correct horse",
			header: []string{httpapi.AuthorizationHeader, testClientHeader}, want: http.StatusForbidden,
		},
		{
			name: "a missing client header", username: "Ada", password: "correct horse",
			header: nil, want: http.StatusBadRequest,
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]string{"Username": row.username, "Pw": row.password})
			if err != nil {
				t.Fatalf("encoding the body: %v", err)
			}
			answer := post(t, handler, string(body), row.header...)

			if answer.status != row.want {
				t.Errorf("%s answered %d, want %d", row.name, answer.status, row.want)
			}
			if answer.body != golden {
				t.Errorf("%s answered the body %q, want the golden %q", row.name, answer.body, golden)
			}
			// The charset is the parameter a framework adds for you, and the
			// reference does not send it: text/plain, bare.
			if answer.contentType != "text/plain" {
				t.Errorf("%s answered the content type %q, want %q with no charset parameter", row.name, answer.contentType, "text/plain")
			}
			if answer.length != int64(len(golden)) {
				t.Errorf("%s declared a length of %d, want %d", row.name, answer.length, len(golden))
			}
		})
	}
}

// TestABodyThatIsNotJSONKeepsTheProblemDetailsShape is the one refusal on this
// route that is **not** the twenty-five bytes.
//
// behaviours 1.11 splits refusals by where they happened, and a body the binder
// could not read never reaches the controller: it is answered with RFC 9457
// problem details, under `application/json; charset=utf-8` rather than
// `application/problem+json`, with `errors` keyed `"$"` for the document and
// the **action parameter's own name** for the parameter that could not be
// filled — `request` here
// [source: Jellyfin.Api/Controllers/UserController.cs:211 @ v10.11.11], which
// is nothing the client sent.
//
// The message under `"$"` is a recorded divergence: it is this parser's, where
// the reference's is .NET's, and reproducing it would mean writing a JSON
// parser to fail like another one (behaviours 1.11). The key, the status, the
// content type and the second entry all match, and the test asserts those.
func TestABodyThatIsNotJSONKeepsTheProblemDetailsShape(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := post(t, handler, "not json at all", httpapi.AuthorizationHeader, testClientHeader)

	if answer.status != http.StatusBadRequest {
		t.Fatalf("a body that is not JSON answered %d, want 400", answer.status)
	}
	if answer.contentType != "application/json; charset=utf-8" {
		t.Errorf("the problem document was sent as %q, want %q", answer.contentType, "application/json; charset=utf-8")
	}
	if answer.body == goldenRefusal(t) {
		t.Fatalf("a body that is not JSON answered the controller's twenty-five bytes; it is the one refusal on this route that keeps the problem-details shape")
	}

	var problem struct {
		Type    string              `json:"type"`
		Title   string              `json:"title"`
		Status  int                 `json:"status"`
		Errors  map[string][]string `json:"errors"`
		TraceId string              `json:"traceId"`
	}
	if err := json.Unmarshal([]byte(answer.body), &problem); err != nil {
		t.Fatalf("decoding the problem document %q: %v", answer.body, err)
	}

	if problem.Type != "https://tools.ietf.org/html/rfc9110#section-15.5.1" {
		t.Errorf("the problem type is %q", problem.Type)
	}
	if problem.Title != "One or more validation errors occurred." {
		t.Errorf("the problem title is %q", problem.Title)
	}
	if problem.Status != http.StatusBadRequest {
		t.Errorf("the problem document reports status %d, want 400", problem.Status)
	}
	if _, ok := problem.Errors["$"]; !ok {
		t.Errorf(`the errors map is %v, and has no "$" entry for the document the parser refused`, problem.Errors)
	}
	if got := problem.Errors["request"]; len(got) != 1 || got[0] != "The request field is required." {
		t.Errorf("the errors map names the action parameter as %v, want [\"The request field is required.\"] under the key \"request\"", got)
	}
	// A W3C trace-context identifier is per request by definition, so it is
	// compared by shape and never by value (behaviours 1.11).
	if matched := regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-00$`).MatchString(problem.TraceId); !matched {
		t.Errorf("the traceId is %q, which is not a W3C trace-context identifier", problem.TraceId)
	}
}

// TestAuthenticatingTwiceFromOneDeviceLeavesOneSessionAndOneLiveToken is AC-5.
//
// Three assertions and each catches a different wrong build: one session row
// catches an implementation that accumulates a session per login; the first
// token failing to resolve catches one that mints without revoking; and the
// second token resolving catches the opposite mistake, an OpenSession that
// revokes and deletes the credential it is in the middle of issuing.
func TestAuthenticatingTwiceFromOneDeviceLeavesOneSessionAndOneLiveToken(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	clock := &settableClock{at: aTestInstant}
	handler := newUsersHandler(t, store, clock)

	first := tokenFrom(t, authenticate(t, handler, "Ada", "correct horse"))
	clock.advance(time.Second)
	second := tokenFrom(t, authenticate(t, handler, "Ada", "correct horse"))

	if first == second {
		t.Fatalf("two authentications answered the same token %q; a bearer credential is minted, never derived", first)
	}

	open, err := store.Sessions(context.Background())
	if err != nil {
		t.Fatalf("reading the sessions: %v", err)
	}
	if len(open) != 1 {
		t.Errorf("two authentications from one device left %d sessions, want 1", len(open))
	}

	if _, _, found, err := store.SessionByTokenDigest(context.Background(), sessions.TokenDigest(first)); err != nil {
		t.Fatalf("resolving the first token: %v", err)
	} else if found {
		t.Errorf("the token issued by the first authentication still resolves; re-authenticating from one device replaces the session and invalidates the prior token")
	}
	if _, _, found, err := store.SessionByTokenDigest(context.Background(), sessions.TokenDigest(second)); err != nil {
		t.Fatalf("resolving the second token: %v", err)
	} else if !found {
		t.Errorf("the token the second authentication answered does not resolve, so the revocation deleted the credential it was issuing")
	}
}

// tokenFrom reads the access token out of a successful authentication.
func tokenFrom(t *testing.T, answer response) string {
	t.Helper()

	if answer.status != http.StatusOK {
		t.Fatalf("authenticating answered %d and the body %q, want 200", answer.status, answer.body)
	}
	var result struct{ AccessToken string }
	if err := json.Unmarshal([]byte(answer.body), &result); err != nil {
		t.Fatalf("decoding the result: %v", err)
	}
	return result.AccessToken
}

// TestAClientHeaderWithNoDeviceIdIsFatalHereAndOnNoOtherRoute is behaviours
// 2.13, asserted as the pair rather than as one half of it.
//
// The **identical** header value goes to two routes. The login route refuses it
// `400` because it needs the components to open a session; the other route
// answers `200`, because a missing DeviceId is fatal to a route and not to a
// parse. Asserting only the refusal would pass on the mistake plan 6.3 names as
// the one that matters — a parser that raised, refusing on every route at once
// requests the reference serves.
//
// The second route is `GET /System/Info` over a **real** authenticator, against
// an installation whose setup is finished, so that the header is load-bearing
// there: the request is admitted because that same header value yielded a Token
// the store could resolve. A route that never read the header would answer 200
// under any parser at all, which would make the second half prove nothing.
func TestAClientHeaderWithNoDeviceIdIsFatalHereAndOnNoOtherRoute(t *testing.T) {
	store := openStore(t)
	id := createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	const token = "0123456789abcdef0123456789abcdef"
	openSessionFor(t, store, id, "Atrium Test", "device-1", token, aTestInstant)

	// Four components minus DeviceId, plus the token this account holds.
	headerWithoutDeviceID := `MediaBrowser Client="Atrium Test", Device="A Device", Version="1.0.0", Token="` + token + `"`

	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})
	body, err := json.Marshal(map[string]string{"Username": "Ada", "Pw": "correct horse"})
	if err != nil {
		t.Fatalf("encoding the body: %v", err)
	}
	refused := post(t, handler, string(body), httpapi.AuthorizationHeader, headerWithoutDeviceID)
	if refused.status != http.StatusBadRequest {
		t.Errorf("the login route answered %d to a header carrying no DeviceId, want 400", refused.status)
	}
	if golden := goldenRefusal(t); refused.body != golden {
		t.Errorf("the login route's refusal is %q, want the same golden body as the other three: %q", refused.body, golden)
	}

	// The same header value, on a route that reads it and does not open a
	// session.
	system := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: newAuthenticator(t, store, &settableClock{at: aTestInstant}),
	})
	served := systemInfo(t, system, func(r *http.Request) {
		r.Header.Set(httpapi.AuthorizationHeader, headerWithoutDeviceID)
	})
	if served.Code != http.StatusOK {
		t.Errorf("the identical header answered %d on GET /System/Info, want 200 — a missing DeviceId is fatal to one route and to no header", served.Code)
	}
}

// TestEitherHeaderNameIdentifiesTheClient asserts that both spellings of spec
// 3.2's grammar are read on this route.
//
// spec 3.2's request table names `X-Emby-Authorization`, and every real client
// sends one of the two; the reference reads `Authorization` first and falls
// back to the Emby spelling
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:231-238 @ v10.11.11].
// A route that read only the field name its own specification's table happens
// to print would refuse half the clients in the wild with a `400` naming
// nothing.
func TestEitherHeaderNameIdentifiesTheClient(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	body, err := json.Marshal(map[string]string{"Username": "Ada", "Pw": "correct horse"})
	if err != nil {
		t.Fatalf("encoding the body: %v", err)
	}

	for _, field := range []string{httpapi.AuthorizationHeader, httpapi.EmbyAuthorizationHeader} {
		t.Run(field, func(t *testing.T) {
			answer := post(t, handler, string(body), field, testClientHeader)
			if answer.status != http.StatusOK {
				t.Errorf("a client identified through %s answered %d and the body %q, want 200", field, answer.status, answer.body)
			}
		})
	}
}

// TestTheClientComponentsAreCheckedBeforeTheCredential is the refusal order,
// and it is observable on exactly one request.
//
// A request with an unusable header **and** a wrong password answers `400` and
// not `401`. The reference checks its four arguments at the session manager
// before it looks a user up
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1589-1592 @ v10.11.11],
// and a build that verified the credential first would answer `401` here — the
// same body, a different status, and a client that logs the user out and
// reprompts for a password that was correct.
func TestTheClientComponentsAreCheckedBeforeTheCredential(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	body, err := json.Marshal(map[string]string{"Username": "Ada", "Pw": "wrong horse"})
	if err != nil {
		t.Fatalf("encoding the body: %v", err)
	}
	answer := post(t, handler, string(body))
	if answer.status != http.StatusBadRequest {
		t.Errorf("a bad header beside a bad password answered %d, want 400 — the four components are checked first", answer.status)
	}
}

// TestTheBodyIsBoundBeforeTheClientComponents is the other half of the same
// order.
//
// The reference binds the action parameter before the action runs, so a body
// that is not JSON is the problem-details `400` **whatever the headers
// carried** — including no client header at all, which would otherwise be the
// twenty-five bytes. One status, two shapes, and the order decides which.
func TestTheBodyIsBoundBeforeTheClientComponents(t *testing.T) {
	store := openStore(t)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := post(t, handler, "not json at all")
	if answer.status != http.StatusBadRequest {
		t.Fatalf("a body that is not JSON with no client header answered %d, want 400", answer.status)
	}
	if answer.body == goldenRefusal(t) {
		t.Errorf("the refusal is the controller's twenty-five bytes, so the client header was checked before the body was bound")
	}
}

// TestABodyMissingItsMembersIsACredentialRefusalAndNotAValidationOne is the
// reading of plan 7's row that the reference's own model settles.
//
// Both members of the request body are nullable there
// [source: Jellyfin.Api/Models/UserDtos/AuthenticateUserByName.cs @ v10.11.11]
// and the `[Required]` is on the parameter rather than on either member, so
// `{}` binds successfully and is refused as a credential — the twenty-five
// bytes with `401` — rather than as a validation failure. A build that made
// Username or Pw required would answer the problem document here and tell a
// client that its request was malformed when its password was wrong.
func TestABodyMissingItsMembersIsACredentialRefusalAndNotAValidationOne(t *testing.T) {
	store := openStore(t)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := post(t, handler, `{}`, httpapi.AuthorizationHeader, testClientHeader)
	if answer.status != http.StatusUnauthorized {
		t.Errorf("an empty JSON object answered %d, want 401", answer.status)
	}
	if golden := goldenRefusal(t); answer.body != golden {
		t.Errorf("an empty JSON object answered %q, want the credential refusal %q", answer.body, golden)
	}
}

// TestAStoreFailureIsFiveHundredAndNeverARefusal is plan 7's last row, and the
// reason it has one: a client answered 401 discards a credential that was fine.
func TestAStoreFailureIsFiveHundredAndNeverARefusal(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", nil)

	broken := errors.New("the session store is unreadable")
	handler, err := httpapi.NewUsersHandler(httpapi.UsersHandlerConfig{
		InstallationID: testInstallationID,
		Login:          users.NewLogin(store, &settableClock{at: aTestInstant}),
		Accounts:       store,
		Sessions:       failingSessions{err: broken},
		Clock:          &settableClock{at: aTestInstant},
	})
	if err != nil {
		t.Fatalf("building the users handler: %v", err)
	}

	answer := authenticate(t, handler, "Ada", "correct horse")
	if answer.status != http.StatusInternalServerError {
		t.Errorf("a login whose session store failed answered %d, want 500", answer.status)
	}
	if answer.body != "" {
		t.Errorf("the 500 carried the body %q, want none", answer.body)
	}
}

// TestTheUsersHandlerRefusesToBeBuiltWithoutItsPorts asserts that every member
// is required, one at a time.
//
// A handler that defaulted any of them would answer requests against a store
// nobody wired, or stamp a session from a clock architecture 2 forbids it to
// read directly.
func TestTheUsersHandlerRefusesToBeBuiltWithoutItsPorts(t *testing.T) {
	store := openStore(t)
	clock := &settableClock{at: aTestInstant}
	whole := httpapi.UsersHandlerConfig{
		InstallationID: testInstallationID,
		Login:          users.NewLogin(store, clock),
		Accounts:       store,
		Sessions:       store,
		Clock:          clock,
	}

	for _, row := range []struct {
		missing string
		remove  func(*httpapi.UsersHandlerConfig)
	}{
		{"the installation identity", func(c *httpapi.UsersHandlerConfig) { c.InstallationID = "" }},
		{"the login path", func(c *httpapi.UsersHandlerConfig) { c.Login = nil }},
		{"the account store", func(c *httpapi.UsersHandlerConfig) { c.Accounts = nil }},
		{"the session store", func(c *httpapi.UsersHandlerConfig) { c.Sessions = nil }},
		{"the clock", func(c *httpapi.UsersHandlerConfig) { c.Clock = nil }},
	} {
		t.Run(row.missing, func(t *testing.T) {
			cfg := whole
			row.remove(&cfg)
			if _, err := httpapi.NewUsersHandler(cfg); err == nil {
				t.Errorf("a users handler was built without %s", row.missing)
			}
		})
	}
}
