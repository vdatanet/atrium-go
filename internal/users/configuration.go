package users

import (
	"encoding/json"
	"fmt"
)

// Configuration is the per-account preference set spec 3.6 replaces and
// spec 3.5 returns inside every user object, in the reference's own declaration
// order [source: MediaBrowser.Model/Configuration/UserConfiguration.cs:35-76 @
// v10.11.11].
//
// Sixteen properties, all of which travel
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. Every
// one is stored and returned faithfully; v1 acts on the two language
// preferences and DisplayMissingEpisodes, and spec 3.6 argues why the rest
// being inert is a bounded gap rather than an oversight.
//
// # Declaration order, for the same reason Policy's matters
//
// This object travels inside the same L3 body Policy does, so the order below
// is contract and not presentation. See Policy.
//
// # Where this model is less certain than Policy's
//
// The reference declares AudioLanguagePreference, SubtitleLanguagePreference
// and CastReceiverId as nullable strings, and fills them per account: it
// coerces the subtitle preference to the empty string, leaves the audio
// preference as the account holds it, and answers CastReceiverId with the first
// cast receiver application the installation has
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:426-447 @
// v10.11.11]. Under behaviours 1.7 a null would be omitted, which would make
// the measured count 15 rather than the 16 spec 3.6 records.
//
// The three are therefore plain strings here — the empty string travels, and
// sixteen members arrive, which is what was measured. ⚠️ UNVERIFIED what the
// reference puts in them on a fresh account, and CastReceiverId is the one that
// cannot match: 001 answers /System/Info with an empty
// CastReceiverApplications because Atrium ships no cast receiver, so the only
// value this server can honestly send is the empty string. It is one request to
// settle, 010's differential run would surface it, and it is stated here rather
// than resolved because spec 3.6 is not this change's to amend.
type Configuration struct {
	AudioLanguagePreference    string
	PlayDefaultAudioTrack      bool
	SubtitleLanguagePreference string
	DisplayMissingEpisodes     bool

	// The four Guid arrays are arrays of identifiers, which .NET writes as
	// 32-character hex without dashes
	// [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:37 @ v10.11.11] and
	// behaviours 1.4 records as this project's identifier shape.
	GroupedFolders []string

	// SubtitleMode is an enum name and not a number — "Default" is what a
	// differential run sends to this route
	// (docs/compatibility/request-cases.yaml, replace-configuration).
	SubtitleMode string

	DisplayCollectionsView bool

	// EnableLocalPassword gates the reference's PIN concept. v1 has none
	// (spec 3.5: HasConfiguredEasyPassword is always false), so this is stored
	// and echoed and acts on nothing.
	EnableLocalPassword bool

	OrderedViews        []string
	LatestItemsExcludes []string
	MyMediaExcludes     []string

	HidePlayedInLatest         bool
	RememberAudioSelections    bool
	RememberSubtitleSelections bool
	EnableNextEpisodeAutoPlay  bool

	CastReceiverId string
}

// subtitleModeDefault is the SubtitlePlaybackMode the reference's User entity
// is constructed with
// [source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/User.cs:55
// @ v10.11.11]. .NET writes an enum as its name
// [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:42 @ v10.11.11].
const subtitleModeDefault = "Default"

// DefaultConfiguration is the configuration a fresh account carries, and the
// value every stored configuration document is decoded over.
//
// It is the reference's two constructors read together: UserConfiguration's own
// [source: MediaBrowser.Model/Configuration/UserConfiguration.cs:16-29 @
// v10.11.11] and the User entity's, which is what actually decides the four
// booleans the account holds
// [source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/User.cs:42-56
// @ v10.11.11]. The two agree on every property they share.
//
// A function rather than a variable, for the reason DefaultPolicy is one: it
// hands out slices.
func DefaultConfiguration() Configuration {
	return Configuration{
		AudioLanguagePreference:    "",
		PlayDefaultAudioTrack:      true,
		SubtitleLanguagePreference: "",
		DisplayMissingEpisodes:     false,

		GroupedFolders: []string{},

		SubtitleMode: subtitleModeDefault,

		DisplayCollectionsView: false,
		EnableLocalPassword:    false,

		OrderedViews:        []string{},
		LatestItemsExcludes: []string{},
		MyMediaExcludes:     []string{},

		HidePlayedInLatest:         true,
		RememberAudioSelections:    true,
		RememberSubtitleSelections: true,
		EnableNextEpisodeAutoPlay:  true,

		CastReceiverId: "",
	}
}

// DecodeConfiguration reads a stored configuration document.
//
// It starts from DefaultConfiguration and unmarshals over it, which is the
// package's one rule.
//
// **An unknown property is dropped, not refused** — spec 3.6's "unknown
// properties are ignored, not rejected". That is what encoding/json does
// without DisallowUnknownFields, and the absence of that call is the whole
// implementation, which is exactly why there is a test naming it: a line
// nobody wrote is a line nobody notices somebody adding.
//
// **This is the opposite of what a session's capabilities do**, and the two are
// opposite because the reference is (plan 6.6, 6.10): behaviours 5.9 records an
// unknown capabilities property surviving into /Sessions as a deliberate
// divergence, and there is no such divergence here. A reader who generalises
// from one to the other gets the wrong answer on both.
func DecodeConfiguration(document []byte) (Configuration, error) {
	configuration := DefaultConfiguration()
	if err := json.Unmarshal(document, &configuration); err != nil {
		return Configuration{}, fmt.Errorf("users: reading a stored configuration document: %w", err)
	}
	return configuration, nil
}

// Document encodes a configuration for the configuration_document column
// (plan 4). See Policy.Document for why it is encoding/json and not
// internal/wire.
func (c Configuration) Document() ([]byte, error) {
	document, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("users: writing a configuration document: %w", err)
	}
	return document, nil
}
