package ports

import (
	"context"
	"fmt"

	"github.com/vdatanet/atrium-go/internal/units"
)

// The four types the two interfaces below are written in terms of are declared
// here, in ports, over the standard library and internal/units and nothing
// else. 002 plan 5 wrote UserStore and SessionStore in terms of User,
// Credential, Session and LoginOutcome without saying where they live, and the
// two candidates are not equivalent.
//
// A port method returning users.User would make this package import a domain
// package, which inverts architecture 2's arrow: Ports is the bottom of the
// diagram and may import "nothing of ours" but the unit types, which are a leaf
// that imports nothing itself. The inversion is not theoretical bookkeeping —
// it is what keeps ADR-0003 arguable after a feature is planned. It would also
// have to be an import of *two* domain packages, because 002 plan 3 splits
// users from sessions deliberately and a single port file naming both would
// re-join in ports what the domain took care to separate.
//
// The cost is that these are store records rather than domain values, and the
// difference shows in one place: the policy and the configuration cross this
// boundary as the **bytes of their stored documents**, not as users.Policy and
// users.Configuration. That is the honest shape, not a workaround. 002 plan 4
// makes both columns serialised models whose declaration order is the wire
// order of an L3 body, and 002 plan 6.6 makes the decode a domain rule — a
// document decodes onto the reference's defaults, never onto Go's zero value,
// and InvalidLoginAttemptCount is overlaid from its own column afterwards. A
// port that handed the domain an already-decoded Policy would have performed
// that rule inside the store, where the reason for it is invisible.
//
// The reverse direction is unaffected: the store implementation may import
// internal/users, because the store is outward of this package and not inward
// of it. It does, for exactly one transition — see RecordLoginOutcome.

// User is one account as the store holds it.
//
// It carries the two documents as bytes for the reason the file comment gives,
// and it does not carry the credential: 002 plan 4 puts the verifier in a table
// of its own because "every read of a user object is a read of users, and none
// of them wants the verifier in memory". Reading a user therefore cannot put a
// password record on the heap, which is a property of the schema and of this
// type rather than of everybody's discipline.
type User struct {
	// ID is 32 lowercase hex (behaviours 1.4's shape), derived from the folded
	// username so that an installation provisioned twice with the same names
	// has the same identifiers (Principle VII, 002 plan 6.9).
	ID string

	// Username is the spelling the operator chose, which is what the user
	// object's Name returns.
	Username string

	// UsernameFolded is the case-folded spelling, and the only column an
	// authentication reads to find a row (spec 3.3, 002 plan 4).
	UsernameFolded string

	// PolicyDocument and ConfigurationDocument are the stored documents, whole.
	// They are decoded by internal/users and never here.
	PolicyDocument        []byte
	ConfigurationDocument []byte

	// InvalidLoginAttemptCount is state and not policy, which is why it is a
	// column and a field of its own rather than a property of the document
	// (002 plan 4). It moves on every failed login; whatever the stored
	// document happens to carry for it is stale by construction, and the user
	// object overlays this value over the decoded policy (002 plan 6.6).
	InvalidLoginAttemptCount int

	// LastLoginAt and LastActivityAt are nil when the column is NULL, which is
	// the whole reason they are pointers: NULL is what makes LastLoginDate
	// *absent* until the first login rather than reported as the minimum date
	// (spec 3.5). These are the only two nullable dates in this feature's
	// schema — every session date is NOT NULL, because the zero tick is a
	// value there and not a missing one (spec 3.3).
	LastLoginAt    *units.Time
	LastActivityAt *units.Time
}

// Credential is the account's password record.
//
// PHC is ADR-0006's record as a string, and deliberately not a type of its own:
// internal/users' Derive returns a string and Verify takes one, and the record
// carries its own parameters, so a store type wrapping it would be a second
// spelling of a value the domain already parses.
type Credential struct {
	UserID string

	// PHC is the record: $argon2id$v=19$m=65536,t=3,p=2$<salt>$<key>.
	PHC string

	// WrittenAt is when the record was derived. It is what a rehash moves, and
	// the only way to see that one happened (002 plan 4).
	WrittenAt units.Time
}

// Session is one session as the store holds it.
//
// It is keyed on (Client, DeviceID) with the user as a field, which is not
// quite the triple spec 3.8 describes and is argued at 002 plan 6.5: the
// reference keys a session on the client and the device alone and updates its
// user when somebody else logs in there, so two users sharing one device and
// one client have two tokens and one session row.
type Session struct {
	// ID is 32 lowercase hex, derived from (Client, DeviceID) — 002 plan 6.5.
	ID string

	// UserID is whoever authenticated here last. It is not necessarily the
	// user a given token belongs to; SessionByTokenDigest says why.
	UserID string

	Client             string
	DeviceID           string
	DeviceName         string
	ApplicationVersion string
	RemoteEndpoint     string

	// CapabilitiesDocument is the declaration POST /Sessions/Capabilities/Full
	// posted, whole, and nil until one is posted. It is stored raw rather than
	// decoded and re-encoded, which is what makes behaviours 5.9's divergence —
	// an unknown capabilities property surviving into /Sessions — the stated
	// one rather than an accident (002 plan 6.10).
	CapabilitiesDocument []byte

	// The three dates a session carries, none of them nullable. The zero tick
	// is a value for LastPlaybackCheckInAt in particular: spec 3.3 measures
	// 0001-01-01T00:00:00.0000000Z for a session that has never played
	// anything, "not null and not absent".
	CreatedAt             units.Time
	LastActivityAt        units.Time
	LastPlaybackCheckInAt units.Time
}

// LoginOutcome is what one authentication attempt did to an account.
//
// It is a single value rather than a set of flags because RecordLoginOutcome is
// a single transition (002 plan 5): the three constants below are the three
// states 002 plan 6.7's rule can reach, and naming them separately is what lets
// the store apply all of a transition or none of it.
//
// The zero value is deliberately none of them. There is no safe default here —
// defaulting to a failure would count an attempt nobody made, and defaulting to
// a success would clear a lockout — so a caller that forgot to say gets an
// error rather than the reading somebody guessed was harmless. This is the one
// place in the project where the zero value is invalid on purpose, and it is
// the opposite decision from httpapi.Authentication's for the opposite reason:
// there, refusing everybody is a safe default and exists precisely so an
// unwired authenticator admits nobody.
type LoginOutcome int

const (
	// LoginOutcomeUnset is the zero value and is not an outcome.
	LoginOutcomeUnset LoginOutcome = iota

	// LoginFailed is a failed attempt that did not reach the threshold. It
	// increments the counter and moves nothing else.
	LoginFailed

	// LoginSucceeded resets the counter to zero and stamps the login date.
	LoginSucceeded

	// LoginLockedOut is the failed attempt that reached the threshold. It
	// increments the counter *and* sets IsDisabled in the stored policy
	// document, which is why the lockout is permanent until an operator clears
	// it and why the next attempt is refused as *disabled* rather than as
	// locked (002 plan 6.7).
	//
	// Which failure reaches the threshold is the domain's to decide and not
	// the store's: LoginAttemptsBeforeLockout is a sentinel and not a count
	// (-1 never locks, 0 means three, anything else is itself), and reading it
	// is 002 plan 6.4's login path. The store applies the transition it is
	// handed.
	LoginLockedOut
)

// String names the outcome, for an error message that would otherwise report an
// integer.
func (o LoginOutcome) String() string {
	switch o {
	case LoginOutcomeUnset:
		return "unset"
	case LoginFailed:
		return "failed"
	case LoginSucceeded:
		return "succeeded"
	case LoginLockedOut:
		return "locked out"
	default:
		return fmt.Sprintf("LoginOutcome(%d)", int(o))
	}
}

// UserStore is what the account domain needs of the store.
//
// Every read returns (value, found, error) rather than a sentinel error for an
// absence, because "no such account" is an ordinary answer on this feature's
// hottest path — an authentication against a username nobody has — and a
// caller that has to distinguish an absence from a failure by matching an error
// is a caller that can get it wrong in the direction of a 500.
type UserStore interface {
	// CreateUser writes a new account, whole.
	//
	// It takes the record rather than a name and a handful of options for the
	// reason OpenSession does: what the store writes is what it was handed, so
	// there is no second place where a default is decided. Every column comes
	// from the value — including the identifier, which is derived from the
	// folded username by the domain (Principle VII, 002 plan 6.9) and not
	// invented here.
	//
	// A fresh account carries no credential: the password record is a row in a
	// table of its own and is written by ReplaceCredential, so an account with
	// no password is the state this method leaves behind and not a special
	// case. It is also a state an operator can ask for (002 plan 6.9's
	// --no-password), which is why the two writes are two calls.
	//
	// A username whose fold is already taken is refused, by the unique index
	// on username_folded rather than by a check here. That constraint is the
	// database's rule precisely so the login's assumption — one row per folded
	// name — cannot be broken by a caller that forgot to look first
	// (002 plan 4).
	CreateUser(ctx context.Context, user User) error

	// UserByFoldedName finds the account whose folded username is folded. It
	// is the only lookup an authentication performs, and the uniqueness it
	// assumes is the database's rule rather than a convention (002 plan 4).
	UserByFoldedName(ctx context.Context, folded string) (User, bool, error)

	// UserByID finds one account by its identifier.
	UserByID(ctx context.Context, id string) (User, bool, error)

	// Users returns every account, in a stated order.
	//
	// The order is the store's to make deterministic — architecture 2 forbids
	// an order that derives from anything but stable input, because L3
	// compares list rows by position. What order the *reference* answers
	// /Users/Public in is a separate and unmeasured question, and it belongs to
	// the route rather than here.
	Users(ctx context.Context) ([]User, error)

	// Credential returns the account's password record, if it has one. A user
	// with no row here is an account with no password, which spec 3.5 reports
	// as HasPassword false and which ADR-0006 rule 4 excludes from the timing
	// equalisation.
	Credential(ctx context.Context, userID string) (Credential, bool, error)

	// ReplaceCredential writes the account's password record, replacing any
	// record it already had. It is what provisioning calls and what a
	// rehash-on-successful-login calls (002 plan 6.4).
	ReplaceCredential(ctx context.Context, userID string, phc string, at units.Time) error

	// ReplaceConfiguration writes the account's configuration document whole.
	// POST /Users/Configuration replaces rather than merges (spec 3.6), and
	// the document handed here has already been decoded onto the defaults and
	// re-encoded by the domain, so an unknown property is already gone.
	ReplaceConfiguration(ctx context.Context, userID string, document []byte) error

	// RecordLoginOutcome applies one authentication attempt's whole effect on
	// the account, at the instant at.
	//
	// It is one method rather than "increment the counter", "reset the
	// counter", "set the disabled flag" and "stamp the login date", because
	// 002 plan 6.7's rule is a single transition and four callers would be
	// four chances to perform three quarters of it. The implementation owes
	// the same atomicity the signature promises: all of the outcome's effect,
	// or none of it.
	//
	// at is used by LoginSucceeded, which stamps it as the login date. The two
	// failing outcomes carry it and do not write it — a failed login is not an
	// activity, and spec 3.5 makes LastLoginDate the date of a login that
	// worked.
	RecordLoginOutcome(ctx context.Context, userID string, outcome LoginOutcome, at units.Time) error

	// TouchActivity records that the account was seen at at.
	TouchActivity(ctx context.Context, userID string, at units.Time) error
}

// SessionStore is what the session domain needs of the store.
type SessionStore interface {
	// OpenSession writes the session and the token that opens it, as one
	// statement.
	//
	// One statement is the contract and not an implementation note: a token
	// row names its session by a foreign key, so a token that outlived a
	// failed session write would be a credential resolving to a caller with no
	// client, no device and no activity to stamp. Either both are there
	// afterwards or neither is.
	//
	// The session is written at its derived identifier, inserted when it is
	// new and updated when the same client authenticates again on the same
	// device — which is spec 3.3's "authenticating again from the same DeviceId
	// replaces that session rather than accumulating one per login". Revoking
	// the tokens that replacement invalidates is RevokeTokensFor's, called
	// first, because 002 plan 6.5 makes it step three of a login and this
	// step four.
	//
	// tokenDigest is the unsalted SHA-256 of the token in lowercase hex, and
	// the token itself never reaches this package: ADR-0006's threat model is
	// the store file leaking, and a stored table of live bearer tokens is that
	// leak with the hashing skipped (002 plan 4).
	OpenSession(ctx context.Context, session Session, tokenDigest string) error

	// SessionByTokenDigest resolves a presented token to the session it opened
	// and to the user it belongs to, which is the second return value.
	//
	// The user is returned separately rather than read off the session because
	// the two can differ, and the difference is the reference's: a session is
	// keyed on (Client, DeviceID) and names whoever authenticated there last,
	// while a token is keyed on (user, device), so two people sharing one
	// client on one device hold two live tokens against one session row
	// (002 plan 6.5). A caller resolved from the session's user would be
	// whoever logged in most recently rather than whoever is holding the
	// token.
	SessionByTokenDigest(ctx context.Context, digest string) (Session, string, bool, error)

	// Sessions returns every session, in a stated order. GET /Sessions filters
	// and narrows the whole list in the domain (002 plan 6.10), which is why
	// this takes no caller and no selection.
	Sessions(ctx context.Context) ([]Session, error)

	// ReplaceCapabilities stores the declaration a client posted, whole and
	// unread. It replaces rather than merges (spec 3.8).
	ReplaceCapabilities(ctx context.Context, sessionID string, document []byte) error

	// TouchSession advances the session's LastActivityDate.
	TouchSession(ctx context.Context, sessionID string, at units.Time) error

	// RevokeTokensFor invalidates every token this user holds on this device,
	// and nothing else.
	//
	// The pair is the whole point. It is 002 plan 6.5's replacement rule, and
	// the reference's own order — it logs out the existing devices for that
	// user and device before creating the new one. A revocation by user alone
	// would log that user out of every other device on every login, which no
	// response would report and which a test asserting "the old token is gone"
	// would pass.
	RevokeTokensFor(ctx context.Context, userID, deviceID string) error
}
