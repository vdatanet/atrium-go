package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/system"
	"github.com/vdatanet/atrium-go/internal/units"
)

// testInstallationID is a well-formed identity: 32 lowercase hex, as
// internal/system validates one. The handler must put it on the wire verbatim.
const testInstallationID = "3f9c1a7e5b2d4e8091a6c3f70d5e2b14"

// fakeInstallations is the store the handler reads the friendly name and the
// setup state from. A failure is a value here rather than a broken database,
// because what is under test is the handler's answer to one, not sqlite's.
type fakeInstallations struct {
	installation ports.Installation
	err          error
}

func (f fakeInstallations) Installation(context.Context) (ports.Installation, error) {
	return f.installation, f.err
}

func (f fakeInstallations) SetServerName(context.Context, string) error { return nil }

func (f fakeInstallations) MarkSetupComplete(context.Context, units.Time) error { return nil }

// freshInstallation is what plan 4 says an installation that has never been
// configured holds: the default name, and setup not finished.
var freshInstallation = ports.Installation{Name: "atrium", SetupCompleted: false}

func newSystemHandler(t *testing.T, installations ports.InstallationStore, addresses system.AddressConfig) *httpapi.SystemHandler {
	t.Helper()
	handler, err := httpapi.NewSystemHandler(testInstallationID, installations, addresses)
	if err != nil {
		t.Fatalf("building the system handler: %v", err)
	}
	return handler
}

// publicInfo issues one request to the handler and hands back the recorder,
// so that a test asserts on bytes rather than on a decoded object
// (Principle VIII).
func publicInfo(t *testing.T, handler *httpapi.SystemHandler, decorate func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
	r.Host = "192.168.1.20:8096"
	r.RemoteAddr = "192.168.1.44:51000"
	if decorate != nil {
		decorate(r)
	}
	w := httptest.NewRecorder()
	handler.PublicInfo().ServeHTTP(w, r)
	return w
}

// The seven fields, in order, with the three literal values spec 3.1 fixes.
// This is the handler-level half of AC-1, AC-2 and AC-3; the conformance golden
// is the same assertion over a real server.
func TestPublicInfoAnswersTheSevenFieldsOfSpecThreeOne(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{})
	w := publicInfo(t, handler, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("status: got %d, want %d", w.Code, http.StatusOK)
	}

	const want = `{"LocalAddress":"http://192.168.1.20:8096",` +
		`"ServerName":"atrium",` +
		`"Version":"10.11.11",` +
		`"ProductName":"Jellyfin Server",` +
		`"OperatingSystem":"",` +
		`"Id":"` + testInstallationID + `",` +
		`"StartupWizardCompleted":false}`

	if got := w.Body.String(); got != want {
		t.Errorf("body:\n got %s\nwant %s", got, want)
	}
}

// ProductName is the discriminator a multi-server client reads, and Version is
// what it gates capabilities on. Neither may be this project's own name or this
// binary's own version (behaviours 4.1, reference-target.md 4). Asserting them
// against the constants alone would pass for any two strings, so the literals
// are written out here.
func TestPublicInfoIdentifiesAsJellyfinAndNotAsAtrium(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{})
	body := publicInfo(t, handler, nil).Body.String()

	for _, want := range []string{`"ProductName":"Jellyfin Server"`, `"Version":"10.11.11"`, `"OperatingSystem":""`} {
		if !contains(body, want) {
			t.Errorf("the body does not carry %s:\n%s", want, body)
		}
	}
	for _, unwanted := range []string{"Atrium", "atrium/"} {
		if contains(body, unwanted) {
			t.Errorf("the body names %q, which no field clients parse may carry:\n%s", unwanted, body)
		}
	}
}

// The handler must hand wire.Write the profile this request negotiated, not a
// constant. A handler that passed ProfilePlain sends a correct-looking body
// under a content type that lies about what was asked for, which is AC-9's
// failure mode and is invisible to a test that only reads the values.
func TestPublicInfoAnswersUnderTheNegotiatedProfile(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{})
	w := publicInfo(t, handler, func(r *http.Request) {
		r.Header.Set("Accept", `application/json; profile="CamelCase"`)
	})

	if got, want := w.Header().Get("Content-Type"), `application/json; profile="CamelCase"; charset=utf-8`; got != want {
		t.Errorf("content type: got %q, want %q", got, want)
	}
	if got := w.Body.String(); !contains(got, `"productName":"Jellyfin Server"`) {
		t.Errorf("the camelCase profile did not rename the properties:\n%s", got)
	}
}

// LocalAddress is spec 3.4's three tiers, and the handler's job is to reach
// them with facts read off the request. A published URL is the tier a request
// cannot influence, so it is what proves the configuration arrives at all.
func TestPublicInfoReportsTheConfiguredPublishedURL(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{
		PublishedURL: "https://jellyfin.example/",
	})
	if got := publicInfo(t, handler, nil).Body.String(); !contains(got, `"LocalAddress":"https://jellyfin.example"`) {
		t.Errorf("the published URL did not reach LocalAddress:\n%s", got)
	}
}

// And the per-caller tier, which is the one that proves the requester's own
// address reached the domain rather than the zero value: two requesters on two
// networks get two answers (AC-8).
func TestPublicInfoAnswersEachNetworkWithItsOwnAddress(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{
		BoundAddresses: []system.BoundAddress{
			{Address: netip.MustParseAddr("192.168.1.20"), Subnet: netip.MustParsePrefix("192.168.1.0/24"), Scheme: "http", Port: 8096},
			{Address: netip.MustParseAddr("10.8.0.1"), Subnet: netip.MustParsePrefix("10.8.0.0/24"), Scheme: "http", Port: 8096},
		},
	})

	for _, row := range []struct{ remoteAddr, want string }{
		{"192.168.1.44:51000", `"LocalAddress":"http://192.168.1.20:8096"`},
		{"10.8.0.66:51000", `"LocalAddress":"http://10.8.0.1:8096"`},
	} {
		body := publicInfo(t, handler, func(r *http.Request) { r.RemoteAddr = row.remoteAddr }).Body.String()
		if !contains(body, row.want) {
			t.Errorf("a requester at %s was not told %s:\n%s", row.remoteAddr, row.want, body)
		}
	}
}

// The friendly name and the setup state come from the store on every request,
// because 002 renames the server through the same port while this process keeps
// running. A handler that read them once at construction would answer the old
// name for ever.
func TestPublicInfoReadsTheInstallationOnEveryRequest(t *testing.T) {
	t.Parallel()

	installations := &mutableInstallations{installation: freshInstallation}
	handler := newSystemHandler(t, installations, system.AddressConfig{})

	if got := publicInfo(t, handler, nil).Body.String(); !contains(got, `"ServerName":"atrium"`) || !contains(got, `"StartupWizardCompleted":false`) {
		t.Fatalf("a fresh installation did not answer its defaults:\n%s", got)
	}

	installations.installation = ports.Installation{Name: "The Library", SetupCompleted: true}

	got := publicInfo(t, handler, nil).Body.String()
	if !contains(got, `"ServerName":"The Library"`) || !contains(got, `"StartupWizardCompleted":true`) {
		t.Errorf("the second request did not see the renamed installation:\n%s", got)
	}
}

// mutableInstallations answers whatever it currently holds, so that a test can
// change the installation between two requests.
type mutableInstallations struct {
	installation ports.Installation
}

func (m *mutableInstallations) Installation(context.Context) (ports.Installation, error) {
	return m.installation, nil
}

func (m *mutableInstallations) SetServerName(context.Context, string) error { return nil }

func (m *mutableInstallations) MarkSetupComplete(context.Context, units.Time) error { return nil }

// A store that cannot be read is a 500 with an empty body: the status is the
// only thing behaviours 1.11 leaves this handler certain about, and plan 7
// records the shape as owed. What matters here is that it is not a 200 carrying
// an empty ServerName, which is the shape an ignored error produces.
func TestPublicInfoRefusesRatherThanInventingAnInstallation(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{err: errors.New("the store is unreadable")}, system.AddressConfig{})
	w := publicInfo(t, handler, nil)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: got %d, want %d", w.Code, http.StatusInternalServerError)
	}
	if body := w.Body.String(); body != "" {
		t.Errorf("body: got %q, want it empty", body)
	}
	if contentType := w.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Content-Type: got %q, want none", contentType)
	}
}

// The constructor's inputs come from the process's own start, so a missing one
// is a failure to start rather than a handler that answers half a body.
func TestNewSystemHandlerRefusesWhatItCannotAnswerWith(t *testing.T) {
	t.Parallel()

	if _, err := httpapi.NewSystemHandler("", fakeInstallations{}, system.AddressConfig{}); err == nil {
		t.Error("an empty installation identity was accepted")
	}
	if _, err := httpapi.NewSystemHandler(testInstallationID, nil, system.AddressConfig{}); err == nil {
		t.Error("a missing installation store was accepted")
	}
}

// Registration goes through the route table, so the pattern a handler answers
// on is surface.yaml's spelling of it and never a second copy typed here.
func TestRoutesRegistersOnTheSurfaceFilesOwnPattern(t *testing.T) {
	t.Parallel()

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{})
	register, err := httpapi.Routes(surface.V1(), httpapi.Handlers{System: handler})
	if err != nil {
		t.Fatalf("building the registration callback: %v", err)
	}

	router := chi.NewRouter()
	register(router)

	registered := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		registered[method+" "+route] = true
		return nil
	}); err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	// The three rows 001 has registered so far, spelled as surface.yaml spells
	// them. /System/Ping appears twice, on the two methods its two operations
	// are named with — which is what a registration keyed by operation buys
	// and what a registration keyed by path could not express (spec 3.3).
	for _, want := range []string{
		"GET /System/Info/Public",
		"GET /System/Ping",
		"POST /System/Ping",
	} {
		if !registered[want] {
			t.Errorf("the router does not serve %q; it serves %v", want, registered)
		}
	}
	if len(registered) != 3 {
		t.Errorf("the router serves %d routes, and 001 has registered three so far: %v", len(registered), registered)
	}
}

// A handler that is not supplied registers nothing, which is what lets a test
// about a stage build a pipeline that routes nothing at all.
func TestRoutesRegistersNothingForAHandlerItWasNotGiven(t *testing.T) {
	t.Parallel()

	register, err := httpapi.Routes(surface.V1(), httpapi.Handlers{})
	if err != nil {
		t.Fatalf("building the registration callback: %v", err)
	}

	router := chi.NewRouter()
	register(router)

	count := 0
	if err := chi.Walk(router, func(string, string, http.Handler, ...func(http.Handler) http.Handler) error {
		count++
		return nil
	}); err != nil {
		t.Fatalf("walking the router: %v", err)
	}
	if count != 0 {
		t.Errorf("a Handlers with no handlers in it registered %d routes", count)
	}
}

// A row the document no longer has is a failure to start naming the operation,
// rather than a router quietly serving a pattern surface.yaml does not contain.
func TestRoutesRefusesATableWithoutItsRow(t *testing.T) {
	t.Parallel()

	table, err := surface.Load([]byte(strings.Join([]string{
		"reference:",
		`  jellyfin_openapi_version: "10.11.11"`,
		`  jellyfin_source_tag: "v10.11.11"`,
		"endpoints:",
		`  - path: "/System/Ping"`,
		"    method: GET",
		"    operation: GetPingSystem",
		"    consumers: []",
		`    feature: "001"`,
		"    level: L2",
		"",
	}, "\n")))
	if err != nil {
		t.Fatalf("loading the fixture table: %v", err)
	}

	handler := newSystemHandler(t, fakeInstallations{installation: freshInstallation}, system.AddressConfig{})
	if _, err := httpapi.Routes(table, httpapi.Handlers{System: handler}); err == nil {
		t.Error("a table with no GetPublicSystemInfo row was accepted")
	} else if !contains(err.Error(), "GetPublicSystemInfo") {
		t.Errorf("the error does not name the operation: %v", err)
	}
}

func contains(haystack, needle string) bool { return strings.Contains(haystack, needle) }
