package conformance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// This file is the half of the two cross-cutting L1 sweeps (spec 6,
// conformance L1) that walks response *bytes*. Its twin walks Go types and
// lives in internal/httpapi/sweep_test.go.
//
// # Why the sweeps are split, and which half this is
//
// architecture 8 puts "the two reflection sweeps" here. architecture 3 forbids
// this package from importing anything under internal/, and
// tools/check_conformance_imports enforces it. A reflection sweep over this
// project's response models needs those models, so the reflection half cannot
// live here and does not; the import rule is the one honoured, and the reason
// is in doc.go — everything a test here knows is something a client could have
// known.
//
// What is left for this half is not a consolation prize. It is the half that
// reaches the rule conformance L1 corrects itself on: **a date is recognised by
// its value, not by its name.** Of nine date-valued fields observed in the
// reference, three — DateCreated, DateLastMediaAdded and LastPlaybackCheckIn —
// do not end in Date `[probe: tools/probe_wire_format, Jellyfin 10.11.11,
// 2026-09-02]`, so a rule keyed on the field name checks six of the nine. A Go
// type cannot answer "is this value a date". These bytes can.
//
// It is also the only half that sees what a body actually contains rather than
// what its declaration allows: 001's two empty arrays are []any, and an
// interface has no fields to reflect over. Whatever a later feature puts in
// one is swept here or nowhere.
//
// # Where the two halves could leave a gap, and what closes it
//
// A field is swept by neither if it is in no registered response model *and*
// in no body this file requests. The first clause is closed in the other half,
// which checks its registry against the router.
//
// **The second was open until T20 and is now closed**, which is worth saying
// plainly because this file used to say the opposite. sweptResponses below is
// still a hand-written list — it has to be, since a request needs a method, a
// path and an Accept header that no table carries — but it is no longer only
// checked against another hand-written list of the same routes:
// TestTheSweepReachesEveryRouteTheServerServes in routes_test.go requires every
// row the *running server* answers to appear here. A route added and not swept
// now fails this package.
//
// architecture 8 carries the amendment, dated.

// sweptResponse is one request whose body must survive both sweeps.
type sweptResponse struct {
	name   string
	method string

	// path is the route as the surface document spells it, parameters and all.
	// It is what TestTheSweepReachesEveryRouteTheServerServes matches on, so
	// it is the document's spelling rather than the one that is sent.
	path string

	// request is what is actually sent, for a row whose path carries a
	// parameter. Empty means the path itself.
	request string

	// accept is the Accept header, empty for a client that asks for nothing in
	// particular. Only the two PascalCase profiles appear below: the CamelCase
	// profile writes camelCase property names *by contract* (spec 3.0.2), so
	// sweeping it would be asserting the opposite of what that profile is for.
	// It has its own test at the end of this file, where the sweep firing is
	// the assertion rather than the failure.
	accept string

	// token says whether the request carries the fixture's access token, in
	// the Authorization header a client really sends it in.
	token bool

	// device replaces the fixture's own DeviceId. It is set on exactly the two
	// login rows, and it has to be: a second authentication from one device
	// revokes the first one's token (002 plan 6.5), so a login sent from the
	// fixture's device would log the rest of this list out half-way through.
	device string

	// body is the request body, for the three routes of 002 that read one.
	body string

	// status is what the request must answer. Zero means 200; the two writes
	// answer 204 and have no body to sweep at all, which is stated here rather
	// than inferred from an empty response.
	status int
}

// sweptResponses is every body this server puts on the wire, under each content
// profile whose names are PascalCase.
//
// ~~/System/Info is requested on a fresh installation, where first-time setup
// is outstanding and the route is therefore admitted without a credential
// (spec 3.2) — the only state 001 can reach it in.~~ **002 reaches the other
// state**: the fixture below provisions an account, so setup is complete and
// the route is requested with a token this server issued.
var sweptResponses = []sweptResponse{
	{name: "the public system info", method: http.MethodGet, path: publicSystemInfoPath},
	{name: "the public system info, PascalCase", method: http.MethodGet, path: publicSystemInfoPath, accept: pascalCaseProfile},
	{name: "the system info", method: http.MethodGet, path: systemInfoPath, token: true},
	{name: "the system info, PascalCase", method: http.MethodGet, path: systemInfoPath, accept: pascalCaseProfile, token: true},
	{name: "a ping", method: http.MethodGet, path: pingPath},
	{name: "a ping, PascalCase", method: http.MethodGet, path: pingPath, accept: pascalCaseProfile},
	{name: "a posted ping", method: http.MethodPost, path: pingPath},
	{name: "a posted ping, PascalCase", method: http.MethodPost, path: pingPath, accept: pascalCaseProfile},

	// Feature 002's seven rows. The tie in routes_test.go requires every row
	// the running server answers to be here, and it does **not** write them:
	// a request needs a method, a credential, a body and a path with its
	// parameters filled, and no table carries any of those.
	{name: "the public users", method: http.MethodGet, path: publicUsersPath},
	{name: "the public users, PascalCase", method: http.MethodGet, path: publicUsersPath, accept: pascalCaseProfile},

	{name: "an authentication", method: http.MethodPost, path: authenticateByNamePath,
		body: sweepLoginBody, device: "sweep-login-plain"},
	{name: "an authentication, PascalCase", method: http.MethodPost, path: authenticateByNamePath,
		body: sweepLoginBody, device: "sweep-login-pascal", accept: pascalCaseProfile},

	{name: "the caller's own user object", method: http.MethodGet, path: currentUserPath, token: true},
	{name: "the caller's own user object, PascalCase", method: http.MethodGet, path: currentUserPath, token: true, accept: pascalCaseProfile},

	// The request path is filled in by the fixture: the identifier is derived
	// from the account name (Principle VII) and read off the login response
	// rather than transcribed, so this list cannot disagree with the server
	// about what it is.
	{name: "a user object by identifier", method: http.MethodGet, path: userByIDPath, token: true},
	{name: "a user object by identifier, PascalCase", method: http.MethodGet, path: userByIDPath, token: true, accept: pascalCaseProfile},

	{name: "the session list", method: http.MethodGet, path: sessionsPath, token: true},
	{name: "the session list, PascalCase", method: http.MethodGet, path: sessionsPath, token: true, accept: pascalCaseProfile},

	{name: "a configuration write", method: http.MethodPost, path: userConfigurationPath,
		token: true, body: "{}", status: http.StatusNoContent},
	{name: "a capabilities declaration", method: http.MethodPost, path: capabilitiesPath,
		token: true, body: sweepCapabilities, status: http.StatusNoContent},
}

// The 002 paths, spelled as the surface document spells them.
const (
	publicUsersPath        = "/Users/Public"
	authenticateByNamePath = "/Users/AuthenticateByName"
	currentUserPath        = "/Users/Me"
	userByIDPath           = "/Users/{userId}"
	userConfigurationPath  = "/Users/Configuration"
	capabilitiesPath       = "/Sessions/Capabilities/Full"
	sessionsPath           = "/Sessions"
)

const (
	pascalCaseProfile = `application/json; profile="PascalCase"`
	camelCaseProfile  = `application/json; profile="CamelCase"`
)

// The account the sweep's requests are made as, and the credential it holds.
//
// The password is a literal in a public repository and that is deliberate: it
// authenticates against an installation this test creates in a temporary
// directory and destroys when it ends, and a value that looked like a real
// secret would be worse rather than better — provisioning_test.go already takes
// the same view.
const (
	sweepAccountName     = "Sweep"
	sweepAccountPassword = "hunter2"
	sweepLoginBody       = `{"Username":"` + sweepAccountName + `","Pw":"` + sweepAccountPassword + `"}`
)

// sweepCapabilities is the declaration the fixture posts, and its property
// names are PascalCase on purpose.
//
// behaviours 5.9: the reference stores the posted document and answers it back
// **unparsed**, so whatever keys a client sent travel into every later
// /Sessions body. internal/wire cannot rename them under a content profile
// either — it renames by walking the document beside the value it was encoded
// from, and a json.Marshaler leaves that walk. So the keys in the response are
// the keys in this literal, and the sweep judges them like any other property
// name. That is the correct treatment and not an oversight: the subtree is
// **not** declared as a dictionary, and
// TestTheCasingSweepFiresOnAnEchoedCapabilitiesDocument below proves the sweep
// still sees into it.
const sweepCapabilities = `{"PlayableMediaTypes":["Audio","Video"],"SupportedCommands":["Play"],"SupportsMediaControl":false}`

// sweepFixture is the installation every request above is issued against: one
// visible administrator with a password, logged in, with a capabilities
// document posted so that /Sessions carries the raw subtree.
type sweepFixture struct {
	*server

	// token is the access token the login answered with.
	token string

	// userID is the account's identifier, read off the login response rather
	// than derived here — this package cannot derive one, and stating it would
	// be transcribing what the server computed.
	userID string
}

// newSweepFixture provisions the account, starts the server and logs in.
//
// The account is created with --hidden=false because a fresh account is hidden
// at the reference [source: Jellyfin.Data/UserEntityExtensions.cs:173 @
// v10.11.11], and /Users/Public excludes hidden accounts (spec 3.4). A hidden
// fixture would answer `[]` and the sweep over that row would walk nothing
// while passing.
func newSweepFixture(t *testing.T) *sweepFixture {
	t.Helper()

	fixture := &sweepFixture{server: startServer(t,
		withProvisionedAccount(sweepAccountName, sweepAccountPassword+"\n",
			"--administrator", "--hidden=false"))}

	got := fixture.send(t, http.MethodPost, authenticateByNamePath, goldenHost,
		http.Header{"Authorization": {clientIdentification(sweepFixtureDevice, "")}},
		[]byte(sweepLoginBody))
	if got.status != http.StatusOK {
		t.Fatalf("authenticating the sweep's account: status %d, want %d\nbody: %s",
			got.status, http.StatusOK, got.body)
	}

	fixture.token = unquote(t, rawField(t, got.body, "AccessToken"))
	fixture.userID = unquote(t, rawField(t, []byte(rawField(t, got.body, "User")), "Id"))
	if fixture.token == "" || fixture.userID == "" {
		t.Fatalf("the login answered no token or no identifier:\n%s", got.body)
	}

	// So that /Sessions carries the raw document rather than nothing. Posted
	// here rather than relying on the swept row of the same route, because the
	// order two rows of one list run in is not something a test should depend
	// on.
	declared := fixture.send(t, http.MethodPost, capabilitiesPath, goldenHost,
		http.Header{"Authorization": {clientIdentification(sweepFixtureDevice, fixture.token)}},
		[]byte(sweepCapabilities))
	if declared.status != http.StatusNoContent {
		t.Fatalf("declaring the fixture's capabilities: status %d, want %d\nbody: %s",
			declared.status, http.StatusNoContent, declared.body)
	}

	return fixture
}

// sweepFixtureDevice is the DeviceId the fixture's own session is keyed on. The
// two login rows use their own, for the reason sweptResponse.device gives.
const sweepFixtureDevice = "sweep-fixture"

// unquote reads a raw JSON string into the value it carries.
//
// rawField hands back the bytes as they arrived (Principle VIII), which is what
// makes it the right reader for an assertion; a token has to be used rather
// than asserted on, and this is where it stops being bytes.
func unquote(t *testing.T, raw string) string {
	t.Helper()

	var value string
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("%s is not a JSON string: %v", raw, err)
	}
	return value
}

// clientIdentification is 002 spec 3.2's grammar, as a client writes it.
//
// Spelled out here rather than imported, which is the boundary doing its job:
// this is the string a real client puts in the header, and if the parser ever
// stopped accepting it these tests would fail rather than agree with it.
func clientIdentification(device, token string) string {
	value := `MediaBrowser Client="Atrium Conformance", Device="A Sweep", DeviceId="` +
		device + `", Version="1.0.0"`
	if token != "" {
		value += `, Token="` + token + `"`
	}
	return value
}

// request turns one row of the list above into the request it names.
func (f *sweepFixture) issue(t *testing.T, swept sweptResponse) *response {
	t.Helper()

	header := http.Header{}
	if swept.accept != "" {
		header.Set("Accept", swept.accept)
	}
	switch {
	case swept.device != "":
		header.Set("Authorization", clientIdentification(swept.device, ""))
	case swept.token:
		header.Set("Authorization", clientIdentification(sweepFixtureDevice, f.token))
	}

	var body []byte
	if swept.body != "" {
		body = []byte(swept.body)
	}
	return f.send(t, swept.method, f.pathOf(swept), goldenHost, header, body)
}

// pathOf is what a row sends: its own request path, or the document's path with
// the fixture's identifier substituted.
func (f *sweepFixture) pathOf(swept sweptResponse) string {
	if swept.request != "" {
		return swept.request
	}
	if swept.path == userByIDPath {
		return "/Users/" + f.userID
	}
	return swept.path
}

// TestEveryResponseSweepsClean is the two sweeps, run over the wire.
//
// One server, every response this build sends, both rules. ~~It finds nothing
// today — 001's bodies are strings, booleans, one integer port and two empty
// arrays~~ **002's bodies are the first with anything in them**: two user
// objects carrying sixty policy and configuration members, a session with a
// date, a tick and an echoed document, and an authentication result carrying
// both. It still finds nothing, which is the state a sweep is meant to be in
// and also the state in which it has proved nothing. The tests below it are
// what make that state mean something.
func TestEveryResponseSweepsClean(t *testing.T) {
	t.Parallel()

	fixture := newSweepFixture(t)

	for _, swept := range sweptResponses {
		t.Run(swept.name, func(t *testing.T) {
			got := fixture.issue(t, swept)

			want := swept.status
			if want == 0 {
				want = http.StatusOK
			}
			if got.status != want {
				t.Fatalf("%s %s: status %d, want %d\nbody: %s",
					swept.method, fixture.pathOf(swept), got.status, want, got.body)
			}

			// A 204 has no body, so there is nothing to walk — and saying so
			// is the row rather than an omission. The assertion is that it
			// really is empty: a sweep handed nothing reports nothing, and a
			// route that started sending a body would otherwise go unswept.
			if want == http.StatusNoContent {
				if len(got.body) != 0 {
					t.Fatalf("%s %s answered %d with a body, which nothing here sweeps:\n%s",
						swept.method, fixture.pathOf(swept), got.status, got.body)
				}
				return
			}

			for _, found := range sweepBody(t, got.body) {
				t.Errorf("%s %s: %s", swept.method, fixture.pathOf(swept), found)
			}
		})
	}
}

// TestTheSweptBodiesAreNotEmpty is what keeps the run above from passing by
// having looked at nothing.
//
// Every assertion in this file is of the form "the sweep found no fault", and
// an empty listing, a session list with no sessions in it or a user object that
// did not arrive would all satisfy it perfectly. So the bodies the new rows
// produce are required to carry the members that make them worth sweeping.
func TestTheSweptBodiesAreNotEmpty(t *testing.T) {
	t.Parallel()

	fixture := newSweepFixture(t)

	for _, row := range []struct {
		name     string
		swept    sweptResponse
		contains []string
	}{
		{
			name:     "the public users list has the visible account in it",
			swept:    sweptResponse{method: http.MethodGet, path: publicUsersPath},
			contains: []string{`"Name":"` + sweepAccountName + `"`, `"Policy"`, `"Configuration"`},
		},
		{
			name:     "the session list has the fixture's own session in it",
			swept:    sweptResponse{method: http.MethodGet, path: sessionsPath, token: true},
			contains: []string{`"DeviceId":"` + sweepFixtureDevice + `"`, `"LastActivityDate"`, `"Capabilities"`},
		},
		{
			name:     "the user object carries both documents",
			swept:    sweptResponse{method: http.MethodGet, path: currentUserPath, token: true},
			contains: []string{`"IsAdministrator"`, `"PlayDefaultAudioTrack"`},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := fixture.issue(t, row.swept)
			if got.status != http.StatusOK {
				t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
			}
			for _, want := range row.contains {
				if !strings.Contains(string(got.body), want) {
					t.Errorf("the body does not contain %s, so the sweep over it walks less than this file claims:\n%s",
						want, got.body)
				}
			}
		})
	}
}

// TestTheSweepsFailOnAFaultPlantedInABodyThisServerReallySent is T17's failure
// proof at the wire, and it is deliberately not the same test as the three
// below.
//
// Those run the sweeps over bytes this file wrote. This one takes a **real
// response** from one of the seven new routes and plants the two faults the
// *Verified by* line names into it — a camelCase property name, and a date with
// three fractional digits where behaviours 1.2 requires seven. What that proves
// is the thing a hand-written body cannot: the walk descends through this
// server's own nesting — a session inside an array, a policy inside a user
// object inside an authentication result — and would have reported a fault at
// that depth.
func TestTheSweepsFailOnAFaultPlantedInABodyThisServerReallySent(t *testing.T) {
	t.Parallel()

	fixture := newSweepFixture(t)

	for _, row := range []struct {
		name  string
		swept sweptResponse

		// property is a PascalCase name in the real body, renamed to its
		// camelCase spelling in the copy.
		property string

		// date is a seven-digit date member in the real body, truncated to
		// three digits in the copy.
		date string
	}{
		{
			name:     "a session in the list",
			swept:    sweptResponse{method: http.MethodGet, path: sessionsPath, token: true},
			property: "DeviceName",
			date:     "LastActivityDate",
		},
		{
			name:     "the authentication result",
			swept:    sweptResponse{method: http.MethodPost, path: authenticateByNamePath, body: sweepLoginBody, device: "sweep-planted"},
			property: "IsAdministrator",
			date:     "LastActivityDate",
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			got := fixture.issue(t, row.swept)
			if got.status != http.StatusOK {
				t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
			}
			if found := sweepBody(t, got.body); len(found) != 0 {
				t.Fatalf("the body this fault is planted in is not clean to begin with:\n%s",
					strings.Join(found, "\n"))
			}

			planted := plantACamelCaseProperty(t, got.body, row.property)
			planted = plantAThreeDigitDate(t, planted, row.date)

			found := sweepBody(t, planted)
			if len(found) != 2 {
				t.Fatalf("the sweeps reported %d findings on a body with two faults planted in it, want 2:\n%s\nbody: %s",
					len(found), strings.Join(found, "\n"), planted)
			}
			report := strings.Join(found, "\n")
			if !strings.Contains(report, lowerFirst(row.property)) {
				t.Errorf("no finding names %q:\n%s", lowerFirst(row.property), report)
			}
			if !strings.Contains(report, row.date) {
				t.Errorf("no finding names %q:\n%s", row.date, report)
			}
		})
	}
}

// plantACamelCaseProperty rewrites one property name of a body to its camelCase
// spelling, wherever it occurs.
//
// It works on the bytes rather than on a decoded document, which is the only
// way to keep everything else — key order included — exactly as it arrived.
func plantACamelCaseProperty(t *testing.T, body []byte, property string) []byte {
	t.Helper()

	before := `"` + property + `":`
	if !bytes.Contains(body, []byte(before)) {
		t.Fatalf("the body carries no property called %q, so nothing was planted:\n%s", property, body)
	}
	return bytes.ReplaceAll(body, []byte(before), []byte(`"`+lowerFirst(property)+`":`))
}

// plantAThreeDigitDate truncates a member's seven fractional digits to three —
// the spelling the reference itself sends on LastPlayedDate, and one this
// server may not send (behaviours 1.2).
func plantAThreeDigitDate(t *testing.T, body []byte, member string) []byte {
	t.Helper()

	truncate := regexp.MustCompile(`("` + member + `":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3})\d{4}Z"`)
	planted := truncate.ReplaceAll(body, []byte(`${1}Z"`))
	if bytes.Equal(planted, body) {
		t.Fatalf("the body carries no seven-digit %s, so nothing was planted:\n%s", member, body)
	}
	return planted
}

// lowerFirst is the camelCase spelling of a PascalCase property name.
func lowerFirst(name string) string {
	return strings.ToLower(name[:1]) + name[1:]
}

// TestTheCasingSweepFiresOnAnEchoedCapabilitiesDocument is the other half of
// what SessionInfo.Capabilities needs said about it, and it is a divergence
// recorded as an assertion rather than a hole opened in the sweep.
//
// behaviours 5.9 measured the reference storing a posted capabilities document
// and answering it back unparsed, unknown properties and all. This server does
// the same, and internal/wire cannot rename those keys under a content profile
// because a json.Marshaler leaves the walk that renames. So a client that posts
// camelCase keys gets camelCase keys back inside every later /Sessions body —
// and the sweep reports them, correctly, because they are property names on
// this server's wire.
//
// The subtree is therefore **not** declared in dictionaryPointers. Declaring it
// would be the loosening that makes the guard stop seeing a real casing fault
// one member over, and the fixture posts a PascalCase document for the same
// reason a real client does.
func TestTheCasingSweepFiresOnAnEchoedCapabilitiesDocument(t *testing.T) {
	t.Parallel()

	fixture := newSweepFixture(t)

	// Two camelCase keys, one of them a property the reference declares and
	// one it does not, posted from the fixture's own device so that the
	// declaration replaces the one the fixture made.
	const camelCased = `{"playableMediaTypes":["Audio"],"somethingNobodyDeclares":true}`
	declared := fixture.send(t, http.MethodPost, capabilitiesPath, goldenHost,
		http.Header{"Authorization": {clientIdentification(sweepFixtureDevice, fixture.token)}},
		[]byte(camelCased))
	if declared.status != http.StatusNoContent {
		t.Fatalf("declaring capabilities: status %d, want %d\nbody: %s",
			declared.status, http.StatusNoContent, declared.body)
	}

	got := fixture.issue(t, sweptResponse{method: http.MethodGet, path: sessionsPath, token: true})
	if got.status != http.StatusOK {
		t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}

	// Echoed unparsed, which is the behaviour under test before the sweep is.
	if !strings.Contains(string(got.body), `"somethingNobodyDeclares":true`) {
		t.Errorf("the posted document did not survive into the session body, which is behaviours 5.9's measurement:\n%s", got.body)
	}

	found := sweepBody(t, got.body)
	if len(found) != 2 {
		t.Fatalf("the casing sweep reported %d findings over an echoed camelCase declaration, want the two keys:\n%s\nbody: %s",
			len(found), strings.Join(found, "\n"), got.body)
	}
	for _, key := range []string{"playableMediaTypes", "somethingNobodyDeclares"} {
		if !strings.Contains(strings.Join(found, "\n"), key) {
			t.Errorf("no finding names %q:\n%s", key, strings.Join(found, "\n"))
		}
	}
}

// TestTheCasingSweepCatchesACamelCaseProperty is one half of the failure proof
// the *Verified by* line asks for: a sweep that has never failed has proved
// nothing.
//
// The body is built from a model declared in this file, which is the strong
// form of "it cannot leak into the served surface": a _test.go file is not part
// of any package the server is built from, and this package is imported by
// nothing at all — the server it talks to is a separate process built from
// cmd/atrium. There is no path by which either type below reaches a response.
func TestTheCasingSweepCatchesACamelCaseProperty(t *testing.T) {
	t.Parallel()

	body := marshalTestOnlyModel(t, modelWithACamelCaseProperty{
		ServerName:   "atrium",
		LocalAddress: "http://192.168.1.20:8096",
		Nested:       nestedModelWithACamelCaseProperty{Version: "10.11.11"},
		Items: []nestedModelWithACamelCaseProperty{
			{Version: "10.11.11"},
		},
	})

	// Three findings: one at the top level, one an object deep, and one inside
	// an array element. The third is there because a sweep that walked objects
	// and not arrays would report two and look correct.
	assertSweptExactly(t, body, "/localAddress", "/Nested/version", "/Items/0/version")
}

// TestTheUnitSweepCatchesAThreeDigitDate is the other half, and it is the one
// the corrected rule is really about.
//
// The model carries two dates and neither is caught by its name: DateCreated
// does not end in Date, and Added says nothing at all. Both are caught by their
// **values**. behaviours 1.2 is the rule — seven fractional digits and a Z —
// and the reference's own three- and six-digit values on LastPlayedDate and
// LastActivityDate are why this is a statement about Atrium rather than a
// shared fact: an L3 comparison of those two fields will differ, and this sweep
// is still right about what this server may send.
func TestTheUnitSweepCatchesAThreeDigitDate(t *testing.T) {
	t.Parallel()

	body := marshalTestOnlyModel(t, modelWithABadlySpelledDate{
		DateCreated:  "2025-06-19T00:00:00.000Z",
		Added:        "2025-06-19T00:00:00Z",
		PremiereDate: "2025-06-19T00:00:00.0000000Z",
	})

	assertSweptExactly(t, body, "DateCreated", "Added")
}

// TestTheUnitSweepCatchesFractionalTicks is the tick half of the same rule.
// behaviours 1.3 makes a tick a whole 100-nanosecond interval; a JSON number
// with a fractional part is a conversion somebody forgot at a boundary.
func TestTheUnitSweepCatchesFractionalTicks(t *testing.T) {
	t.Parallel()

	// Written as raw bytes rather than marshalled from a struct, because Go's
	// encoder writes a float64 with a zero fraction as an integer and would
	// hide the very thing under test.
	body := []byte(`{"RunTimeTicks":9000000000.5,"StartTicks":9000000000}`)

	assertSweptExactly(t, body, "RunTimeTicks")
}

// assertSweptExactly runs both sweeps over one body and requires exactly one
// finding per named location, and no others.
//
// The count is asserted as well as the names, because a sweep that reported
// every field would satisfy "it named the bad one" and prove nothing. The names
// are matched without regard to order: encoding/json decodes an object into a
// map, so the order the sweep visits keys in is lexical rather than the
// document's, and asserting on it would be asserting on the decoder.
func assertSweptExactly(t *testing.T, body []byte, want ...string) {
	t.Helper()

	found := sweepBody(t, body)
	if len(found) != len(want) {
		t.Fatalf("the sweeps reported %d findings on %s, want %d:\n%s",
			len(found), body, len(want), strings.Join(found, "\n"))
	}

	report := strings.Join(found, "\n")
	for _, name := range want {
		matches := 0
		for _, finding := range found {
			if strings.Contains(finding, name) {
				matches++
			}
		}
		if matches != 1 {
			t.Errorf("%d of the findings name %q, want 1:\n%s", matches, name, report)
		}
	}
}

// TestADeclaredDictionarysKeysAreNotSweptAsPropertyNames exercises the guard
// conformance L1 records the cost of.
//
// A dictionary's keys are data. Treating them as property names reported **688
// of 899 keys** as casing failures in one run against the reference, because
// ImageBlurHashes is keyed by image tag `[probe: tools/probe_wire_format,
// Jellyfin 10.11.11, 2026-09-02]`. 001 sends no dictionary, so dictionaryPointers
// is empty and the guard would otherwise be code no case reaches — which is the
// same shape as the guards T4 and T15 each deleted after a surviving mutation.
// So the set is stated here instead, and the body carries both a data key and a
// real property name beside it, so that the guard is proven to be narrow rather
// than merely present.
func TestADeclaredDictionarysKeysAreNotSweptAsPropertyNames(t *testing.T) {
	t.Parallel()

	body := []byte(`{"ImageBlurHashes":{"a1b2c3":"W04,","primary":"W04,"},"itemId":"3f9c"}`)

	found := sweepBodyWithDictionaries(t, body, map[string]bool{"/ImageBlurHashes": true})
	if len(found) != 1 {
		t.Fatalf("the casing sweep reported %d findings, want the one that is not a dictionary key:\n%s",
			len(found), strings.Join(found, "\n"))
	}
	if !strings.Contains(found[0], "itemId") {
		t.Errorf("the finding is %q and does not name %q", found[0], "itemId")
	}

	// And the guard is scoped: the same body with nothing declared reports the
	// key as well, which is what makes the declaration load-bearing.
	if undeclared := sweepBodyWithDictionaries(t, body, nil); len(undeclared) != 3 {
		t.Errorf("with no dictionary declared the sweep reported %d findings, want 3 — "+
			"the two data keys and the property name:\n%s",
			len(undeclared), strings.Join(undeclared, "\n"))
	}
}

// TestTheCasingSweepFiresOnTheCamelCaseProfile is the failure proof that needs
// no test-only model at all: it is a body this server really sends.
//
// spec 3.0.2 makes `profile="CamelCase"` a real content profile, and under it
// every property name of /System/Info/Public is camelCase by contract. So the
// sweep run over that response must report a finding for each of the seven —
// which is a live sweep firing on real bytes, and evidence that
// TestEveryResponseSweepsClean passes because the bodies are clean rather than
// because the sweep looked at nothing.
//
// It is also why that profile is absent from sweptResponses: sweeping it as a
// requirement would assert the opposite of what spec 3.0.2 says it is for.
func TestTheCasingSweepFiresOnTheCamelCaseProfile(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.get(t, publicSystemInfoPath, goldenHost, http.Header{"Accept": {camelCaseProfile}})

	if got.status != http.StatusOK {
		t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}

	// Seven fields (spec 3.1), and every one of them renamed. Six begin with a
	// lower-case letter; Id becomes id.
	const fields = 7
	found := sweepBody(t, got.body)
	if len(found) != fields {
		t.Fatalf("the casing sweep reported %d findings on the CamelCase profile, want %d:\n%s\nbody: %s",
			len(found), fields, strings.Join(found, "\n"), got.body)
	}
}

// modelWithACamelCaseProperty exists only in this file. Its Go field names are
// PascalCase and read fine in review; the tags are where the mistake is really
// made, and where a body's property names really come from.
type modelWithACamelCaseProperty struct {
	ServerName   string
	LocalAddress string `json:"localAddress"`
	Nested       nestedModelWithACamelCaseProperty
	Items        []nestedModelWithACamelCaseProperty
}

// nestedModelWithACamelCaseProperty puts the second failure one level down, so
// that a sweep which only looked at the top level of a body would pass the test
// above with one finding instead of two.
type nestedModelWithACamelCaseProperty struct {
	Version string `json:"version"`
}

// modelWithABadlySpelledDate carries three dates: one spelled as the wire
// spells it, and two that are not. Neither of the two is named in a way the
// suffix rule would find.
type modelWithABadlySpelledDate struct {
	DateCreated  string
	Added        string
	PremiereDate string
}

// marshalTestOnlyModel serialises one of the models above into the bytes the
// sweep works on. It uses encoding/json directly rather than anything of this
// project's, because this package may not import internal/wire — which is the
// boundary doing its job: the sweep is being fed bytes, not a value.
func marshalTestOnlyModel(t *testing.T, model any) []byte {
	t.Helper()
	body, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("serialising a test-only model: %v", err)
	}
	return body
}

// dictionaryPointers names the places in a response body whose object *keys*
// are data rather than property names, as JSON Pointers to the containing
// object.
//
// It is empty, and it exists because the trap it guards has already been fallen
// into: a sweep that treated every JSON object key as a property name reported
// **688 of 899 keys** as casing failures in one run against the reference,
// because ImageBlurHashes is keyed by image tag (conformance L1)
// `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`. No body
// feature 001 sends contains a dictionary, so the correct declaration today is
// an empty one — and the feature that first sends ImageBlurHashes adds
// "/ImageBlurHashes" here rather than discovering the rule from 688 failures.
var dictionaryPointers = map[string]bool{}

// wireDate is the one spelling of a date this server may send: seven fractional
// digits and a Z (behaviours 1.2).
//
// It is written out here rather than borrowed from internal/units, which is the
// import rule again and, here, a benefit: a change to the layout that package
// writes has to be made twice, and the second time is at the wire, where the
// contract is. A sweep that asked internal/units whether internal/units had
// written a date correctly would agree with it by construction.
var wireDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{7}Z$`)

// dateShaped is what makes a string a candidate for the rule above: a full ISO
// date, alone or followed by a time.
//
// It is deliberately narrower than "starts with something date-like". A value
// that begins with a date and continues into prose — an album called
// "2001-01-01 Sessions" — must not be swept as a malformed date, and requiring
// either the whole string or a T and a clock is what keeps it out. A date-only
// value is included because it is exactly ten characters and is never a title;
// it is also a real failure, since behaviours 1.2's output form always carries
// a time.
var dateShaped = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}.*)?$`)

// sweepBody runs both cross-cutting sweeps over one response body and returns
// what they found.
//
// The body is decoded with UseNumber, so a number is still the bytes that were
// sent: 9000000000 and 9000000000.0 are the same float64 and a different
// response, and the tick rule is about which of the two arrived.
func sweepBody(t *testing.T, body []byte) []string {
	t.Helper()
	return sweepBodyWithDictionaries(t, body, dictionaryPointers)
}

// sweepBodyWithDictionaries is sweepBody with the declared dictionaries stated
// rather than taken from the package.
//
// The set is a parameter so that the guard can be exercised: 001 declares none,
// and a guard no case reaches is a guard that has proved nothing — which is the
// finding T4 and T15 each arrived at independently.
func sweepBodyWithDictionaries(t *testing.T, body []byte, dictionaries map[string]bool) []string {
	t.Helper()

	// A bare JSON string, which is what /System/Ping answers, decodes to a
	// string with no property name in it. That is not an empty sweep by
	// accident — it is the whole content of that response.
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var document any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("the body is not JSON: %v\nbody: %s", err, body)
	}

	var found []string
	sweepValue("", "", document, dictionaries, func(finding string) { found = append(found, finding) })
	return found
}

// sweepValue walks one decoded value, applying both rules.
//
// pointer is the JSON Pointer of the value, and name is the property name it
// arrived under — empty at the root and inside an array, because an array's
// elements share their parent's name.
func sweepValue(pointer, name string, value any, dictionaries map[string]bool, report func(string)) {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range sortedObjectKeys(v) {
			child := pointer + "/" + escapePointerToken(key)
			if !dictionaries[pointer] && !isPascalCase(key) {
				report(fmt.Sprintf(
					"%s is the property name %q, which is not PascalCase (behaviours 1.1)",
					child, key))
			}
			sweepValue(child, key, v[key], dictionaries, report)
		}

	case []any:
		for i, element := range v {
			sweepValue(fmt.Sprintf("%s/%d", pointer, i), name, element, dictionaries, report)
		}

	case json.Number:
		if strings.HasSuffix(name, "Ticks") && !isWholeNumber(v.String()) {
			report(fmt.Sprintf(
				"%s is %s; ticks are whole 100-nanosecond intervals and serialise as an integer "+
					"(behaviours 1.3)", pointerOrRoot(pointer), v.String()))
		}

	case string:
		if dateShaped.MatchString(v) && !isWireDate(v) {
			report(fmt.Sprintf(
				"%s is the date %q; a date on this wire carries seven fractional digits and a Z "+
					"(behaviours 1.2)", pointerOrRoot(pointer), v))
		}
	}
}

// isWireDate reports whether a value is a date spelled the way this server must
// spell one.
//
// The shape is checked by the expression and the calendar by the parse: a
// pattern alone accepts 2025-02-30, and a parse alone accepts three fractional
// digits, because Go's fractional layout element matches any number of them.
func isWireDate(s string) bool {
	if !wireDate.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02T15:04:05.0000000Z", s)
	return err == nil
}

// isWholeNumber reports whether a JSON number arrived as an integer. It is a
// question about the bytes, not about the value: 9000000000.0 parses to a whole
// number and is not one on the wire.
func isWholeNumber(number string) bool {
	return !strings.ContainsAny(number, ".eE")
}

// isPascalCase is the rule of spec 3.0.1: an upper-case letter, then letters
// and digits.
//
// It is a second implementation of the rule the model sweep applies, and that
// is the import boundary rather than an oversight. It is held to the same
// standard by the table beside it: the reference's own names include
// EnableIPv4, UICulture, Video3DFormat and Hdr10PlusPresentFlag, and a rule
// spelled "capital then lower-case letters, repeated" refuses three of them and
// then gets loosened by whoever meets it.
func isPascalCase(name string) bool {
	for i, r := range name {
		if i == 0 {
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit {
			return false
		}
	}
	return name != ""
}

// TestThePascalCaseRuleHoldsTheAwkwardNames keeps this copy of the rule from
// drifting away from the one the model sweep applies. The names are the ones
// the pinned document contains that a careless rule refuses, and the two
// spellings that must be refused.
func TestThePascalCaseRuleHoldsTheAwkwardNames(t *testing.T) {
	t.Parallel()

	for _, accepted := range []string{
		"EnableIPv4", "UICulture", "Video3DFormat", "Hdr10PlusPresentFlag", "ETag", "Id",
	} {
		if !isPascalCase(accepted) {
			t.Errorf("the rule refuses %q, which is a property name of the pinned document", accepted)
		}
	}

	for _, refused := range []string{
		"localAddress", "uiCulture", "run_time_ticks", "", "3D", "Package Name",
	} {
		if isPascalCase(refused) {
			t.Errorf("the rule accepts %q, which is not PascalCase", refused)
		}
	}
}

// TestTheDateRuleRecognisesADateByItsValue is the correction conformance L1
// makes to its own sentence, written as a test.
//
// The left column is what must be recognised as a date and judged; the right is
// what must not be touched, because a sweep that flagged an ordinary string
// would be turned off by the first person who met it.
func TestTheDateRuleRecognisesADateByItsValue(t *testing.T) {
	t.Parallel()

	for _, wellSpelled := range []string{
		"2025-06-19T00:00:00.0000000Z",
		"0001-01-01T00:00:00.0000000Z", // the unset date the reference sends
	} {
		if !dateShaped.MatchString(wellSpelled) || !isWireDate(wellSpelled) {
			t.Errorf("%q is a date this server may send and the rule does not accept it", wellSpelled)
		}
	}

	for _, badlySpelled := range []string{
		"2025-06-19T00:00:00.000Z",    // three digits, as the reference sends on LastPlayedDate
		"2025-06-19T00:00:00.000000Z", // six, as it sends on LastActivityDate
		"2025-06-19T00:00:00Z",        // none
		"2025-06-19T00:00:00.0000000", // no zone
		"2025-06-19T02:00:00.0000000+02:00",
		"2025-06-19",
		"2025-02-30T00:00:00.0000000Z", // the right shape and not a day
	} {
		if !dateShaped.MatchString(badlySpelled) {
			t.Errorf("%q is a date and the sweep does not recognise it as one", badlySpelled)
			continue
		}
		if isWireDate(badlySpelled) {
			t.Errorf("%q is not a date this server may send and the rule accepts it", badlySpelled)
		}
	}

	for _, notADate := range []string{
		"atrium", "Jellyfin Server", "10.11.11", "/var/lib/atrium",
		"2001-01-01 Sessions", // an album, not a date
		"3f9c1a7e5b2d4e8091a6c3f70d5e2b14",
	} {
		if dateShaped.MatchString(notADate) {
			t.Errorf("the sweep reads %q as a date, and it is a value a library could really hold", notADate)
		}
	}
}

// sortedObjectKeys is the keys of a decoded object in the order they must be
// reported in.
//
// encoding/json decodes an object into a map and the wire order is gone by
// then, so this is the lexical order rather than the document's. That costs
// nothing here — the sweep reports every failing key whatever the order — and
// it is why a *key order* assertion cannot be built on this function.
// system_info_test.go's property-name assertion reads the bytes directly for
// exactly that reason.
func sortedObjectKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// pointerOrRoot names a location for a failure message.
func pointerOrRoot(pointer string) string {
	if pointer == "" {
		return "the body"
	}
	return pointer
}

// escapePointerToken is RFC 6901's escaping, so that a property name containing
// a slash does not make a pointer that reads as two.
func escapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	return strings.ReplaceAll(token, "/", "~1")
}
