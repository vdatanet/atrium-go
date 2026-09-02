package surface

import (
	"fmt"
	"sort"
	"strings"
)

// Load reads a surface document and returns the table it describes.
//
// # Why this reads the document itself rather than through a YAML library
//
// ADR-0002 puts the standard library first and argues a further dependency
// "where it is needed, in the plan that needs it". Plan 3 does not name one,
// and this document does not need one: it is generated, its shape is four
// kinds of line, and every value in it is a scalar or a flow sequence of
// scalars.
//
// The reader is strict in the direction a general parser is lenient. A YAML
// library reads a document that has drifted — a key nobody consumes, an item
// missing a field, a second document — and hands back something that loads. A
// row that fails to say what level it must reach is not a row with a default;
// it is a row nobody has decided about, and Principle VI is the reason this
// refuses rather than assumes. So: an unknown key, a missing key, a repeated
// key, a line at an unexpected indent and a value it cannot read are each an
// error naming the line.
//
// What it does not accept is therefore worth stating, because it is not YAML:
// no tabs, no trailing comments after a value, no escapes inside a quoted
// scalar, no block sequences or nested maps below an endpoint, no anchors, no
// multiple documents. Every one of those is an error rather than a
// misreading, and if the generator ever emits one the failure is loud.
func Load(data []byte) (*Table, error) {
	doc, err := readDocument(data)
	if err != nil {
		return nil, err
	}
	return doc.table()
}

// rawDocument is the parsed shape of the file, one step short of a Table: the
// values are still strings and nothing has been validated beyond the syntax.
type rawDocument struct {
	reference map[string]field
	endpoints []rawEndpoint
}

// rawEndpoint is one item of the endpoints sequence.
type rawEndpoint struct {
	fields map[string]field
	line   int
}

// field is a value and the line it was written on, so that every refusal can
// point at one.
type field struct {
	value string
	line  int
}

// endpointKeys are the keys an endpoint must carry, in the order the generator
// writes them. Every one is required: this is the list the reader checks in
// both directions, so a key absent from an item and a key nobody expects are
// the same kind of error.
var endpointKeys = []string{"path", "method", "operation", "consumers", "feature", "level"}

// referenceKeys are the keys the reference block must carry, same rule.
var referenceKeys = []string{"jellyfin_openapi_version", "jellyfin_source_tag"}

// readDocument turns the bytes into a document, or into the first syntax
// error it meets.
func readDocument(data []byte) (*rawDocument, error) {
	doc := &rawDocument{}

	// section is which top-level key the reader is under, "" before the first.
	section := ""

	for number, raw := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
		lineNumber := number + 1
		text := strings.TrimSuffix(raw, "\r")

		if strings.ContainsRune(text, '\t') {
			return nil, fmt.Errorf("line %d: a tab is not an indent", lineNumber)
		}
		trimmed := strings.TrimLeft(text, " ")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.TrimRight(trimmed, " ") != trimmed {
			return nil, fmt.Errorf("line %d: trailing whitespace", lineNumber)
		}
		indent := len(text) - len(trimmed)

		switch indent {
		case 0:
			switch trimmed {
			case "reference:":
				if doc.reference != nil {
					return nil, fmt.Errorf("line %d: reference: appears twice", lineNumber)
				}
				doc.reference = map[string]field{}
			case "endpoints:":
				if doc.endpoints != nil {
					return nil, fmt.Errorf("line %d: endpoints: appears twice", lineNumber)
				}
				doc.endpoints = []rawEndpoint{}
			default:
				return nil, fmt.Errorf("line %d: unknown top-level key %q, expected reference: or endpoints:", lineNumber, trimmed)
			}
			section = strings.TrimSuffix(trimmed, ":")

		case 2:
			switch section {
			case "reference":
				key, value, err := splitPair(trimmed, lineNumber)
				if err != nil {
					return nil, err
				}
				if _, repeated := doc.reference[key]; repeated {
					return nil, fmt.Errorf("line %d: key %q appears twice in reference", lineNumber, key)
				}
				doc.reference[key] = field{value: value, line: lineNumber}
			case "endpoints":
				rest, isItem := strings.CutPrefix(trimmed, "- ")
				if !isItem {
					return nil, fmt.Errorf("line %d: expected a new endpoint starting with %q", lineNumber, "- ")
				}
				key, value, err := splitPair(rest, lineNumber)
				if err != nil {
					return nil, err
				}
				doc.endpoints = append(doc.endpoints, rawEndpoint{
					fields: map[string]field{key: {value: value, line: lineNumber}},
					line:   lineNumber,
				})
			default:
				return nil, fmt.Errorf("line %d: indented line before any top-level key", lineNumber)
			}

		case 4:
			if section != "endpoints" || len(doc.endpoints) == 0 {
				return nil, fmt.Errorf("line %d: indented line outside an endpoint", lineNumber)
			}
			key, value, err := splitPair(trimmed, lineNumber)
			if err != nil {
				return nil, err
			}
			item := &doc.endpoints[len(doc.endpoints)-1]
			if _, repeated := item.fields[key]; repeated {
				return nil, fmt.Errorf("line %d: key %q appears twice in the endpoint at line %d", lineNumber, key, item.line)
			}
			item.fields[key] = field{value: value, line: lineNumber}

		default:
			return nil, fmt.Errorf("line %d: unexpected indent of %d spaces", lineNumber, indent)
		}
	}

	if doc.reference == nil {
		return nil, fmt.Errorf("the document has no reference: block")
	}
	if doc.endpoints == nil {
		return nil, fmt.Errorf("the document has no endpoints: sequence")
	}
	if len(doc.endpoints) == 0 {
		return nil, fmt.Errorf("the document declares no endpoints")
	}
	return doc, nil
}

// splitPair reads `key: value` out of one already-trimmed line.
func splitPair(text string, lineNumber int) (key, value string, err error) {
	colon := strings.Index(text, ":")
	if colon < 0 {
		return "", "", fmt.Errorf("line %d: expected `key: value`, got %q", lineNumber, text)
	}
	key = text[:colon]
	if key == "" {
		return "", "", fmt.Errorf("line %d: empty key", lineNumber)
	}
	for _, r := range key {
		if r != '_' && (r < 'a' || r > 'z') {
			return "", "", fmt.Errorf("line %d: key %q is not lower case", lineNumber, key)
		}
	}
	rest := text[colon+1:]
	if rest == "" {
		return "", "", fmt.Errorf("line %d: key %q has no value", lineNumber, key)
	}
	if !strings.HasPrefix(rest, " ") {
		return "", "", fmt.Errorf("line %d: expected a space after %q:", lineNumber, key)
	}
	return key, strings.TrimPrefix(rest, " "), nil
}

// scalar reads one value: either bare, or double-quoted with no escapes.
func scalar(f field, key string) (string, error) {
	value := f.value
	if strings.HasPrefix(value, `"`) {
		if len(value) < 2 || !strings.HasSuffix(value, `"`) {
			return "", fmt.Errorf("line %d: %s: unterminated quoted value %s", f.line, key, value)
		}
		value = value[1 : len(value)-1]
	}
	if strings.ContainsAny(value, `"'[]{}#`) && !strings.HasPrefix(f.value, `"`) {
		return "", fmt.Errorf("line %d: %s: %q is not a plain scalar", f.line, key, value)
	}
	if strings.ContainsRune(value, '"') {
		return "", fmt.Errorf("line %d: %s: an escape inside a quoted value is not read here", f.line, key)
	}
	if value == "" {
		return "", fmt.Errorf("line %d: %s is empty", f.line, key)
	}
	return value, nil
}

// sequence reads a flow sequence of scalars: `[]`, or `[a, b]`.
func sequence(f field, key string) ([]string, error) {
	inner, ok := strings.CutPrefix(f.value, "[")
	if !ok {
		return nil, fmt.Errorf("line %d: %s: expected a list in [brackets], got %q", f.line, key, f.value)
	}
	inner, ok = strings.CutSuffix(inner, "]")
	if !ok {
		return nil, fmt.Errorf("line %d: %s: unterminated list %q", f.line, key, f.value)
	}
	if strings.TrimSpace(inner) == "" {
		// An empty list is a declared state, not a missing one: the endpoint
		// is included by design and the prose twin says why.
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(inner, ",") {
		element, err := scalar(field{value: strings.TrimSpace(part), line: f.line}, key)
		if err != nil {
			return nil, err
		}
		out = append(out, element)
	}
	return out, nil
}

// table validates the document and indexes it.
func (d *rawDocument) table() (*Table, error) {
	if err := exactKeys(d.reference, referenceKeys, "reference", 0); err != nil {
		return nil, err
	}
	openAPIVersion, err := scalar(d.reference[referenceKeys[0]], referenceKeys[0])
	if err != nil {
		return nil, err
	}
	sourceTag, err := scalar(d.reference[referenceKeys[1]], referenceKeys[1])
	if err != nil {
		return nil, err
	}

	t := &Table{
		reference:     Reference{OpenAPIVersion: openAPIVersion, SourceTag: sourceTag},
		methodsByPath: map[string][]string{},
		rowByRoute:    map[route]int{},
	}

	// folded remembers the line each path's canonical spelling first appeared
	// on, keyed case-insensitively: two spellings that differ only in case
	// would make canonicalisation ambiguous, and there would be no way to
	// choose between them at a request (spec 3.6).
	folded := map[string]field{}
	operations := map[string]field{}

	for _, item := range d.endpoints {
		if err := exactKeys(item.fields, endpointKeys, "the endpoint", item.line); err != nil {
			return nil, err
		}

		endpoint, err := item.endpoint()
		if err != nil {
			return nil, err
		}

		key := route{method: endpoint.Method, path: endpoint.Path}
		if first, duplicate := t.rowByRoute[key]; duplicate {
			return nil, fmt.Errorf("line %d: %s %s is already declared at line %d",
				item.line, endpoint.Method, endpoint.Path, d.endpoints[first].line)
		}
		if first, duplicate := operations[endpoint.Operation]; duplicate {
			return nil, fmt.Errorf("line %d: operation %s is already declared at line %d",
				item.line, endpoint.Operation, first.line)
		}
		if first, seen := folded[strings.ToLower(endpoint.Path)]; seen && first.value != endpoint.Path {
			return nil, fmt.Errorf("line %d: path %s differs only in casing from %s at line %d",
				item.line, endpoint.Path, first.value, first.line)
		}

		if _, seen := t.methodsByPath[endpoint.Path]; !seen {
			t.paths = append(t.paths, endpoint.Path)
		}
		t.rowByRoute[key] = len(t.endpoints)
		operations[endpoint.Operation] = field{value: endpoint.Operation, line: item.line}
		folded[strings.ToLower(endpoint.Path)] = field{value: endpoint.Path, line: item.line}
		t.methodsByPath[endpoint.Path] = append(t.methodsByPath[endpoint.Path], endpoint.Method)
		t.endpoints = append(t.endpoints, endpoint)
	}

	// Sorting here rather than at the caller is deliberate: this is the
	// `Allow` header's order (spec 3.6, plan 3), and one place that sorts is
	// one place to be wrong.
	for _, methods := range t.methodsByPath {
		sort.Strings(methods)
	}
	return t, nil
}

// endpoint validates one item's fields and returns the row.
func (r rawEndpoint) endpoint() (Endpoint, error) {
	var e Endpoint
	var err error

	if e.Path, err = scalar(r.fields["path"], "path"); err != nil {
		return Endpoint{}, err
	}
	if !strings.HasPrefix(e.Path, "/") || strings.HasSuffix(e.Path, "/") ||
		strings.Contains(e.Path, "//") || strings.ContainsAny(e.Path, " ?#") {
		return Endpoint{}, fmt.Errorf("line %d: path %q is not a canonical spelling", r.fields["path"].line, e.Path)
	}

	if e.Method, err = scalar(r.fields["method"], "method"); err != nil {
		return Endpoint{}, err
	}
	for _, c := range e.Method {
		if c < 'A' || c > 'Z' {
			return Endpoint{}, fmt.Errorf("line %d: method %q is not an upper-case token", r.fields["method"].line, e.Method)
		}
	}

	if e.Operation, err = scalar(r.fields["operation"], "operation"); err != nil {
		return Endpoint{}, err
	}
	if strings.ContainsRune(e.Operation, ' ') {
		return Endpoint{}, fmt.Errorf("line %d: operation %q contains a space", r.fields["operation"].line, e.Operation)
	}

	if e.Consumers, err = sequence(r.fields["consumers"], "consumers"); err != nil {
		return Endpoint{}, err
	}

	if e.Feature, err = scalar(r.fields["feature"], "feature"); err != nil {
		return Endpoint{}, err
	}
	if len(e.Feature) != 3 || strings.Trim(e.Feature, "0123456789") != "" {
		return Endpoint{}, fmt.Errorf("line %d: feature %q is not a three-digit specs/ directory number", r.fields["feature"].line, e.Feature)
	}

	rawLevel, err := scalar(r.fields["level"], "level")
	if err != nil {
		return Endpoint{}, err
	}
	level, known := parseLevel(rawLevel)
	if !known {
		return Endpoint{}, fmt.Errorf("line %d: unknown level %q, expected one of %s",
			r.fields["level"].line, rawLevel, strings.Join(levelNames[:], ", "))
	}
	e.Level = level

	return e, nil
}

// exactKeys refuses a map that is missing an expected key or carries one
// nobody expects. Both directions matter: the first is a row nobody decided
// about, and the second is a row somebody decided about in a vocabulary this
// loader does not read, which is the same thing with a friendlier face.
func exactKeys(fields map[string]field, expected []string, what string, line int) error {
	where := ""
	if line > 0 {
		where = fmt.Sprintf(" at line %d", line)
	}
	for _, key := range expected {
		if _, ok := fields[key]; !ok {
			return fmt.Errorf("%s%s has no %s:", what, where, key)
		}
	}
	if len(fields) == len(expected) {
		return nil
	}
	unknown := make([]string, 0, len(fields)-len(expected))
	for key, f := range fields {
		if !contains(expected, key) {
			unknown = append(unknown, fmt.Sprintf("%q (line %d)", key, f.line))
		}
	}
	sort.Strings(unknown)
	return fmt.Errorf("%s%s carries unknown key %s", what, where, strings.Join(unknown, ", "))
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
