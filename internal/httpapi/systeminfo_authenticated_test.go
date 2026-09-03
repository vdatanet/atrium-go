package httpapi_test

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/system"
)

// configuredInstallation is an installation whose setup has been finished,
// which is the state spec 3.2 refuses an uncredentialed request in. 001 has no
// way to reach it over HTTP — 002 owns the endpoint that finishes setup — so it
// is stated here through the store, which is why the 401 is asserted at this
// level and not in conformance/.
var configuredInstallation = ports.Installation{Name: "The Library", SetupCompleted: true}

// stubAuthenticator is the port 002 fills, standing in for it. It answers one
// decision, so a test names the state it is putting the server in rather than a
// credential the server could not recognise anyway.
type stubAuthenticator struct {
	access httpapi.Access
	err    error
}

func (s stubAuthenticator) Authenticate(*http.Request) (httpapi.Access, error) {
	return s.access, s.err
}

// systemInfo issues one GET /System/Info to the handler and hands back the
// recorder.
func systemInfo(t *testing.T, handler *httpapi.SystemHandler, decorate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/System/Info", nil)
	r.Host = "192.168.1.20:8096"
	r.RemoteAddr = "192.168.1.44:51000"
	if decorate != nil {
		decorate(r)
	}
	w := httptest.NewRecorder()
	handler.Info().ServeHTTP(w, r)
	return w
}

// The whole body, as bytes, on the installation state 001 can actually serve
// this route in: setup outstanding, which the reference admits without a
// credential.
//
// It is written out rather than assembled from the fields it is being compared
// against, because a test that builds its expectation the way the code does
// asserts that the code is self-consistent and nothing else. The three things
// only this form catches are the key order, the absence of PackageName, and the
// JSON *type* of every value — an array that had become null, a boolean that
// had become the string "false".
func TestSystemInfoAnswersSpecThreeTwosSuperset(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
	})
	w := systemInfo(t, handler, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	const want = `{"LocalAddress":"http://192.168.1.20:8096",` +
		`"ServerName":"atrium",` +
		`"Version":"10.11.11",` +
		`"ProductName":"Jellyfin Server",` +
		`"OperatingSystem":"",` +
		`"Id":"` + testInstallationID + `",` +
		`"StartupWizardCompleted":false,` +
		`"OperatingSystemDisplayName":"",` +
		`"HasPendingRestart":false,` +
		`"IsShuttingDown":false,` +
		`"SupportsLibraryMonitor":false,` +
		`"WebSocketPortNumber":34567,` +
		`"CompletedInstallations":[],` +
		`"CanSelfRestart":false,` +
		`"CanLaunchWebBrowser":false,` +
		`"ProgramDataPath":"/var/lib/atrium",` +
		`"WebPath":"/var/lib/atrium/web",` +
		`"ItemsByNamePath":"/var/lib/atrium/metadata",` +
		`"CachePath":"/var/lib/atrium/cache",` +
		`"LogPath":"/var/lib/atrium/log",` +
		`"InternalMetadataPath":"/var/lib/atrium/metadata",` +
		`"TranscodingTempPath":"/var/lib/atrium/cache/transcodes",` +
		`"CastReceiverApplications":[],` +
		`"HasUpdateAvailable":false,` +
		`"EncoderLocation":"",` +
		`"SystemArchitecture":""}`

	if got := w.Body.String(); got != want {
		t.Errorf("body:\n got %s\nwant %s", got, want)
	}
}

// AC-5's superset half, at the handler.
//
// It compares the two bodies' **raw JSON**, member by member, so that a shared
// field which had become a different JSON type — `"StartupWizardCompleted":
// "true"` beside `true` — fails here rather than agreeing after two decodes.
//
// The installation is a renamed one whose setup is finished, on purpose: with
// the defaults, four of the seven shared values are constants and two are
// derived from the same request, so a handler that rebuilt the public half
// wrongly could still agree. Here ServerName and StartupWizardCompleted both
// carry a value only the store can have supplied.
func TestSystemInfoAgreesWithThePublicBodyOnEverySharedField(t *testing.T) {
	t.Parallel()

	installations := fakeInstallations{installation: configuredInstallation}
	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: installations,
		// Setup is complete here, so the route needs a credential; this is the
		// state 002 will fill for real.
		Authenticator: stubAuthenticator{access: httpapi.AccessGranted},
	})

	public := publicInfo(t, handler, nil)
	if public.Code != http.StatusOK {
		t.Fatalf("the public route answered %d", public.Code)
	}
	authenticated := systemInfo(t, handler, nil)
	if authenticated.Code != http.StatusOK {
		t.Fatalf("the authenticated route answered %d: %s", authenticated.Code, authenticated.Body)
	}

	publicNames, publicFields := jsonMembers(t, public.Body.Bytes())
	_, authenticatedFields := jsonMembers(t, authenticated.Body.Bytes())

	if len(publicNames) != 7 {
		t.Fatalf("the public body carries %d fields, and spec 3.1 has seven: %v", len(publicNames), publicNames)
	}

	for _, name := range publicNames {
		got, present := authenticatedFields[name]
		if !present {
			t.Errorf("%s is in the public body and not in the authenticated one, so the second is not a superset of the first", name)
			continue
		}
		if string(got) != string(publicFields[name]) {
			t.Errorf("%s: the authenticated body says %s, the public one says %s", name, got, publicFields[name])
		}
	}

	if len(authenticatedFields) <= len(publicNames) {
		t.Errorf("the authenticated body carries %d fields and the public one %d; spec 3.2 makes the first a strict superset",
			len(authenticatedFields), len(publicNames))
	}
}

// behaviours 1.7 measured that the reference declares PackageName on this
// response and does not send it, so an empty string here would be a property no
// reference server carries.
//
// The assertion is on the bytes rather than on a decoded map, because a decoder
// answers the same zero value for a member that is absent and one that is
// present and empty — which is the whole of the distinction being made.
func TestSystemInfoDoesNotSendPackageName(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
	})
	if body := systemInfo(t, handler, nil).Body.String(); strings.Contains(body, "PackageName") {
		t.Errorf("the body names PackageName, which the reference declares and does not send:\n%s", body)
	}
}

// WebSocketPortNumber is the port this server is listening on, which the
// handler cannot know when it is built. A handler that answered a constant, or
// the port out of the request's Host header, passes every other test in this
// file.
func TestSystemInfoReportsThePortTheServerIsListeningOn(t *testing.T) {
	t.Parallel()

	const boundPort = 41999
	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
		HTTPPort:      func() int { return boundPort },
	})

	// The request names port 8096 in its Host header and arrives at a server
	// listening on 41999, so the two candidate answers are different numbers.
	if body := systemInfo(t, handler, nil).Body.String(); !contains(body, `"WebSocketPortNumber":41999`) {
		t.Errorf("the port the server is listening on did not reach WebSocketPortNumber:\n%s", body)
	}
}

// The port is read per request rather than at construction, because the entry
// layer only learns it after it has bound and the handler is built before that
// (app.Run).
func TestSystemInfoReadsThePortOnEveryRequest(t *testing.T) {
	t.Parallel()

	port := 0
	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
		HTTPPort:      func() int { return port },
	})

	if body := systemInfo(t, handler, nil).Body.String(); !contains(body, `"WebSocketPortNumber":0`) {
		t.Fatalf("a server that has not bound yet did not answer 0:\n%s", body)
	}

	port = 8096
	if body := systemInfo(t, handler, nil).Body.String(); !contains(body, `"WebSocketPortNumber":8096`) {
		t.Errorf("the second request did not see the bound port:\n%s", body)
	}
}

// spec 3.2: the reference permits this route during first-time setup, before
// any user exists — its authorisation handler succeeds on
// !IsStartupWizardCompleted before it looks at a role
// [source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31 @ v10.11.11].
//
// Both rows matter. A credential does not change the answer, because an
// unrecognised token is "no result" rather than a failure and authorisation
// still decides
// [source: Jellyfin.Api/Auth/CustomAuthenticationHandler.cs:48-51,79-83 @ v10.11.11]
// — so a server that refused a request *because* it carried a token nobody
// issued would be refusing exactly the client trying to complete its setup.
func TestSystemInfoIsServedDuringFirstTimeSetupWithoutACredential(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
	})

	for _, row := range []struct {
		what     string
		decorate func(*http.Request)
	}{
		{"no credential at all", nil},
		{"a token nothing issued", func(r *http.Request) { r.Header.Set("X-Emby-Token", "not-a-token") }},
	} {
		w := systemInfo(t, handler, row.decorate)
		if w.Code != http.StatusOK {
			t.Errorf("%s: status %d, want %d — setup is outstanding, and the reference admits the request",
				row.what, w.Code, http.StatusOK)
		}
	}
}

// The 401 of spec 3.2 and plan 7, in behaviours 1.11's shape.
//
// # Why this goes over a connection
//
// Three of the four things being asserted are invisible to a recorder. An
// httptest.ResponseRecorder synthesises none of what net/http adds on the way
// out, so it shows no Content-Length at all — and net/http only declares one on
// a body-less response for some methods, which is why the length is set
// explicitly in refusal.go and why a test that cannot see it proves nothing
// about it (T11's finding).
//
// # Why a token is sent as well
//
// Until 002 there is no credential this server could recognise, so "no token"
// and "a token" are the same state seen twice — and that is the point: both
// must be refused, and a handler that admitted anything carrying a header would
// pass a test that only ever sent none.
func TestSystemInfoRefusesWithoutACredentialOnceSetupIsComplete(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
	})

	for _, row := range []struct {
		what   string
		header []string
	}{
		{"no credential at all", nil},
		{"a token nothing issued", []string{`X-Emby-Token: not-a-token`}},
		{"an Authorization header", []string{`Authorization: MediaBrowser Token="not-a-token"`}},
	} {
		response := send(t, handler.Info(), http.MethodGet, "/System/Info", row.header...)
		assertEmptyRefusal(t, response, http.StatusUnauthorized, row.what)

		// The fourth thing 1.11 measures about this shape, and the one that is
		// only about the 401: the reference sends no WWW-Authenticate, where a
		// framework adding one for correctness's sake would put a difference on
		// every authenticated route at once.
		if response.has("WWW-Authenticate") {
			t.Errorf("%s: WWW-Authenticate = %v, want the header absent (behaviours 1.11)",
				row.what, response.values("WWW-Authenticate"))
		}
	}
}

// The port exists so that 002 can admit a request, and this is the assertion
// that it is wired to the answer rather than declared and ignored.
//
// **It does not prove AC-5's second half.** The credential here is a stub: no
// token exists, nothing issues one, and nothing validates one. What is proven
// is that the handler asks and obeys. "200 with a valid token" is carried into
// 002 (tasks.md T18).
func TestSystemInfoAdmitsWhatTheAuthenticatorAdmits(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: stubAuthenticator{access: httpapi.AccessGranted},
	})

	w := systemInfo(t, handler, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d: %s", w.Code, http.StatusOK, w.Body)
	}
	if body := w.Body.String(); !contains(body, `"ServerName":"The Library"`) || !contains(body, `"StartupWizardCompleted":true`) {
		t.Errorf("the admitted request did not get the configured installation's own body:\n%s", body)
	}
}

// An authenticator that could not decide is not an authenticator that said no.
//
// A client answered 401 discards its credential and logs in again, so
// answering one because a store was briefly unreadable makes every client in
// the house re-authenticate over a transient fault.
func TestSystemInfoDoesNotTurnAFailureToDecideIntoARefusal(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: stubAuthenticator{err: errors.New("the session store is unreadable")},
	})

	if w := systemInfo(t, handler, nil); w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// The rule internal/wire set for an unknown Profile, held here for an unknown
// Access: a value this package does not recognise is an error, never a
// fall-through.
//
// Both directions a fall-through could take are wrong silently. This is the
// test that fails the day 002 adds the 403 without teaching the handler what to
// do with it — which is the day it would otherwise start answering 200 or 401
// to a caller the reference forbids.
func TestSystemInfoRefusesAnAccessItDoesNotUnderstand(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: configuredInstallation},
		Authenticator: stubAuthenticator{access: httpapi.Access(99)},
	})

	w := systemInfo(t, handler, nil)
	if w.Code == http.StatusOK {
		t.Errorf("an Access this package does not know admitted the request: %s", w.Body)
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// AccessUnauthenticated is the zero value, so a value nobody filled in refuses
// rather than admits. The direction is the whole point and it is one keystroke
// wide.
func TestTheZeroAccessAdmitsNobody(t *testing.T) {
	t.Parallel()

	var zero httpapi.Access
	if zero != httpapi.AccessUnauthenticated {
		t.Errorf("the zero Access is %d, and it must be AccessUnauthenticated (%d)", zero, httpapi.AccessUnauthenticated)
	}
	if zero == httpapi.AccessGranted {
		t.Error("the zero Access admits the request, which is the wrong direction for a value that decides admission")
	}
}

// A store that cannot be read is a 500, on this route as on the public one —
// and, in particular, not a 401. The setup state comes out of the store, so a
// handler that treated an unreadable store as "setup is complete" would refuse
// every request with a credential nobody could supply.
func TestSystemInfoRefusesRatherThanGuessingTheSetupState(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{err: errors.New("the store is unreadable")},
	})

	if w := systemInfo(t, handler, nil); w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// AC-9 on this route: the profile the request negotiated is the one the body is
// written under, and the content type says which.
func TestSystemInfoAnswersUnderTheNegotiatedProfile(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
	})
	w := systemInfo(t, handler, func(r *http.Request) {
		r.Header.Set("Accept", `application/json; profile="CamelCase"`)
	})

	if got, want := w.Header().Get("Content-Type"), `application/json; profile="CamelCase"; charset=utf-8`; got != want {
		t.Errorf("content type: got %q, want %q", got, want)
	}
	// One name from each half of the superset, so that a renamer which had
	// stopped at the embedded struct's boundary fails here.
	for _, want := range []string{`"productName":"Jellyfin Server"`, `"webSocketPortNumber":34567`, `"itemsByNamePath":`} {
		if !contains(w.Body.String(), want) {
			t.Errorf("the camelCase profile did not carry %s:\n%s", want, w.Body.String())
		}
	}
}

// The seven paths are one layout, derived once, rather than seven strings the
// handler assembles. ItemsByNamePath and InternalMetadataPath carry the same
// value, which is the reference's own assignment
// [source: Emby.Server.Implementations/SystemManager.cs:71-72 @ v10.11.11].
func TestSystemInfoReportsTheInstallationsOwnPaths(t *testing.T) {
	t.Parallel()

	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{
		Installations: fakeInstallations{installation: freshInstallation},
		Paths:         system.PathsFor("/srv/atrium-data"),
	})

	_, fields := jsonMembers(t, systemInfo(t, handler, nil).Body.Bytes())
	for _, row := range []struct{ field, want string }{
		{"ProgramDataPath", `"/srv/atrium-data"`},
		{"WebPath", `"/srv/atrium-data/web"`},
		{"ItemsByNamePath", `"/srv/atrium-data/metadata"`},
		{"CachePath", `"/srv/atrium-data/cache"`},
		{"LogPath", `"/srv/atrium-data/log"`},
		{"InternalMetadataPath", `"/srv/atrium-data/metadata"`},
		{"TranscodingTempPath", `"/srv/atrium-data/cache/transcodes"`},
	} {
		if got := string(fields[row.field]); got != row.want {
			t.Errorf("%s: got %s, want %s", row.field, got, row.want)
		}
	}
}

// jsonMembers reads an object's members in the order the bytes carry them,
// keeping each value as the raw JSON it was written as.
//
// Raw, because Principle VIII: a decoded value has already lost the difference
// between 8096 and "8096", between an absent member and a null one, and between
// [] and null.
func jsonMembers(t *testing.T, body []byte) ([]string, map[string]json.RawMessage) {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	open, err := decoder.Token()
	if err != nil {
		t.Fatalf("reading the body: %v\n%s", err, body)
	}
	if delimiter, ok := open.(json.Delim); !ok || delimiter != '{' {
		t.Fatalf("the body is not a JSON object: %s", body)
	}

	var names []string
	values := map[string]json.RawMessage{}
	for decoder.More() {
		name, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading a property name: %v\n%s", err, body)
		}
		text, ok := name.(string)
		if !ok {
			t.Fatalf("a property name is not a string: %v", name)
		}
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil && err != io.EOF {
			t.Fatalf("reading the value of %s: %v", text, err)
		}
		names = append(names, text)
		values[text] = value
	}
	return names, values
}
