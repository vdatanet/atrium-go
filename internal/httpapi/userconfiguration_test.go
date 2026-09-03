package httpapi_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/store/sqlite"
)

// postConfiguration sends one POST /Users/Configuration and reads the whole
// answer, headers included.
//
// It goes over a real server for the reason post and getUser do — an absent
// Content-Type is part of this response's shape and httptest.ResponseRecorder
// cannot express one — and it hands the header map back as well as the four
// fields of response, because a `204` is measured by what it does **not**
// carry: net/http drops a Content-Length from a 204 whether or not a handler
// set one [measurement: net/http, Go 1.27.0, 2026-09-03], so the assertion that
// this shape declares no length is an assertion about a header being absent
// rather than about a number.
func postConfiguration(t *testing.T, handler *httpapi.UsersHandler, body string, headers ...string) (response, http.Header) {
	t.Helper()
	return postConfigurationWithQuery(t, handler, "", body, headers...)
}

// postConfigurationWithQuery is postConfiguration for the one case that carries
// a query: U-14's `userId`, which spec 3.6 does not declare and this route
// therefore never reads. query is the string after the `?`, empty for none.
func postConfigurationWithQuery(t *testing.T, handler *httpapi.UsersHandler, query, body string, headers ...string) (response, http.Header) {
	t.Helper()

	if len(headers)%2 != 0 {
		t.Fatalf("postConfiguration was given %d header parts, which is not a whole number of name/value pairs", len(headers))
	}

	server := httptest.NewServer(userRoutes(handler))
	defer server.Close()

	target := server.URL + "/Users/Configuration"
	if query != "" {
		target += "?" + query
	}
	request, err := http.NewRequest(http.MethodPost, target, strings.NewReader(body))
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
	}, answer.Header
}

// replaceConfiguration is postConfiguration plus the one thing every round-trip
// caller does with it: it insists on the `204`.
func replaceConfiguration(t *testing.T, handler *httpapi.UsersHandler, token, body string) {
	t.Helper()

	answer, _ := postConfiguration(t, handler, body, httpapi.EmbyTokenHeader, token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("POST /Users/Configuration answered %d and the body %q, want 204", answer.status, answer.body)
	}
}

// configurationOf reads one account's `Configuration` member **as its own
// bytes**, through GET /Users/Me or GET /Users/{userId}.
//
// json.RawMessage rather than a decode, for the reason spec 3.5's assertions
// give: an absent member, an explicit null and a reordering are all invisible
// once a document is parsed, and this route's whole claim is that what was
// posted comes back unchanged.
func configurationOf(t *testing.T, handler *httpapi.UsersHandler, path, token string) []byte {
	t.Helper()

	_, values := members(t, []byte(readUser(t, handler, path, token)))
	configuration, present := values["Configuration"]
	if !present {
		t.Fatalf("the user object read from %s carries no Configuration member", path)
	}
	return configuration
}

// configurationProperties is spec 3.6's sixteen, in the reference's declaration
// order [source: MediaBrowser.Model/Configuration/UserConfiguration.cs:35-76 @
// v10.11.11], each with a value that is **not** the default.
//
// Not the default is the whole point of the table. users.DefaultConfiguration
// is what a posted document is decoded over, so a handler that dropped the body
// on the floor and stored the defaults would round-trip perfectly against a
// fixture whose values happened to be them — which is 002 T13's finding twice
// over: an equality proves nothing over data with only one possible answer.
//
// The two language preferences differ from each other for the same reason. A
// build that assigned one from the other would agree with itself.
var configurationProperties = []struct {
	name string

	// value is the JSON this test posts and the JSON it expects to read back.
	// One string for both halves, because "unchanged" is the claim: they are
	// written compactly so that they are the encoder's own bytes.
	value string
}{
	{"AudioLanguagePreference", `"eng"`},
	{"PlayDefaultAudioTrack", `false`},
	{"SubtitleLanguagePreference", `"fra"`},
	{"DisplayMissingEpisodes", `true`},
	{"GroupedFolders", `["0123456789abcdef0123456789abcdef"]`},
	{"SubtitleMode", `"Always"`},
	{"DisplayCollectionsView", `true`},
	{"EnableLocalPassword", `true`},
	{"OrderedViews", `["11111111111111111111111111111111","22222222222222222222222222222222"]`},
	{"LatestItemsExcludes", `["33333333333333333333333333333333"]`},
	{"MyMediaExcludes", `["44444444444444444444444444444444"]`},
	{"HidePlayedInLatest", `false`},
	{"RememberAudioSelections", `false`},
	{"RememberSubtitleSelections", `false`},
	{"EnableNextEpisodeAutoPlay", `false`},
	{"CastReceiverId", `"atrium-cast"`},
}

// wholeConfigurationBody is the sixteen properties as a request body, in
// **reverse** declaration order.
//
// Reversed deliberately. The order the sixteen come back in is the model's
// declaration order and is contract under L3 (plan 6.6), and a body posted in
// that same order would let a handler that echoed the client's document pass
// the order assertion. Sent backwards, the read-back order is the server's own
// or it is wrong.
func wholeConfigurationBody() string {
	pairs := make([]string, 0, len(configurationProperties))
	for i := len(configurationProperties) - 1; i >= 0; i-- {
		property := configurationProperties[i]
		pairs = append(pairs, `"`+property.name+`":`+property.value)
	}
	return "{" + strings.Join(pairs, ",") + "}"
}

// configurationFixture is one installation with two accounts that can both ask
// and both be read: the caller, and the account U-14's request names.
type configurationFixture struct {
	store   *sqlite.Store
	handler *httpapi.UsersHandler
	caller  occupant

	// other is a second seat rather than a subject nobody can log in as,
	// because U-14's assertion is made on **both** accounts: that the caller's
	// configuration changed and that the named user's did not. The second half
	// is only an assertion if the second account can be read, and reading it
	// through its own /Users/Me as well as through the caller's
	// /Users/{userId} is what makes it one whether or not this route ever
	// learns to redact.
	other occupant
}

func newConfigurationFixture(t *testing.T) configurationFixture {
	t.Helper()

	store := openStore(t)
	handler := newUsersHandler(t, store, &settableClock{at: aTestInstant})

	return configurationFixture{
		store:   store,
		handler: handler,
		caller:  seat(t, store, handler, "Ada", "correct horse", "device-ada", ordinary),
		other:   seat(t, store, handler, "Bob", "battery staple", "device-bob", ordinary),
	}
}

// TestEverySixteenPropertyPostedComesBackThroughUsersMe is AC-8, asserted
// property by property and as an order.
//
// spec 3.6 sends sixteen properties and v1 acts on three of them; the criterion
// is that every one round-trips, **including the ones v1 does not act on**,
// because a client that sets a preference and reads it back is the only thing
// that ever observes the thirteen inert ones. Storing them is the feature.
//
// Two assertions, and they fail on different wrong handlers. The values catch a
// handler that dropped, defaulted or transposed a property; the **order**
// catches a model whose declaration drifted from the reference's, which no
// value comparison can see and which is contract under L3 (plan 6.6). The body
// is posted backwards so that the order assertion is about this server and not
// about the request.
func TestEverySixteenPropertyPostedComesBackThroughUsersMe(t *testing.T) {
	fixture := newConfigurationFixture(t)

	replaceConfiguration(t, fixture.handler, fixture.caller.token, wholeConfigurationBody())

	names, values := members(t, configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token))

	want := make([]string, 0, len(configurationProperties))
	for _, property := range configurationProperties {
		want = append(want, property.name)
	}
	if !slices.Equal(names, want) {
		t.Errorf("the configuration read back carries\n%v\nand spec 3.6's sixteen in the reference's declaration order are\n%v",
			names, want)
	}

	for _, property := range configurationProperties {
		if got := string(values[property.name]); got != property.value {
			t.Errorf("%s was posted as %s and came back as %s", property.name, property.value, got)
		}
	}
}

// TestASecondPostReplacesTheConfigurationRatherThanMergingIntoIt is the
// assertion the round trip above cannot make.
//
// A handler that decoded the posted body over the **stored** document instead
// of over the reference's defaults passes every round-trip test ever written:
// everything posted is still there afterwards. What it gets wrong is what was
// *not* posted, and the reference is unambiguous — its binder constructs a
// fresh UserConfiguration and the update assigns fifteen of the sixteen from it
// unconditionally
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:760-799 @ v10.11.11],
// so an omitted property returns to its default.
//
// The comparison is with a **freshly provisioned** account's configuration, and
// not with a transcribed list of defaults: what the second post must produce is
// the state a client would get from an account that had never been posted to,
// and asserting that as bytes is the only spelling of it that cannot drift from
// users.DefaultConfiguration.
func TestASecondPostReplacesTheConfigurationRatherThanMergingIntoIt(t *testing.T) {
	fixture := newConfigurationFixture(t)

	untouched := configurationOf(t, fixture.handler, "/Users/"+fixture.other.id, fixture.caller.token)

	replaceConfiguration(t, fixture.handler, fixture.caller.token, wholeConfigurationBody())
	replaceConfiguration(t, fixture.handler, fixture.caller.token, `{"DisplayMissingEpisodes":true}`)

	_, values := members(t, configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token))
	if got := string(values["DisplayMissingEpisodes"]); got != "true" {
		t.Errorf("the one property the second post carried came back as %s, want true", got)
	}

	_, defaults := members(t, untouched)
	for _, property := range configurationProperties {
		if property.name == "DisplayMissingEpisodes" {
			continue
		}
		if got, want := string(values[property.name]), string(defaults[property.name]); got != want {
			t.Errorf("%s was set to %s by the first post and the second post did not carry it, so it should have returned to %s; it is %s",
				property.name, property.value, want, got)
		}
	}
}

// TestAnUnknownPropertyIsAcceptedAndDroppedWhileTheDeclaredOnesSurvive is
// spec 3.6's "unknown properties are ignored, not rejected", at the wire.
//
// Both halves are the assertion. **Accepted** is the `204`: a handler that
// refused the document would break every client that posts a property a later
// Jellyfin added. **Dropped** is the read-back: the unknown name is not in the
// answer, and the declared properties beside it in the same body are.
//
// This is deliberately the opposite of what a session's capabilities do.
// behaviours 5.9 records an unknown capabilities property **surviving** into
// /Sessions as a declared divergence; there is no such divergence here, and
// internal/users' own
// TestAnUnknownPropertyInAStoredConfigurationIsDroppedAndTheDeclaredOnesSurvive
// is the same claim one layer down. A reader who generalises from either
// route to the other gets the wrong answer on both.
func TestAnUnknownPropertyIsAcceptedAndDroppedWhileTheDeclaredOnesSurvive(t *testing.T) {
	fixture := newConfigurationFixture(t)

	answer, _ := postConfiguration(t, fixture.handler,
		`{"SubtitleMode":"Always","NoSuchPreference":"and yet","DisplayCollectionsView":true}`,
		httpapi.EmbyTokenHeader, fixture.caller.token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("a body carrying an unknown property answered %d and the body %q; spec 3.6 ignores it rather than rejecting it",
			answer.status, answer.body)
	}

	names, values := members(t, configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token))
	if slices.Contains(names, "NoSuchPreference") {
		t.Errorf("the unknown property survived into the user object: %v", names)
	}
	if len(names) != len(configurationProperties) {
		t.Errorf("the configuration carries %d members and spec 3.6 declares %d: %v", len(names), len(configurationProperties), names)
	}
	if got := string(values["SubtitleMode"]); got != `"Always"` {
		t.Errorf(`SubtitleMode came back as %s, want "Always" — a declared property beside an unknown one must survive`, got)
	}
	if got := string(values["DisplayCollectionsView"]); got != "true" {
		t.Errorf("DisplayCollectionsView came back as %s, want true", got)
	}
}

// TestTheStoredDocumentIsTheDeclaredSixteenAndNotTheBytesThatWerePosted is the
// assertion no request can make, and it is here because a mutation proved it.
//
// Handing ports.UserStore.ReplaceConfiguration the **posted bytes** rather than
// the re-encoded document passes every test above. It has to: the read side
// decodes the stored document over the reference's defaults and drops unknown
// properties there too (plan 6.6), so the normalisation happens twice and
// removing the first one changes no byte on the wire. What it changes is what
// is in the column — the port's own contract says the document it is handed has
// already been decoded and re-encoded, and an unknown property that reaches the
// column is a property some later reader has to remember to drop.
//
// This is 002 T7's finding in the same shape: an assertion phrased in the
// vocabulary the wire uses cannot see state the wire does not carry. So this
// one is phrased in the store's, and it is the only test in the file that reads
// anything but a response.
func TestTheStoredDocumentIsTheDeclaredSixteenAndNotTheBytesThatWerePosted(t *testing.T) {
	fixture := newConfigurationFixture(t)

	replaceConfiguration(t, fixture.handler, fixture.caller.token,
		`{"SubtitleMode":"Always","NoSuchPreference":"and yet"}`)

	account, found, err := fixture.store.UserByID(context.Background(), fixture.caller.id)
	if err != nil {
		t.Fatalf("reading the account back: %v", err)
	}
	if !found {
		t.Fatalf("the account %s is gone", fixture.caller.id)
	}

	names := memberNamesInOrder(t, account.ConfigurationDocument)
	want := make([]string, 0, len(configurationProperties))
	for _, property := range configurationProperties {
		want = append(want, property.name)
	}
	if !slices.Equal(names, want) {
		t.Errorf("the stored configuration document carries\n%v\nand the sixteen declared properties are\n%v",
			names, want)
	}
}

// TestTheAnswerIsA204WithNoBodyAndNoContentType is the shape of the success.
//
// Four things, three of which are invisible to a test that reads the status.
// The content type is the one with teeth: net/http keeps a Content-Type an
// earlier stage left in the header map even on a 204
// [measurement: net/http, Go 1.27.0, 2026-09-03], so the absence is this
// project's Del rather than the runtime's good manners. The length is asserted
// as an **absent header** rather than as a zero, because that same measurement
// found net/http dropping a declared one from a 204 — a test asserting
// `Content-Length: 0` here would be asserting something no handler could
// change.
func TestTheAnswerIsA204WithNoBodyAndNoContentType(t *testing.T) {
	fixture := newConfigurationFixture(t)

	answer, header := postConfiguration(t, fixture.handler, wholeConfigurationBody(),
		httpapi.EmbyTokenHeader, fixture.caller.token)

	if answer.status != http.StatusNoContent {
		t.Errorf("the answer is %d, want 204", answer.status)
	}
	if answer.body != "" {
		t.Errorf("the answer carried the body %q, and this shape has none", answer.body)
	}
	if answer.contentType != "" {
		t.Errorf("the answer declared the content type %q, and this shape declares none", answer.contentType)
	}
	if _, declared := header["Content-Length"]; declared {
		t.Errorf("the answer declared Content-Length: %v, and a 204 carries no length", header["Content-Length"])
	}
}

// TestNoCredentialIsTheEmptyUnauthorizedAndWritesNothing is spec 3.6's `401`.
//
// The refusal is the empty shape of behaviours 1.11, which is four things and
// three of them are invisible to a status check. And the second half is what
// makes this route's refusal different from a read's: a write refused after it
// had already written is a refusal a client believes and a server has already
// acted on, so the assertion is that the account's configuration is the same
// bytes afterwards as before.
func TestNoCredentialIsTheEmptyUnauthorizedAndWritesNothing(t *testing.T) {
	fixture := newConfigurationFixture(t)

	before := configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token)

	answer, _ := postConfiguration(t, fixture.handler, wholeConfigurationBody())
	if answer.status != http.StatusUnauthorized {
		t.Errorf("a request with no credential answered %d and the body %q, want 401", answer.status, answer.body)
	}
	if answer.body != "" {
		t.Errorf("the refusal carried the body %q, and this shape has none", answer.body)
	}
	if answer.contentType != "" {
		t.Errorf("the refusal declared the content type %q, and this shape declares none", answer.contentType)
	}
	if answer.length != 0 {
		t.Errorf("the refusal declared a length of %d, want 0", answer.length)
	}

	after := configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token)
	if string(before) != string(after) {
		t.Errorf("a refused request changed the configuration:\nbefore: %s\n after: %s", before, after)
	}
}

// TestAUserIdNamingSomebodyElseUpdatesTheCallersOwnConfiguration encodes
// spec 3.6 **against the reference's own source**, which is the point of it.
//
// spec 3.6 names no parameter and says "the authenticated user's
// configuration". The reference declares `[FromQuery] Guid? userId`, defaults
// it to the caller's identifier and updates the account it names
// [source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11], so
// an administrator naming somebody else changes that user there and changes
// **their own** here. behaviours 1.12 is what makes ignoring an unrecognised
// query value the specified behaviour, and U-14 in
// docs/compatibility/reference-target.md is the register row: the first one
// where what this project does is not a different answer but a different act —
// a silent write to the wrong account rather than a status a client can branch
// on.
//
// It is written as a test rather than as a comment for the reason T13 and T14
// wrote their divergences as tests: the day somebody sends this request to a
// running Jellyfin, this is a **failing test naming the behaviour that moved**
// instead of a rediscovery. If the measurement agrees with the source, the fix
// is spec 3.6 and this test moves with it; if it agrees with this, U-14 closes.
//
// Both accounts are asserted, because either half alone passes a wrong build. A
// handler that honoured the parameter would leave the caller's configuration
// untouched **and** change the named user's, and a test that only checked one
// of the two would call one of those a pass.
func TestAUserIdNamingSomebodyElseUpdatesTheCallersOwnConfiguration(t *testing.T) {
	fixture := newConfigurationFixture(t)

	namedBefore := configurationOf(t, fixture.handler, "/Users/Me", fixture.other.token)

	answer, _ := postConfigurationWithQuery(t, fixture.handler,
		theUserIdQuery+"="+fixture.other.id,
		`{"AudioLanguagePreference":"deu"}`,
		httpapi.EmbyTokenHeader, fixture.caller.token)
	if answer.status != http.StatusNoContent {
		t.Fatalf("a request naming another user answered %d and the body %q; behaviours 1.12 ignores the value rather than refusing it",
			answer.status, answer.body)
	}

	_, callers := members(t, configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token))
	if got := string(callers["AudioLanguagePreference"]); got != `"deu"` {
		t.Errorf(`the caller's own AudioLanguagePreference is %s, want "deu" — spec 3.6 replaces the authenticated user's configuration`, got)
	}

	namedAfter := configurationOf(t, fixture.handler, "/Users/Me", fixture.other.token)
	if string(namedBefore) != string(namedAfter) {
		t.Errorf("the account named by userId was changed, and spec 3.6 leaves it alone"+
			" — the reference updates it [source: Jellyfin.Api/Controllers/UserController.cs:488-511 @ v10.11.11],"+
			" so if this failure is a measurement rather than a mutation, U-14 has its answer:\nbefore: %s\n after: %s",
			namedBefore, namedAfter)
	}
}

// theUserIdQuery is the parameter U-14 is about, spelled the way the reference
// declares it.
//
// It is the reference's spelling `[source: Jellyfin.Api/Controllers/UserController.cs:493 @ v10.11.11]`
// and it appears nowhere in this server's own code, which is the finding rather
// than an omission: nothing here reads a query on this route, so there is no
// constant in the handler for this one to be compared against.
const theUserIdQuery = "userId"

// TestACastReceiverIdTheInstallationDoesNotDeclareIsStoredHere is the second
// place this route's specification and the reference's source disagree, and it
// is recorded the same way as the first.
//
// Fifteen of the sixteen properties are assigned unconditionally by the
// reference's update. CastReceiverId is not: it is kept only when the posted
// value is non-empty **and** names a cast receiver application the server's own
// configuration declares, so an unknown one leaves the stored value alone
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:785-789 @ v10.11.11].
//
// Atrium declares no cast receiver applications at all — 001 answers
// /System/Info with an empty CastReceiverApplications — so replicating the
// condition would mean discarding every value this route is ever sent, and
// spec 3.6 stores and returns every property faithfully. It is implemented as
// specified and asserted here, so that the day the reference is asked what it
// does with this request the answer arrives as a failing test. T2 already
// records CastReceiverId as the one member of this model whose **value** cannot
// match; this is the same member from the writing side.
func TestACastReceiverIdTheInstallationDoesNotDeclareIsStoredHere(t *testing.T) {
	fixture := newConfigurationFixture(t)

	replaceConfiguration(t, fixture.handler, fixture.caller.token, `{"CastReceiverId":"no-such-receiver"}`)

	_, values := members(t, configurationOf(t, fixture.handler, "/Users/Me", fixture.caller.token))
	if got := string(values["CastReceiverId"]); got != `"no-such-receiver"` {
		t.Errorf(`CastReceiverId came back as %s, want "no-such-receiver" — spec 3.6 stores every property faithfully`, got)
	}
}

// TestABodyThatIsNotJsonIsTheValidationProblem is the refusal spec 3.6 does not
// name and this route cannot avoid having.
//
// A document that cannot be read is not an empty one: storing the defaults for
// it would be the silent wrong write U-14 already costs this route once, and a
// 500 would blame the server for a request. The reference's parameter is
// `[FromBody, Required]`
// [source: Jellyfin.Api/Controllers/UserController.cs:494 @ v10.11.11], so its
// binder answers behaviours 1.11's validation 400 with two keys: `"$"` for the
// text it could not read and the action parameter's own declared name beside
// it. That is the login route's shape with one word changed, and the word is
// `userConfig` rather than `request` — two routes, two action parameters, and
// neither name is guessable from the other.
//
// The message under `"$"` is Go's and the reference's is .NET's, which
// behaviours 1.11 already declares as the one half of this shape that cannot be
// matched. What is asserted here is the half that can: the status, the content
// type, the two keys — and that the message is **the same text the login route
// sends for the same bytes**. That last one is not about the reference at all.
// The configuration is decoded by internal/users, which wraps its decoder's
// error with its own package name; a handler that passed the wrapped text
// through would make two of this server's own routes answer one unreadable
// document in two spellings, which is a difference between Atrium and Atrium
// before it is a difference between Atrium and Jellyfin.
func TestABodyThatIsNotJsonIsTheValidationProblem(t *testing.T) {
	fixture := newConfigurationFixture(t)

	answer, _ := postConfiguration(t, fixture.handler, "not a document at all",
		httpapi.EmbyTokenHeader, fixture.caller.token)

	if answer.status != http.StatusBadRequest {
		t.Fatalf("a body that is not JSON answered %d and the body %q, want 400", answer.status, answer.body)
	}
	if answer.contentType != "application/json; charset=utf-8" {
		t.Errorf("the refusal declared %q, want application/json; charset=utf-8", answer.contentType)
	}

	var problem struct {
		Status int                 `json:"status"`
		Errors map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(answer.body), &problem); err != nil {
		t.Fatalf("reading the problem document %q: %v", answer.body, err)
	}
	if problem.Status != http.StatusBadRequest {
		t.Errorf("the problem document declares status %d, want 400", problem.Status)
	}
	keys := make([]string, 0, len(problem.Errors))
	for key := range problem.Errors {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	if want := []string{"$", "userConfig"}; !slices.Equal(keys, want) {
		t.Errorf("the errors map is keyed on %v, want %v", keys, want)
	}
	if got := problem.Errors["userConfig"]; len(got) != 1 || got[0] != "The userConfig field is required." {
		t.Errorf("the message under userConfig is %v, want [\"The userConfig field is required.\"]", got)
	}

	login := post(t, fixture.handler, "not a document at all", httpapi.AuthorizationHeader, testClientHeader)
	var loginProblem struct {
		Errors map[string][]string `json:"errors"`
	}
	if err := json.Unmarshal([]byte(login.body), &loginProblem); err != nil {
		t.Fatalf("reading the login route's problem document %q: %v", login.body, err)
	}
	if got, want := problem.Errors["$"], loginProblem.Errors["$"]; !slices.Equal(got, want) {
		t.Errorf("the two routes describe one unreadable document differently:\n  this route: %v\nlogin route: %v", got, want)
	}
}

// TestTheCredentialIsReadBeforeTheBodyIsBound asserts an order observable in
// exactly one request: no credential **and** a body that is not JSON.
//
// It answers the 401 rather than the 400, which is GET /Users/{userId}'s order
// on this route's own two refusals — the reference's authorization filter runs
// ahead of its model binder, measured on another route (009 spec 3.8,
// 2026-09-01) and read across to this one. A handler that bound first would
// tell an unauthenticated caller what this server thinks of its body, and no
// assertion about either refusal on its own can see it.
func TestTheCredentialIsReadBeforeTheBodyIsBound(t *testing.T) {
	fixture := newConfigurationFixture(t)

	answer, _ := postConfiguration(t, fixture.handler, "not a document at all")
	if answer.status != http.StatusUnauthorized {
		t.Errorf("a request with no credential and an unreadable body answered %d and the body %q,"+
			" want the 401 — the credential is read before the body is bound", answer.status, answer.body)
	}
	if answer.body != "" {
		t.Errorf("the refusal carried the body %q, so the binder answered it and not the authenticator", answer.body)
	}
}
