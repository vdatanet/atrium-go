package conformance_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// pingPath is the route under test, spelled as a client spells it.
const pingPath = "/System/Ping"

// pingMethods is the pair spec 3.3 gives one request shape and one response.
// Every assertion below runs over both, because "both methods" is half of
// AC-6 and a test that only sends GET would pass on a server that never
// registered POST at all.
var pingMethods = []string{http.MethodGet, http.MethodPost}

// The byte-compared golden of AC-6, over both methods.
//
// One recorded file for two requests, on purpose: comparing both against the
// same bytes is what makes "both methods answer identically" an assertion
// rather than two assertions that happen to have been written the same way.
//
// Nothing about this response derives from the run — no identity, no address,
// no clock — so unlike the /System/Info/Public golden it needs no fixture to
// hold it still, and the installation it is recorded on is genuinely empty.
func TestPingMatchesItsGoldenOnBothMethods(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	// AC-6 says nothing about configuration, and this is the record that the
	// response does not depend on any: if a later change seeds this
	// installation, this line says the assertion stopped being about a server
	// with nothing on disk.
	if len(server.seeded) != 0 {
		t.Fatalf("this installation was not empty when it started: %v", server.seeded)
	}

	for _, method := range pingMethods {
		t.Run(method, func(t *testing.T) {
			got := server.do(t, method, pingPath, "", nil)

			if got.status != http.StatusOK {
				t.Fatalf("status: got %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
			}
			if contentType := got.header.Get("Content-Type"); contentType != "application/json; charset=utf-8" {
				t.Errorf("Content-Type: got %q, want %q", contentType, "application/json; charset=utf-8")
			}

			assertGolden(t, "system_ping.json", got.body)
		})
	}
}

// The golden above proves nothing unless the friendly name is a value this
// server could have answered with instead, and this is what says it is.
//
// spec 3.3's compatibility note is the trap: the reference's documentation
// comment on the operation says it returns "the server name" and its code
// returns the product name, so a reimplementation that read the comment
// answers the operator's chosen name here. Against a fixture whose friendly
// name nobody looked at, that mistake can pass.
//
// So the friendly name is read off this same server, from the response that
// does report it, and the test fails loudly if it has become the product name
// — at which point no ping assertion in this file discriminates any more.
//
// **This is the weakest link in T17 and it is stated rather than hidden.**
// An operator has no way to rename an installation — the SetServerName port
// exists and nothing calls it — so the friendly name this binary can be started
// with is 001 plan 4's default, "atrium", and not a value the fixture chose. The
// handler-level test in internal/httpapi sets a deliberately unlike name through
// the store and is where the discrimination is really proven; this one asserts,
// at the wire, that the two names differ on the server the golden was recorded
// against.
//
// # The condition this caveat drops on, corrected at 002 T21
//
// ~~When 002 lands a rename, this test should send it and drop the caveat.~~
// **002 lands no rename, and no v1 feature can.** The reference renames a server
// at POST /Startup/Configuration
// [source: Jellyfin.Api/Controllers/StartupController.cs:74-78 @ v10.11.11],
// which is not one of surface.yaml's fifty-nine rows and has no named consumer,
// so "when the rename endpoint lands" is a condition this surface can never
// satisfy and a caveat waiting on it would outlive the project (002 spec 5's
// amendment, 002 plan 8.2).
//
// What can satisfy it is the friendly name becoming **operator configuration** —
// one more subcommand over the port 001 wrote and nobody calls — at which point
// a fixture can start a server under a name it chose and this test can send it.
// 002 deliberately does not add that surface on the way past: the name is 001's
// datum, and deciding to configure it is not this feature's decision to take
// while it is passing.
func TestPingAnswersTheProductNameAndNotThisServersFriendlyName(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	var info struct {
		ServerName  string
		ProductName string
	}
	body := server.get(t, publicSystemInfoPath, goldenHost, nil).body
	if err := json.Unmarshal(body, &info); err != nil {
		t.Fatalf("reading the public system info: %v\n%s", err, body)
	}

	if info.ProductName != "Jellyfin Server" {
		t.Fatalf("ProductName is %q, want %q — the whole of AC-6 is that ping carries this value", info.ProductName, "Jellyfin Server")
	}
	if info.ServerName == "" {
		t.Fatalf("this server reports no ServerName, so there is no friendly name for ping to be confused with")
	}
	if info.ServerName == info.ProductName {
		t.Fatalf("this server's friendly name is %q, which is the product name; no assertion in this file can tell a ping that returns ServerName from one that returns ProductName", info.ServerName)
	}

	for _, method := range pingMethods {
		got := server.do(t, method, pingPath, "", nil)
		if string(got.body) == `"`+info.ServerName+`"` {
			t.Errorf("%s %s answered the operator's friendly name %s; spec 3.3 requires the product name", method, pingPath, got.body)
		}
		if want := `"` + info.ProductName + `"`; string(got.body) != want {
			t.Errorf("%s %s: body: got %s, want %s", method, pingPath, got.body, want)
		}
	}
}

// A method /System/Ping does not have is still refused, and this test is here
// for two reasons neither of which is the refusal shape.
//
// **The first is that it is what makes every "both methods" assertion above
// mean anything.** A harness that ignored the method it was handed and sent
// GET every time passes all three of them — measured, by making it do exactly
// that: every test in this package stayed green. Nothing that answers 200 to
// both methods can tell the two apart. A request that must answer *differently*
// by method can, and this is the cheapest one: sent as GET it is a 200, and the
// test fails.
//
// The second is that /System/Ping is the path this feature serves on two
// methods, so it is the path where registering a second real handler could
// disturb what a third method is answered. T11 owns the Allow computation and
// proves it over the route table and over an assembled pipeline; this row
// re-runs none of that reasoning, it just records that the value reaching a
// client of the real binary did not move when the two handlers arrived.
func TestAMethodPingDoesNotHaveIsStillRefused(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.do(t, http.MethodPut, pingPath, "", nil)

	if got.status != http.StatusMethodNotAllowed {
		t.Fatalf("PUT %s: status: got %d, want %d. A 200 here means the request was sent as a GET, and every assertion in this file about POST is then an assertion about GET.",
			pingPath, got.status, http.StatusMethodNotAllowed)
	}
	if len(got.body) != 0 {
		t.Errorf("PUT %s: body: got %s, want it empty", pingPath, got.body)
	}
	if allow := got.header.Values("Allow"); len(allow) != 1 || allow[0] != "GET, POST" {
		t.Errorf("PUT %s: Allow: got %v, want exactly [\"GET, POST\"] — the two methods this path is now served on", pingPath, allow)
	}
}

// spec 3.0 applies to every response in this specification, so ping echoes the
// profile that matched (AC-9).
//
// A bare JSON string has no property name to rename, so the three profiles
// answer one body under three content types — and that is the assertion. A
// handler passing a constant profile sends a correct body advertised as
// something the client did not ask for, which is invisible to any test that
// only reads the bytes of the body.
func TestPingEchoesTheProfileItWasAskedFor(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, testCase := range []struct {
		name        string
		accept      string
		contentType string
	}{
		{"the plain type", "application/json", "application/json; charset=utf-8"},
		{"the PascalCase profile", `application/json; profile="PascalCase"`, `application/json; profile="PascalCase"; charset=utf-8`},
		{"the CamelCase profile", `application/json; profile="CamelCase"`, `application/json; profile="CamelCase"; charset=utf-8`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			for _, method := range pingMethods {
				got := server.do(t, method, pingPath, "", http.Header{"Accept": {testCase.accept}})

				if contentType := got.header.Get("Content-Type"); contentType != testCase.contentType {
					t.Errorf("%s under %q: Content-Type: got %q, want %q", method, testCase.accept, contentType, testCase.contentType)
				}
				assertGolden(t, "system_ping.json", got.body)
			}
		})
	}
}
