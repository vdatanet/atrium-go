package httpapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// PathFolder rewrites a request path to the spelling the route table declares,
// so that the router only ever matches a canonical path.
//
// # What it is for
//
// The reference routes /system/info/public, /SYSTEM/INFO/PUBLIC and
// /System/info/Public to the handler of /System/Info/Public, and answers
// /System/Info/Public/ with the same body; /System/Info/Public// is a 404
// (behaviours 1.14, spec 3.6)
// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-26]. Clients rely
// on that without ever having had a reason to notice: a hand-written path
// literal typed in the wrong case, a URL normaliser, a proxy that
// canonicalises, an edited configuration file.
//
// No redirect is ever issued. behaviours 1.14 records that the framework
// default here is a third behaviour rather than a smaller divergence — a 307
// for the unmatched trailing slash is a round trip the reference does not
// make, and a 307 for the doubled slash points at a URL that works where the
// reference refuses. Two divergences, in opposite directions, is what
// accepting a default would have cost.
//
// # Literal segments fold; parameters do not
//
// Only the runs a route spells itself are matched without regard to case and
// rewritten to the route's own spelling. Whatever occupies a parameter is
// data, and reaches the handler exactly as the client wrote it — lowercasing
// an identifier is invisible until something case-sensitive reads one.
//
// The distinction is finer than one per segment, because a segment can be
// both: /Audio/{itemId}/stream.{container} spells stream. literally and takes
// the container from the client, so /audio/AbC/STREAM.MP4 canonicalises to
// /Audio/AbC/stream.MP4 — the literal respelled, both parameters untouched.
//
// # Which route wins
//
// A fully literal path is looked up before any parametrised one, so
// /Items/Filters is itself rather than /Items/{itemId} with an item called
// Filters. Among parametrised paths the table's own order decides, which is
// document order (Principle VII wants an order derived from a stable input,
// and the document is one). No two paths in the v1 table are ambiguous under
// that rule; the rule is written down so that a row added later is a decision
// somebody takes rather than one that falls out of a map iteration.
type PathFolder struct {
	// literal maps the folded spelling of a path with no parameters to its
	// canonical spelling. Most of the table is in here, and every route 001
	// serves is.
	literal map[string]string

	// patterns holds the paths with at least one parameter, in the order the
	// document first names them.
	patterns []pathPattern
}

// pathPattern is one canonical path, split into segments and each segment
// split into the runs that fold and the runs that pass through.
type pathPattern struct {
	// canonical is the spelling the document writes, kept whole so that a
	// path with no parameter at all can be returned without rebuilding it.
	canonical string

	// segments are the path's segments after the leading slash, in order.
	segments [][]pathPart
}

// pathPart is one run of a segment: literal text the route spells itself, or a
// parameter whose value belongs to the client.
type pathPart struct {
	// text is the canonical spelling of a literal run, and is empty for a
	// parameter.
	text string

	// parameter says which of the two this is, rather than leaving it to be
	// inferred from an empty text.
	parameter bool
}

// NewPathFolder builds the fold map from a route table.
//
// It reads the canonical spelling of every distinct path, which is what
// surface.Table.Paths answers, and nothing else: canonicalisation is a
// property of a path, not of a route, so a path served by two methods is one
// entry here.
func NewPathFolder(table *surface.Table) (*PathFolder, error) {
	return newPathFolder(table.Paths())
}

// newPathFolder is the builder a test can hand paths the loader would have
// refused, so that the refusals below are reachable rather than asserted.
func newPathFolder(paths []string) (*PathFolder, error) {
	folder := &PathFolder{literal: make(map[string]string, len(paths))}
	folded := make(map[string]string, len(paths))

	for _, path := range paths {
		if !strings.HasPrefix(path, "/") {
			return nil, fmt.Errorf("httpapi: path %q does not begin with a slash", path)
		}
		pattern, parametrised, err := compilePath(path)
		if err != nil {
			return nil, err
		}

		key := pattern.key()
		if first, clash := folded[key]; clash {
			return nil, fmt.Errorf("httpapi: paths %q and %q fold together, so there is no rule for choosing between them", first, path)
		}
		folded[key] = path

		if parametrised {
			folder.patterns = append(folder.patterns, pattern)
			continue
		}
		folder.literal[key] = path
	}
	return folder, nil
}

// parameterMark stands for a parameter in a pattern's key. It is a byte no
// path can carry, so a key is unambiguous.
const parameterMark = "\x00"

// key is the pattern reduced to what it actually matches: literal runs folded,
// every parameter the same mark whatever it is called.
//
// Two paths with the same key accept exactly the same requests, which makes
// choosing between them a coin toss rather than a rule — so newPathFolder
// refuses the pair. surface.Load already refuses two paths differing only in
// casing (plan 3, at T8), and this catches the case it cannot see: two paths
// that differ only in what they call a parameter.
func (p pathPattern) key() string {
	var out strings.Builder
	out.Grow(len(p.canonical))
	for _, parts := range p.segments {
		out.WriteByte('/')
		for _, part := range parts {
			if part.parameter {
				out.WriteString(parameterMark)
				continue
			}
			out.WriteString(foldASCII(part.text))
		}
	}
	return out.String()
}

// compilePath splits a canonical path into its segments and each segment into
// its runs, and reports whether it carries a parameter at all.
func compilePath(path string) (pathPattern, bool, error) {
	pattern := pathPattern{canonical: path}
	parametrised := false

	for i, segment := range splitSegments(path) {
		if segment == "" {
			return pathPattern{}, false, fmt.Errorf("httpapi: path %q has an empty segment at position %d", path, i+1)
		}
		parts, err := compileSegment(segment, path)
		if err != nil {
			return pathPattern{}, false, err
		}
		for _, part := range parts {
			if part.parameter {
				parametrised = true
			}
		}
		pattern.segments = append(pattern.segments, parts)
	}
	return pattern, parametrised, nil
}

// compileSegment splits one segment into alternating literal and parameter
// runs.
//
// Two shapes are refused rather than guessed at. An unclosed or empty brace is
// not a pattern anybody meant to write. Two adjacent parameters — {a}{b} — have
// no literal between them to anchor the first one's end, so where one stops is
// a choice this function would be making on the route author's behalf; the
// router does not support it either.
func compileSegment(segment, path string) ([]pathPart, error) {
	var parts []pathPart
	rest := segment

	for rest != "" {
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			if strings.IndexByte(rest, '}') >= 0 {
				return nil, fmt.Errorf("httpapi: segment %q of path %q closes a parameter it never opened", segment, path)
			}
			parts = append(parts, pathPart{text: rest})
			break
		}
		if open > 0 {
			literal := rest[:open]
			if strings.IndexByte(literal, '}') >= 0 {
				return nil, fmt.Errorf("httpapi: segment %q of path %q closes a parameter it never opened", segment, path)
			}
			parts = append(parts, pathPart{text: literal})
		}

		closed := strings.IndexByte(rest[open:], '}')
		if closed < 0 {
			return nil, fmt.Errorf("httpapi: segment %q of path %q opens a parameter it never closes", segment, path)
		}
		name := rest[open+1 : open+closed]
		if name == "" || strings.IndexByte(name, '{') >= 0 {
			return nil, fmt.Errorf("httpapi: segment %q of path %q does not name its parameter", segment, path)
		}
		if len(parts) > 0 && parts[len(parts)-1].parameter {
			return nil, fmt.Errorf("httpapi: segment %q of path %q puts two parameters side by side, so where the first one ends is undecidable", segment, path)
		}
		parts = append(parts, pathPart{parameter: true})
		rest = rest[open+closed+1:]
	}
	return parts, nil
}

// Canonicalise answers the spelling to route, given a request's escaped path.
//
// The three steps are plan 6.1's:
//
//  1. Two or more trailing slashes are a refusal — false is returned, and the
//     caller answers the empty 404 of spec 3.6. One trailing slash is trimmed.
//  2. The remaining path is folded and looked up. A literal segment matches
//     without regard to case; a parameter takes whatever is there, byte for
//     byte.
//  3. The canonical spelling is returned, with each parameter's bytes in place.
//
// A path that matches nothing in the table is returned as it arrived, minus
// the trailing slash. That is deliberately not a refusal here: the 404 for a
// path matching no route is the router's, computed from the same table, and
// making it two refusals in two places would mean two shapes to keep the same.
func (f *PathFolder) Canonicalise(path string) (string, bool) {
	if strings.HasSuffix(path, "//") {
		return "", false
	}
	if !strings.HasPrefix(path, "/") {
		// Not an origin-form path — OPTIONS * is the one that reaches a
		// server. There is nothing to fold and nothing to trim.
		return path, true
	}
	if len(path) > 1 && strings.HasSuffix(path, "/") {
		path = path[:len(path)-1]
	}

	if canonical, ok := f.literal[foldASCII(path)]; ok {
		return canonical, true
	}
	for _, pattern := range f.patterns {
		if canonical, ok := pattern.match(path); ok {
			return canonical, true
		}
	}
	return path, true
}

// match rebuilds path in the pattern's own spelling, or reports that the
// pattern does not describe it.
func (p pathPattern) match(path string) (string, bool) {
	segments := splitSegments(path)
	if len(segments) != len(p.segments) {
		return "", false
	}

	var out strings.Builder
	out.Grow(len(p.canonical) + len(path))
	for i, parts := range p.segments {
		out.WriteByte('/')
		if !matchSegment(parts, segments[i], &out) {
			return "", false
		}
	}
	return out.String(), true
}

// matchSegment matches one segment against one segment's runs, writing the
// canonical spelling of each literal run and the client's own bytes for each
// parameter.
//
// A parameter ends where the next literal run begins, at that run's leftmost
// occurrence from one byte in — one byte, because a parameter that matched
// nothing would let /Audio/x/.mp4 stand in for a stream with no name.
func matchSegment(parts []pathPart, segment string, out *strings.Builder) bool {
	at := 0
	for i, part := range parts {
		if !part.parameter {
			if len(segment)-at < len(part.text) || !equalFoldASCII(segment[at:at+len(part.text)], part.text) {
				return false
			}
			out.WriteString(part.text)
			at += len(part.text)
			continue
		}

		if i == len(parts)-1 {
			if at == len(segment) {
				return false
			}
			out.WriteString(segment[at:])
			at = len(segment)
			continue
		}

		// Parts alternate, so the next one is a literal.
		offset := indexFoldASCII(segment[at+1:], parts[i+1].text)
		if offset < 0 {
			return false
		}
		out.WriteString(segment[at : at+1+offset])
		at += 1 + offset
	}
	return at == len(segment)
}

// splitSegments splits a path on its slashes, dropping the leading empty
// segment the leading slash produces.
func splitSegments(path string) []string {
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

// Wrap is the middleware. It rewrites the request's path to the canonical
// spelling before the next handler — which is the router — sees it, and
// answers the empty 404 of spec 3.6 for a path carrying two or more trailing
// slashes.
//
// The incoming request is never mutated: a rewrite copies the request and its
// URL, the way http.StripPrefix does, because the caller's *http.Request may
// be read by something that outlives this frame.
func (f *PathFolder) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		escaped := r.URL.EscapedPath()
		canonical, ok := f.Canonicalise(escaped)
		if !ok {
			// 404, empty body, no Content-Type (behaviours 1.11). Writing
			// nothing is what leaves the content type off: the standard
			// library sniffs one only for a body, and there is none.
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if canonical == escaped {
			next.ServeHTTP(w, r)
			return
		}

		unescaped, err := url.PathUnescape(canonical)
		if err != nil {
			// Unreachable: EscapedPath returns a valid encoding, and the
			// rewrite only ever substitutes unreserved ASCII from the table
			// for bytes that folded to it. Route what arrived rather than
			// invent a refusal the reference does not send.
			next.ServeHTTP(w, r)
			return
		}

		rewritten := *r.URL
		rewritten.Path = unescaped
		// net/http keeps RawPath only where it differs from the default
		// encoding of Path, and so does this.
		rewritten.RawPath = ""
		if rewritten.EscapedPath() != canonical {
			rewritten.RawPath = canonical
		}

		routed := *r
		routed.URL = &rewritten
		next.ServeHTTP(w, &routed)
	})
}

// foldASCII lower-cases the ASCII letters of s and leaves every other byte
// alone.
//
// The fold is ASCII rather than Unicode on purpose. Every literal run in the
// route table is ASCII, so a Unicode fold would decide nothing this table asks
// about — and it would decide it differently in different places, because
// simple folding equates strings of different byte lengths and a fixed-width
// window silently would not.
func foldASCII(s string) string {
	if !strings.ContainsFunc(s, isUpperASCII) {
		return s
	}
	folded := []byte(s)
	for i, b := range folded {
		if b >= 'A' && b <= 'Z' {
			folded[i] = b + ('a' - 'A')
		}
	}
	return string(folded)
}

func isUpperASCII(r rune) bool { return r >= 'A' && r <= 'Z' }

// equalFoldASCII reports whether a and b are the same string once ASCII
// letters are folded.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if x >= 'A' && x <= 'Z' {
			x += 'a' - 'A'
		}
		if y >= 'A' && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// indexFoldASCII returns the leftmost index at which s carries sub, folding
// ASCII letters, or -1.
func indexFoldASCII(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if equalFoldASCII(s[i:i+len(sub)], sub) {
			return i
		}
	}
	return -1
}
