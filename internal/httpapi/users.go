package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/users"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// UsersHandlerConfig is what the /Users routes of feature 002 are built from.
//
// A struct rather than a parameter list, for the reason SystemHandlerConfig is
// one and with the same consequence: the tasks after this one add routes to
// this handler, and they add a field rather than change a signature every
// caller has already written.
type UsersHandlerConfig struct {
	// InstallationID is the server identity of 001 spec 3.1. It is the
	// `ServerId` of the authentication result and of every user object, and it
	// is the same string `/System/Info/Public` answers as `Id`.
	InstallationID string

	// Login is spec 3.3's credential check: the whole of what happens before a
	// session exists. It answers two sentinels and never a status
	// (internal/users' login.go).
	Login *users.Login

	// Accounts is read for one thing at this layer — whether an account has a
	// password record, which is the user object's `HasPassword` (plan 6.6).
	Accounts ports.UserStore

	// Sessions is what an authentication writes to: the revocation and the
	// session-and-token insert of plan 6.5.
	Sessions ports.SessionStore

	// Clock stamps the session. It is a port rather than a call to time.Now so
	// that a test can hold it still (architecture 2), and it is refused when
	// nil for the reason the authenticator refuses it: a defaulted clock makes
	// the port an option.
	Clock ports.Clock

	// Authenticator is what the routes that require a token ask (plan 6.2's
	// table). Three of this handler's routes require one — /Users/Me,
	// /Users/{userId} and T15's /Users/Configuration — and two do not:
	// AuthenticateByName reads the credential out of its body, and
	// /Users/Public reads no credential at all even when one is present.
	//
	// It is refused when nil for the reason every other member is, and the
	// reason is sharper here: a nil Authenticator admits nobody, so a server
	// wired without one would answer 401 to every authenticated /Users route
	// while answering the two that need no token perfectly. That is a failure
	// that looks like a working server.
	Authenticator Authenticator
}

// UsersHandler answers the /Users routes of feature 002.
type UsersHandler struct {
	installationID string
	login          *users.Login
	accounts       ports.UserStore
	sessions       ports.SessionStore
	clock          ports.Clock
	authenticator  Authenticator
}

// NewUsersHandler builds the handler for the /Users routes.
//
// Every member is required and a missing one is a failure to start, for the
// reason NewSystemHandler gives: the inputs come from the process's own start,
// so nothing here can recover from one being absent, and a handler that
// defaulted a port would answer requests with a store nobody wired.
func NewUsersHandler(cfg UsersHandlerConfig) (*UsersHandler, error) {
	if cfg.InstallationID == "" {
		return nil, errors.New("httpapi: the users handler needs an installation identity, and was given none")
	}
	if cfg.Login == nil {
		return nil, errors.New("httpapi: the users handler needs a login path, and was given none")
	}
	if cfg.Accounts == nil {
		return nil, errors.New("httpapi: the users handler needs an account store, and was given none")
	}
	if cfg.Sessions == nil {
		return nil, errors.New("httpapi: the users handler needs a session store, and was given none")
	}
	if cfg.Clock == nil {
		return nil, errors.New("httpapi: the users handler needs a clock, and was given none")
	}
	if cfg.Authenticator == nil {
		return nil, errors.New("httpapi: the users handler needs an authenticator, and was given none")
	}
	return &UsersHandler{
		installationID: cfg.InstallationID,
		login:          cfg.Login,
		accounts:       cfg.Accounts,
		sessions:       cfg.Sessions,
		clock:          cfg.Clock,
		authenticator:  cfg.Authenticator,
	}, nil
}

// authenticateUserByName is spec 3.3's request body.
//
// # The action parameter's name, and why it is a constant here
//
// The reference declares this body as an action parameter called `request`
// [source: Jellyfin.Api/Controllers/UserController.cs:211 @ v10.11.11], and
// that name is on the wire in exactly one place: the `errors` map of the
// validation refusal, which names the parameter the binder could not fill
// (behaviours 1.11). It is never anything the client sent, so it cannot be
// derived from the request and has to be transcribed.
//
// # Both members are optional at the reference, and that decides a refusal
//
// `Username` and `Pw` are both `string?`
// [source: Jellyfin.Api/Models/UserDtos/AuthenticateUserByName.cs @ v10.11.11],
// and the `[Required]` sits on the *parameter* rather than on either member. So
// a body that is valid JSON and carries neither is bound successfully and
// refused later as a credential — the 25-byte `401` — while a body that is not
// JSON at all fails the binder and is the validation `400`. plan 7's row reads
// "a body that is not JSON, **or a required member missing**", and the required
// member the reference has is the parameter itself: it is reported under the
// second key of the same refusal, beside `"$"`, on one response rather than on
// two.
type authenticateUserByName struct {
	Username string
	Pw       string
}

// loginActionParameter is the reference's own name for the body above.
const loginActionParameter = "request"

// The two keys the login route's validation refusal names.
//
// `"$"` is the deserialiser's key for a document that could not be read at all,
// and the message under it is **this parser's** where the reference's is
// .NET's. behaviours 1.11 records that as the one half of this shape that
// cannot be matched at any price short of writing a JSON parser to fail like
// another one; the key, the status, the content type and the second entry all
// match.
const jsonDocumentKey = "$"

// AuthenticationResult is spec 3.3's `200` body.
//
// The four members are the reference's, in its declaration order
// [source: MediaBrowser.Controller/Authentication/AuthenticationResult.cs:12-28 @ v10.11.11],
// which is also the order spec 3.3's sample body shows.
type AuthenticationResult struct {
	// User is the whole user object of spec 3.5, built by the one filler every
	// route returning one calls.
	User UserObject

	// SessionInfo is the session this authentication opened.
	SessionInfo SessionInfo

	// AccessToken is 32 lowercase hex characters
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
	AccessToken string

	// ServerId is the installation identity, the same value the user object
	// carries under the same name.
	ServerId string
}

// SessionInfo is spec 3.8's session as this feature answers it.
//
// # The order is the reference's declaration order
//
// [source: MediaBrowser.Model/Dto/SessionInfoDto.cs:17-185 @ v10.11.11], with
// the members v1 does not send left out rather than reordered, so that the
// members that do travel appear in the positions the reference puts them in.
//
// # What v1 does not send, and why it is stated here rather than discovered
//
// The reference's DTO declares twenty-eight members and this model declares
// fifteen. Every omission is a member spec 3.8 does not name, and they fall
// into three groups:
//
//   - `PlayState`, `NowPlayingItem`, `NowViewingItem`, `TranscodingInfo`,
//     `NowPlayingQueue`, `NowPlayingQueueFullItems`, `LastPausedDate` and
//     `PlaylistItemId` are playback state, which spec 3.8 gives to feature 007
//     ("while something is playing"). Three of them are non-null on the
//     reference for a session that has never played anything — `PlayState` is
//     constructed eagerly and the two queues are empty arrays
//     [source: MediaBrowser.Controller/Session/SessionInfo.cs:44-48 @ v10.11.11]
//     — so their absence here is a **difference on a fresh session** and not
//     only a deferred feature. It is stated rather than silently accepted, and
//     it is a register row 010's run would raise.
//   - `AdditionalUsers`, `IsActive`, `HasCustomDeviceName`, `DeviceType`,
//     `UserPrimaryImageTag` and `ServerId` are members of a session concept v1
//     does not have: no second user on a session, no liveness a
//     request-response server can report, no operator-renamed device, no
//     device typing and no user avatar (spec 3.5's `PrimaryImageTag` is absent
//     for the same reason).
//   - `Capabilities` is spec 3.8's declared capabilities, and it is added by
//     the task that serves `POST /Sessions/Capabilities/Full`: a session that
//     has just authenticated has posted none, so it would be absent on this
//     route's body whichever task declared it.
//
// spec 3.8 lists what a session carries and this model is that list; the
// counting is written down because Principle I is a count as much as it is a
// spelling, and thirteen absent members is the kind of gap that is only ever
// found by counting.
type SessionInfo struct {
	// RemoteEndPoint is the requester's address as this server normalised it,
	// which is the reference's own choice of value
	// [source: Jellyfin.Api/Controllers/UserController.cs:223 @ v10.11.11].
	RemoteEndPoint string

	// PlayableMediaTypes is hoisted verbatim from the client's declared
	// capabilities (spec 3.8). It is non-nullable at the reference and
	// initialised empty, so it travels as `[]` for a session that has declared
	// nothing — never as `null`, which is what a nil slice would serialise as.
	PlayableMediaTypes []string

	// Id is the derived session identifier: 32 lowercase hex over the client
	// name and the device identifier (sessions.DeriveID, plan 6.5). It is
	// deliberately not the reference's MD5 of the two concatenated, and
	// allowlist.yaml already declares the difference.
	Id string

	// UserId is whoever authenticated on this client and device last, which on
	// this route is always the caller who just did.
	UserId string

	UserName string
	Client   string

	// LastActivityDate is the instant of this authentication.
	LastActivityDate units.Time

	// LastPlaybackCheckIn is `0001-01-01T00:00:00.0000000Z` for a session that
	// has never played anything — .NET's minimum date, and a **value** rather
	// than an absence
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
	// It is a non-pointer for exactly that reason: the zero units.Time
	// serialises to those bytes, and a pointer would invite an implementation
	// to send nothing where the reference sends a date. The column T1 shipped
	// for it is NOT NULL for the same reason.
	LastPlaybackCheckIn units.Time

	DeviceName         string
	DeviceId           string
	ApplicationVersion string

	// SupportsMediaControl and SupportsRemoteControl are the server's judgement
	// about the client rather than the client's declaration, and v1's judgement
	// is always false: it has no control channel. The reference reports false
	// here for a request-response client too, while echoing a declared `true`
	// back inside `Capabilities`
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28],
	// so this is measured parity rather than a gap (behaviours 2.14).
	SupportsMediaControl  bool
	SupportsRemoteControl bool

	// SupportedCommands is hoisted verbatim from the declared capabilities, and
	// is `[]` until something is declared, for the reason PlayableMediaTypes
	// is.
	SupportedCommands []string
}

// AuthenticateByName answers POST /Users/AuthenticateByName (spec 3.3).
//
// # The order of the refusals is the reference's, and it is observable
//
//  1. **The body is bound first.** The reference binds the action parameter
//     before the action runs, so a request whose body is not JSON is refused
//     with problem details whatever its headers carried.
//  2. **Then the four client components.** `Client`, `Device`, `DeviceId` and
//     `Version` must all be non-empty. The reference checks exactly those four,
//     at the session manager rather than at the header parser
//     [source: Emby.Server.Implementations/Session/SessionManager.cs:1589-1592 @ v10.11.11],
//     and this is the route behaviours 2.13 calls "fatal to a route, not to a
//     parse": the identical header carrying no `DeviceId` is served `200`
//     everywhere else, and a parser that refused it would refuse requests the
//     reference serves on every route at once (plan 6.3).
//  3. **Then the credential**, which is the domain's (internal/users). It
//     answers 401 for an unknown username or a wrong password and 403 for a
//     disabled account, and the check happens *after* the four components
//     because the reference's does — so a request with a bad header and a bad
//     password is a `400` and never a `401`.
//
// Steps 2 and 3 answer the same twenty-five bytes and differ only in status,
// which is why they are compared against one golden body rather than four
// written alike (behaviours 1.11, spec 3.3).
func (h *UsersHandler) AuthenticateByName() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body authenticateUserByName
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			// The body could not be read off the connection. Nothing was
			// wrong with the request as far as this server can tell, so it is
			// not a refusal of it.
			WriteInternalServerError(w)
			return
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				jsonDocumentKey:      {err.Error()},
				loginActionParameter: {requiredBodyMessage(loginActionParameter)},
			})
			return
		}

		client := presentedClientIdentification(r)
		if client.Client == "" || client.Device == "" || client.DeviceID == "" || client.Version == "" {
			WriteControllerRefusal(w, http.StatusBadRequest)
			return
		}

		// The plaintext is built here and nowhere else. users.Plaintext redacts
		// through every formatting verb and through slog, which is half of what
		// AC-11 stands on; the other half is that this struct is never logged
		// whole (plan 7).
		account, err := h.login.Authenticate(r.Context(), body.Username, users.NewPlaintext(body.Pw))
		switch {
		case err == nil:
		case errors.Is(err, users.ErrCredentialsRefused):
			WriteControllerRefusal(w, http.StatusUnauthorized)
			return
		case errors.Is(err, users.ErrAccountDisabled):
			WriteControllerRefusal(w, http.StatusForbidden)
			return
		default:
			// A store failure is a 500 and never a 401: a client told 401
			// discards a credential that was fine (plan 7).
			WriteInternalServerError(w)
			return
		}

		result, err := h.openSession(r, client, account)
		if err != nil {
			WriteInternalServerError(w)
			return
		}

		if err := wire.Write(w, http.StatusOK, result, NegotiateProfile(r)); err != nil {
			WriteInternalServerError(w)
		}
	}
}

// openSession is plan 6.5's transaction and the body it produces.
//
// The four steps, in the order plan 6.5 writes them and the reference performs
// them:
//
//  1. Mint the token — sixteen bytes of system randomness as thirty-two
//     lowercase hex. It is the one identifier in this project that must not be
//     derived: a bearer credential that is a function of anything an attacker
//     knows is not a credential.
//  2. Derive the session identifier from the client name and the device
//     identifier.
//  3. Revoke every token this user holds on this device, which is spec 3.3's
//     "authenticating again from the same DeviceId replaces that session" and
//     the reference's own order — it logs the existing devices out before it
//     creates the new one
//     [source: Emby.Server.Implementations/Session/SessionManager.cs:1653-1681 @ v10.11.11].
//     It is a separate call because OpenSession must not revoke: it would
//     delete the token it is minting.
//  4. Insert or update the session row at the derived identifier, with the
//     token, as one statement.
//
// # The response describes the row, not the value that was written
//
// The session is read back through the token that was just minted rather than
// assembled from the struct handed to the store. The two differ on a
// re-authentication: OpenSession leaves `created_at`, `capabilities_document`
// and `last_playback_check_in_at` alone when it updates, so a body built from
// the value written would report a zero `LastPlaybackCheckIn` for a session
// that had played something — invisible today, because nothing in v1 writes
// that column, and a wire bug the day feature 007 does. Reading it back through
// the digest is one indexed lookup and makes the body a description of the
// stored row.
func (h *UsersHandler) openSession(r *http.Request, client ClientIdentification, account ports.User) (AuthenticationResult, error) {
	token, err := newAccessToken()
	if err != nil {
		return AuthenticationResult{}, err
	}
	digest := sessions.TokenDigest(token)
	now := h.clock.Now()

	if err := h.sessions.RevokeTokensFor(r.Context(), account.ID, client.DeviceID); err != nil {
		return AuthenticationResult{}, err
	}
	written := ports.Session{
		ID:                 sessions.DeriveID(client.Client, client.DeviceID),
		UserID:             account.ID,
		Client:             client.Client,
		DeviceID:           client.DeviceID,
		DeviceName:         client.Device,
		ApplicationVersion: client.Version,
		RemoteEndpoint:     remoteEndpoint(r),
		CreatedAt:          now,
		LastActivityAt:     now,
	}
	if err := h.sessions.OpenSession(r.Context(), written, digest); err != nil {
		return AuthenticationResult{}, err
	}

	stored, _, found, err := h.sessions.SessionByTokenDigest(r.Context(), digest)
	if err != nil {
		return AuthenticationResult{}, err
	}
	if !found {
		return AuthenticationResult{}, errors.New("httpapi: the session just opened cannot be resolved by the token just minted")
	}

	object, err := userObject(r.Context(), h.accounts, h.installationID, account)
	if err != nil {
		return AuthenticationResult{}, err
	}

	return AuthenticationResult{
		User:        object,
		SessionInfo: sessionInfo(stored, account.Username),
		AccessToken: token,
		ServerId:    h.installationID,
	}, nil
}

// sessionInfo builds spec 3.8's object from the stored row.
//
// The two hoisted arrays are empty and non-nil rather than nil: internal/wire
// serialises a nil slice as `null`, and the reference's own members are
// non-nullable and initialised empty. The task that serves
// POST /Sessions/Capabilities/Full fills them from the declaration.
func sessionInfo(session ports.Session, username string) SessionInfo {
	return SessionInfo{
		RemoteEndPoint:        session.RemoteEndpoint,
		PlayableMediaTypes:    []string{},
		Id:                    session.ID,
		UserId:                session.UserID,
		UserName:              username,
		Client:                session.Client,
		LastActivityDate:      session.LastActivityAt,
		LastPlaybackCheckIn:   session.LastPlaybackCheckInAt,
		DeviceName:            session.DeviceName,
		DeviceId:              session.DeviceID,
		ApplicationVersion:    session.ApplicationVersion,
		SupportsMediaControl:  false,
		SupportsRemoteControl: false,
		SupportedCommands:     []string{},
	}
}

// accessTokenBytes is the size of a minted token before it is rendered as hex:
// the same 128 bits the reference mints
// [source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/Security/Device.cs:29 @ v10.11.11],
// and thirty-two lowercase hex characters afterwards, which is the shape spec
// 3.3 measures.
const accessTokenBytes = 16

// newAccessToken mints one bearer credential.
//
// A failure to read the system's randomness is returned rather than survived:
// the alternative is issuing a predictable credential, and answering 500 is the
// right answer to a server that cannot produce one.
func newAccessToken() (string, error) {
	var raw [accessTokenBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

// remoteEndpoint is the requester's address, normalised the way the rest of
// this package normalises it (requestFacts): the host half of RemoteAddr,
// parsed, and unmapped so that a dual-stack listener's `::ffff:192.168.1.44`
// is the IPv4 address an operator would recognise.
//
// A request that never came off a connection has no address and reports the
// empty string rather than netip's "invalid IP", which would be a sentence on
// the wire.
func remoteEndpoint(r *http.Request) string {
	address := requestFacts(r).RemoteAddress
	if !address.IsValid() {
		return ""
	}
	return address.String()
}
