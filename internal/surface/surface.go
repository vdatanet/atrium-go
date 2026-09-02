// Package surface holds the v1 route table: the 59 rows of
// docs/compatibility/surface.yaml, each a method, a path, an operation, the
// clients observed calling it, the feature that owns it and the conformance
// level it must reach.
//
// # Why this is a package and not a slice in the router
//
// Three unrelated things read the same table (plan 3): the router, which
// registers the rows of implemented features; the `Allow` computation, which
// needs every method a *path* has rather than every method the matched *route*
// has (spec 3.6); and the L0 check, which asserts the server exposes exactly
// this file and nothing else (Principle VI, plan 8.5). A table that lived
// inside the router could not be used to check the router — the check would
// compare the router against itself and pass by construction.
//
// So this package knows nothing about HTTP. It parses a document and answers
// questions about it. Canonicalisation (T9, T10), routing (T11) and `Allow`
// itself (T11) are elsewhere and read from here.
//
// # How the file reaches the binary
//
// The document lives under docs/, which is where the paired artefacts live
// (docs/README.md) and where it must stay: a machine-readable half whose prose
// twin is api-surface-v1.md is not this package's to move. But go:embed cannot
// reach outside its own package directory, and ADR-0002 wants one static
// binary — a server started from any directory has to find its own route
// table, so reading docs/compatibility/surface.yaml from disk at run time is
// not available: it would make the working directory part of the deployment.
//
// The copy beside this file is therefore embedded, and
// TestTheEmbeddedCopyIsTheDocument asserts it is byte-identical to the
// document. The copy is derived, not a second opinion: docs/README.md records
// it beside the pairs, and the test's failure message carries the one command
// that fixes it.
package surface

import (
	_ "embed"
	"fmt"
	"sync"
)

// EmbeddedFile is the path, relative to the repository root, of the document
// this package embeds. It is here so that a test or a tool can name the
// canonical half rather than spelling the path again.
const EmbeddedFile = "docs/compatibility/surface.yaml"

// document is the byte-identical copy of EmbeddedFile that ships in the
// binary. Everything above about go:embed is why it exists.
//
//go:embed surface.yaml
var document []byte

// Level is the conformance level a row is required to reach, defined in
// docs/compatibility/conformance.md.
//
// It is an integer rather than the string it is written as, so that the
// loader is the only thing that can produce one: a level a caller could
// construct out of an arbitrary string would make the "unknown level refuses
// to load" rule a suggestion.
type Level int

const (
	// L0 — routed: the path exists and answers a sane status.
	L0 Level = iota
	// L1 — shape: fields, casing, types and units.
	L1
	// L2 — semantic: the values are right for a known library.
	L2
	// L3 — differential: byte-comparable to what the reference sends.
	L3
)

// levelNames is the spelling of each level in the document, indexed by the
// level. It is both halves of the mapping: String reads it and parseLevel
// searches it, so a level added to one is added to the other.
var levelNames = [...]string{L0: "L0", L1: "L1", L2: "L2", L3: "L3"}

// String returns the level as the document spells it.
func (l Level) String() string {
	if l < 0 || int(l) >= len(levelNames) {
		return fmt.Sprintf("Level(%d)", int(l))
	}
	return levelNames[l]
}

// parseLevel reads a level as the document spells it. The second result is
// false for anything else, which is what refuses a row.
func parseLevel(s string) (Level, bool) {
	for i, name := range levelNames {
		if name == s {
			return Level(i), true
		}
	}
	return 0, false
}

// Endpoint is one row of the table.
//
// Path carries the canonical spelling — the one this repository writes
// everywhere and the one a request is folded to before routing (spec 3.6).
// Method carries it in upper case, which is what `Allow` sends.
type Endpoint struct {
	// Path is the canonical spelling, chi's pattern syntax included: a
	// parameter is written `{name}` and may share a segment with literal
	// text, as in `/Audio/{itemId}/stream.{container}`.
	Path string

	// Method is the HTTP method, upper case.
	Method string

	// Operation is the pinned OpenAPI document's operationId. It is what a
	// probe, an allowlist entry and a request case name a route by.
	Operation string

	// Consumers are the real clients observed calling this endpoint. An empty
	// list means the endpoint is included by design — the prose twin's Notes
	// column says why (Principle VI).
	Consumers []string

	// Feature is the specs/ directory that owns the row, as three digits.
	Feature string

	// Level is the conformance level the row is required to reach.
	Level Level
}

// clone returns a copy that shares nothing with the receiver, so that a caller
// holding one row cannot reach into the table through its consumers slice.
func (e Endpoint) clone() Endpoint {
	e.Consumers = append([]string(nil), e.Consumers...)
	return e
}

// Reference is the pin the document was generated against. It is not the
// project's pin — docs/compatibility/reference-target.md is — but a table
// generated against a different reference than the one this server targets is
// worth being able to notice.
type Reference struct {
	// OpenAPIVersion is the version of the pinned OpenAPI document.
	OpenAPIVersion string

	// SourceTag is the Jellyfin source tag the rows were validated against.
	SourceTag string
}

// Table is a loaded route table. It is read-only once loaded, and every
// accessor that hands out a slice hands out a copy.
type Table struct {
	reference Reference

	// endpoints is in document order, which groups the rows by owning
	// feature. Nothing here sorts them: Principle VII wants an order that
	// derives from a stable input, and the document is one.
	endpoints []Endpoint

	// paths is the canonical spelling of each distinct path, in the order it
	// first appears. It is what canonicalisation builds its fold map from.
	paths []string

	// methodsByPath answers `Allow`: every method a path has, sorted
	// alphabetically (spec 3.6 wants every method the *path* has, which for
	// /System/Ping is two rows).
	methodsByPath map[string][]string

	// rowByRoute finds one row by its method and path.
	rowByRoute map[route]int
}

// route is a method-and-path pair: the key the document may not repeat.
type route struct {
	method string
	path   string
}

// Len returns the number of rows.
func (t *Table) Len() int { return len(t.endpoints) }

// Reference returns the pin the document was generated against.
func (t *Table) Reference() Reference { return t.reference }

// Endpoints returns every row, in document order.
func (t *Table) Endpoints() []Endpoint {
	out := make([]Endpoint, 0, len(t.endpoints))
	for _, e := range t.endpoints {
		out = append(out, e.clone())
	}
	return out
}

// ForFeature returns the rows one feature owns, in document order. It is how
// the router registers the features that are implemented and no others: a row
// belonging to a feature whose spec is still draft is listed in the document
// and not yet served (surface.yaml's own header).
func (t *Table) ForFeature(feature string) []Endpoint {
	var out []Endpoint
	for _, e := range t.endpoints {
		if e.Feature == feature {
			out = append(out, e.clone())
		}
	}
	return out
}

// Paths returns the canonical spelling of every distinct path, in the order it
// first appears in the document. 59 rows share 51 paths, because a path served
// by two methods is two rows and one path.
func (t *Table) Paths() []string {
	return append([]string(nil), t.paths...)
}

// Methods returns every method registered on a path, sorted alphabetically —
// which is the `Allow` header spec 3.6 asks for, already in its order. It
// returns nil for a path the table does not have.
//
// The lookup is exact. Folding a client's spelling to the canonical one is
// canonicalisation's job (spec 3.6), and doing it here would hide from that
// middleware the fact that it had not run.
func (t *Table) Methods(path string) []string {
	methods, ok := t.methodsByPath[path]
	if !ok {
		return nil
	}
	return append([]string(nil), methods...)
}

// Lookup returns the row for one method and path.
func (t *Table) Lookup(method, path string) (Endpoint, bool) {
	i, ok := t.rowByRoute[route{method: method, path: path}]
	if !ok {
		return Endpoint{}, false
	}
	return t.endpoints[i].clone(), true
}

// v1 parses the embedded document once, the first time it is asked for.
//
// It panics on a parse failure because there is no useful thing a caller could
// do about a malformed table compiled into the binary, and because
// TestTheEmbeddedDocumentLoads makes it a failure of the build rather than of
// a run.
var v1 = sync.OnceValue(func() *Table {
	table, err := Load(document)
	if err != nil {
		panic("surface: the embedded " + EmbeddedFile + " does not load: " + err.Error())
	}
	return table
})

// V1 returns the v1 surface table.
func V1() *Table { return v1() }
