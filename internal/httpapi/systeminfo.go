package httpapi

import (
	"errors"
	"net/http"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/system"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// PublicSystemInfo is the body of GET /System/Info/Public: spec 3.1's seven
// fields, and no others.
//
// # Seven, exactly
//
// The reference builds this response from six assignments and never sets
// OperatingSystem, which keeps the empty string its declaration gives it
// [source: Emby.Server.Implementations/SystemManager.cs:112-125 @ v10.11.11]
// [source: MediaBrowser.Model/System/PublicSystemInfo.cs:37-38 @ v10.11.11].
// Principle I is the rest: an eighth field is a delta whether or not any client
// reads it, and a missing seventh is a client on the unknown-server path.
//
// # The declaration order is the wire order, and it is spec 3.1's
//
// Go writes a struct's fields in declaration order, so this list is the key
// order of every body this server sends — and L3 compares bytes, so key order
// is contract rather than presentation.
//
// The order below is spec 3.1's table, which is also the reference model's own
// declaration order
// [source: MediaBrowser.Model/System/PublicSystemInfo.cs:14-53 @ v10.11.11].
// ⚠️ UNVERIFIED that this is the order the reference actually *sends*: no probe
// in this repository records the key order of a body, the two sample bodies
// here disagree about it (spec 3.1 opens with LocalAddress, reference-target.md
// 4 opens with ServerName), and .NET's serialiser writing properties in
// declaration order is a property of that serialiser rather than something
// measured. It is one request to settle, and until it is settled it is a
// difference 010's differential run would surface.
//
// # Why the fields are not pointers
//
// ADR-0002 makes an optional field a pointer, and none of these is optional.
// The reference declares StartupWizardCompleted as a nullable bool — "nullable
// for OpenAPI specification only to retain backwards compatibility in api
// clients"
// [source: MediaBrowser.Model/System/PublicSystemInfo.cs:49-53 @ v10.11.11] —
// and assigns it on every construction, so it never serialises as null and a
// pointer here would be an optionality the wire does not have. OperatingSystem
// is an empty string rather than a null, which is why behaviours 1.7's
// omit-when-null rule does not reach it either.
//
// # Why Id is spelled Id
//
// The property name on the wire is the field name here: the PascalCase policy
// writes names exactly as a model declares them (internal/wire). Go's
// convention would spell an identifier ID, and doing so would rename the
// property. The wire's spelling wins in a response model, everywhere in this
// project.
type PublicSystemInfo struct {
	// LocalAddress is the address this server advertises to this requester,
	// chosen by spec 3.4's three tiers.
	LocalAddress string

	// ServerName is the operator-chosen friendly name. A fresh installation
	// carries "atrium" (spec 3.1, plan 4).
	ServerName string

	// Version is the pinned reference version, not this binary's own — see
	// system.ReportedVersion.
	Version string

	// ProductName is exactly "Jellyfin Server" (behaviours 4.1).
	ProductName string

	// OperatingSystem is always the empty string.
	OperatingSystem string

	// Id is the installation identity: 32 lowercase hex, generated once and
	// persisted beside the store so that it survives a rebuild of it (AC-4).
	Id string

	// StartupWizardCompleted reports whether initial setup is finished.
	StartupWizardCompleted bool
}

// SystemInfo is the body of GET /System/Info: spec 3.2's superset of the seven
// public fields.
//
// # The superset is structural, not asserted
//
// AC-5 asks that this body agree with /System/Info/Public "on every shared
// field". The cheapest way to guarantee that is for there to be nothing to
// keep in agreement: PublicSystemInfo is embedded, encoding/json flattens an
// embedded struct into its parent, and the seven shared members are therefore
// the *same seven values* the public route sends, filled in by the same
// function. A test can still fail here — and one does, over the wire, on bytes
// — but what it is checking is that the two routes were reached, not that two
// lists of assignments were kept in step.
//
// The alternative, twenty-seven fields written out flat and seven of them
// filled in twice, is a body that agrees today and drifts on the first change
// to either route. This is the difference between a property and a promise.
//
// # Key order, and what is not known about it
//
// Go writes an embedded struct's fields where the embedded field sits, so this
// is the seven public fields in spec 3.1's order followed by spec 3.2's
// additions in the reference model's own declaration order
// [source: MediaBrowser.Model/System/SystemInfo.cs:29-143 @ v10.11.11].
//
// ⚠️ UNVERIFIED that this is the order the reference sends. It is a stronger
// doubt than the one on PublicSystemInfo: there the model's declaration order
// is at least the order a serialiser walking one type's properties would
// produce, whereas SystemInfo *derives* from PublicSystemInfo, and where a
// serialiser puts an inherited property relative to a declared one is a
// property of that serialiser which no probe here has recorded. Both orders
// are plausible and this project has measured neither. One request settles it;
// until then it is a difference 010's differential run would surface, and the
// route is L2 rather than L3 so no golden in this repository asserts it.
//
// # What this response does not carry
//
// PackageName is declared below and deliberately never sent — see the field.
// Nothing else the reference declares is missing, and nothing it does not
// declare is here: Principle I is a count as much as it is a spelling.
type SystemInfo struct {
	// The seven fields of spec 3.1, flattened. Not a member named
	// "PublicSystemInfo": an embedded struct with no tag is promoted by
	// encoding/json, so these arrive at the top level of the object.
	PublicSystemInfo

	// OperatingSystemDisplayName is empty, for the same reason
	// OperatingSystem is: the reference marks the property obsolete and never
	// assigns it, leaving the empty string its declaration gives it
	// [source: MediaBrowser.Model/System/SystemInfo.cs:28-29 @ v10.11.11].
	OperatingSystemDisplayName string

	// PackageName is the one member of this model that is never sent.
	//
	// The reference declares it and does not send it — measured, not inferred:
	// its whole JSON pipeline omits a null property, and /System/Info is the
	// worked example behaviours 1.7 gives for the rule
	// [probe: tools/probe_public_info.py, Jellyfin 10.11.11, 2026-08-28]. The
	// value would be the '-package' command-line argument's, which nothing
	// sets [source: Emby.Server.Implementations/SystemManager.cs:80 @
	// v10.11.11], and Atrium has no such concept to set it from either.
	//
	// spec 3.2's table asks for "empty string otherwise" and spec 3.0.3 —
	// "absent optional values are omitted or null exactly as the reference
	// server does, verified per field rather than by rule" — is what decides
	// between them, because for this field the per-field verification exists.
	// An empty string here would be a property the reference never sends,
	// which is a delta by Principle I on the count. spec 3.2 carries the
	// amendment, dated.
	//
	// It is declared rather than left out so that the absence is a decision a
	// reader can see, with the measurement beside it, instead of a field
	// somebody notices is missing and adds. ADR-0002 makes an optional field a
	// pointer, and this is the only optional field in 001.
	PackageName *string `json:",omitempty"`

	// HasPendingRestart is false: Atrium has no self-update and nothing that
	// asks for a restart (spec 3.2).
	HasPendingRestart bool

	// IsShuttingDown is false. A request answered while this process is
	// draining is answered by a process that is stopping, but 001 has no way
	// to observe that from a handler and spec 3.2 fixes the value.
	IsShuttingDown bool

	// SupportsLibraryMonitor is false in v1: filesystem watching is not
	// implemented (spec 3.2). The reference answers true unconditionally
	// [source: Emby.Server.Implementations/SystemManager.cs:79 @ v10.11.11],
	// so this is a deliberate difference and it is honest rather than
	// cosmetic — a client that started a watch-dependent flow on the strength
	// of a true here would wait for a notification that never came.
	SupportsLibraryMonitor bool

	// WebSocketPortNumber is the port this server is actually listening on.
	//
	// v1 serves no WebSocket. The field is a number clients read
	// unconditionally (spec 3.2), so it is answered with the truth about where
	// this server can be reached rather than with a zero that reads as "no
	// port" or a constant that reads as a promise.
	WebSocketPortNumber int

	// CompletedInstallations is empty: Atrium installs no packages (spec 3.2).
	//
	// The element type is deliberately unspecified. The reference's is
	// InstallationInfo, a model of a package manager Atrium does not have, and
	// declaring its members here would be a schema for a value nothing can
	// produce — Principle VI's plausible-looking stub, in a response body. The
	// feature that ever fills this array declares the type it fills it with.
	CompletedInstallations []any

	// CanSelfRestart is false: Atrium cannot restart itself (spec 3.2). The
	// reference's is always true and marked obsolete saying so
	// [source: MediaBrowser.Model/System/SystemInfo.cs:67-69 @ v10.11.11].
	CanSelfRestart bool

	// CanLaunchWebBrowser is false, which is also what the reference always
	// answers [source: MediaBrowser.Model/System/SystemInfo.cs:71-73 @
	// v10.11.11].
	CanLaunchWebBrowser bool

	// The seven paths of spec 3.2, from system.Paths. They differ on every
	// installation, which is why 010's request case for this route calls them
	// "triage rather than allowlist rows" (request-cases.yaml).
	ProgramDataPath      string
	WebPath              string
	ItemsByNamePath      string
	CachePath            string
	LogPath              string
	InternalMetadataPath string
	TranscodingTempPath  string

	// CastReceiverApplications is empty: Atrium ships no cast receiver
	// (spec 3.2). The element type is unspecified for the same reason
	// CompletedInstallations' is.
	CastReceiverApplications []any

	// HasUpdateAvailable is false: Atrium checks for no update (spec 3.2), and
	// the reference's is false and marked obsolete saying updates are the
	// package manager's business
	// [source: MediaBrowser.Model/System/SystemInfo.cs:133-135 @ v10.11.11].
	HasUpdateAvailable bool

	// EncoderLocation is empty. v1 negotiates no playback and holds no
	// encoder, so there is no location to report; the reference's default is
	// the string "System" under an obsolete marker saying it "isn't set
	// correctly anymore"
	// [source: MediaBrowser.Model/System/SystemInfo.cs:137-139 @ v10.11.11],
	// which is spec 3.2's "empty string otherwise" rather than its "real
	// values where meaningful".
	EncoderLocation string

	// SystemArchitecture is empty, and that is a choice rather than an
	// oversight.
	//
	// The reference never assigns it and its default is the literal "X64"
	// under an obsolete marker saying it "is no longer set"
	// [source: MediaBrowser.Model/System/SystemInfo.cs:141-143 @ v10.11.11],
	// so the value it sends is true of no machine in particular. Copying that
	// constant would be asserting something false about this host; reporting
	// this host's real architecture would put a string on the wire that no
	// reference server sends, on a field whose own declaration says it is not
	// set. spec 3.2's rule — real where meaningful, empty otherwise — makes it
	// empty, and plan 8 records the difference for 010 to surface.
	SystemArchitecture string
}

// SystemHandlerConfig is what the /System handlers are built from.
//
// It is a struct rather than a parameter list because it grew past the point
// where a reader could tell the arguments apart at a call site, and because
// the features after this one add to it: 002 fills Authenticator, and whichever
// feature gives an operator a published URL fills Addresses.
type SystemHandlerConfig struct {
	// InstallationID is the Id field, verbatim, as internal/system validated
	// it before the server bound.
	InstallationID string

	// Installations is where ServerName and StartupWizardCompleted come from.
	Installations ports.InstallationStore

	// Addresses is the configuration half of spec 3.4's three tiers.
	Addresses system.AddressConfig

	// Paths is the layout spec 3.2's seven path fields report.
	Paths system.Paths

	// HTTPPort answers the port this server is listening on, for
	// WebSocketPortNumber.
	//
	// It is a function because of the order a start happens in: the handlers
	// are built before the pipeline, the pipeline before the server, and the
	// server is what binds — so at the moment this handler is constructed the
	// port is not known, and when --bind-address names port 0 it is not even
	// knowable. The entry layer fills the answer in after it binds and before
	// it serves.
	HTTPPort func() int

	// Authenticator decides whether a credential admits a request to
	// /System/Info. 001 leaves it nil, which admits nobody; 002 fills it.
	Authenticator Authenticator
}

// SystemHandler answers the /System routes of feature 001.
//
// It holds what those responses are built from and nothing else: the
// installation identity the entry layer read before it bound, the store the
// friendly name and the setup state come from, and the address configuration
// spec 3.4's tiers consult. Everything per-request is read off the request.
type SystemHandler struct {
	// installationID is the Id field, verbatim. internal/system has already
	// validated it as 32 lowercase hex, so nothing here re-derives or re-cases
	// it: a second spelling of an identifier is a second identifier.
	installationID string

	// installations is where ServerName and StartupWizardCompleted come from.
	// It is read per request rather than cached, because 002 renames the
	// server through the same port while this process keeps running.
	installations ports.InstallationStore

	// addresses is the configuration half of spec 3.4. The per-request half is
	// requestFacts.
	addresses system.AddressConfig

	// paths is the layout the seven path fields of spec 3.2 report. It is a
	// value because it is configuration: derived once, from --data-dir.
	paths system.Paths

	// httpPort answers the port this server is listening on, for
	// WebSocketPortNumber. See SystemHandlerConfig for why it is a function.
	httpPort func() int

	// authenticator is nil until 002 fills it, and a nil one admits nobody.
	// That is not a placeholder: a server that has issued no token really does
	// recognise no credential.
	authenticator Authenticator
}

// NewSystemHandler builds the handler for the /System routes.
//
// It returns an error rather than panicking, for the reason every stage in this
// package does: its inputs come from the process's own start, so a failure here
// is a failure to start (plan 7) and the entry layer is where that is reported.
//
// The address configuration is a value rather than a pointer because it is
// configuration: the same for every request, decided once. 001 gives an
// operator no way to set any of it — there is no published-URL flag, no derive
// flag and no bound-address list — so what this binary passes is the zero
// value, and system.LocalAddress then answers from the request itself, which
// plan 6.6 states as the deliberate answer for an installation with none of the
// three. The parameter exists so that the feature which adds the configuration
// surface has somewhere to put it, and so that this handler's tests can reach
// tiers 1 and 3 that a v1 deployment cannot.
func NewSystemHandler(cfg SystemHandlerConfig) (*SystemHandler, error) {
	if cfg.InstallationID == "" {
		return nil, errors.New("httpapi: the system handler needs an installation identity, and was given none")
	}
	if cfg.Installations == nil {
		return nil, errors.New("httpapi: the system handler needs an installation store, and was given none")
	}
	// A missing port would answer WebSocketPortNumber with a zero that reads
	// like a real answer, so it is a failure to start rather than a field
	// nobody notices is wrong.
	if cfg.HTTPPort == nil {
		return nil, errors.New("httpapi: the system handler needs to be able to answer which port this server is listening on, and was given no way to")
	}
	return &SystemHandler{
		installationID: cfg.InstallationID,
		installations:  cfg.Installations,
		addresses:      cfg.Addresses,
		paths:          cfg.Paths,
		httpPort:       cfg.HTTPPort,
		authenticator:  cfg.Authenticator,
	}, nil
}

// PublicInfo answers GET /System/Info/Public (spec 3.1).
//
// It is unauthenticated and it answers before any user exists and before any
// library is configured (AC-2, AC-3) — it reads one row of a table every start
// creates, and nothing else. That is what makes it testable with nothing on
// disk but a data directory.
func (h *SystemHandler) PublicInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		info, err := h.publicSystemInfo(r)
		if err != nil {
			WriteInternalServerError(w)
			return
		}

		// The profile is negotiated here rather than carried in from a
		// middleware, and the body and its content type are set by one call:
		// behaviours 1.10 puts the content type on whatever produced the body.
		if err := wire.Write(w, http.StatusOK, info, NegotiateProfile(r)); err != nil {
			// wire.Write writes nothing to w unless the whole body serialised,
			// so there is still a refusal to send. Nothing in this model can
			// fail to serialise today — seven strings and a bool — but the
			// error is handled rather than ignored, because the day a field
			// with a custom encoder joins it is not the day to discover that
			// half a 200 was already on the wire.
			WriteInternalServerError(w)
		}
	}
}

// publicSystemInfo builds the body, and is separate from the handler because
// /System/Info is a superset of it (spec 3.2, T18): the two responses must
// agree on every shared field, and the cheapest way to guarantee that is for
// there to be one place the shared fields are filled in.
func (h *SystemHandler) publicSystemInfo(r *http.Request) (PublicSystemInfo, error) {
	installation, err := h.installations.Installation(r.Context())
	if err != nil {
		return PublicSystemInfo{}, err
	}

	return PublicSystemInfo{
		LocalAddress:           system.LocalAddress(requestFacts(r), h.addresses),
		ServerName:             installation.Name,
		Version:                system.ReportedVersion,
		ProductName:            system.ProductName,
		OperatingSystem:        system.OperatingSystem,
		Id:                     h.installationID,
		StartupWizardCompleted: installation.SetupCompleted,
	}, nil
}

// Info answers GET /System/Info (spec 3.2).
//
// # The order the two questions are asked in
//
// The installation is read *before* admission is decided, which looks like the
// wrong way round and is not: the route's policy depends on the setup state.
// The reference admits any request at all while first-time setup is
// outstanding — its authorisation handler succeeds on
// "!IsStartupWizardCompleted" before it looks at a role
// [source: Jellyfin.Api/Auth/FirstTimeSetupPolicy/FirstTimeSetupHandler.cs:29-31
// @ v10.11.11] — and an unrecognised token does not change that, because the
// authentication handler answers "no result" rather than a failure and leaves
// authorisation to decide
// [source: Jellyfin.Api/Auth/CustomAuthenticationHandler.cs:48-51,79-83 @
// v10.11.11]. So the answer to "may this request in" cannot be reached without
// the answer to "is setup finished", and StartupWizardCompleted is the field
// that carries it into the body afterwards. One read, two uses.
//
// # What is reachable here, and what is not
//
// Until 002 there is no Authenticator, so the two reachable states are the two
// spec 3.2 describes with no credential in existence: setup outstanding, which
// is admitted, and setup complete, which is refused whatever the request
// carried. AC-5's other half — "200 with a valid one" — needs a token nothing
// can issue yet and is **carried into 002** rather than claimed here
// (tasks.md T18, T21).
func (h *SystemHandler) Info() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		public, err := h.publicSystemInfo(r)
		if err != nil {
			WriteInternalServerError(w)
			return
		}

		if public.StartupWizardCompleted && !h.admits(w, r) {
			return
		}

		if err := wire.Write(w, http.StatusOK, h.systemInfo(public), NegotiateProfile(r)); err != nil {
			WriteInternalServerError(w)
		}
	}
}

// admits asks the authenticator whether this request may have the response,
// and writes the refusal itself when it may not. It reports whether the caller
// should carry on.
//
// A nil authenticator is not a special case worth a branch of its own: 001
// ships without one, and "this server recognises no credential" is the true
// answer for a server that has issued none.
func (h *SystemHandler) admits(w http.ResponseWriter, r *http.Request) bool {
	var authentication Authentication
	if h.authenticator != nil {
		decided, err := h.authenticator.Authenticate(r)
		if err != nil {
			// Not a 401. A client answered 401 discards its credential and
			// logs in again, and a store that was briefly unreadable is not a
			// reason to make it.
			WriteInternalServerError(w)
			return false
		}
		authentication = decided
	}

	switch authentication.Access {
	case AccessGranted:
		return true
	case AccessForbidden:
		// 002 plan 7's row for a live token whose user was disabled after it
		// was issued: behaviours 1.11's *policy* refusal — the status, an
		// empty body and no content type at all, which is what 001's refuse
		// already writes. 002 T11 gives that shape a name of its own
		// (WriteForbidden) beside 001's four writers, and this call site takes
		// it when it lands; what it must not do meanwhile is fall through to
		// the default, which would answer a disabled account 500.
		refuse(w, http.StatusForbidden)
		return false
	case AccessUnauthenticated:
		// behaviours 1.11's empty shape, written in one place (refusal.go):
		// no body, Content-Length: 0, no Content-Type and no
		// WWW-Authenticate. None of those four is visible to a test that
		// parses the body, and three of them are invisible to one that only
		// reads the status.
		WriteUnauthorized(w)
		return false
	default:
		// An Access this package does not know is an error rather than a
		// fall-through, which is internal/wire's rule for an unknown Profile
		// and holds for the same reason: the two directions a fall-through
		// could take are "admit everybody" and "refuse everybody", and both
		// are wrong silently. 002 adding a value without teaching this switch
		// about it fails loudly here.
		WriteInternalServerError(w)
		return false
	}
}

// systemInfo builds spec 3.2's body around the seven fields the public route
// already answered with.
//
// The public half arrives whole rather than being rebuilt, which is what makes
// AC-5's agreement structural: there is one filler for those seven values and
// this function cannot disagree with it, because it never sees them
// individually.
func (h *SystemHandler) systemInfo(public PublicSystemInfo) SystemInfo {
	return SystemInfo{
		PublicSystemInfo:           public,
		OperatingSystemDisplayName: system.OperatingSystem,

		// PackageName is left nil, and therefore off the wire entirely.

		HasPendingRestart:      false,
		IsShuttingDown:         false,
		SupportsLibraryMonitor: false,
		WebSocketPortNumber:    h.httpPort(),

		// Empty and non-nil, because a nil slice serialises as null and
		// spec 3.2 asks for empty arrays. That difference is exactly the kind
		// behaviours 1.7 is about, and it is one keystroke wide.
		CompletedInstallations: []any{},

		CanSelfRestart:      false,
		CanLaunchWebBrowser: false,

		ProgramDataPath:      h.paths.ProgramData,
		WebPath:              h.paths.Web,
		ItemsByNamePath:      h.paths.ItemsByName,
		CachePath:            h.paths.Cache,
		LogPath:              h.paths.Log,
		InternalMetadataPath: h.paths.InternalMetadata,
		TranscodingTempPath:  h.paths.TranscodingTemp,

		CastReceiverApplications: []any{},

		HasUpdateAvailable: false,
		EncoderLocation:    "",
		SystemArchitecture: "",
	}
}
