package users

import (
	"encoding/json"
	"fmt"
)

// Policy is the permission set spec 3.5 returns inside every user object, in
// the reference's own declaration order.
//
// # Forty-four declared, forty-two sent
//
// The reference declares 44 properties
// [source: MediaBrowser.Model/Users/UserPolicy.cs:16-68,112-114 @ v10.11.11]
// and sends 42
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. The
// two that do not travel are MaxParentalRating and MaxParentalSubRating, which
// are nullable integers and null on an account nothing has restricted, so
// behaviours 1.7's global omit-when-null rule drops them. Everything else
// travels, including the six arrays, because the reference fills every one of
// them from a preference rather than leaving it null
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:448-493 @
// v10.11.11].
//
// 44 − 2 = 42 is the arithmetic, and it is written as a subtraction in the test
// rather than as the literal 42 so that the measured count is a check on this
// model. A model that lost a property and a test that asserted "42 members"
// would agree with each other and disagree with the reference.
//
// # Why declaration order is not presentation
//
// Go writes a struct's fields in declaration order, so this list is the key
// order of the Policy object inside POST /Users/AuthenticateByName's body —
// which is this feature's one L3 row, and L3 compares bytes (conformance L3).
// A property moved for tidiness is a wire change. The test that guards it reads
// the member names out of the encoded document in the order the bytes carry
// them, because a set comparison passes on a reordered model.
//
// # What v1 honours
//
// Fourteen of these are acted on and the rest are stored and echoed unchanged;
// spec 3.5 lists which, argues why the gap is bounded, and carries the
// amendment recording the two — EnableRemoteAccess and AccessSchedules — that
// the reference enforces at authentication and v1 does not. This model is the
// whole set either way: a policy that returned only the honoured flags would be
// a delta by Principle I on the count.
type Policy struct {
	IsAdministrator            bool
	IsHidden                   bool
	EnableCollectionManagement bool
	EnableSubtitleManagement   bool
	EnableLyricManagement      bool
	IsDisabled                 bool

	// MaxParentalRating and MaxParentalSubRating are the two members that do
	// not reach the wire on a default account. They are nullable in the
	// reference [source: MediaBrowser.Model/Users/UserPolicy.cs:112-114 @
	// v10.11.11], so they are pointers here — ADR-0002 makes an optional field
	// a pointer project-wide, and `omitempty` on a non-pointer is banned
	// because it would also drop a real zero.
	MaxParentalRating    *int `json:",omitempty"`
	MaxParentalSubRating *int `json:",omitempty"`

	BlockedTags                []string
	AllowedTags                []string
	EnableUserPreferenceAccess bool

	// AccessSchedules is always empty in v1, and its element type is
	// deliberately unspecified — 001's rule for CompletedInstallations, for
	// the same reason: the reference's element is a model of a feature v1 does
	// not have, and declaring its members would be a schema for a value
	// nothing here can produce (Principle VI's plausible-looking stub, in a
	// response body). spec 3.5's amendment records that the reference enforces
	// these at authentication and that v1 does not, as U-15. The feature that
	// ever fills this array declares the type it fills it with.
	AccessSchedules []any

	// BlockUnratedItems is an array of enum names. .NET writes an enum as its
	// name [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:42 @
	// v10.11.11], so the wire type is a string and not a number.
	BlockUnratedItems []string

	EnableRemoteControlOfOtherUsers bool
	EnableSharedDeviceControl       bool
	EnableRemoteAccess              bool
	EnableLiveTvManagement          bool
	EnableLiveTvAccess              bool
	EnableMediaPlayback             bool
	EnableAudioPlaybackTranscoding  bool
	EnableVideoPlaybackTranscoding  bool
	EnablePlaybackRemuxing          bool
	ForceRemoteSourceTranscoding    bool
	EnableContentDeletion           bool

	EnableContentDeletionFromFolders []string

	EnableContentDownloading bool
	EnableSyncTranscoding    bool
	EnableMediaConversion    bool

	EnabledDevices    []string
	EnableAllDevices  bool
	EnabledChannels   []string
	EnableAllChannels bool
	EnabledFolders    []string
	EnableAllFolders  bool

	// InvalidLoginAttemptCount is reported here and stored on the account's
	// own row rather than inside the document (plan 4): it moves on every
	// failed login, and keeping it in the document would rewrite the whole
	// policy on each failure. Whatever a stored document happens to carry for
	// it is overlaid from the column when the user object is built (plan 6.6),
	// which is the one place the wire's shape and the store's shape are
	// allowed to disagree.
	InvalidLoginAttemptCount int

	// LoginAttemptsBeforeLockout is a sentinel and not a count: the reference
	// maps -1 to "never lock", 0 to three attempts, and anything else to
	// itself [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821
	// @ v10.11.11]. -1 is what it sends for a default account
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28],
	// so the default account never locks — which is why decoding a document
	// onto Policy{} rather than onto DefaultPolicy is not a cosmetic mistake.
	LoginAttemptsBeforeLockout int

	// MaxActiveSessions caps concurrent sessions; 0 means unlimited and 0 is
	// what the reference sends (spec 3.5, plan 6.7).
	MaxActiveSessions int

	EnablePublicSharing bool

	BlockedMediaFolders []string
	BlockedChannels     []string

	RemoteClientBitrateLimit int

	// AuthenticationProviderId and PasswordResetProviderId name the
	// reference's pluggable providers. v1 has one of each and no plugin
	// surface, so these carry the reference's own default names: a client that
	// reads them expects the shape, and an empty string would be a value no
	// reference server sends.
	AuthenticationProviderId string
	PasswordResetProviderId  string

	// SyncPlayAccess is an enum name, for the reason BlockUnratedItems is.
	SyncPlayAccess string
}

// The reference's two provider identifiers. It creates every account naming
// the default providers by their .NET type name
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:281-284 @
// v10.11.11], and those two names are the namespace and class below
// [source: Jellyfin.Server.Implementations/Users/DefaultAuthenticationProvider.cs:10,15,
// Jellyfin.Server.Implementations/Users/DefaultPasswordResetProvider.cs:17,22 @ v10.11.11].
//
// Copying a .NET type name into a Go server reads oddly and is Principle I:
// these are two of the 42 properties that travel, and a server that sent an
// empty string where the reference sends a name would differ on a field a
// client can read. ⚠️ UNVERIFIED as *values*: no probe in this repository
// records what these two properties carry, only that 42 members arrive. It is
// one request to settle and 010's differential run would surface it.
const (
	defaultAuthenticationProviderID = "Jellyfin.Server.Implementations.Users.DefaultAuthenticationProvider"
	defaultPasswordResetProviderID  = "Jellyfin.Server.Implementations.Users.DefaultPasswordResetProvider"
)

// syncPlayCreateAndJoinGroups is the SyncPlayAccess the reference's constructor
// assigns [source: MediaBrowser.Model/Users/UserPolicy.cs:67 @ v10.11.11].
const syncPlayCreateAndJoinGroups = "CreateAndJoinGroups"

// DefaultPolicy is the policy a fresh account carries, and the value every
// stored document is decoded over.
//
// It is the reference's constructor, transcribed
// [source: MediaBrowser.Model/Users/UserPolicy.cs:16-68 @ v10.11.11]. Thirteen
// booleans are true, LoginAttemptsBeforeLockout is -1 and every array is empty
// and non-nil — a nil slice would serialise as null, and the reference sends
// [] (behaviours 1.7 is about the difference, and it is one keystroke wide).
//
// It is a function rather than a package-level variable because it hands out
// slices: a variable would hand every caller the same backing arrays, and one
// account's policy would then be able to change another's.
func DefaultPolicy() Policy {
	return Policy{
		IsAdministrator:            false,
		IsHidden:                   true,
		EnableCollectionManagement: false,
		EnableSubtitleManagement:   false,
		EnableLyricManagement:      false,
		IsDisabled:                 false,

		// Null, and therefore the two members that do not reach the wire.
		MaxParentalRating:    nil,
		MaxParentalSubRating: nil,

		BlockedTags:                []string{},
		AllowedTags:                []string{},
		EnableUserPreferenceAccess: true,
		AccessSchedules:            []any{},
		BlockUnratedItems:          []string{},

		EnableRemoteControlOfOtherUsers: false,
		EnableSharedDeviceControl:       true,
		EnableRemoteAccess:              true,
		EnableLiveTvManagement:          true,
		EnableLiveTvAccess:              true,
		EnableMediaPlayback:             true,
		EnableAudioPlaybackTranscoding:  true,
		EnableVideoPlaybackTranscoding:  true,
		EnablePlaybackRemuxing:          true,
		ForceRemoteSourceTranscoding:    false,
		EnableContentDeletion:           false,

		EnableContentDeletionFromFolders: []string{},

		EnableContentDownloading: true,
		EnableSyncTranscoding:    true,
		EnableMediaConversion:    true,

		EnabledDevices:    []string{},
		EnableAllDevices:  true,
		EnabledChannels:   []string{},
		EnableAllChannels: true,
		EnabledFolders:    []string{},
		EnableAllFolders:  true,

		InvalidLoginAttemptCount:   0,
		LoginAttemptsBeforeLockout: -1,
		MaxActiveSessions:          0,

		EnablePublicSharing: true,

		BlockedMediaFolders: []string{},
		BlockedChannels:     []string{},

		RemoteClientBitrateLimit: 0,

		AuthenticationProviderId: defaultAuthenticationProviderID,
		PasswordResetProviderId:  defaultPasswordResetProviderID,

		SyncPlayAccess: syncPlayCreateAndJoinGroups,
	}
}

// DecodePolicy reads a stored policy document.
//
// It starts from DefaultPolicy and unmarshals the document over it, which is
// the whole point of the function existing at all: a property the document does
// not carry keeps the reference's default rather than Go's zero value. See the
// package comment for what that costs when it is got wrong.
//
// An unknown property is ignored rather than refused, for the reason
// DecodeConfiguration's is (spec 3.6) — a document written by a later build
// must still be readable by this one, and the alternative is a store that
// cannot be rolled back.
func DecodePolicy(document []byte) (Policy, error) {
	policy := DefaultPolicy()
	if err := json.Unmarshal(document, &policy); err != nil {
		return Policy{}, fmt.Errorf("users: reading a stored policy document: %w", err)
	}
	return policy, nil
}

// Document encodes a policy for the policy_document column (plan 4).
//
// It is encoding/json rather than internal/wire because a stored document is
// not a response; the package comment argues it. What the two encodings share
// is the struct, and therefore the order.
func (p Policy) Document() ([]byte, error) {
	document, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("users: writing a policy document: %w", err)
	}
	return document, nil
}
