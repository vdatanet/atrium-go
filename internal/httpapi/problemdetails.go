package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/vdatanet/atrium-go/internal/wire"
)

// The fourth shape of behaviours 1.11, and the one 001 and 002 T11 both left
// unwritten because it is a *model* rather than a shape.
//
// The other three empty refusals and the three T11 added are functions that set
// headers and write fixed bytes. This one carries a body with members, one of
// which — traceId — is per request by definition, so it goes through
// internal/wire like any other response and is declared as a struct like any
// other model.
//
// # What is measured, and what is a rule applied
//
// The shape, the content type and the two keys are measured
// [probe: tools/probe_playstate.py, Jellyfin 10.11.11, 2026-08-28]
// [probe: tools/probe_playlist_rename.py, Jellyfin 10.11.11, 2026-09-01]:
//
//	{"type": "https://tools.ietf.org/html/rfc9110#section-15.5.1",
//	 "title": "One or more validation errors occurred.", "status": 400,
//	 "errors": {"$": ["…"], "request": ["The request field is required."]},
//	 "traceId": "00-…-…-00"}
//
// The content type is `application/json; charset=utf-8` and **not**
// `application/problem+json`, which behaviours 1.11 records as the detail that
// costs an override rather than a default in every framework that has one.
//
// # The five member names are lower case, and that is not a slip
//
// behaviours 1.1's PascalCase rule is 1003 of the pinned document's 1026
// property names, and the five RFC 7807 members are among the twenty-three
// that are not. The casing sweep walks *registered response models*
// (architecture 4), and this type is deliberately not one: it is a refusal
// body, it is not the response of any operation, and registering it would make
// the sweep fail on names the reference really sends.
type problemDetails struct {
	// The five members in the order the reference sends them. Key order is
	// contract under L3 and a struct is the only thing in Go that fixes it.
	Type    string              `json:"type"`
	Title   string              `json:"title"`
	Status  int                 `json:"status"`
	Errors  map[string][]string `json:"errors"`
	TraceId string              `json:"traceId"`
}

// The two constants of the validation refusal, transcribed from the measured
// body rather than assembled from http.StatusText.
const (
	validationProblemType  = "https://tools.ietf.org/html/rfc9110#section-15.5.1"
	validationProblemTitle = "One or more validation errors occurred."
)

// requiredBodyMessage is what the reference says about an action parameter it
// could not bind, with the parameter's own name interpolated.
//
// The name is the *reference's* action parameter, never anything the client
// sent: three of 007's routes answer `playbackStartInfo`, `playbackProgressInfo`
// and `playbackStopInfo`, 009's rename answers `request`, and so does this
// feature's login route
// [source: Jellyfin.Api/Controllers/UserController.cs:211 @ v10.11.11].
func requiredBodyMessage(parameter string) string {
	return "The " + parameter + " field is required."
}

// WriteValidationProblem answers behaviours 1.11's validation `400`: RFC 9457
// problem details under `application/json; charset=utf-8`, with the errors map
// the caller built.
//
// # Why the caller builds the map
//
// The keys are the whole of what varies. A path or query parameter's refusal
// is keyed on the parameter's **declared** spelling and never the client's; a
// body's refusal is keyed on `"$"` when the text is not JSON, on `""` when it
// parses and a value inside it does not bind, and on the route's own action
// parameter name beside either. One writer with a map argument is the shape
// that lets each route name its own keys without a second spelling of the
// envelope.
//
// # The map's key order is the encoder's, and it agrees with both measured pairs
//
// encoding/json writes a map's keys in sorted order. That is not a decision
// this project took, and it matters because key order is contract under L3 —
// so it is recorded rather than assumed: the two measured pairs are
// `{"": …, "playbackProgressInfo": …}` and `{"$": …, "request": …}`, and in
// both the reference's order is the sorted one. A future refusal whose two keys
// sort the other way would need an ordered type here, not a comment.
//
// # The profile is pinned to plain
//
// For the reason WriteJSONMessage pins it: the three probes above measured
// `application/json; charset=utf-8`, this signature takes no request, and what
// the reference echoes for a refusal on a request that negotiated
// `profile="CamelCase"` is unmeasured. The five member names here are already
// lower case, so the camelCase policy would rename nothing anyway — but the
// content type would differ, and that is the byte a measurement would settle.
func WriteValidationProblem(w http.ResponseWriter, status int, errors map[string][]string) {
	header := w.Header()
	// wire.Write sets the content type; the challenge is deleted for the
	// reason refuse() deletes it, since an earlier stage may have set one.
	header.Del("WWW-Authenticate")

	body := problemDetails{
		Type:    validationProblemType,
		Title:   validationProblemTitle,
		Status:  status,
		Errors:  errors,
		TraceId: newTraceID(),
	}
	if err := wire.Write(w, status, body, wire.ProfilePlain); err != nil {
		// Unreachable — five members of three kinds, none with a custom
		// encoder — and handled for the reason WriteJSONMessage handles it:
		// wire.Write writes nothing to w unless the whole body was produced,
		// so there is still a refusal to send.
		WriteInternalServerError(w)
	}
}

// newTraceID is a W3C trace-context identifier in the shape the reference's
// traceId carries: version `00`, a 16-byte trace id, an 8-byte span id, and
// `00` flags, all lower-case hex.
//
// It is generated here rather than propagated from a `traceparent` header,
// because this server runs no tracer and joining a caller's trace would be a
// claim about a span nothing here ever ends. behaviours 1.11 records that the
// value is per request by definition and is therefore compared by **shape**
// and never by value, which is what makes generating one honest: the member has
// to be there and no value of it is more right than another.
//
// A failure to read the system's randomness is not survivable and there is no
// second answer to send, so it panics rather than answering a refusal with a
// constant identifier that would look like a real one. crypto/rand.Read cannot
// fail on any platform this project supports.
func newTraceID() string {
	var raw [24]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic("httpapi: the system's randomness is unreadable: " + err.Error())
	}
	return "00-" + hex.EncodeToString(raw[:16]) + "-" + hex.EncodeToString(raw[16:]) + "-00"
}
