package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/users"
)

// visible is the policy shape a login screen shows.
//
// It is a shape rather than the default because a bare account is **hidden**:
// the reference grants PermissionKind.IsHidden to a freshly created account
// [source: Jellyfin.Data/UserEntityExtensions.cs:173 @ v10.11.11], and
// users.DefaultPolicy transcribes it. So `IsHidden = false` is what puts a user
// on a login screen, and the fixture for "every user is hidden" is the one that
// sets nothing at all.
func visible(policy *users.Policy) { policy.IsHidden = false }

// getPublic sends one GET /Users/Public and reads the whole answer.
//
// It goes over a real server for the reason post does: a declared
// Content-Length and an absent Content-Type are part of a response's shape and
// httptest.ResponseRecorder can express neither.
func getPublic(t *testing.T, handler *httpapi.UsersHandler, headers ...string) response {
	t.Helper()

	if len(headers)%2 != 0 {
		t.Fatalf("getPublic was given %d header parts, which is not a whole number of name/value pairs", len(headers))
	}

	server := httptest.NewServer(handler.PublicUsers())
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/Users/Public", nil)
	if err != nil {
		t.Fatalf("building the request: %v", err)
	}
	for i := 0; i < len(headers); i += 2 {
		request.Header.Set(headers[i], headers[i+1])
	}

	answer, err := server.Client().Do(request)
	if err != nil {
		t.Fatalf("sending the request: %v", err)
	}
	defer answer.Body.Close()

	read := make([]byte, 0, 4096)
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

// publicUsers is getPublic plus the one thing every caller does with it: it
// insists on a 200 and hands back the array elements **as their own bytes**.
//
// json.RawMessage keeps the bytes the server sent, in the order it sent them,
// which is what every assertion in this file needs — a decode into a struct
// would answer the same value for four different documents (spec 3.5's
// absences, an explicit null, and a reordering are all invisible once parsed).
func publicUsers(t *testing.T, handler *httpapi.UsersHandler, headers ...string) []json.RawMessage {
	t.Helper()

	answer := getPublic(t, handler, headers...)
	if answer.status != http.StatusOK {
		t.Fatalf("GET /Users/Public answered %d and the body %q, want 200", answer.status, answer.body)
	}

	var listed []json.RawMessage
	if err := json.Unmarshal([]byte(answer.body), &listed); err != nil {
		t.Fatalf("reading the array %q: %v", answer.body, err)
	}
	return listed
}

// members reads one JSON object's member names **in the order the bytes carry
// them**, with each member's raw value.
//
// The technique is T2's (internal/users' policy_test.go) and it is here for the
// reason that task's handoff gives: a map comparison passes on a reordered
// model, and key order is exactly what spec 3.5 records having got wrong once.
// json.Decoder's token stream is what preserves it; each value is consumed
// whole with Decode, so a nested object's own members are never mistaken for
// this one's.
func members(t *testing.T, document []byte) ([]string, map[string]json.RawMessage) {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(document))
	open, err := decoder.Token()
	if err != nil {
		t.Fatalf("reading the document: %v", err)
	}
	if delimiter, ok := open.(json.Delim); !ok || delimiter != '{' {
		t.Fatalf("the document is not a JSON object: %s", document)
	}

	var names []string
	values := make(map[string]json.RawMessage)
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading the document: %v", err)
		}
		name, ok := token.(string)
		if !ok {
			t.Fatalf("expected a member name and read %v", token)
		}
		names = append(names, name)

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("reading the value of %s: %v", name, err)
		}
		values[name] = value
	}
	return names, values
}

// memberNamesInOrder is members' first half, for the assertions that only care
// about order and absence.
func memberNamesInOrder(t *testing.T, document []byte) []string {
	t.Helper()
	names, _ := members(t, document)
	return names
}

// TestPublicUsersIsByteIdenticalToTheAuthenticatedReadingOfTheSameUser is
// spec 3.4's measured equality, asserted as an equality.
//
// The criterion is deliberately not two independent shape checks over the two
// bodies. Two such checks drift the same way — a member added to the model
// appears on both sides and neither notices — and what spec 3.4 measured is
// that the *bytes* agree
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. One
// comparison of one pair of documents is the whole assertion, and it is what
// fails the day somebody writes a second filler (plan 6.6).
//
// The authenticated reading here is the `User` member of the authentication
// result, which is the only authenticated route returning a user object until
// T14 lands /Users/Me and /Users/{userId}. The public read happens *after* the
// login on purpose: it is the harder direction, because the login stamps
// last_login_at and a body built from a stale read of the account would differ
// on exactly that member.
func TestPublicUsersIsByteIdenticalToTheAuthenticatedReadingOfTheSameUser(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", visible)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	login := authenticate(t, handler, "Ada", "correct horse")
	if login.status != http.StatusOK {
		t.Fatalf("authenticating answered %d and the body %q, want 200", login.status, login.body)
	}
	var result struct {
		User json.RawMessage
	}
	if err := json.Unmarshal([]byte(login.body), &result); err != nil {
		t.Fatalf("reading the authentication result %q: %v", login.body, err)
	}

	listed := publicUsers(t, handler)
	if len(listed) != 1 {
		t.Fatalf("GET /Users/Public answered %d users, want 1", len(listed))
	}

	if !bytes.Equal(listed[0], result.User) {
		t.Errorf("/Users/Public and the authenticated reading of the same account are different bytes,"+
			" and spec 3.4 measured them identical:\npublic: %s\n  auth: %s", listed[0], result.User)
	}
}

// TestPublicUsersReadsNoCredential is plan 6.2's row for this route.
//
// Three requests — no credential, a token that resolves to a live session, and
// a token that resolves to nothing — must answer the same bytes. The middle one
// is the one that matters: the reference answers an authenticated caller
// exactly what it answers a stranger, and the cheapest way to be sure of that
// is a handler with no branch on the credential at all.
//
// The third request is worth its line too. A token that is not a token is a
// `401` on every other route in this feature; here it is not read, so it cannot
// refuse, and a handler that had started validating credentials would fail here
// before it failed anywhere a client would notice.
//
// # The fixture carries a hidden account, and that is the load-bearing half
//
// An installation whose only account is visible cannot tell these three
// requests apart whatever the handler does with the credential: every plausible
// wrong build answers the same one user. The mistake worth catching is the
// generous one — *an authenticated caller may see the accounts a login screen
// hides* — and it is invisible until there is an account to hide. A mutation
// making the exclusion conditional on the credential passed this file until the
// hidden account was in it.
func TestPublicUsersReadsNoCredential(t *testing.T) {
	store := openStore(t)
	id := createAccount(t, store, "Ada", visible)
	createAccount(t, store, "Bob", nil)
	openSessionFor(t, store, id, "Atrium Test", "device-1", "a-live-token", aTestInstant)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	anonymous := getPublic(t, handler)
	authenticated := getPublic(t, handler, httpapi.EmbyTokenHeader, "a-live-token")
	nonsense := getPublic(t, handler, httpapi.EmbyTokenHeader, "not-a-token")

	for _, answer := range []struct {
		what string
		got  response
	}{
		{"no credential", anonymous},
		{"a live token", authenticated},
		{"a token naming no session", nonsense},
	} {
		if answer.got.status != http.StatusOK {
			t.Fatalf("GET /Users/Public with %s answered %d and the body %q, want 200",
				answer.what, answer.got.status, answer.got.body)
		}
	}

	if authenticated.body != anonymous.body {
		t.Errorf("a caller holding a live token is answered different bytes from a stranger,"+
			" and this route reads no credential:\n  token: %s\nno token: %s", authenticated.body, anonymous.body)
	}
	if nonsense.body != anonymous.body {
		t.Errorf("a caller presenting a token naming no session is answered different bytes from a stranger,"+
			" and this route reads no credential:\nnonsense: %s\nno token: %s", nonsense.body, anonymous.body)
	}
}

// TestAHiddenUserIsExcludedAndTheOthersTravelWhole is AC-6's first two clauses.
//
// The two counts are the assertion that the excluded user is the only thing
// missing. spec 3.5's `Configuration` and `Policy` are **sent by
// /Users/Public too** — this specification asserted the opposite until it was
// measured, on the sound-sounding premise that a pre-authentication route must
// not disclose what a user is allowed to do — so a body carrying the right
// names and an empty policy would pass every check but this one. The reference
// sends 42 of the 44 policy properties it declares and 16 configuration
// properties
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
func TestAHiddenUserIsExcludedAndTheOthersTravelWhole(t *testing.T) {
	store := openStore(t)
	createAccount(t, store, "Ada", visible)
	createAccount(t, store, "Bob", nil) // A bare account is hidden.
	createAccount(t, store, "Cleo", visible)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	listed := publicUsers(t, handler)
	if len(listed) != 2 {
		t.Fatalf("GET /Users/Public answered %d users over an installation with one hidden account of three, want 2:\n%s",
			len(listed), listed)
	}

	var names []string
	for _, element := range listed {
		_, values := members(t, element)

		var name string
		if err := json.Unmarshal(values["Name"], &name); err != nil {
			t.Fatalf("reading the Name of %s: %v", element, err)
		}
		names = append(names, name)

		if got := len(memberNamesInOrder(t, values["Policy"])); got != 42 {
			t.Errorf("%s travels with %d policy properties, want 42 — the whole object is what spec 3.4 measured", name, got)
		}
		if got := len(memberNamesInOrder(t, values["Configuration"])); got != 16 {
			t.Errorf("%s travels with %d configuration properties, want 16", name, got)
		}
	}

	if want := []string{"Ada", "Cleo"}; strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("GET /Users/Public answered %v, want %v — the hidden account is the one excluded and the order is the store's", names, want)
	}
}

// TestAnInstallationWhereEveryUserIsHiddenAnswersAnEmptyArray is AC-6's third
// clause, and the assertion is on the **bytes**.
//
// `[]` and `null` are one value once parsed and two documents on the wire, and
// internal/wire spells a nil slice as `null` — which is why the handler builds
// the slice non-nil and why this test compares the body as a string. A client
// reading `null` where the reference sends an empty array is a crash, not a
// cosmetic difference.
//
// The fixture is one line because a bare account is hidden: plan 6.9's
// amendment records that `--hidden` alone is a no-op at the provisioning
// command for the same reason.
func TestAnInstallationWhereEveryUserIsHiddenAnswersAnEmptyArray(t *testing.T) {
	store := openStore(t)
	createAccount(t, store, "Ada", nil)
	createAccount(t, store, "Bob", nil)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := getPublic(t, handler)
	if answer.status != http.StatusOK {
		t.Fatalf("GET /Users/Public over an installation of hidden users answered %d, want 200", answer.status)
	}
	if answer.body != "[]" {
		t.Errorf("GET /Users/Public answered %q over an installation where every user is hidden, want []", answer.body)
	}
}

// TestAnInstallationWithNoAccountsAnswersAnEmptyArray is the same document from
// the other direction: the empty array must not depend on there being rows to
// filter.
func TestAnInstallationWithNoAccountsAnswersAnEmptyArray(t *testing.T) {
	store := openStore(t)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	answer := getPublic(t, handler)
	if answer.status != http.StatusOK {
		t.Fatalf("GET /Users/Public over an empty installation answered %d, want 200", answer.status)
	}
	if answer.body != "[]" {
		t.Errorf("GET /Users/Public answered %q over an installation with no accounts, want []", answer.body)
	}
}

// TestLastLoginDateIsAbsentBeforeAFirstLoginAndPresentAfter is the
// NULL-versus-zero distinction, asserted in both directions because one
// direction proves nothing.
//
// A non-pointer date would answer `0001-01-01T00:00:00.0000000Z` for an account
// that has never logged in, which is .NET's minimum date and a **value** where
// the reference sends no member at all (spec 3.5). It is the exact opposite of
// SessionInfo.LastPlaybackCheckIn, where the zero tick *is* what the reference
// sends for a session that has never played anything, and getting the two the
// wrong way round is the mistake this pair of assertions exists to catch. The
// absent half is asserted twice — the member is missing, and the minimum date's
// bytes appear nowhere in the body — because a `null` member would satisfy the
// first check on its own and is a third document again.
func TestLastLoginDateIsAbsentBeforeAFirstLoginAndPresentAfter(t *testing.T) {
	store := openStore(t)
	createAccountWithPassword(t, store, "Ada", "correct horse", visible)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	before := publicUsers(t, handler)
	if len(before) != 1 {
		t.Fatalf("GET /Users/Public answered %d users, want 1", len(before))
	}
	if names := memberNamesInOrder(t, before[0]); slices.Contains(names, "LastLoginDate") {
		t.Errorf("an account that has never logged in carries LastLoginDate: %s", before[0])
	}
	if strings.Contains(string(before[0]), "0001-01-01T00:00:00.0000000Z") {
		t.Errorf("an account that has never logged in carries the minimum date somewhere,"+
			" which is a value where the reference sends no member: %s", before[0])
	}

	if answer := authenticate(t, handler, "Ada", "correct horse"); answer.status != http.StatusOK {
		t.Fatalf("authenticating answered %d and the body %q, want 200", answer.status, answer.body)
	}

	after := publicUsers(t, handler)
	if len(after) != 1 {
		t.Fatalf("GET /Users/Public answered %d users after a login, want 1", len(after))
	}
	_, values := members(t, after[0])
	got, present := values["LastLoginDate"]
	if !present {
		t.Fatalf("an account that has logged in carries no LastLoginDate: %s", after[0])
	}
	if want := `"2026-09-03T10:30:00.0000000Z"`; string(got) != want {
		t.Errorf("LastLoginDate is %s after a login at the test instant, want %s", got, want)
	}
}

// TestTheUserObjectSendsNoImageMembers is spec 3.5's two conditional members.
//
// Both are conditioned on the user having an avatar, and v1 gives a user no way
// to have one (plan 6.6) — 006 owns the day that changes. They are asserted
// absent rather than null because behaviours 1.7 omits a null globally, so a
// model that sent `"PrimaryImageTag": null` would be a delta on a route the
// video client reads before anybody has logged in.
func TestTheUserObjectSendsNoImageMembers(t *testing.T) {
	store := openStore(t)
	createAccount(t, store, "Ada", visible)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	listed := publicUsers(t, handler)
	if len(listed) != 1 {
		t.Fatalf("GET /Users/Public answered %d users, want 1", len(listed))
	}

	names := memberNamesInOrder(t, listed[0])
	for _, absent := range []string{"PrimaryImageTag", "PrimaryImageAspectRatio"} {
		if slices.Contains(names, absent) {
			t.Errorf("the user object carries %s, and no account in v1 has an avatar: %s", absent, listed[0])
		}
	}
}

// TestServerIdIsWrittenBeforeId asserts the key order of the encoded bytes.
//
// spec 3.5 records that this project's own table had `ServerId` and `Id` the
// other way round until the route was measured
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], and a
// member-by-member assertion cannot see a transposition: both spellings, both
// values and both types are identical under any check that reads the parsed
// object. The order is contract because L3 compares bytes.
//
// The whole list is asserted rather than the one pair, so that a member added
// in the wrong place or an absence that stopped being absent fails here too.
// The expected list is the reference's declaration order
// [source: MediaBrowser.Model/Dto/UserDto.cs:26-105 @ v10.11.11] with the five
// members v1 never sends removed: `ServerName` and `PrimaryImageAspectRatio`
// are null on every account this binary can hold, the two image members need an
// avatar, and the two dates are NULL columns on an account that has neither
// logged in nor been stamped.
func TestServerIdIsWrittenBeforeId(t *testing.T) {
	store := openStore(t)
	createAccount(t, store, "Ada", visible)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	listed := publicUsers(t, handler)
	if len(listed) != 1 {
		t.Fatalf("GET /Users/Public answered %d users, want 1", len(listed))
	}

	got := memberNamesInOrder(t, listed[0])
	want := []string{
		"Name",
		"ServerId",
		"Id",
		"HasPassword",
		"HasConfiguredPassword",
		"HasConfiguredEasyPassword",
		"EnableAutoLogin",
		"Configuration",
		"Policy",
	}
	if len(got) != len(want) {
		t.Fatalf("the user object carries %d members, want %d:\n got: %s\nwant: %s",
			len(got), len(want), strings.Join(got, ", "), strings.Join(want, ", "))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("the user object writes %q at position %d, want %q — declaration order is wire order",
				got[i], i, want[i])
		}
	}

	server, identifier := slices.Index(got, "ServerId"), slices.Index(got, "Id")
	if server > identifier {
		t.Errorf("ServerId is written at position %d and Id at %d; the reference writes ServerId first"+
			" and this project's own table had the two the other way round until it was measured", server, identifier)
	}
}

// TestADisabledUserIsListedHereAndTheReferencesSourceExcludesIt is a divergence
// written as a test rather than as a comment.
//
// spec 3.4 excludes users "flagged hidden from login screens" and names no
// other exclusion, and the measurement behind it is of the hidden flag
// `[prior-probe: Jellyfin 10.11.11, 2026-06-13]`. The reference's own source
// excludes disabled accounts as well, in the same call
// [source: Jellyfin.Api/Controllers/UserController.cs:114-117,625-633 @ v10.11.11],
// and that reading has never been measured against a running server. So the
// specification is implemented as written and the difference is asserted, which
// is what makes the day a probe lands a **failing test naming the behaviour
// that moved** instead of a rediscovery. plan 6.2 carries the item and the
// register owes it a row.
func TestADisabledUserIsListedHereAndTheReferencesSourceExcludesIt(t *testing.T) {
	store := openStore(t)
	createAccount(t, store, "Ada", func(policy *users.Policy) {
		policy.IsHidden = false
		policy.IsDisabled = true
	})
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	listed := publicUsers(t, handler)
	if len(listed) != 1 {
		t.Fatalf("GET /Users/Public answered %d users over an installation whose one account is disabled and not hidden,"+
			" want 1 — spec 3.4 excludes the hidden and says nothing about the disabled:\n%s", len(listed), listed)
	}
}

// TestAStoreThatCannotListAccountsIsAFiveHundred is plan 7's rule at this
// route: a failure to read is a server fault and never an empty login screen.
//
// The alternative a handler falls into by accident is worse than it looks — a
// store error swallowed into an empty array is a `200` telling every client
// that this installation has no users, which is indistinguishable from a real
// answer and which the video client would render as a login screen with nothing
// on it.
func TestAStoreThatCannotListAccountsIsAFiveHundred(t *testing.T) {
	store := openStore(t)
	clock := &settableClock{at: aTestInstant}
	handler, err := httpapi.NewUsersHandler(httpapi.UsersHandlerConfig{
		InstallationID: testInstallationID,
		Login:          users.NewLogin(store, clock),
		Accounts:       unreadableAccounts{},
		Sessions:       store,
		Clock:          clock,
	})
	if err != nil {
		t.Fatalf("building the users handler: %v", err)
	}

	answer := getPublic(t, handler)
	if answer.status != http.StatusInternalServerError {
		t.Errorf("GET /Users/Public over a store that cannot be read answered %d and the body %q, want 500",
			answer.status, answer.body)
	}
	if answer.body != "" {
		t.Errorf("the failure answered the body %q, and a fault carries none", answer.body)
	}
}

// unreadableAccounts is a ports.UserStore whose account list cannot be read.
//
// The interface is embedded rather than implemented so that this fake declares
// exactly the method the test is about: any other call is a nil dereference
// naming itself, rather than a plausible zero value the test would pass over.
type unreadableAccounts struct{ ports.UserStore }

func (unreadableAccounts) Users(context.Context) ([]ports.User, error) {
	return nil, errors.New("the accounts cannot be read")
}
