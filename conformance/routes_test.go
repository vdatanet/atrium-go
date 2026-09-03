package conformance_test

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// This file is the **reachability** half of the L0 check of plan 8.5: a real
// request is issued to every row of surface.yaml, and what comes back says
// whether the server serves that row (spec 3.6, AC-11, Principle VI).
//
// Its twin is the **registration** half in internal/httpapi/registration_test.go,
// which walks the built router with chi.Walk. plan 8.5 keeps both, and calls
// this one the stronger of the two: a router walked agrees with the table it
// was built from almost by construction, where a request that has been through
// the whole pipeline agrees with nothing unless the pipeline works. It is the
// only one of the two that catches a route registered correctly and made
// unreachable by something above it — a canonicalisation bug, a middleware that
// swallows it, a gate that never opens.
//
// Neither half covers a route that is registered, reachable and answers the
// wrong thing. That is what the goldens and the per-field tests are for.
//
// # Where the row list comes from, and why it is read here rather than imported
//
// This package may not import internal/surface (architecture 3, enforced by
// tools/check_conformance_imports), so the rows are read from the document
// itself by the small reader below. The alternative — a list generated into
// testdata — was rejected because a copy is a thing that goes out of date
// quietly, and the failure it produces is a check that agrees with what the
// document said last time somebody remembered to regenerate it.
//
// Two readers of one document is the same trade the wire sweep already makes
// for the PascalCase rule and the date layout, and it buys the same thing: the
// two halves of this check read the surface through different code, so a change
// to the loader cannot make both agree by construction. It also reads the
// **document** — docs/compatibility/surface.yaml, the file a reviewer edits —
// rather than the derived copy internal/surface embeds, so the two halves would
// disagree if the copy ever drifted from it.
//
// The reader is deliberately strict: it refuses any line it does not
// understand, rather than skipping it. A lenient reader would answer this check
// with a subset of the surface and report that everything in it was fine.

// surfaceDocument is the machine-readable v1 surface, relative to this
// package's directory — which is where `go test` runs a test binary.
//
// A relative path into the repository's layout is what harness_test.go
// deliberately avoids for the *binary*, by naming it with an import path. A
// document has no import path, and naming this one is the point of the check
// rather than an incidental dependency: this file is about that file.
var surfaceDocument = filepath.Join("..", "docs", "compatibility", "surface.yaml")

// surfaceRow is one endpoint of the document — the three columns this check
// needs, and the operation, which is what a failure message should name.
type surfaceRow struct {
	path      string
	method    string
	operation string
	feature   string
}

func (r surfaceRow) String() string {
	return fmt.Sprintf("%s %s (%s, feature %s)", r.method, r.path, r.operation, r.feature)
}

// TestTheServerIsReachableOnExactlyTheImplementedRowsOfTheSurfaceDocument is
// the check.
//
// # What "implemented" means here
//
// The same derivation the registration half uses, reached from the other side:
// **a feature the server answers any row of must answer every row of it.** A
// list of implemented features written into this file would be right until 002
// lands and then quietly wrong. A feature no row of which is reachable is a
// feature this build does not implement, which is a reading of the server
// rather than a claim about the roadmap.
func TestTheServerIsReachableOnExactlyTheImplementedRowsOfTheSurfaceDocument(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	served, refused := reachability(t, server, readSurfaceRows(t))

	// A server that answered nothing would satisfy the rule below vacuously —
	// there would be no served feature left to be incomplete.
	if len(served) == 0 {
		t.Fatalf("no row of the surface document is reachable, so this check has nothing to be right about")
	}

	// And a server that answered *everything* would mean this run never
	// observed an unreachable row, which is the primitive the rule is built
	// on. Today that cannot happen — 55 of the 59 rows belong to features
	// nothing here serves — and the day it can, v1 is complete and this
	// assertion is removed deliberately rather than discovered failing.
	if len(refused) == 0 {
		t.Fatalf("every row of the surface document is reachable; either v1 is complete, " +
			"or this check has stopped issuing the requests it thinks it is issuing")
	}

	servedFeatures := map[string]bool{}
	for _, row := range served {
		servedFeatures[row.feature] = true
	}
	t.Logf("features this server implements, read off the wire: %s",
		strings.Join(slices.Sorted(maps.Keys(servedFeatures)), ", "))

	for _, row := range refused {
		if servedFeatures[row.feature] {
			t.Errorf("feature %s is implemented — this server answers other rows of it — and %s is not reachable",
				row.feature, row)
		}
	}
}

// TestTheSweepReachesEveryRouteTheServerServes ties the wire sweep's
// hand-written request list to the surface document, which is the gap T19 left
// open and named.
//
// The wire sweep walks the bodies of the responses in sweptResponses, and that
// list is written by hand: a route added without a row in it is a route whose
// values nothing sweeps. Until this file existed there was nothing here that
// could enumerate what the server serves. Now there is, and the list is checked
// against it rather than against a second hand-written list of the same four
// routes.
func TestTheSweepReachesEveryRouteTheServerServes(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	served, _ := reachability(t, server, readSurfaceRows(t))

	swept := map[string]bool{}
	for _, response := range sweptResponses {
		swept[response.method+" "+response.path] = true
	}

	for _, row := range served {
		if !swept[row.method+" "+row.path] {
			t.Errorf("this server serves %s and no swept response covers it, so nothing sweeps the values it sends", row)
		}
	}
}

// reachability issues one request per row and splits the rows in two.
//
// # What counts as "not served"
//
// The empty refusal of behaviours 1.11: a 404 or a 405, with no body and no
// content type. That is what spec 3.6 says a path matching no route answers,
// and what a path whose row is registered under a different method answers —
// both of which are this row not being served.
//
// The body and the content type are part of the test and not decoration. A
// handler that looked something up and did not find it also answers 404, and a
// future feature's "no item with that identifier" must not be read here as "the
// route does not exist". Distinguishing them by the refusal *shape* is the best
// signal available at the wire; it is not perfect, and the case it would still
// misread — a handler answering an empty 404 with no content type — is written
// down in this feature's handoff rather than left to be discovered.
func reachability(t *testing.T, server *server, rows []surfaceRow) (served, refused []surfaceRow) {
	t.Helper()

	for _, row := range rows {
		got := server.do(t, row.method, requestPathFor(row), goldenHost, nil)

		notRouted := (got.status == http.StatusNotFound || got.status == http.StatusMethodNotAllowed) &&
			len(got.body) == 0 && got.header.Get("Content-Type") == ""
		if notRouted {
			refused = append(refused, row)
			continue
		}
		served = append(served, row)
	}
	return served, refused
}

// pathParameter is a `{name}` run in a row's path.
//
// It matches a run inside a segment as well as a whole segment: five of the
// document's paths put a literal and a parameter in one segment, such as
// /Audio/{itemId}/stream.{container}.
var pathParameter = regexp.MustCompile(`\{[^/{}]+\}`)

// parameterValue is what every path parameter is filled with.
//
// It is a value no route in the document spells literally, so that filling a
// parameter cannot accidentally address a different row: /Users/{userId} must
// not become /Users/Public.
const parameterValue = "conformance-placeholder"

// requestPathFor turns a row's pattern into a path a client can send.
func requestPathFor(row surfaceRow) string {
	return pathParameter.ReplaceAllLiteralString(row.path, parameterValue)
}

// readSurfaceRows reads the document, and fails the test rather than returning
// an empty list if it cannot: a check that silently found no rows would pass.
func readSurfaceRows(t *testing.T) []surfaceRow {
	t.Helper()

	document, err := os.ReadFile(surfaceDocument)
	if err != nil {
		t.Fatalf("reading the v1 surface document: %v", err)
	}
	rows, err := parseSurfaceRows(document)
	if err != nil {
		t.Fatalf("reading %s: %v", surfaceDocument, err)
	}
	return rows
}

// parseSurfaceRows reads the endpoints of the surface document.
//
// It is not a YAML parser and does not pretend to be one — it understands the
// one shape this document is generated in, and refuses everything else. The
// keys are known, so an unknown one is an error rather than a line to skip:
// a reader that skipped what it did not recognise would answer a renamed
// `feature` column with rows that have no feature at all.
func parseSurfaceRows(document []byte) ([]surfaceRow, error) {
	var (
		rows        []surfaceRow
		inEndpoints bool
		current     map[string]string
	)

	finish := func() error {
		if current == nil {
			return nil
		}
		row := surfaceRow{
			path:      current["path"],
			method:    current["method"],
			operation: current["operation"],
			feature:   current["feature"],
		}
		current = nil
		if err := validateSurfaceRow(row); err != nil {
			return err
		}
		rows = append(rows, row)
		return nil
	}

	for number, line := range strings.Split(string(document), "\n") {
		at := func(problem string) error {
			return fmt.Errorf("line %d: %s: %q", number+1, problem, line)
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if !inEndpoints {
			inEndpoints = trimmed == "endpoints:"
			continue
		}

		switch {
		case strings.HasPrefix(line, "  - "):
			if err := finish(); err != nil {
				return nil, err
			}
			current = map[string]string{}
			if err := readSurfaceKey(current, strings.TrimPrefix(line, "  - ")); err != nil {
				return nil, at(err.Error())
			}

		case strings.HasPrefix(line, "    "):
			if current == nil {
				return nil, at("a key outside any endpoint")
			}
			if err := readSurfaceKey(current, strings.TrimPrefix(line, "    ")); err != nil {
				return nil, at(err.Error())
			}

		default:
			return nil, at("a line this reader does not understand")
		}
	}
	if err := finish(); err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, errors.New("the document names no endpoints")
	}
	seen := map[string]string{}
	for _, row := range rows {
		route := row.method + " " + row.path
		if first, ok := seen[route]; ok {
			return nil, fmt.Errorf("%s appears twice, as %s and as %s", route, first, row.operation)
		}
		seen[route] = row.operation
	}
	return rows, nil
}

// surfaceKeys are the columns an endpoint has. The two this check does not read
// are listed so that an unknown key is an error and a known one is not.
var surfaceKeys = map[string]bool{
	"path": true, "method": true, "operation": true, "consumers": true, "feature": true, "level": true,
}

// readSurfaceKey reads one `key: value` into an endpoint under construction.
func readSurfaceKey(into map[string]string, text string) error {
	key, value, ok := strings.Cut(text, ": ")
	if !ok {
		return errors.New("not a `key: value`")
	}
	if !surfaceKeys[key] {
		return fmt.Errorf("%q is not a column of this document", key)
	}
	if _, repeated := into[key]; repeated {
		return fmt.Errorf("%q appears twice in one endpoint", key)
	}
	into[key] = strings.Trim(strings.TrimSpace(value), `"`)
	return nil
}

// featureNumber is what a feature column holds: the three digits that name a
// directory under specs/.
var featureNumber = regexp.MustCompile(`^\d{3}$`)

// httpMethod is a method token, which this document spells in upper case.
var httpMethod = regexp.MustCompile(`^[A-Z]+$`)

// validateSurfaceRow refuses a row this check could not act on.
func validateSurfaceRow(row surfaceRow) error {
	switch {
	case !strings.HasPrefix(row.path, "/"):
		return fmt.Errorf("the path %q does not begin with a slash", row.path)
	case !httpMethod.MatchString(row.method):
		return fmt.Errorf("%q is not an HTTP method", row.method)
	case row.operation == "":
		return fmt.Errorf("%s %s names no operation", row.method, row.path)
	case !featureNumber.MatchString(row.feature):
		return fmt.Errorf("%s %s is owned by %q, which is not a feature number", row.method, row.path, row.feature)
	}
	return nil
}

// TestTheSurfaceDocumentReaderRefusesWhatItCannotRead is the failure proof for
// the reader.
//
// The reader is what supplies every row this file checks, so a reader that
// answered a malformed document with a short list would turn the whole check
// into a statement about the rows it happened to understand.
func TestTheSurfaceDocumentReaderRefusesWhatItCannotRead(t *testing.T) {
	t.Parallel()

	const wellFormed = "endpoints:\n" +
		"  - path: \"/System/Ping\"\n" +
		"    method: GET\n" +
		"    operation: GetPingSystem\n" +
		"    consumers: []\n" +
		"    feature: \"001\"\n" +
		"    level: L2\n"

	if rows, err := parseSurfaceRows([]byte(wellFormed)); err != nil || len(rows) != 1 {
		t.Fatalf("the reader refuses a document it must accept: %d rows, %v", len(rows), err)
	}

	for _, refused := range []struct {
		name     string
		document string
	}{
		{"an empty document", ""},
		{"a document with no endpoints", "reference:\n  jellyfin_source_tag: \"v10.11.11\"\n"},
		{"a renamed column", strings.Replace(wellFormed, "feature:", "owner:", 1)},
		{"a missing feature", strings.Replace(wellFormed, "    feature: \"001\"\n", "", 1)},
		{"a feature that is not a number", strings.Replace(wellFormed, "\"001\"", "\"one\"", 1)},
		{"a path that is not a path", strings.Replace(wellFormed, "\"/System/Ping\"", "\"System/Ping\"", 1)},
		{"a lower-case method", strings.Replace(wellFormed, "method: GET", "method: get", 1)},
		{"a row with no operation", strings.Replace(wellFormed, "    operation: GetPingSystem\n", "", 1)},
		{"an indentation this reader does not know", strings.Replace(wellFormed, "    method: GET", "      method: GET", 1)},
		{"a repeated route", wellFormed + strings.TrimPrefix(wellFormed, "endpoints:\n")},
	} {
		if rows, err := parseSurfaceRows([]byte(refused.document)); err == nil {
			t.Errorf("the reader accepts %s, and found %d rows in it", refused.name, len(rows))
		}
	}
}

// TestTheSurfaceDocumentIsWhatThisCheckThinksItIs states the two properties the
// check above depends on and cannot assert about itself.
func TestTheSurfaceDocumentIsWhatThisCheckThinksItIs(t *testing.T) {
	t.Parallel()

	rows := readSurfaceRows(t)

	// More than one feature, because the rule "a feature is all of its rows or
	// none of them" says nothing about a document that names one feature.
	features := map[string]bool{}
	for _, row := range rows {
		features[row.feature] = true
	}
	if len(features) < 2 {
		t.Errorf("the document names %d feature(s); this check needs an unimplemented one to have anything to compare against", len(features))
	}

	// And every parameter is filled, because a path still carrying braces
	// would be sent verbatim and answered 404 for a reason that has nothing to
	// do with what is registered.
	for _, row := range rows {
		if sent := requestPathFor(row); strings.ContainsAny(sent, "{}") {
			t.Errorf("%s becomes %q, which is not a path a client can send", row, sent)
		}
	}
}
