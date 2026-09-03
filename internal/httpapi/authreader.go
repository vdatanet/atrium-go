package httpapi

import (
	"net/http"
	"net/url"
	"strings"
)

// The three header names and the two query names a credential may arrive in.
//
// They are constants rather than literals at the call sites because the reader
// below and the login route both spell them, and a header name spelled twice
// is a header name that can be spelled differently twice.
const (
	// AuthorizationHeader is the standard field, carrying 002 spec 3.2's
	// grammar rather than a Bearer token.
	AuthorizationHeader = "Authorization"

	// EmbyAuthorizationHeader is the historical Emby spelling of the same
	// grammar. It was missing from 002's specification until it was measured
	// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28],
	// and a server implementing only the other four would refuse clients that
	// have worked against the reference for years (behaviours 2.4).
	EmbyAuthorizationHeader = "X-Emby-Authorization"

	// EmbyTokenHeader carries a bare token and nothing else.
	EmbyTokenHeader = "X-Emby-Token"
)

// The query names, in the two spellings 002 spec 3.1 lists. They are two
// names rather than one name in two cases: neither folds to the other, so a
// request may carry both.
const (
	apiKeyQueryName           = "ApiKey"
	apiKeyUnderscoreQueryName = "api_key"
)

// The five component names of 002 spec 3.2's grammar, spelled the way a client
// must spell them. Matching is case-sensitive, so these literals are the whole
// of the rule (see ParseClientIdentification).
const (
	clientComponent   = "Client"
	deviceComponent   = "Device"
	deviceIDComponent = "DeviceId"
	versionComponent  = "Version"
	tokenComponent    = "Token"
)

// ClientIdentification is what one Authorization or X-Emby-Authorization field
// value yields: the four components of 002 spec 3.2 by which a client names
// itself, and the Token component by which it may also authenticate.
//
// The zero value is what a field value carrying no scheme word, or a scheme
// word this grammar does not know, yields — and it is also what an absent
// header yields. Those are deliberately the same value: a header that is
// present but says nothing is, to every caller, a header that said nothing.
//
// # Every member is optional, and that is not this type's decision to make
//
// A missing DeviceId is fatal on exactly one route and on no header. A request
// to any other route carrying a header with no DeviceId is served normally,
// measured 200
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], and
// it is POST /Users/AuthenticateByName that refuses with 400, because that
// route needs the components to open a session (behaviours 2.13). The
// reference draws the line in the same place — its four arguments are checked
// at the session manager rather than at the parser
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1589-1592 @ v10.11.11].
//
// So this type carries no notion of valid. A parser that raised would refuse,
// on every route at once, requests the reference serves.
type ClientIdentification struct {
	// Client is the application name, e.g. "Jellyfin Android".
	Client string

	// Device is the human-readable device name shown in session lists.
	Device string

	// DeviceID is the stable per-installation identifier. It is what
	// identifies a session (002 spec 3.2), and it is spelled DeviceId on the
	// wire.
	DeviceID string

	// Version is the client version string.
	Version string

	// Token is the access token, when the header carried one. It is a
	// component of the same grammar rather than a separate mechanism: the
	// reference reads both header names with one parser and authenticates a
	// Token found in either.
	Token string
}

// ParseClientIdentification reads 002 spec 3.2's grammar out of one field
// value:
//
//	scheme-word  1*(component)
//	component    name "=" ( quoted-string | bare )  [ "," ]
//
// # It never fails
//
// There is no error return and there is no "unparseable" answer. It returns
// what it could read, and the zero ClientIdentification — nothing at all — is
// a legitimate result rather than a refusal (plan 6.3). Whether that is fatal
// is the route's question, not the parser's, and the route that answers yes is
// POST /Users/AuthenticateByName alone.
//
// # The scheme word decides whether anything is read
//
// It is compared case-insensitively against MediaBrowser and Emby. Any other
// word — Bearer, a made-up word, or a field value with no space in it at all —
// and nothing is read out of the header, not even components that would
// otherwise have parsed
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:246-268 @ v10.11.11].
// This is the case a lenient parser passes and the reference refuses.
//
// # Lenient in six ways, strict in two, and the two are the interesting half
//
// Accepted: components in any order, values quoted or bare, no space after a
// comma, a space before one, extra spaces after the scheme word, a trailing
// comma, and unknown components, which are ignored. Refused: whitespace around
// the "=", and a lowercase component name — each yields *no component*, not a
// different value.
//
// An earlier version of 002 spec 3.2 claimed whitespace around the "=" was
// accepted. It is not
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28], and
// matching the reference matters more than being kind: no working client sends
// either form, because the reference refuses both, and accepting them would
// let somebody build a client against Atrium that fails against Jellyfin
// (behaviours 2.12, and 6's non-improvements).
//
// # Where the reference's own mechanism differs from this reading
//
// The reference does not have a rule about whitespace around the "="; it has a
// value scanner whose consequence is one. It trims the *name* it reads before
// the "=" and does not trim the *value* after it, then strips quotes from the
// value's ends
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:276-317 @ v10.11.11].
// So `Token = "x"` yields, there, a Token component whose value is ` "x` —
// which is not a token, which is why the measurement recorded 401 — while
// `Token ="x"` (whitespace only *before* the "=") appears to yield a clean `x`.
//
// The probe measured the refusal, not the mechanism: both readings answer 401
// to the request it sent. 002 spec 3.2 and plan 6.3 both state the rule as
// "whitespace around the =", symmetrically, and that is what is implemented
// here. The disagreement is recorded rather than resolved silently
// (AGENTS 1.3): it is observable on a request nobody has sent — a header
// carrying `Client ="x"`, whose Client this server drops and the reference
// appears to keep — and one probe settles it.
func ParseClientIdentification(field string) ClientIdentification {
	rest, ok := afterSchemeWord(field)
	if !ok {
		return ClientIdentification{}
	}

	var read ClientIdentification
	for _, component := range splitComponents(rest) {
		name, value, ok := readComponent(component)
		if !ok {
			continue
		}

		// The five names are matched case-sensitively, and this switch is the
		// only place that decides it. A lowercase name is not a component
		// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28];
		// the reference's parts are a dictionary with an ordinal comparer,
		// read by five exact spellings
		// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:86-90 @ v10.11.11],
		// so `token=` there is a key nothing ever asks for. An unknown name —
		// including a lowercase one — falls through and is ignored rather than
		// refused, which is 002 spec 3.2's own row.
		switch name {
		case clientComponent:
			read.Client = value
		case deviceComponent:
			read.Device = value
		case deviceIDComponent:
			read.DeviceID = value
		case versionComponent:
			read.Version = value
		case tokenComponent:
			read.Token = value
		}
	}
	return read
}

// schemeWords are the two scheme words that admit a field value to the
// grammar, compared case-insensitively.
//
// Emby is the historical spelling and it is not a synonym this project
// invented: the reference accepts it beside MediaBrowser
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:258-259 @ v10.11.11].
var schemeWords = [...]string{"MediaBrowser", "Emby"}

// afterSchemeWord splits the scheme word off a field value and reports whether
// it is one this grammar knows.
//
// The separator is a single space, and the first one: a field value with no
// space carries no components, whatever else it says. That is the reference's
// own rule and its own spelling
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:248-254 @ v10.11.11],
// so a tab after the scheme word is not a separator here either.
func afterSchemeWord(field string) (string, bool) {
	at := strings.IndexByte(field, ' ')
	if at < 0 {
		return "", false
	}
	word := field[:at]
	for _, known := range schemeWords {
		if equalFoldASCII(word, known) {
			return field[at+1:], true
		}
	}
	return "", false
}

// splitComponents cuts a field value's remainder on the commas that are not
// inside a quoted value.
//
// Quoting is tracked by toggling on every double quote, which is what the
// reference does
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:287-302 @ v10.11.11],
// so a comma inside a quoted device name does not end its component. An empty
// piece — from a trailing comma, or from two commas in a row — carries no "="
// and is dropped by readComponent, which is how a trailing comma is accepted
// without being a component.
func splitComponents(rest string) []string {
	var components []string
	quoted := false
	start := 0
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case '"':
			quoted = !quoted
		case ',':
			if !quoted {
				components = append(components, rest[start:i])
				start = i + 1
			}
		}
	}
	return append(components, rest[start:])
}

// readComponent reads one name="value" pair, and reports whether it is a
// component at all.
//
// Surrounding whitespace belongs to the separator rather than to the
// component, so it is removed first: that is what makes extra spaces after the
// scheme word, a missing space after a comma and a space before one all
// accepted. What survives that trim is the strict half — whitespace between
// the name and the "=", or between the "=" and the value, is whitespace
// *around the "="*, and such a piece is not a component.
func readComponent(component string) (name, value string, ok bool) {
	component = strings.Trim(component, " \t")

	at := strings.IndexByte(component, '=')
	if at < 0 {
		return "", "", false
	}

	name, value = component[:at], component[at+1:]
	if endsInWhitespace(name) || beginsWithWhitespace(value) {
		return "", "", false
	}

	// Quotes come off the ends. The reference trims the quote character
	// rather than removing a matched pair
	// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:313 @ v10.11.11],
	// so an unbalanced quote is stripped there and is stripped here; a quote
	// *inside* a value is data and survives.
	return name, strings.Trim(value, `"`), true
}

func beginsWithWhitespace(s string) bool { return s != "" && (s[0] == ' ' || s[0] == '\t') }

func endsInWhitespace(s string) bool {
	return s != "" && (s[len(s)-1] == ' ' || s[len(s)-1] == '\t')
}

// PresentedToken reports the access token a request presents, over the five
// mechanisms of 002 spec 3.1, in the order the reference resolves them:
//
//	Authorization > X-Emby-Authorization > X-Emby-Token > ?ApiKey= / ?api_key=
//
// measured pair by pair, in both directions each time
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. An
// empty string is a request that presented no token; it is never an error,
// because there is nothing here that can fail.
//
// All five are required rather than a choice. Clients use the headers for API
// calls and the query forms for the URLs they hand to media players and image
// loaders, which set no headers — so a server implementing only the headers
// breaks playback and artwork while leaving browsing intact, which looks like
// a bug in the client (002 spec 3.1).
func PresentedToken(r *http.Request) string {
	return presentedToken(
		r.Header.Get(AuthorizationHeader),
		r.Header.Get(EmbyAuthorizationHeader),
		r.Header.Get(EmbyTokenHeader),
		r.URL.RawQuery,
	)
}

// presentedToken is the pure core: five mechanisms read off four strings, with
// no request and no route.
//
// # A header that is present but yields nothing does not stop the search
//
// A request carrying `Authorization: Bearer x` and `?api_key=<good>`
// authenticates. The first header contributes no Token component at all —
// ParseClientIdentification reads nothing out of a header whose scheme word is
// neither MediaBrowser nor Emby — so it is not a mechanism that disagreed, it
// is a mechanism that was not used. The measured precedence is between
// mechanisms that each produced a token, which is what the probe sent pair by
// pair (plan 6.1). An implementation shaped `if hasHeader { return }` answers
// 401 to that request.
//
// ⚠️ UNVERIFIED, and it is the one place this reader is knowingly more
// generous than the reference's source: the reference falls back from
// Authorization to X-Emby-Authorization only when the first header is
// *absent*, not when it is present and unparseable
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:231-238 @ v10.11.11].
// So a request carrying `Authorization: Bearer x` beside
// `X-Emby-Authorization: MediaBrowser Token="<good>"` is 401 there and 200
// here. No probe has sent that pair — the measured pairs each carried a
// readable scheme word — and plan 6.1 states the search rule in general terms,
// which is what is implemented. One request settles it.
//
// # The query names are read here and not by query canonicalisation
//
// 001 plan 6.2 keys a declared spelling by route, on the argument that the
// spelling that matters is the one this server's own handler binds. No handler
// binds a credential: it is a property of the request, accepted on every
// authenticated route in the project. Declaring ApiKey and api_key on all
// fifty-nine rows to make one stage cover them would be a declaration nobody
// reads — and, decisively, a reader that went through canonicalisation would
// stop working on a request whose path matches no route, which is exactly the
// request a credential reader is most needed for.
func presentedToken(authorization, embyAuthorization, embyToken, rawQuery string) string {
	if token := ParseClientIdentification(authorization).Token; token != "" {
		return token
	}
	if token := ParseClientIdentification(embyAuthorization).Token; token != "" {
		return token
	}
	// X-Emby-Token is the whole field value, trimmed of surrounding
	// whitespace and nothing else (plan 6.1). net/http has already trimmed
	// optional whitespace off a field value it read from a connection; the
	// trim is here because this is a function over strings and its rows are
	// written directly.
	if token := strings.Trim(embyToken, " \t"); token != "" {
		return token
	}
	return queryToken(rawQuery)
}

// queryToken reads ApiKey and api_key off the raw query string.
//
// # Off the raw query, and folded here
//
// The names are matched case-insensitively, so ApiKey, api_key, APIKEY and
// apikey all name a credential — behaviours 1.15's rule that the reference
// treats a query name case-insensitively, applied to the two names 002 spec
// 3.1 lists. Reading the raw query rather than r.URL.Query() is what keeps
// this working on a request nobody routed, and reading it here rather than in
// the canonicalisation stage is plan 6.1's decision.
//
// # First occurrence wins
//
// ApiKey and api_key are two names, not one name in two cases: they do not
// fold together, so no precedence between them is needed and none was
// measured — behaviours 2.4 records that "the two query spellings were never
// set against each other". When both are present and disagree, this reader
// takes whichever comes first in the raw query string, which plan 6.1 states
// so that it is a decision rather than a property of a map, and names in its
// 9 as the one place a client could observe an order nobody measured.
//
// ⚠️ UNVERIFIED: the reference's source resolves it the other way. It reads
// ApiKey and consults api_key only when that is empty
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:103-111 @ v10.11.11],
// which is a name precedence rather than a positional one, so
// `?api_key=a&ApiKey=b` yields b there and a here. Nothing has measured it and
// plan 6.1 took the positional decision deliberately, so it is implemented as
// planned and recorded here for the register.
func queryToken(rawQuery string) string {
	if rawQuery == "" {
		return ""
	}
	for _, fragment := range strings.Split(rawQuery, "&") {
		name, value, hasValue := strings.Cut(fragment, "=")
		if !hasValue {
			continue
		}
		if !equalFoldASCII(name, apiKeyQueryName) && !equalFoldASCII(name, apiKeyUnderscoreQueryName) {
			continue
		}
		if decoded, err := url.QueryUnescape(value); err == nil {
			value = decoded
		}
		if value != "" {
			return value
		}
	}
	return ""
}
