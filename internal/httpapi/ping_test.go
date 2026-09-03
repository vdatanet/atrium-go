package httpapi_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/system"
)

// pingBody is the whole response of spec 3.3, as bytes: a bare JSON string
// carrying the product name.
//
// It is written out here rather than built from system.ProductName, because a
// test that composes its expectation from the same constant the handler reads
// agrees with the handler about the wrong value just as readily as about the
// right one. This is the literal spec 3.3 prints.
const pingBody = `"Jellyfin Server"`

// renamedInstallation is the fixture that makes this a test of the product
// name rather than a test that two strings happen to differ.
//
// The friendly name is deliberately unlike the product name in every way a
// mistake could survive: it is not "Jellyfin Server", it does not contain it,
// and it is not the "atrium" default either — so a handler that returned the
// friendly name would fail here whatever the default becomes, and the
// assertion does not quietly rest on a value plan 4 owns.
//
// It is also, deliberately, a *plausible* operator name. An installation
// nobody has renamed carries "atrium", and a ping that returned ServerName
// would already fail against that; but the day a fresh installation is named
// something else is the day such a test stops proving anything, and this one
// does not depend on that day never arriving.
var renamedInstallation = ports.Installation{Name: "Front Room Media", SetupCompleted: true}

// ping issues one request to the ping handler and hands back the recorder, so
// that a test asserts on bytes rather than on a decoded value (Principle VIII).
func ping(t *testing.T, handler *httpapi.SystemHandler, method string, decorate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, "/System/Ping", nil)
	if decorate != nil {
		decorate(r)
	}
	w := httptest.NewRecorder()
	handler.Ping().ServeHTTP(w, r)
	return w
}

// pingMethods is the pair spec 3.3 gives one response: both, every time,
// because "both methods" is half of what AC-6 asks.
var pingMethods = []string{http.MethodGet, http.MethodPost}

// AC-6: both methods answer 200 with the JSON string "Jellyfin Server".
//
// The body is compared as bytes and in full, which is what tells the required
// response apart from three plausible near misses at once: an object with one
// field, an empty body, and the same string under a different name.
func TestPingAnswersTheProductNameOnBothMethods(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: renamedInstallation}, system.AddressConfig{})

	for _, method := range pingMethods {
		t.Run(method, func(t *testing.T) {
			w := ping(t, handler, method, nil)

			if w.Code != http.StatusOK {
				t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
			}
			if got := w.Body.String(); got != pingBody {
				t.Errorf("body: got %s, want %s", got, pingBody)
			}
			if got, want := w.Header().Get("Content-Type"), "application/json; charset=utf-8"; got != want {
				t.Errorf("content type: got %q, want %q", got, want)
			}
		})
	}
}

// The test above proves nothing unless the friendly name was a value the
// handler could have returned, and this is what says it was.
//
// The same handler, holding the same store, puts "Front Room Media" on the
// wire when asked for /System/Info/Public — so ServerName is reachable, is
// populated, and is not the product name. spec 3.3's compatibility note is
// exactly this trap: the reference's own documentation comment says the
// operation returns "the server name", and a reimplementation that followed
// the comment would pass a ping test written against a fixture whose friendly
// name nobody set.
//
// The two halves are asserted together, in one test, because separating them
// is how the second one gets deleted as redundant.
func TestPingIgnoresTheFriendlyNameTheSameHandlerReports(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: renamedInstallation}, system.AddressConfig{})

	// The friendly name is a value this fixture set, and it is unlike the
	// product name. If this ever stops holding, every ping assertion in this
	// file has stopped discriminating and this is the line that says so.
	if renamedInstallation.Name == system.ProductName {
		t.Fatalf("the fixture's friendly name is the product name, so no ping test in this file can fail on the difference between them")
	}

	// It reaches the wire on the response that does report it...
	info := publicInfo(t, handler, nil).Body.String()
	if want := `"ServerName":"` + renamedInstallation.Name + `"`; !contains(info, want) {
		t.Fatalf("the fixture's friendly name did not reach /System/Info/Public, so the ping assertion below is vacuous.\nwant %s in:\n%s", want, info)
	}

	// ...and not on the one that must not.
	for _, method := range pingMethods {
		got := ping(t, handler, method, nil).Body.String()
		if contains(got, renamedInstallation.Name) {
			t.Errorf("%s /System/Ping answered %s, which carries the operator's friendly name; spec 3.3 requires the product name", method, got)
		}
		if got != pingBody {
			t.Errorf("%s /System/Ping: body: got %s, want %s", method, got, pingBody)
		}
	}
}

// spec 3.0 applies to every response in this specification, ping included, so
// the content type echoes the profile that matched (AC-9).
//
// A bare JSON string carries no property names, so all three profiles produce
// one body — which is the assertion, not an excuse to skip the negotiation. A
// handler that passed a constant profile would send a correct body under a
// content type that lies about what was asked for, and nothing about the bytes
// of this particular response would show it.
func TestPingEchoesTheNegotiatedProfileOverAnUnchangedBody(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: renamedInstallation}, system.AddressConfig{})

	for _, testCase := range []struct {
		name        string
		accept      string
		contentType string
	}{
		{"no Accept at all", "", "application/json; charset=utf-8"},
		{"the plain type", "application/json", "application/json; charset=utf-8"},
		{"the PascalCase profile", `application/json; profile="PascalCase"`, `application/json; profile="PascalCase"; charset=utf-8`},
		{"the CamelCase profile", `application/json; profile="CamelCase"`, `application/json; profile="CamelCase"; charset=utf-8`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			for _, method := range pingMethods {
				w := ping(t, handler, method, func(r *http.Request) {
					if testCase.accept != "" {
						r.Header.Set("Accept", testCase.accept)
					}
				})
				if got := w.Header().Get("Content-Type"); got != testCase.contentType {
					t.Errorf("%s: content type: got %q, want %q", method, got, testCase.contentType)
				}
				if got := w.Body.String(); got != pingBody {
					t.Errorf("%s: body under %q: got %s, want %s — a bare string has no property name to rename", method, testCase.accept, got, pingBody)
				}
			}
		})
	}
}
