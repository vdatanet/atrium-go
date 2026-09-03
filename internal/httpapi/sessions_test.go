package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// This file is spec 3.8's two routes, asserted at the HTTP boundary against a
// real store. T17 is what registers them on the server's router; these are the
// handlers, reachable, and every assertion is on the bytes a client would read
// off a connection.

// The two devices the fixture's two sessions are opened from. A session is
// keyed on (Client, DeviceId), so two seats sharing a device would be one
// session row and a second login would revoke the first one's token
// (002 plan 6.5) — which is exactly the fixture spec 3.8's parameter matrix
// cannot be measured over.
const (
	adminDevice    = "device-ada"
	strangerDevice = "device-cleo"
)

// sessionsFixture is spec 3.8's "two sessions on two devices": an
// administrator, who sees both, and a restricted non-administrator, who sees
// one — with the two logins an hour apart so that a window can exclude the
// older of them.
type sessionsFixture struct {
	store   *sqlite.Store
	clock   *settableClock
	handler *httpapi.SessionsHandler

	// admin is the seat every "an administrator asks" row is sent from, and
	// the only seat whose visibility rule is not its own identifier.
	admin occupant

	// stranger is the restricted non-administrator: the seat whose
	// controllableByUserId naming somebody else is the 403 this whole matrix
	// turns on.
	stranger occupant
}

func newSessionsFixture(t *testing.T) sessionsFixture {
	t.Helper()

	store := openStore(t)
	clock := &settableClock{at: aTestInstant}
	users := newUsersHandler(t, store, clock)

	admin := seat(t, store, users, "Ada", "correct horse", adminDevice, administrator)
	clock.advance(time.Hour)
	stranger := seat(t, store, users, "Cleo", "a third password", strangerDevice, restricted)

	return sessionsFixture{
		store:    store,
		clock:    clock,
		handler:  newSessionsHandler(t, store, clock),
		admin:    admin,
		stranger: stranger,
	}
}

// newSessionsHandler builds the handler over a real store, a real
// authenticator and a clock a test can hold still.
//
// The authenticator is the real one for the reason newUsersHandler's is: the
// whole of this route is *which caller* a live token resolves to, and a
// stand-in would let the matrix pass over an admission rule nothing exercises.
func newSessionsHandler(t *testing.T, store *sqlite.Store, clock ports.Clock) *httpapi.SessionsHandler {
	t.Helper()

	handler, err := httpapi.NewSessionsHandler(httpapi.SessionsHandlerConfig{
		Sessions:      store,
		Accounts:      store,
		Authenticator: newAuthenticator(t, store, clock),
		Clock:         clock,
	})
	if err != nil {
		t.Fatalf("building the sessions handler: %v", err)
	}
	return handler
}

// sessionRoutes mounts spec 3.8's two routes on a router of this test's own.
//
// Both are on one router because every assertion the capabilities route makes
// is read back through GET /Sessions: a second router would be a second answer
// to which handler a request reaches, and a round trip is only a round trip
// while both halves are on one (T15's rule, one route pair over).
func sessionRoutes(handler *httpapi.SessionsHandler) http.Handler {
	router := chi.NewRouter()
	router.Get("/Sessions", handler.Sessions())
	router.Post("/Sessions/Capabilities/Full", handler.PostFullCapabilities())
	return router
}

// sessionRequest sends one request to the routes above and reads the whole
// answer off a real connection.
//
// Over a real server for the reason post and getUser are: an absent
// Content-Type and a declared Content-Length are part of a response's shape,
// and httptest.ResponseRecorder can express neither. Two of this file's
// assertions — the 204 that carries no content type, and the 403 that carries
// `text/plain` and twenty-five bytes — are exactly those.
func sessionRequest(t *testing.T, handler http.Handler, method, target, body, token string) response {
	t.Helper()

	server := httptest.NewServer(handler)
	defer server.Close()

	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	var request *http.Request
	var err error
	if reader != nil {
		request, err = http.NewRequest(method, server.URL+target, reader)
	} else {
		request, err = http.NewRequest(method, server.URL+target, nil)
	}
	if err != nil {
		t.Fatalf("building the %s %s request: %v", method, target, err)
	}
	if token != "" {
		request.Header.Set(httpapi.EmbyTokenHeader, token)
	}

	answer, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("sending the %s %s request: %v", method, target, err)
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

// listSessions sends one GET /Sessions and insists on a 200.
func listSessions(t *testing.T, f sessionsFixture, query, token string) string {
	t.Helper()

	answer := sessionRequest(t, sessionRoutes(f.handler), http.MethodGet, "/Sessions"+query, "", token)
	if answer.status != http.StatusOK {
		t.Fatalf("GET /Sessions%s answered %d and the body %q, want 200", query, answer.status, answer.body)
	}
	return answer.body
}

// postCapabilities sends one POST /Sessions/Capabilities/Full.
func postCapabilities(t *testing.T, f sessionsFixture, query, body, token string) response {
	t.Helper()
	return sessionRequest(t, sessionRoutes(f.handler), http.MethodPost, "/Sessions/Capabilities/Full"+query, body, token)
}

// declare posts a capabilities document and insists on the 204.
func declare(t *testing.T, f sessionsFixture, body, token string) {
	t.Helper()

	answer := postCapabilities(t, f, "", body, token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("posting %s answered %d and the body %q, want 204", body, answer.status, answer.body)
	}
}

// elements splits a JSON array into its elements, in the order the bytes carry
// them.
//
// Position is the assertion for a list body: L3 compares list rows by position
// (architecture 2), and a test that sorted or searched would pass on a build
// that answered a different order.
func elements(t *testing.T, document string) []json.RawMessage {
	t.Helper()

	var list []json.RawMessage
	if err := json.Unmarshal([]byte(document), &list); err != nil {
		t.Fatalf("reading the session list %s: %v", document, err)
	}
	return list
}

// sessionIDs is the identifier of every session in a /Sessions body, in order.
func sessionIDs(t *testing.T, document string) []string {
	t.Helper()

	ids := make([]string, 0, 4)
	for _, element := range elements(t, document) {
		_, values := members(t, element)
		raw, ok := values["Id"]
		if !ok {
			t.Fatalf("a session in %s carries no Id", document)
		}
		var id string
		if err := json.Unmarshal(raw, &id); err != nil {
			t.Fatalf("reading a session's Id: %v", err)
		}
		ids = append(ids, id)
	}
	return ids
}

// assertSessions insists that a body names exactly these sessions, in this
// order.
func assertSessions(t *testing.T, what, document string, want ...string) {
	t.Helper()

	got := sessionIDs(t, document)
	if len(got) != len(want) {
		t.Fatalf("%s answered %d sessions, want %d — the body was %s", what, len(got), len(want), document)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s answered session %d = %s, want %s — the body was %s", what, i, got[i], want[i], document)
		}
	}
}

// adminSession and strangerSession are the two rows the fixture holds, named
// by the derivation the login used.
func (f sessionsFixture) adminSession() string {
	return sessions.DeriveID(testSessionClient, adminDevice)
}

func (f sessionsFixture) strangerSession() string {
	return sessions.DeriveID(testSessionClient, strangerDevice)
}

// testSessionClient is the client name seat() logs in under. A session
// identifier is derived from it and the device, case-sensitively in both
// (sessions.DeriveID), so it is a constant rather than a spelling repeated.
const testSessionClient = "Atrium Test"

// TestSessionsAnswersTheCallersOwnAndAnAdministratorsIsEverybodys is spec
// 3.8's visibility rule at the wire: their own always, and all sessions for an
// administrator.
func TestSessionsAnswersTheCallersOwnAndAnAdministratorsIsEverybodys(t *testing.T) {
	f := newSessionsFixture(t)

	assertSessions(t, "a restricted seat's own reading",
		listSessions(t, f, "", f.stranger.token), f.strangerSession())

	// The order is the store's — created_at, id (002 plan 6.5 and T4) — and the
	// administrator logged in first.
	assertSessions(t, "an administrator's reading",
		listSessions(t, f, "", f.admin.token), f.adminSession(), f.strangerSession())
}

// TestASessionThatHasPostedNoCapabilitiesCarriesNoneAndTwoEmptyLists is the
// state every session starts in.
//
// `Capabilities` is **absent** rather than null: the reference's constructor
// initialises AdditionalUsers, PlayState and both queues and leaves this one
// unset [source: MediaBrowser.Controller/Session/SessionInfo.cs:39-49 @ v10.11.11],
// and behaviours 1.7 omits a null. The two hoisted lists are `[]` and not
// `null`, which is a distinction invisible to anything that parses the body.
func TestASessionThatHasPostedNoCapabilitiesCarriesNoneAndTwoEmptyLists(t *testing.T) {
	f := newSessionsFixture(t)

	document := listSessions(t, f, "", f.stranger.token)
	names, values := members(t, elements(t, document)[0])

	for _, name := range names {
		if name == "Capabilities" {
			t.Errorf("a session that has declared nothing carries Capabilities: %s", document)
		}
	}
	for _, name := range []string{"PlayableMediaTypes", "SupportedCommands"} {
		if string(values[name]) != "[]" {
			t.Errorf("%s = %s on a session that has declared nothing, want [] — null is a different document", name, values[name])
		}
	}
}

// TestPostingCapabilitiesAnswersTwoHundredAndFourWithNoBodyAndNoContentType is
// spec 3.8's `204`, and the L1 evidence surface.yaml asks of this route.
func TestPostingCapabilitiesAnswersTwoHundredAndFourWithNoBodyAndNoContentType(t *testing.T) {
	f := newSessionsFixture(t)

	answer := postCapabilities(t, f, "", `{"PlayableMediaTypes":["Audio"]}`, f.stranger.token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 — the body was %q", answer.status, answer.body)
	}
	if answer.body != "" {
		t.Errorf("body = %q, want none", answer.body)
	}
	if answer.contentType != "" {
		t.Errorf("Content-Type = %q, want none — a 204 describes nothing", answer.contentType)
	}
}

// TestTheDeclarationIsHoistedWhileTheControlFlagsStayTheServersJudgement is
// behaviours 2.14, which is the case that separates the client's declaration
// from the server's judgement about it.
//
// The reference echoes `SupportsMediaControl: true` back inside `Capabilities`
// for a session that posted it while reporting `false` at the **top level** of
// the same session
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], and
// hoists `PlayableMediaTypes` and `SupportedCommands` verbatim. So this is
// measured parity and not a gap — and the assertion is a `false` beside a
// declared `true`, which is the only fixture that can tell a hoist from a
// judgement.
func TestTheDeclarationIsHoistedWhileTheControlFlagsStayTheServersJudgement(t *testing.T) {
	f := newSessionsFixture(t)

	const declaration = `{"PlayableMediaTypes":["Audio","Video"],` +
		`"SupportedCommands":["Play","Pause"],` +
		`"SupportsMediaControl":true,"SupportsPersistentIdentifier":true}`
	declare(t, f, declaration, f.stranger.token)

	_, values := members(t, elements(t, listSessions(t, f, "", f.stranger.token))[0])

	if got := string(values["PlayableMediaTypes"]); got != `["Audio","Video"]` {
		t.Errorf(`PlayableMediaTypes = %s, want ["Audio","Video"] hoisted verbatim`, got)
	}
	if got := string(values["SupportedCommands"]); got != `["Play","Pause"]` {
		t.Errorf(`SupportedCommands = %s, want ["Play","Pause"] hoisted verbatim`, got)
	}
	if got := string(values["SupportsMediaControl"]); got != "false" {
		t.Errorf("SupportsMediaControl = %s beside a declared true, want false: the declaration is the client's and the flag is the server's (behaviours 2.14)", got)
	}
	if got := string(values["SupportsRemoteControl"]); got != "false" {
		t.Errorf("SupportsRemoteControl = %s, want false: v1 has no control channel", got)
	}
	if got := string(values["Capabilities"]); got != declaration {
		t.Errorf("Capabilities = %s, want the posted declaration echoed whole: %s", got, declaration)
	}
	if !bytes.Contains(values["Capabilities"], []byte(`"SupportsMediaControl":true`)) {
		t.Errorf("the echoed declaration lost the true the client declared: %s", values["Capabilities"])
	}
}

// TestAnUnknownCapabilitiesPropertySurvivesIntoSessions is behaviours 5.9's
// **recorded divergence**, asserted as a divergence.
//
// The reference accepts a property outside its schema — the `204` — and
// **drops** it, so its `Capabilities` echoes the declared fields and not the
// stranger `[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
// 2026-08-28]`. This server keeps it. AGENTS.md 3 makes a conformance
// assertion a declared inequality where **a declared difference that has gone
// away fails too**, and this is the one place 002 owns one: a build that
// quietly became parity here fails this test rather than passing a round trip.
//
// # It is deliberately the opposite of what POST /Users/Configuration does
//
// internal/users' TestAnUnknownPropertyInAStoredConfigurationIsDroppedAndTheDeclaredOnesSurvive
// and userconfiguration_test.go's wire twin assert that an unknown
// *configuration* property does **not** survive, and the declared ones do. The
// two look like one question and are not: a configuration document decodes onto
// the reference's defaults before the store sees it (spec 3.6, plan 6.6), and a
// capabilities document is stored as the bytes the client posted (spec 3.8,
// plan 6.10). Inheriting either rule by proximity is the mistake both comments
// exist to prevent.
func TestAnUnknownCapabilitiesPropertySurvivesIntoSessions(t *testing.T) {
	f := newSessionsFixture(t)

	const declaration = `{"PlayableMediaTypes":["Audio"],"AProperyNoSchemaDeclares":"kept"}`
	declare(t, f, declaration, f.stranger.token)

	_, values := members(t, elements(t, listSessions(t, f, "", f.stranger.token))[0])

	if !bytes.Contains(values["Capabilities"], []byte(`"AProperyNoSchemaDeclares":"kept"`)) {
		t.Errorf("the unknown property is gone from %s.\n"+
			"behaviours 5.9 declares this server keeping it and the reference dropping it. "+
			"If this server now drops it too, that is a declared difference that has gone away, "+
			"which AGENTS.md 3 makes a failure — amend behaviours 5.9 in the same change or restore the behaviour",
			values["Capabilities"])
	}
	if got := string(values["PlayableMediaTypes"]); got != `["Audio"]` {
		t.Errorf("PlayableMediaTypes = %s beside the unknown property, want the declared one hoisted anyway", got)
	}
}

// TestASecondPostReplacesTheCapabilitiesRatherThanMergingIntoThem is spec
// 3.8's replacement, and it is the one assertion no round trip can make.
//
// A handler that decoded the posted document over the **stored** one passes
// every "post it, read it back" test in this file, because everything posted is
// still there. Only a second post carrying **fewer** properties catches it, and
// what it is compared against is the second document itself rather than a
// transcribed list — T15's shape on the other document of this feature.
func TestASecondPostReplacesTheCapabilitiesRatherThanMergingIntoThem(t *testing.T) {
	f := newSessionsFixture(t)

	declare(t, f, `{"PlayableMediaTypes":["Audio","Video"],"SupportedCommands":["Play"],"SupportsMediaControl":true}`, f.stranger.token)

	const second = `{"PlayableMediaTypes":["Audio"]}`
	declare(t, f, second, f.stranger.token)

	_, values := members(t, elements(t, listSessions(t, f, "", f.stranger.token))[0])

	if got := string(values["Capabilities"]); got != second {
		t.Errorf("Capabilities = %s after a second, smaller post, want exactly %s — the route is named Full and replaces (behaviours 2.14)", got, second)
	}
	if got := string(values["SupportedCommands"]); got != "[]" {
		t.Errorf("SupportedCommands = %s after a post that declared none, want [] — a merge would have kept the first post's", got)
	}
}

// TestPostingCapabilitiesNamingAnotherSessionUpdatesTheCallersOwn is U-14's
// shape a second time, on a second route, and it is written as a test for
// U-14's reason.
//
// The reference declares `[FromQuery] string? id` and writes the declaration
// into the session it names
// [source: Jellyfin.Api/Controllers/SessionController.cs:380-389 @ v10.11.11].
// spec 3.8 declares no parameter on this route, and an unrecognised query value
// is ignored rather than rejected (behaviours 1.12), so a request naming
// somebody else's session writes to its own. Asserting it on **both** rows is
// what makes the day a probe measures it a failing test naming the behaviour
// that moved rather than a rediscovery.
func TestPostingCapabilitiesNamingAnotherSessionUpdatesTheCallersOwn(t *testing.T) {
	f := newSessionsFixture(t)

	const declaration = `{"PlayableMediaTypes":["Audio"]}`
	answer := postCapabilities(t, f, "?id="+f.adminSession(), declaration, f.stranger.token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("posting with another session's id answered %d and the body %q, want 204", answer.status, answer.body)
	}

	stored := storedSessions(t, f.store)
	if got := string(stored[f.strangerSession()].CapabilitiesDocument); got != declaration {
		t.Errorf("the caller's own session holds %q, want the posted declaration %q", got, declaration)
	}
	if got := stored[f.adminSession()].CapabilitiesDocument; len(got) != 0 {
		t.Errorf("the session named in the query holds %q, want nothing written to it", got)
	}
}

// storedSessions reads every session row back out of the store, by identifier.
//
// The only assertion in this file that reads anything but a response, and it is
// there for T7's reason: an assertion phrased in the wire's vocabulary cannot
// see state the wire does not carry. GET /Sessions never shows the
// administrator's row to the restricted seat, so "the named session was left
// alone" has no request that could prove it.
func storedSessions(t *testing.T, store *sqlite.Store) map[string]ports.Session {
	t.Helper()

	open, err := store.Sessions(context.Background())
	if err != nil {
		t.Fatalf("reading the sessions: %v", err)
	}
	byID := make(map[string]ports.Session, len(open))
	for _, session := range open {
		byID[session.ID] = session
	}
	return byID
}

// TestABodyThisRouteCannotReadIsTheValidationRefusal is the second refusal spec
// 3.8 does not name, and it is the login route's rule applied with one word
// changed.
//
// The action parameter is `capabilities` where the login route's is `request`
// and the configuration route's is `userConfig`
// [source: Jellyfin.Api/Controllers/SessionController.cs:380-382 @ v10.11.11].
// The alternative to refusing was storing bytes that are not a document, which
// would put an unparseable subtree into every later /Sessions body — a refusal
// that is not measured is a better answer than a response nobody can decode.
//
// The message under `"$"` is asserted to be the **same text the login route
// sends for the same bytes**, which is a rule about two of this server's own
// routes rather than about the reference (behaviours 1.11 already declares that
// half unmatchable).
func TestABodyThisRouteCannotReadIsTheValidationRefusal(t *testing.T) {
	f := newSessionsFixture(t)
	users := newUsersHandler(t, f.store, f.clock)

	rows := []struct {
		name string
		body string
	}{
		{"a body that is not JSON", "not a document"},
		{"a JSON array", `["Audio"]`},
		{"a JSON string", `"Audio"`},
		{"the null document", "null"},
		{"an empty body", ""},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			answer := postCapabilities(t, f, "", row.body, f.stranger.token)
			if answer.status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 — the body was %q", answer.status, answer.body)
			}
			if answer.contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type = %q, want application/json; charset=utf-8 (behaviours 1.11)", answer.contentType)
			}

			names, values := members(t, []byte(answer.body))
			var errorsMap map[string][]string
			if err := json.Unmarshal(values["errors"], &errorsMap); err != nil {
				t.Fatalf("reading the errors map of %q: %v", answer.body, err)
			}
			if _, ok := errorsMap["$"]; !ok {
				t.Errorf("the errors map %v has no \"$\" key", errorsMap)
			}
			if got, ok := errorsMap["capabilities"]; !ok || len(got) != 1 || got[0] != "The capabilities field is required." {
				t.Errorf("the errors map %v does not name this route's own action parameter", errorsMap)
			}
			if len(names) == 0 {
				t.Errorf("the problem document carries no members: %q", answer.body)
			}
		})
	}

	// The two routes describe one unreadable document in one spelling. The
	// null document is deliberately not in this comparison: the login route's
	// body binds it successfully and refuses it as a credential (users.go), so
	// there is no second message to agree with.
	const unreadable = "not a document"
	mine := postCapabilities(t, f, "", unreadable, f.stranger.token)
	theirs := post(t, users, unreadable, httpapi.AuthorizationHeader, testClientHeader)
	if messageUnderDollar(t, mine.body) != messageUnderDollar(t, theirs.body) {
		t.Errorf("this route says %q about an unreadable document and the login route says %q; two of this server's own routes should spell one failure one way",
			messageUnderDollar(t, mine.body), messageUnderDollar(t, theirs.body))
	}
}

// messageUnderDollar reads the deserialiser's own words out of a problem
// document.
func messageUnderDollar(t *testing.T, document string) string {
	t.Helper()

	_, values := members(t, []byte(document))
	var errorsMap map[string][]string
	if err := json.Unmarshal(values["errors"], &errorsMap); err != nil {
		t.Fatalf("reading the errors map of %q: %v", document, err)
	}
	if len(errorsMap["$"]) != 1 {
		t.Fatalf("the errors map of %q carries no single message under \"$\"", document)
	}
	return errorsMap["$"][0]
}

// TestBothSessionRoutesRefuseARequestCarryingNoCredential is spec 3.8's `401`,
// in behaviours 1.11's empty shape.
func TestBothSessionRoutesRefuseARequestCarryingNoCredential(t *testing.T) {
	f := newSessionsFixture(t)
	routes := sessionRoutes(f.handler)

	rows := []struct {
		name           string
		method, target string
		body           string
	}{
		{"GET /Sessions", http.MethodGet, "/Sessions", ""},
		{"POST /Sessions/Capabilities/Full", http.MethodPost, "/Sessions/Capabilities/Full", `{"PlayableMediaTypes":[]}`},
	}

	for _, row := range rows {
		answer := sessionRequest(t, routes, row.method, row.target, row.body, "")
		if answer.status != http.StatusUnauthorized {
			t.Errorf("%s without a token answered %d, want 401", row.name, answer.status)
		}
		if answer.body != "" {
			t.Errorf("%s without a token answered the body %q, want none", row.name, answer.body)
		}
		if answer.contentType != "" {
			t.Errorf("%s without a token declared Content-Type %q, want none", row.name, answer.contentType)
		}
	}

	// The credential is read before the body is bound: a request with no token
	// and a body this route cannot read is the empty 401 and never the
	// validation 400. No assertion about either refusal alone can see the order
	// (T14's rule).
	answer := sessionRequest(t, routes, http.MethodPost, "/Sessions/Capabilities/Full", "not a document", "")
	if answer.status != http.StatusUnauthorized || answer.body != "" {
		t.Errorf("no credential beside an unreadable body answered %d and %q, want the empty 401", answer.status, answer.body)
	}
}

// TestTheParameterMatrix is spec 3.8's table, sent as requests.
//
// Every row is one request and one expected list, and the two that are not a
// list — the 403 and the two 400s — are asserted separately below, because
// their whole content is a shape rather than a set of rows.
func TestTheParameterMatrix(t *testing.T) {
	f := newSessionsFixture(t)

	// Far enough past the second login that a sixty-second window excludes it.
	// The administrator's own row is stamped by the request itself, which is
	// why every activeWithinSeconds row is sent from that seat.
	f.clock.advance(time.Hour)

	rows := []struct {
		name  string
		query string
		token string
		want  []string
	}{
		{"no parameter, from the administrator", "", f.admin.token,
			[]string{f.adminSession(), f.strangerSession()}},
		{"deviceId naming the administrator's device", "?deviceId=" + adminDevice, f.admin.token,
			[]string{f.adminSession()}},
		{"deviceId in another case", "?deviceId=" + strings.ToUpper(adminDevice), f.admin.token,
			[]string{f.adminSession()}},
		{"deviceId matching nothing", "?deviceId=no-such-device", f.admin.token,
			nil},
		{"an empty deviceId is ignored", "?deviceId=", f.admin.token,
			[]string{f.adminSession(), f.strangerSession()}},
		{"a restricted seat naming another user's device is an empty 200", "?deviceId=" + adminDevice, f.stranger.token,
			nil},
		{"activeWithinSeconds at 0 is ignored", "?activeWithinSeconds=0", f.admin.token,
			[]string{f.adminSession(), f.strangerSession()}},
		{"a negative activeWithinSeconds is ignored", "?activeWithinSeconds=-5", f.admin.token,
			[]string{f.adminSession(), f.strangerSession()}},
		{"controllableByUserId naming the caller, from a restricted seat", "?controllableByUserId=" + f.stranger.id, f.stranger.token,
			nil},
		{"controllableByUserId named by an administrator", "?controllableByUserId=" + f.stranger.id, f.admin.token,
			nil},
		{"controllableByUserId beside deviceId", "?deviceId=" + adminDevice + "&controllableByUserId=" + f.admin.id, f.admin.token,
			nil},
		{"the all-zero controllableByUserId is the caller's own", "?controllableByUserId=00000000000000000000000000000000", f.stranger.token,
			nil},
		{"a parameter this route does not declare is ignored", "?notAParameter=x", f.stranger.token,
			[]string{f.strangerSession()}},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			assertSessions(t, row.name, listSessions(t, f, row.query, row.token), row.want...)
		})
	}
}

// TestActiveWithinSecondsExcludesASessionOlderThanItsWindow is the third of
// spec 3.8's three activeWithinSeconds rows, and the only one that filters.
//
// It has a fixture of its own rather than a row in the matrix above, and that
// is the finding rather than tidiness: **an authenticated request advances its
// own session's LastActivityDate** (002 plan 6.10), so any earlier row sent
// from the restricted seat stamps the very row this window is meant to exclude,
// and the assertion silently becomes "both sessions are recent". The window is
// therefore measured from the administrator's seat, against a session no
// request in this test has touched.
func TestActiveWithinSecondsExcludesASessionOlderThanItsWindow(t *testing.T) {
	f := newSessionsFixture(t)
	f.clock.advance(time.Hour)

	assertSessions(t, "a sixty-second window an hour after the second login",
		listSessions(t, f, "?activeWithinSeconds=60", f.admin.token), f.adminSession())

	// And the same request with a window wide enough to reach it, so that the
	// row above is a filter rather than a list that was empty anyway.
	assertSessions(t, "a window that reaches both",
		listSessions(t, f, "?activeWithinSeconds=7200", f.admin.token), f.adminSession(), f.strangerSession())
}

// TestAnEmptyAnswerIsAnEmptyArrayAndNotNull is the distinction a parsed body
// cannot see (Principle VIII).
//
// internal/wire serialises a nil slice as `null`, and spec 3.8 answers `[]`.
// The whole route's shape rests on the two places that is decided —
// sessions.Visible never returning nil and the handler's own slice — and this
// is the assertion that sees either of them going away.
func TestAnEmptyAnswerIsAnEmptyArrayAndNotNull(t *testing.T) {
	f := newSessionsFixture(t)

	document := listSessions(t, f, "?controllableByUserId="+f.stranger.id, f.stranger.token)
	if document != "[]" {
		t.Errorf("an empty answer is %q, want exactly []", document)
	}
}

// TestControllableByUserIdNamingSomebodyElseFromARestrictedSeatIsTheGoldenRefusal
// is AC-4's second half and the load-bearing half of spec 3.8's matrix.
//
// It is the **one request in 002 where a valid token is refused for who its
// holder is**: none of the other six routes gates on a permission, spec 3.7
// refuses nobody, and 001's /System/Info admits any authenticated caller once
// setup is complete. A route declaring neither parameter would answer both this
// request and the deviceId one above `200` with the caller's own sessions — a
// success where the reference refuses, and a client branching on the refusal
// would take the success path.
//
// The status and the media type are measured
// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29]; the
// twenty-five bytes are behaviours 1.11's rule applied to a refusal the
// reference raises the same way, which is register U-18 and is the reason to
// assert them against the same golden the login route's four refusals use
// rather than to assume them. Six responses now stand on that one file.
func TestControllableByUserIdNamingSomebodyElseFromARestrictedSeatIsTheGoldenRefusal(t *testing.T) {
	f := newSessionsFixture(t)

	answer := sessionRequest(t, sessionRoutes(f.handler), http.MethodGet,
		"/Sessions?controllableByUserId="+f.admin.id, "", f.stranger.token)

	if answer.status != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 — the body was %q", answer.status, answer.body)
	}
	if answer.contentType != "text/plain" {
		t.Errorf("Content-Type = %q, want text/plain with no charset", answer.contentType)
	}
	if golden := goldenRefusal(t); answer.body != golden {
		t.Errorf("body = %q, want the same golden the login route's refusals carry: %q", answer.body, golden)
	}
	if answer.length != int64(len(goldenRefusal(t))) {
		t.Errorf("Content-Length = %d, want %d", answer.length, len(goldenRefusal(t)))
	}
}

// TestNamingAnotherUsersDeviceAndNamingAnotherUserAnswerDifferently is the
// contrast spec 3.8 calls the observable half of the order.
//
// One route, two parameters that each name somebody else's property, two
// answers: the device is an empty `200` and the user is a `403`. The empty list
// is not a redaction of the refusal — it is the ordinary result of a filter
// that matched a row the caller may not see — and a build that answered them
// alike would be wrong whichever way it chose.
func TestNamingAnotherUsersDeviceAndNamingAnotherUserAnswerDifferently(t *testing.T) {
	f := newSessionsFixture(t)
	routes := sessionRoutes(f.handler)

	device := sessionRequest(t, routes, http.MethodGet, "/Sessions?deviceId="+adminDevice, "", f.stranger.token)
	user := sessionRequest(t, routes, http.MethodGet, "/Sessions?controllableByUserId="+f.admin.id, "", f.stranger.token)

	if device.status != http.StatusOK || device.body != "[]" {
		t.Errorf("naming another user's device answered %d and %q, want 200 and []", device.status, device.body)
	}
	if user.status != http.StatusForbidden {
		t.Errorf("naming another user answered %d, want 403", user.status)
	}
	if device.status == user.status {
		t.Errorf("both requests answered %d; spec 3.8 makes them two answers", device.status)
	}
}

// TestAParameterThatCannotBeBoundIsTheValidationRefusal is spec 3.8's
// ⚠️ UNVERIFIED row, implemented as the specification writes it and asserted so
// that the day register U-17's probe lands, a failing test names the behaviour
// that moved.
//
// behaviours 1.12 forgives an unrecognised *token* and this refuses a value
// that cannot parse as its declared *type* — the same shape spec 3.7 measured
// for `userId`, keyed on the parameter's own declared spelling.
func TestAParameterThatCannotBeBoundIsTheValidationRefusal(t *testing.T) {
	f := newSessionsFixture(t)

	rows := []struct {
		name, query, key, value string
	}{
		{"an activeWithinSeconds that is not an integer", "?activeWithinSeconds=soon", "activeWithinSeconds", "soon"},
		{"a controllableByUserId that is not an identifier", "?controllableByUserId=not-an-identifier", "controllableByUserId", "not-an-identifier"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			answer := sessionRequest(t, sessionRoutes(f.handler), http.MethodGet, "/Sessions"+row.query, "", f.stranger.token)
			if answer.status != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 — the body was %q", answer.status, answer.body)
			}

			_, values := members(t, []byte(answer.body))
			var errorsMap map[string][]string
			if err := json.Unmarshal(values["errors"], &errorsMap); err != nil {
				t.Fatalf("reading the errors map of %q: %v", answer.body, err)
			}
			got, ok := errorsMap[row.key]
			if !ok {
				t.Fatalf("the errors map %v is not keyed on the parameter's declared spelling %q", errorsMap, row.key)
			}
			if len(got) != 1 || got[0] != "The value '"+row.value+"' is not valid." {
				t.Errorf("the message under %s is %v, want the reference's own sentence quoting the value back", row.key, got)
			}
			if len(errorsMap) != 1 {
				t.Errorf("the errors map %v carries a second key; a query parameter's refusal names one parameter", errorsMap)
			}
		})
	}
}

// TestAControllableByUserIdIsBoundBeforeItIsRefused is the order between this
// route's two refusals, and it is one request.
//
// The reference binds an action's parameters before the action runs, so a
// `controllableByUserId` that is not an identifier is the binder's `400` before
// it is anybody's `403`. No assertion about either refusal alone can see it.
func TestAControllableByUserIdIsBoundBeforeItIsRefused(t *testing.T) {
	f := newSessionsFixture(t)

	answer := sessionRequest(t, sessionRoutes(f.handler), http.MethodGet,
		"/Sessions?controllableByUserId=somebody-else", "", f.stranger.token)
	if answer.status != http.StatusBadRequest {
		t.Errorf("a malformed controllableByUserId from a restricted seat answered %d, want the binder's 400 before the 403", answer.status)
	}
}

// TestAnActiveWithinSecondsAboveInt32IsAcceptedHereAndTheReferencesBinderRefusesIt
// records a delta rather than discovering it later.
//
// The reference declares `int?`, which is Int32
// [source: Jellyfin.Api/Controllers/SessionController.cs:58 @ v10.11.11], so a
// value above 2147483647 fails its binder. Here it binds, and
// sessions.Visible's saturating window answers the unfiltered list — the same
// answer a client asking for "every session ever" wanted. Accepting more than
// the reference is the safe direction (no request that succeeds there is
// refused here), and narrowing to Int32 at the edge would make that saturation
// unreachable from the wire. The register at T23 is owed the row.
func TestAnActiveWithinSecondsAboveInt32IsAcceptedHereAndTheReferencesBinderRefusesIt(t *testing.T) {
	f := newSessionsFixture(t)

	document := listSessions(t, f, "?activeWithinSeconds=4611686018427387904", f.admin.token)
	assertSessions(t, "an activeWithinSeconds beyond Int32", document, f.adminSession(), f.strangerSession())
}

// TestASessionsBodyDeclaresCapabilitiesFirst is the reference's declaration
// order, asserted as key order in the bytes.
//
// The reference declares `Capabilities` third, behind `PlayState` and
// `AdditionalUsers`, and this model declares neither of those
// [source: MediaBrowser.Model/Dto/SessionInfoDto.cs:17-29 @ v10.11.11] — so the
// member this task added is **first**. A map comparison passes on a reordered
// model, which is why this reads the names out of the bytes in the order the
// bytes carry them (T2's technique).
func TestASessionsBodyDeclaresCapabilitiesFirst(t *testing.T) {
	f := newSessionsFixture(t)
	declare(t, f, `{"PlayableMediaTypes":["Audio"]}`, f.stranger.token)

	want := []string{
		"Capabilities", "RemoteEndPoint", "PlayableMediaTypes", "Id", "UserId",
		"UserName", "Client", "LastActivityDate", "LastPlaybackCheckIn",
		"DeviceName", "DeviceId", "ApplicationVersion",
		"SupportsMediaControl", "SupportsRemoteControl", "SupportedCommands",
	}
	got := memberNamesInOrder(t, elements(t, listSessions(t, f, "", f.stranger.token))[0])

	if len(got) != len(want) {
		t.Fatalf("a session carries %d members %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("member %d is %s, want %s — the whole list was %v", i, got[i], want[i], got)
		}
	}
}

// TestTheSessionsHandlerRefusesToBeBuiltWithoutItsPorts is the failure-to-start
// rule of plan 7, one row per member.
//
// The whole configuration is asserted to build first. Without that line every
// row could be failing for the member the table forgot to fill rather than for
// the one it removed, which is how T14 found this same table passing vacuously.
func TestTheSessionsHandlerRefusesToBeBuiltWithoutItsPorts(t *testing.T) {
	store := openStore(t)
	clock := &settableClock{at: aTestInstant}
	whole := httpapi.SessionsHandlerConfig{
		Sessions:      store,
		Accounts:      store,
		Authenticator: newAuthenticator(t, store, clock),
		Clock:         clock,
	}
	if _, err := httpapi.NewSessionsHandler(whole); err != nil {
		t.Fatalf("the whole configuration does not build, so every row below proves nothing: %v", err)
	}

	rows := []struct {
		name   string
		remove func(*httpapi.SessionsHandlerConfig)
	}{
		{"no session store", func(c *httpapi.SessionsHandlerConfig) { c.Sessions = nil }},
		{"no account store", func(c *httpapi.SessionsHandlerConfig) { c.Accounts = nil }},
		{"no authenticator", func(c *httpapi.SessionsHandlerConfig) { c.Authenticator = nil }},
		{"no clock", func(c *httpapi.SessionsHandlerConfig) { c.Clock = nil }},
	}

	for _, row := range rows {
		configuration := whole
		row.remove(&configuration)
		if _, err := httpapi.NewSessionsHandler(configuration); err == nil {
			t.Errorf("%s built a handler, want a failure to start", row.name)
		}
	}
}

// TestTheQueryFoldStageFoldsThisRoutesNamesOnARealRequest is the arrival 001
// plan 6.2 shipped an empty map waiting for.
//
// 001 T10 built the canonicalisation stage and shipped **no** declarations, and
// its own mutation testing recorded that no request this server could answer
// exercised the stage: removing it from the chain broke nothing. These are the
// first route-keyed entries the map has ever had, so this is the first
// assertion that the stage runs on a request a client can really send.
//
// It goes through the **assembled pipeline** over the real route table and the
// real declaration set, because that is what makes it a claim about the server
// rather than about a folder built in a test.
func TestTheQueryFoldStageFoldsThisRoutesNamesOnARealRequest(t *testing.T) {
	f := newSessionsFixture(t)

	rows := []struct {
		name, query string
		want        []string
	}{
		{"the declared spelling", "?deviceId=" + adminDevice, []string{f.adminSession()}},
		{"upper case", "?DEVICEID=" + adminDevice, []string{f.adminSession()}},
		{"lower case", "?deviceid=" + adminDevice, []string{f.adminSession()}},
		{"PascalCase, which is what the reference's own clients send", "?DeviceId=" + adminDevice, []string{f.adminSession()}},
		{"activeWithinSeconds in another case", "?ACTIVEWITHINSECONDS=0", []string{f.adminSession(), f.strangerSession()}},
		{"controllableByUserId in another case", "?CONTROLLABLEBYUSERID=" + f.admin.id, nil},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			answer := throughPipeline(t, f, httpapi.V1QuerySpellings(), row.query, f.admin.token)
			if answer.status != http.StatusOK {
				t.Fatalf("status = %d, want 200 — the body was %q", answer.status, answer.body)
			}
			assertSessions(t, row.name, answer.body, row.want...)
		})
	}
}

// TestTheFoldIsWhatMakesAnotherSpellingWork is the other half of the assertion
// above: it fails, deliberately, when the declaration is removed.
//
// A stage that has never been shown to matter has proved nothing. The same
// request through a pipeline built with an **empty** declaration set — which is
// exactly what this server shipped until this change — leaves `DEVICEID` a name
// the handler does not read, and the answer is the wider list that behaviours
// 1.15 calls a wrong answer with a 200 on it.
func TestTheFoldIsWhatMakesAnotherSpellingWork(t *testing.T) {
	f := newSessionsFixture(t)

	const query = "?DEVICEID=" + adminDevice
	folded := throughPipeline(t, f, httpapi.V1QuerySpellings(), query, f.admin.token)
	unfolded := throughPipeline(t, f, httpapi.QuerySpellings{}, query, f.admin.token)

	assertSessions(t, "with the declaration", folded.body, f.adminSession())
	assertSessions(t, "without the declaration", unfolded.body, f.adminSession(), f.strangerSession())
}

// throughPipeline sends one GET /Sessions through the whole assembled pipeline
// — the response-time stamp, the Server header, the readiness gate, path
// canonicalisation, query canonicalisation and the router — rather than
// straight at the handler.
func throughPipeline(t *testing.T, f sessionsFixture, spellings httpapi.QuerySpellings, query, token string) response {
	t.Helper()

	pipeline, err := httpapi.NewPipeline(surface.V1(), spellings, func(router chi.Router) {
		router.Get("/Sessions", f.handler.Sessions())
	})
	if err != nil {
		t.Fatalf("assembling the pipeline: %v", err)
	}
	pipeline.Gate().MarkReady()

	return sessionRequest(t, pipeline, http.MethodGet, "/Sessions"+query, "", token)
}
