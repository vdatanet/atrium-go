// Package wire writes every response body this server sends.
//
// It exists so that three wire facts are decided in one place instead of at
// fifty-nine handlers, and all three are here now:
//
//   - Property names are PascalCase (spec 3.0.1), and a second, camelCase
//     policy is chosen per request (spec 3.0.2). The policy is a parameter of
//     the write rather than a property of the model, because it is negotiated.
//   - Every non-ASCII character and seven ASCII ones leave as an upper-case
//     \uXXXX escape (behaviours 1.16), which no JSON encoder in the standard
//     library does.
//   - The response's content type names the profile that was matched, before
//     the charset (behaviours 1.13).
//
// # Why the status, the value and the content type go out together
//
// behaviours 1.10 puts `charset=utf-8` on every JSON response, and the source
// project records that it did this "through a response class rather than a
// middleware, so the content type belongs to the thing that produced the body".
// A middleware bolted on after cannot make that claim: it stamps a content type
// on a body it never saw, and the first handler to send something that is not
// JSON — an image, a media segment, an empty refusal (behaviours 1.11) — gets
// the header anyway.
//
// So Write takes the status and the value in one call. A caller that has not
// gone through this package has not set a content type either, which is the
// property the pipeline wants: the two are set by the same act or by neither.
//
// # Why the last argument is a Profile and not a Naming
//
// Three declared content types, two naming policies. A body written under
// NamingPascal cannot say which of the two PascalCase content types asked for
// it, so Naming alone could not carry the echo and the negotiation's single
// winner had to arrive whole. See profile.go.
package wire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
)

// Naming is the property-naming policy a body is written under.
//
// It is negotiated per request from Accept (spec 3.0.2), so it travels as an
// argument to the write and never as a property of a model: the same struct is
// written under both policies, to two different clients, in the same second.
type Naming int

// NamingPascal writes property names exactly as the models declare them, which
// is PascalCase project-wide (spec 3.0.1). It is the policy the plain content
// type and the `PascalCase` profile both answer under — three names for two
// behaviours, and this is the one two of them share.
const NamingPascal Naming = 0

// Write serialises v as this server's JSON, sets the content type the profile
// echoes and the status, and sends the body.
//
// Nothing is written to w unless the whole body was produced. A model can fail
// to serialise — units.Time refuses a year outside 1 to 9999, because the
// layout behaviours 1.2 measured has room for four year digits — and a writer
// that had already sent a 200 and half a body could only follow it with more
// body. The caller gets the error while it can still send a refusal instead.
//
// A profile this package does not know is such an error, and for the reason
// marshal refuses an unknown naming policy: a fall-through to the plain type
// would answer a client that asked for camelCase with PascalCase, which is an
// empty object out of its decoder rather than a visible failure
// (behaviours 1.13).
func Write(w http.ResponseWriter, status int, v any, profile Profile) error {
	naming, known := profile.Naming()
	if !known {
		return fmt.Errorf("wire: no content profile numbered %d", int(profile))
	}
	contentType, _ := profile.ContentType()

	body, err := marshal(v, naming)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_, err = w.Write(body)
	return err
}

// marshal produces the bytes Write sends: the encoder's output, renamed under
// the requested policy, with the escape pass applied.
//
// The order is the one thing about this function that is not free. The rename
// walks the document beside the value it came from, because that is the only
// place a property name can be told from a dictionary key (plan 10); the escape
// pass works on the finished document, because behaviours 1.16 applies to every
// string in it, keys included. So the rename runs first, on bytes the encoder
// wrote, and the escape pass last, over both halves of every member.
//
// An unrecognised policy is an error. A Naming this package does not know would
// otherwise fall through to PascalCase, which is the one failure the argument
// exists to prevent: a client that asked for camelCase and was answered in
// PascalCase gets an empty object out of its decoder (behaviours 1.13).
func marshal(v any, naming Naming) ([]byte, error) {
	body, err := encodeCompact(v)
	if err != nil {
		return nil, err
	}

	switch naming {
	case NamingPascal:
		// The models already declare PascalCase names (spec 3.0.1), so the
		// policy has nothing to do and the document is the encoder's own.
	case NamingCamel:
		body, err = renameProperties(body, reflect.ValueOf(v))
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("wire: no naming policy numbered %d", int(naming))
	}

	return escape(body), nil
}

// encodeCompact encodes one value with this package's encoder settings, and
// without the newline Encode appends.
func encodeCompact(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)

	// The encoder's own HTML escaping is switched off (plan 6.4). Left on it
	// writes three of behaviours 1.16's seven characters — < > & — as
	// lower-case escapes and none of the other four, so the escape pass would
	// have to undo its casing as well as do its own job. Off, the pass sees
	// them raw and treats all seven alike.
	encoder.SetEscapeHTML(false)

	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// Encode appends a newline; Marshal is the same encoder without one, and
	// has no way to switch the HTML escaping off. The reference sends no
	// trailing newline, so it is dropped here rather than tolerated by every
	// golden.
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
