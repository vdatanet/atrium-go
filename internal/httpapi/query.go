package httpapi

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// Route names one row of the route table: an HTTP method and the canonical
// spelling of a path, both written the way docs/compatibility/surface.yaml
// writes them.
type Route struct {
	// Method is the HTTP method, upper case.
	Method string

	// Path is the canonical spelling, chi's pattern syntax included, so a
	// parametrised route is named `/Items/{itemId}` rather than by any one
	// request that matches it.
	Path string
}

// String renders the route the way a log line or an error names one.
func (r Route) String() string { return r.Method + " " + r.Path }

// QuerySpellings declares, per route, the query parameter names that route's
// handler binds. A name is written exactly as the handler reads it, and every
// spelling that folds to it is rewritten to it before the handler runs.
//
// # Where these declarations live, and why not in surface.yaml
//
// plan 6.2 says "each route declares its parameter spellings" and does not say
// where. Nowhere does: surface.yaml carries a path, a method, an operation,
// its consumers, the owning feature and the required level, and no parameters
// at all — and 001's own four routes take no query parameter, so this feature
// cannot supply an example either. Three ways were open, and this is the
// decision rather than an omission.
//
// Extending surface.yaml was the expensive one, and it buys nothing yet. It is
// a paired artefact (docs/README.md): its prose twin api-surface-v1.md would
// have to gain the same column in the same commit, the derived copy beside
// internal/surface would have to be recopied, and the strict loader would gain
// a key — all to write an empty list on 59 rows. The document's own header
// says it is generated and validated against the pinned OpenAPI document by a
// tool that stayed in the source repository, so a column added here by hand is
// a column the next generation would not know to keep.
//
// Declaring in code, beside the routes that have parameters, is what happens
// here, and the reason is not only cost. **The declared spelling is a property
// of this server's handler, not a fact about the reference's surface.** The
// pinned document spells every parameter camelCase (`startIndex`), the
// reference's own clients send PascalCase (`StartIndex`), and both work
// (behaviours 1.15) — so there is no single spelling the surface file could
// state. What this stage needs is the one spelling *our* handler binds, which
// is the handler's to declare.
//
// The mechanism ships now with an empty declaration set, and the first feature
// with a query parameter supplies the first rows. If that list ever grows
// unwieldy — 005's item query alone has dozens — moving it into surface.yaml
// remains available, and moving a list that exists is a smaller decision than
// inventing a column for an empty one.
type QuerySpellings map[Route][]string

// V1QuerySpellings is the declaration set this server runs on: every query
// parameter name a route it serves binds.
//
// ~~**It is empty, and that is the whole of 001's answer.**~~ **002 T16 filled
// it, with the three names GET /Sessions binds.** 001's four routes —
// /System/Info/Public, /System/Info and the two on /System/Ping — still take no
// query parameter at all, and 002's other six routes take none either: the
// login route reads its client identification from a header, /Users/Public
// reads nothing, and the two reads and the two writes each ignore the query
// parameter the reference declares (U-14, and sessions.go's `id`). So one route
// declares three names and the other ten declare none.
//
// The stage shipped ahead of them because the order of the pipeline is contract
// (001 plan 6.7) and a stage inserted later is a change to that contract rather
// than a row in a map. Until this change, no request this server could answer
// exercised it — 001's own mutation testing recorded that — so the assertions
// that it folds on a real route are 002 T16's and are in query_test.go and
// sessions_test.go.
func V1QuerySpellings() QuerySpellings {
	return QuerySpellings{
		// spec 3.8's three, spelled as the reference declares them
		// [source: Jellyfin.Api/Controllers/SessionController.cs:52-59 @ v10.11.11].
		// A name is written here exactly as sessions.go reads it, which is what
		// makes `?DEVICEID=x` and `?deviceid=x` the same request as `?deviceId=x`
		// (behaviours 1.15).
		{Method: http.MethodGet, Path: "/Sessions"}: {
			controllableByUserIDParameter,
			deviceIDParameter,
			activeWithinSecondsParameter,
		},
	}
}

// QueryFolder rewrites a request's query parameter *names* to the spellings
// the matched route declares, so that a handler reads one spelling.
//
// # What it is for
//
// The reference treats Limit=1, limit=1 and LIMIT=1 as the same parameter, and
// a lowercased sortby=PremiereDate&sortorder=Descending reorders /Items
// exactly as the PascalCase spelling does (behaviours 1.15)
// [probe: tools/probe_query_envelope.py, Jellyfin 10.11.11, 2026-08-28]. Every
// client depends on at least one half of that, and which half is a per-client
// accident: the pinned document spells every parameter camelCase and the
// reference's own clients send PascalCase.
//
// The framework default is a silent third behaviour rather than a smaller
// divergence. An unrecognised spelling is not rejected but *ignored*, which
// for StartIndex against a camelCase route means every page is page one —
// a wrong answer with a 200 on it.
//
// # Values are never touched
//
// Only names fold. A value is data, and a value that differs from its own
// name only in case is still a value: ?Limit=LIMIT canonicalises to
// ?limit=LIMIT. This is the query counterpart of the path rewrite's rule that
// a path parameter reaches the handler byte for byte (architecture 4), and it
// is why the rewrite is a splice into the raw query rather than a parse and
// re-encode — re-encoding would rewrite every percent-escape a client chose.
//
// # An unrecognised name is left in place, not dropped
//
// behaviours 1.12: an unrecognised query parameter is ignored rather than
// rejected, and 010 3.6 counts what was ignored. A stage that dropped an
// unknown name would leave that tally nothing to count, so a fragment this
// folder does not recognise — including one that is not a name=value pair at
// all — survives in its own position, byte for byte.
//
// # Which route's declarations apply
//
// The declarations are keyed by route, so the folder has to know which row of
// the table a request belongs to before the router does. It does the same fold
// the path stage does, and reads the row's *pattern*: a request for
// /Items/AbC is /Items/{itemId}'s. Keying by route rather than by path is
// plan 6.2's own word, and it matters — the reference binds parameters per
// action, and a path served by two methods can bind different ones on each, so
// a path-keyed declaration would rewrite a name on a method that never
// declared it. That name is unrecognised there, and behaviours 1.12 says an
// unrecognised name is left alone.
//
// architecture 4 puts case-insensitive query names in "the same middleware" as
// case-insensitive paths. The point of that row is where the behaviour may not
// be — "a handler reading two spellings" — and plan 6.7 fixes the pipeline
// with the two as adjacent stages. They are two stages here, sharing one fold.
type QueryFolder struct {
	// paths is the fold this stage borrows to name the route a request
	// belongs to. It is built from the same table, so a path the router will
	// serve is a path this stage recognises.
	paths *PathFolder

	// byRoute maps a route to its declared names, each keyed by its folded
	// spelling. An absent route declares nothing and folds nothing.
	byRoute map[Route]map[string]string
}

// NewQueryFolder builds the stage from the route table and the declarations.
//
// It refuses a declaration the table does not have a row for. A name declared
// against a route that was renamed, or misspelled in the declaration itself,
// would otherwise fold nothing and say nothing — and the failure it hides is
// the one behaviours 1.15 is about, where a client's spelling is ignored and
// the answer is wrong with a 200 on it.
func NewQueryFolder(table *surface.Table, spellings QuerySpellings) (*QueryFolder, error) {
	paths, err := NewPathFolder(table)
	if err != nil {
		return nil, err
	}

	folder := &QueryFolder{paths: paths, byRoute: make(map[Route]map[string]string, len(spellings))}
	for _, route := range sortedRoutes(spellings) {
		if _, ok := table.Lookup(route.Method, route.Path); !ok {
			return nil, fmt.Errorf("httpapi: query parameters are declared for %s, which the route table does not have", route)
		}

		declared := make(map[string]string, len(spellings[route]))
		for _, name := range spellings[route] {
			if err := checkQueryName(route, name); err != nil {
				return nil, err
			}
			folded := foldASCII(name)
			if first, clash := declared[folded]; clash {
				return nil, fmt.Errorf("httpapi: %s declares both %q and %q, which fold together, so there is no rule for choosing between them", route, first, name)
			}
			declared[folded] = name
		}
		folder.byRoute[route] = declared
	}
	return folder, nil
}

// sortedRoutes orders the declared routes, so that a document with two faults
// in it is refused for the same one every time. Ranging over the map directly
// would make the error message depend on Go's map iteration order, which is
// the kind of non-determinism Principle VII exists to keep out.
func sortedRoutes(spellings QuerySpellings) []Route {
	routes := make([]Route, 0, len(spellings))
	for route := range spellings {
		routes = append(routes, route)
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes
}

// unmatchableInAQueryName are the bytes a declared name may not carry.
//
// The comparison below is on the query string's **own bytes**, undecoded — the
// same choice the path fold made for the same reason (plan 6.1): decoding
// first would fold on a separator a client percent-encoded precisely so that
// it would not be one. So `&` and `=` are what a fragment is split on and
// could never appear inside a name; `%` and `+` say the raw form is an encoded
// form, which is not what is being compared; and `#` starts a fragment, so it
// never reaches a query string at all.
const unmatchableInAQueryName = "&=+%#"

// checkQueryName refuses a declared name that no request could ever match.
//
// A declaration that cannot match is worse than a missing one: the route looks
// as though it folds a parameter, and the client that sends the other spelling
// is ignored silently.
func checkQueryName(route Route, name string) error {
	if name == "" {
		return fmt.Errorf("httpapi: %s declares an empty query parameter name", route)
	}
	if i := strings.IndexAny(name, unmatchableInAQueryName); i >= 0 {
		return fmt.Errorf("httpapi: %s declares the query parameter name %q, which carries %q — no raw query string can spell that, so the declaration could never match", route, name, name[i:i+1])
	}
	for i := 0; i < len(name); i++ {
		if name[i] <= ' ' || name[i] >= 0x7f {
			return fmt.Errorf("httpapi: %s declares the query parameter name %q, which carries a byte outside printable ASCII at position %d — such a byte reaches the server percent-encoded, so the declaration could never match", route, name, i)
		}
	}
	return nil
}

// Canonicalise answers the query string to bind, given a request's method, its
// escaped path and its raw query.
//
// The path is expected to be the canonical one the path stage produced, but
// nothing here depends on that: the lookup folds, so an out-of-order pipeline
// gets the right route rather than a quietly different answer.
//
// A route that declares nothing, and a path the table does not describe, each
// return the query unchanged. Neither is a refusal — the 404 for a path
// matching no route is the router's, computed from the same table.
func (f *QueryFolder) Canonicalise(method, escapedPath, rawQuery string) string {
	if rawQuery == "" || len(f.byRoute) == 0 {
		return rawQuery
	}
	pattern, ok := f.paths.pattern(escapedPath)
	if !ok {
		return rawQuery
	}
	declared, ok := f.byRoute[Route{Method: method, Path: pattern}]
	if !ok || len(declared) == 0 {
		return rawQuery
	}
	return foldQueryNames(declared, rawQuery)
}

// foldQueryNames rewrites the names of a raw query string and nothing else.
//
// The split is on `&` alone. A fragment with no `=` is a name with no value
// and folds like any other; a fragment that is empty, or that carries only a
// value, is not a name this stage recognises and survives untouched. The
// bytes after the name — the `=`, the value, its escapes — are spliced back
// unread.
func foldQueryNames(declared map[string]string, rawQuery string) string {
	fragments := strings.Split(rawQuery, "&")
	changed := false

	for i, fragment := range fragments {
		name := fragment
		if at := strings.IndexByte(fragment, '='); at >= 0 {
			name = fragment[:at]
		}
		canonical, ok := declared[foldASCII(name)]
		if !ok || canonical == name {
			continue
		}
		fragments[i] = canonical + fragment[len(name):]
		changed = true
	}

	if !changed {
		return rawQuery
	}
	return strings.Join(fragments, "&")
}

// Wrap is the middleware. It rewrites the request's query string to the
// declared spellings before the next handler — the router, and then the
// handler — reads it.
//
// The incoming request is never mutated, for the reason the path stage gives:
// the caller's *http.Request may be read by something that outlives this
// frame.
func (f *QueryFolder) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		canonical := f.Canonicalise(r.Method, r.URL.EscapedPath(), r.URL.RawQuery)
		if canonical == r.URL.RawQuery {
			next.ServeHTTP(w, r)
			return
		}

		rewritten := *r.URL
		rewritten.RawQuery = canonical

		routed := *r
		routed.URL = &rewritten
		next.ServeHTTP(w, &routed)
	})
}
