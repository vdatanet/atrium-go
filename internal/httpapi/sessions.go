package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// The three query parameter names GET /Sessions binds, spelled the way spec 3.8
// declares them and the way the reference declares them
// [source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11].
//
// They are constants because each is written in three places that must agree —
// the declaration V1QuerySpellings folds every spelling of a name to, the read
// below, and the key of the validation refusal that names a value the route
// could not bind. A parameter read under a spelling the folder does not declare
// is ignored on every request that does not happen to send it in exactly that
// case, which behaviours 1.15 makes a wrong answer with a 200 on it.
const (
	deviceIDParameter             = "deviceId"
	controllableByUserIDParameter = "controllableByUserId"
	activeWithinSecondsParameter  = "activeWithinSeconds"
)

// capabilitiesActionParameter is the reference's own name for the body of
// POST /Sessions/Capabilities/Full
// [source: Jellyfin.Api/Controllers/SessionController.cs:380-382 @ v10.11.11].
//
// Like the login route's `request` and the configuration route's `userConfig`,
// it reaches the wire in exactly one place — the second key of the validation
// refusal, which names the action parameter the binder could not fill
// (behaviours 1.11) — and it is never anything the client sent, so it is
// transcribed rather than derived.
const capabilitiesActionParameter = "capabilities"

// SessionsHandlerConfig is what the two /Sessions routes of feature 002 are
// built from.
type SessionsHandlerConfig struct {
	// Sessions is the list GET /Sessions narrows and the row
	// POST /Sessions/Capabilities/Full writes a declaration into.
	Sessions ports.SessionStore

	// Accounts answers the one thing a session row does not carry: the
	// username of whoever authenticated on it. spec 3.8 puts `UserName` among
	// the identity fields, and ports.Session holds an identifier because
	// 002 plan 3 makes the session domain know a user by identifier and never
	// the reverse.
	Accounts ports.UserStore

	// Authenticator is what both routes ask. Both require a token: spec 3.8
	// answers a caller's own sessions and nobody else's, so a route that
	// admitted an unauthenticated request would have no caller to narrow with.
	Authenticator Authenticator

	// Clock is `now` for activeWithinSeconds, which is a window measured back
	// from the instant the request is answered. It is a port rather than a call
	// to time.Now for the reason every other clock in this package is one
	// (architecture 2), and it is refused when nil because a defaulted clock
	// makes the port an option.
	Clock ports.Clock
}

// SessionsHandler answers the two /Sessions routes of feature 002.
type SessionsHandler struct {
	sessions      ports.SessionStore
	accounts      ports.UserStore
	authenticator Authenticator
	clock         ports.Clock
}

// NewSessionsHandler builds the handler for the /Sessions routes.
//
// Every member is required and a missing one is a failure to start, for the
// reason NewUsersHandler gives: the inputs come from the process's own start,
// so nothing here can recover from one being absent.
func NewSessionsHandler(cfg SessionsHandlerConfig) (*SessionsHandler, error) {
	if cfg.Sessions == nil {
		return nil, errors.New("httpapi: the sessions handler needs a session store, and was given none")
	}
	if cfg.Accounts == nil {
		return nil, errors.New("httpapi: the sessions handler needs an account store, and was given none")
	}
	if cfg.Authenticator == nil {
		return nil, errors.New("httpapi: the sessions handler needs an authenticator, and was given none")
	}
	if cfg.Clock == nil {
		return nil, errors.New("httpapi: the sessions handler needs a clock, and was given none")
	}
	return &SessionsHandler{
		sessions:      cfg.Sessions,
		accounts:      cfg.Accounts,
		authenticator: cfg.Authenticator,
		clock:         cfg.Clock,
	}, nil
}

// clientCapabilities is the part of a posted declaration this server reads.
//
// Two members, because spec 3.8 hoists exactly two to the top level of a
// session — `PlayableMediaTypes` and `SupportedCommands`, verbatim — and reads
// nothing else. `SupportsMediaControl` is deliberately **not** here: hoisting it
// is the mistake behaviours 2.14 measures against, and a struct member that
// existed would be a member somebody could wire to the flag.
//
// It is a decode target and never a store shape. What is stored is the posted
// document whole (ReplaceCapabilities), which is what keeps behaviours 5.9's
// divergence — an unknown property surviving into /Sessions — the stated one
// rather than an accident. This type is why a document that could not be read
// is refused at the door rather than discovered on the way out.
type clientCapabilities struct {
	PlayableMediaTypes []string
	SupportedCommands  []string
}

// jsonNull is the one document that unmarshals into a struct without error and
// is not an object. It is the reference's missing body: the action parameter is
// `[FromBody, Required]`, so a null body is the validation refusal rather than
// an empty declaration
// [source: Jellyfin.Api/Controllers/SessionController.cs:380-382 @ v10.11.11].
var jsonNull = []byte("null")

// decodeCapabilities reads the two hoisted lists out of a posted declaration,
// and reports a document this route cannot serve.
//
// Every JSON value that is not an object fails: encoding/json refuses a number,
// a string, a boolean and an array against a struct, and `null` is refused here
// because it is the one that would not. That is the reference's binder, whose
// parameter is a `ClientCapabilitiesDto` and required.
func decodeCapabilities(document []byte) (clientCapabilities, error) {
	if bytes.Equal(bytes.TrimSpace(document), jsonNull) {
		return clientCapabilities{}, errors.New("json: the document is null, where an object was required")
	}
	var declaration clientCapabilities
	if err := json.Unmarshal(document, &declaration); err != nil {
		return clientCapabilities{}, err
	}
	return declaration, nil
}

// PostFullCapabilities answers POST /Sessions/Capabilities/Full (spec 3.8,
// AC-9, AC-13).
//
// # It stores the posted document whole, which is the divergence and not a
// shortcut
//
// behaviours 5.9: the reference accepts a property outside its own schema — the
// `204` — and **drops** it, so the session's `Capabilities` in GET /Sessions
// echoes the declared fields and not the stranger. This server keeps it. That
// is a recorded divergence with a closing mechanism written down, and it is the
// **opposite** of what POST /Users/Configuration does to an unknown property
// (spec 3.6): there the document decodes onto the reference's defaults and the
// stranger is gone before the store sees it. The two look like one question and
// are not, which is why the store is handed raw bytes here and a re-encoded
// document there.
//
// # Replacement, not merge, and there is no decode to make it one
//
// The route is named `Full` and behaves like it (behaviours 2.14): the stored
// document is overwritten, so a second post carrying fewer properties leaves
// none of the first behind. Nothing here reads the stored declaration, which is
// what makes the replacement structural rather than a rule somebody has to
// remember — but no round trip can see the difference, because everything a
// merge kept was posted at some point. Only a second, smaller post can.
//
// # The `id` query parameter is ignored, and it is U-14's shape a second time
//
// The reference declares `[FromQuery] string? id` and falls back to the
// caller's own session when it is absent or blank
// [source: Jellyfin.Api/Controllers/SessionController.cs:380-387 @ v10.11.11].
// spec 3.8 names no parameter on this route, and an unrecognised query value is
// ignored rather than rejected (behaviours 1.12) — so this route reads no query
// at all and a request naming somebody else's session writes to its **own**.
// That is the same silent write to the wrong row POST /Users/Configuration
// already costs this project once (U-14), and it is asserted as a test rather
// than left in this comment so that the day a probe measures it, a failing test
// names the behaviour that moved. It is also why this route contributes no row
// to V1QuerySpellings: a name nothing binds is a name nothing should fold.
//
// # The credential is read before the body is bound
//
// T14's order on this route's own two refusals, and the same reading of the
// reference's filter running ahead of its model binder (009 spec 3.8, measured
// 2026-09-01): a request with no credential and a body that is not JSON is the
// empty 401 and never the validation 400.
func (h *SessionsHandler) PostFullCapabilities() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := admitted(w, r, h.authenticator)
		if !ok {
			return
		}

		posted, err := io.ReadAll(r.Body)
		if err != nil {
			// The body could not be read off the connection, which is not the
			// request being wrong (users.go takes the same view).
			WriteInternalServerError(w)
			return
		}

		if _, err := decodeCapabilities(posted); err != nil {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				jsonDocumentKey:             {deserialiserMessage(err)},
				capabilitiesActionParameter: {requiredBodyMessage(capabilitiesActionParameter)},
			})
			return
		}

		if err := h.sessions.ReplaceCapabilities(r.Context(), caller.SessionID, posted); err != nil {
			WriteInternalServerError(w)
			return
		}

		writeNoContent(w)
	}
}

// Sessions answers GET /Sessions (spec 3.8, AC-4, AC-15).
//
// The three parameters are bound here and applied by the domain, in the one
// ordered function that owns their order (sessions.Visible). What this handler
// decides is the two things a domain function must not: what a query string
// means, and the 403.
//
// # The 403 is the handler's, before the domain is asked
//
// `controllableByUserId` naming anybody but the caller, from a caller who is
// not an administrator, is refused — the reference raises it in the controller,
// ahead of the session manager
// [source: Jellyfin.Api/Helpers/RequestHelpers.cs:67-85 @ v10.11.11]
// [source: Jellyfin.Api/Controllers/SessionController.cs:60-61 @ v10.11.11] —
// and it is 002 plan 6.10's decision for the reason 001 gives about shapes: a
// refusal decided in the domain would have to travel back out as a status the
// domain does not own.
//
// **This is AC-4's second half and the one request in 002 where a valid token
// is refused for who its holder is.** A route that declared neither parameter
// would answer `200` with the caller's own sessions to a caller the reference
// refuses, and a client that branches on the refusal would take the success
// path. It is the same twenty-five bytes as spec 3.3's four refusals — the
// status and the media type are measured
// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29] and
// the bytes are behaviours 1.11's rule applied, which is register U-18 — so it
// is byte-compared against the same golden those four are.
//
// # Naming another user's device is not the same answer as naming another user
//
// `deviceId` narrows the whole list before the visibility rule, so a caller who
// is not an administrator naming somebody else's device is `200` with `[]`. One
// route, two parameters that name somebody else's property, two answers (spec
// 3.8). The empty list is not a redaction of the refusal; it is the ordinary
// result of a filter that matched a row the caller may not see.
func (h *SessionsHandler) Sessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caller, ok := admitted(w, r, h.authenticator)
		if !ok {
			return
		}

		selection, ok := h.selection(w, r, caller)
		if !ok {
			return
		}

		open, err := h.sessions.Sessions(r.Context())
		if err != nil {
			WriteInternalServerError(w)
			return
		}

		// The reduction 002 plan 5 and T9 both name: the edge's caller carries
		// a users.Policy and the domain's carries the one flag this rule reads,
		// and internal/sessions may import neither internal/httpapi nor
		// internal/users. One visible line, here, rather than the domain
		// reaching up for a type it must not see.
		visible := sessions.Visible(open, sessions.Caller{
			UserID:          caller.UserID,
			IsAdministrator: caller.Policy.IsAdministrator,
		}, selection, h.clock.Now())

		body, err := h.describe(r, visible)
		if err != nil {
			WriteInternalServerError(w)
			return
		}
		if err := wire.Write(w, http.StatusOK, body, NegotiateProfile(r)); err != nil {
			WriteInternalServerError(w)
		}
	}
}

// describe turns the sessions the caller may see into spec 3.8's bodies.
//
// The slice is non-nil however empty it is, for the reason sessions.Visible
// never returns nil: internal/wire serialises a nil slice as `null` and spec
// 3.8 answers `[]`, which is a difference invisible to anything that parses the
// body (Principle VIII).
//
// The usernames come from one read of the accounts rather than one read per
// session. A session row holds an identifier and spec 3.8's body carries a
// name, and the alternative — a lookup inside the loop — is a request whose
// cost is the number of open sessions.
func (h *SessionsHandler) describe(r *http.Request, visible []ports.Session) ([]SessionInfo, error) {
	body := make([]SessionInfo, 0, len(visible))
	if len(visible) == 0 {
		return body, nil
	}

	accounts, err := h.accounts.Users(r.Context())
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(accounts))
	for _, account := range accounts {
		names[account.ID] = account.Username
	}

	for _, session := range visible {
		// A session naming an account this store does not have carries an
		// empty name rather than failing the request. v1 serves no route that
		// removes an account, so the state is unreachable; answering it with a
		// 500 would make a list of everybody's sessions fail for one row, which
		// is a worse answer than the member the reference declares nullable
		// [source: MediaBrowser.Model/Dto/SessionInfoDto.cs:59 @ v10.11.11].
		info, err := sessionInfo(session, names[session.UserID])
		if err != nil {
			return nil, err
		}
		body = append(body, info)
	}
	return body, nil
}

// selection binds spec 3.8's three parameters, writing the refusal itself when
// one of them cannot be bound or must not be honoured.
//
// # The order is bind, then refuse
//
// A `controllableByUserId` that is not an identifier is the binder's 400 before
// it is anybody's 403, because the reference binds the action's parameters
// before the action runs — the same order the login route's body takes
// (users.go).
//
// # Absent and empty are one request
//
// spec 3.8 measures `deviceId=` as ignored rather than as a device nothing is
// named after
// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29], and
// takes the same direction for the two it has not measured: an empty
// `activeWithinSeconds` and an empty `controllableByUserId` are absent
// (⚠️ UNVERIFIED, register U-17). The reference's own binder agrees for the
// declared types — an empty value binds a `Guid?` and an `int?` to null — and
// what stays unmeasured is the *route's* answer rather than the binding.
func (h *SessionsHandler) selection(w http.ResponseWriter, r *http.Request, caller *Caller) (sessions.Selection, bool) {
	query := r.URL.Query()

	selection := sessions.Selection{DeviceID: query.Get(deviceIDParameter)}

	// The width is Go's int and the reference's is Int32
	// [source: Jellyfin.Api/Controllers/SessionController.cs:58 @ v10.11.11],
	// so a value above 2147483647 binds here and fails the reference's binder.
	// Accepting more than the reference is the safe direction — no request that
	// succeeds there meets a refusal here — and narrowing to Int32 would make
	// sessions.Visible's saturating window unreachable from the wire, which is
	// the one arithmetic this route can get wrong. It is asserted as a
	// divergence test rather than left in this comment, and the register at T23
	// is owed the row.
	if presented := query.Get(activeWithinSecondsParameter); presented != "" {
		seconds, err := strconv.Atoi(presented)
		if err != nil {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				activeWithinSecondsParameter: {invalidValueMessage(presented)},
			})
			return sessions.Selection{}, false
		}
		selection.ActiveWithinSeconds = seconds
	}

	if presented := query.Get(controllableByUserIDParameter); presented != "" {
		named, wellFormed := canonicalIdentifier(presented)
		if !wellFormed {
			WriteValidationProblem(w, http.StatusBadRequest, map[string][]string{
				controllableByUserIDParameter: {invalidValueMessage(presented)},
			})
			return sessions.Selection{}, false
		}
		// The all-zero identifier is the caller's own, not nobody's: the
		// reference treats an empty Guid here exactly as it treats an absent
		// one and falls back to the authenticated user
		// [source: Jellyfin.Api/Helpers/RequestHelpers.cs:67-75 @ v10.11.11].
		// Without this line it would be "anybody else" and answered 403, where
		// the reference answers the caller's own rule — a refusal for a request
		// the reference serves, which is the direction behaviours 3.0.3 calls
		// the dangerous one. It differs from the *absent* case, which does not
		// take the controllable path at all.
		if named == emptyIdentifier {
			named = caller.UserID
		}
		if named != caller.UserID && !caller.Policy.IsAdministrator {
			WriteControllerRefusal(w, http.StatusForbidden)
			return sessions.Selection{}, false
		}
		selection.ControllableByUser = named
	}

	return selection, true
}
