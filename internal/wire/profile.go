package wire

// Profile is the content profile a response is written under: the outcome of
// negotiating one request's `Accept` header (spec 3.0.2).
//
// # Why this and not the naming policy
//
// There are three declared content types and two behaviours
// `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`.
// The plain type and the `PascalCase` profile answer identically; only
// `CamelCase` changes the property names. So the naming policy is a two-valued
// thing and the content type is a three-valued one, and a body written under
// NamingPascal cannot say on its own whether it is answering `application/json`
// or `application/json; profile="PascalCase"`.
//
// That is why Write takes a Profile rather than a Naming. The negotiation has
// one winner and that winner decides both halves together (plan 6.3) — which
// keeps behaviours 1.10's rule intact: the content type belongs to the thing
// that produced the body, and a middleware that stamped it afterwards would be
// guessing at exactly the distinction this type carries.
//
// The zero value is ProfilePlain, which is what a request with no `Accept`
// header gets and what the overwhelming majority of clients ask for. A caller
// that forgets therefore sends a declared content type rather than an empty
// header — and never the wrong naming policy, because both PascalCase answers
// share one.
type Profile int

const (
	// ProfilePlain is `application/json` with no profile: PascalCase names,
	// and a content type that names no profile. It is what an `Accept` header
	// that asks for nothing in particular gets, and what every fallback in
	// negotiation lands on.
	ProfilePlain Profile = iota

	// ProfilePascal is `application/json; profile="PascalCase"`. It answers
	// with the same bytes as ProfilePlain — spec 3.0.2's "three names for two
	// behaviours" — and differs only in echoing the profile it matched, which
	// is what AC-9's first two requests assert.
	ProfilePascal

	// ProfileCamel is `application/json; profile="CamelCase"`: the same values
	// under camelCase property names, at every depth, with dictionary keys
	// untouched (behaviours 1.13).
	ProfileCamel
)

// answer is what a profile decides. The two halves are in one table because
// they are one decision: plan 6.3's winner "decides two things together".
type answer struct {
	naming      Naming
	contentType string
}

// profileAnswers is behaviours 1.13's measured table, transcribed — including
// the charset on all three and the profile before it
// `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`.
//
// The echoed profile carries the canonical spelling, quoted, whatever spelling
// the request used. A request matches by *naming* a profile — leniently, see
// negotiate.go — and what comes back is the media type the matched formatter
// was registered with, which is a constant
// `[source: Jellyfin.Api/Formatters/CamelCaseJsonProfileFormatter.cs:15-18,
// src/Jellyfin.Extensions/Json/JsonDefaults.cs:16,21 @ v10.11.11]`. The probe
// measured the canonical request; the source is what says the echo does not
// vary with how the request spelled it.
var profileAnswers = map[Profile]answer{
	ProfilePlain:  {NamingPascal, "application/json; charset=utf-8"},
	ProfilePascal: {NamingPascal, `application/json; profile="PascalCase"; charset=utf-8`},
	ProfileCamel:  {NamingCamel, `application/json; profile="CamelCase"; charset=utf-8`},
}

// Naming reports the property-naming policy this profile is answered under, and
// whether the profile is one this package knows.
//
// The second return value is not decoration. A Profile this package does not
// recognise must not become PascalCase by default: a client that asked for
// camelCase and was answered in PascalCase gets an empty object out of its
// decoder rather than a degraded response (behaviours 1.13), and that is
// invisible from here. Callers get an unusable answer they have to look at.
func (p Profile) Naming() (Naming, bool) {
	known, ok := profileAnswers[p]
	return known.naming, ok
}

// ContentType reports the `Content-Type` this profile is echoed with, and
// whether the profile is one this package knows, for the same reason.
func (p Profile) ContentType() (string, bool) {
	known, ok := profileAnswers[p]
	return known.contentType, ok
}

// String names the profile as the wire spells it, so a test failure reads as
// `CamelCase` rather than as `2`.
func (p Profile) String() string {
	switch p {
	case ProfilePlain:
		return "plain"
	case ProfilePascal:
		return "PascalCase"
	case ProfileCamel:
		return "CamelCase"
	default:
		return "unknown profile"
	}
}
