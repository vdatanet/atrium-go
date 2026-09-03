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
func NewSystemHandler(installationID string, installations ports.InstallationStore, addresses system.AddressConfig) (*SystemHandler, error) {
	if installationID == "" {
		return nil, errors.New("httpapi: the system handler needs an installation identity, and was given none")
	}
	if installations == nil {
		return nil, errors.New("httpapi: the system handler needs an installation store, and was given none")
	}
	return &SystemHandler{
		installationID: installationID,
		installations:  installations,
		addresses:      addresses,
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
