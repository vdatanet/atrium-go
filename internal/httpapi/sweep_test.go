package httpapi_test

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
	"github.com/vdatanet/atrium-go/internal/units"
)

// This file is the half of the two cross-cutting L1 sweeps (spec 6,
// conformance L1) that walks Go *types*. Its twin walks response *bytes* and
// lives in conformance/sweep_test.go.
//
// # Why there are two halves, and which document each one honours
//
// architecture 8 puts "the two reflection sweeps" in conformance/.
// architecture 3 forbids conformance/ from importing anything under internal/,
// and tools/check_conformance_imports enforces it over `go list -deps`. A
// reflection sweep over this project's response models needs those models, so
// the two rules cannot both be honoured by one file. **The import rule wins.**
// It is mechanically enforced, ADR-0007 depends on the boundary it draws, and
// conformance/doc.go states what it buys: everything a conformance test knows
// is something a client could have known. Relaxing that to make a placement
// sentence true would trade the boundary for a directory name.
//
// So the sweeps are split by what each one can see, and the split is not a
// compromise — each half reaches something the other cannot:
//
//   - This half sees **every field of every registered response model**,
//     including the ones no request has yet produced a value for. PackageName
//     is the worked example: it is never sent, so no body sweep will ever see
//     it, and its name and its optionality are still contract.
//   - The other half sees **the values**, which is where the date rule lives:
//     conformance L1's own correction is that DateCreated, DateLastMediaAdded
//     and LastPlaybackCheckIn are dates whose names do not end in Date, so a
//     rule keyed on the field name checks six of the nine date-valued fields
//     observed `[probe: tools/probe_wire_format, Jellyfin 10.11.11,
//     2026-09-02]`. A type cannot answer "is this a date"; a value can.
//
// # Where the split could leave a gap, and what closes it
//
// A field is swept by neither half if it is in no registered model *and* in no
// body the other half issues. The first clause is closed here, by
// TestEveryRegisteredOperationDeclaresItsResponseModel: the registry below is
// compared against the operations the router is actually built with, so a
// route added without a model fails this package. The second is the wire
// sweep's own coverage, and it is the weaker of the two — see the note there.
//
// architecture 8 carries the amendment, dated.

// responseModels is the type of the body each registered operation answers
// with: the "registered response models" architecture 4 and behaviours 1.1
// both say the casing sweep walks.
//
// It is test-only on purpose. A registry compiled into the server would be a
// second list of what this API sends, kept in step with the handlers by hand;
// what makes this one trustworthy is not that it is complete by construction
// but that its completeness is *checked* against the router, below.
//
// A route whose body is not an object has its type here all the same:
// /System/Ping answers a bare JSON string (spec 3.3), which has no property
// name to sweep, and saying so with a type is how the row is distinguished
// from a row somebody forgot.
var responseModels = map[string]reflect.Type{
	"GetPublicSystemInfo": reflect.TypeOf(httpapi.PublicSystemInfo{}),
	"GetSystemInfo":       reflect.TypeOf(httpapi.SystemInfo{}),
	"GetPingSystem":       reflect.TypeOf(""),
	"PostPingSystem":      reflect.TypeOf(""),
}

// TestEveryRegisteredOperationDeclaresItsResponseModel is what stops the split
// between the two sweeps from becoming a hole.
//
// The registry above is a hand-written list, and a hand-written list of what a
// server sends is exactly the thing that goes stale. So it is not trusted: the
// routes callback is built from the real table and applied to a real router,
// the router is walked, and every method-and-pattern it exposes is mapped back
// to the operation surface.yaml names it by. The two sets must be equal.
//
// A feature that adds a handler and forgets its model fails here, naming the
// operation — before any of its fields have gone unswept.
func TestEveryRegisteredOperationDeclaresItsResponseModel(t *testing.T) {
	registered := registeredOperations(t)

	for _, operation := range registered {
		if _, ok := responseModels[operation]; !ok {
			t.Errorf("the router registers %s and responseModels has no entry for it, "+
				"so nothing sweeps the fields of whatever it answers with", operation)
		}
	}

	for operation := range responseModels {
		if !slices.Contains(registered, operation) {
			t.Errorf("responseModels declares %s and the router registers no such operation, "+
				"so the sweep is walking a model this server does not send", operation)
		}
	}
}

// TestEveryResponseFieldNameIsPascalCase is the casing sweep (AC-10,
// behaviours 1.1).
//
// It walks the declared type of every registered response model, through
// pointers, slices, arrays, dictionary *values* and embedded structs, and fails
// on a property name that is not PascalCase.
//
// behaviours 1.1 calls this "the single most likely source of a silent, total
// incompatibility", and it is worth being clear about why a sweep is the answer
// rather than a review habit: a camelCase body is not a degraded response, it
// is an empty object to a client's decoder, and the failure is total on the
// route that has it and invisible on every other.
func TestEveryResponseFieldNameIsPascalCase(t *testing.T) {
	for _, operation := range slices.Sorted(maps.Keys(responseModels)) {
		t.Run(operation, func(t *testing.T) {
			for _, found := range sweepCasing(responseModels[operation]) {
				t.Errorf("%s", found)
			}
		})
	}
}

// TestEveryUnitCarryingFieldHasAUnitType is the type half of the unit sweep
// (spec 6, conformance L1).
//
// conformance L1 states the rule as "every field whose name ends in Ticks must
// be an integer, and every field whose name ends in Date must serialise with
// seven fractional digits and a Z", and then corrects the second half itself:
// the suffix is a heuristic for finding a date, not the rule. What a *type*
// can answer is therefore narrower than the whole rule, and this test asks only
// the questions a type can answer:
//
//   - a field whose name ends in Ticks has an integer type;
//   - a field whose name ends in Date has type units.Time;
//   - no field has type time.Time, whatever it is called.
//
// The third is the one that does real work. time.Time is the type a date
// reaches for by reflex, and encoding/json writes it as RFC 3339 with trailing
// zeros trimmed — "2025-06-19T00:00:00Z" for the value behaviours 1.2 says the
// reference spells "2025-06-19T00:00:00.0000000Z". A field of that type cannot
// be right, so it is refused by its type rather than by the value it happens to
// carry in a test, and DateCreated is caught along with PremiereDate.
//
// The rest of the rule is the wire sweep's, over values, in conformance/.
func TestEveryUnitCarryingFieldHasAUnitType(t *testing.T) {
	for _, operation := range slices.Sorted(maps.Keys(responseModels)) {
		t.Run(operation, func(t *testing.T) {
			for _, found := range sweepUnits(responseModels[operation]) {
				t.Errorf("%s", found)
			}
		})
	}
}

// 001 sends no tick and no date — spec 3.1 and spec 3.2 are strings, booleans,
// one integer port and two empty arrays — so the two tests above are, today,
// a sweep that finds nothing. That is the state a sweep is supposed to be in
// and it is also the state in which it proves nothing, which is what the next
// three tests are for.

// modelWithACamelCaseField is a test-only model, and it is the whole evidence
// that the casing sweep can fail.
//
// It is declared in a _test.go file, which is the strong form of "it cannot
// leak into the served surface": the file is not part of the package the server
// is built from, so this type does not exist in the binary at all. The weaker
// guarantee is beside it — responseModels is checked against the router above,
// so a model that is not answered by a registered operation cannot be swept as
// though it were, and one that is not registered cannot be served.
type modelWithACamelCaseField struct {
	// The wire name comes from the tag, which is what a struct tag is for and
	// therefore where the mistake would really be made: the Go field name is
	// PascalCase and reads fine in review.
	LocalAddress string `json:"localAddress"`

	// Nested, so that the failure is one the walk had to descend to find. A
	// sweep that only looked at the top level of each registered model would
	// pass this file and fail the first response with a nested object in it.
	Nested struct {
		ServerName string `json:"serverName"`
	}
}

// modelWithACamelCaseFieldBehindAnEmbeddedStruct is the shape T18 warned about:
// SystemInfo embeds PublicSystemInfo, and encoding/json flattens an embedded
// struct's fields into the parent object. A sweep that read reflect.Field(i)
// without recursing into an anonymous field would see one field named after a
// type and miss the seven inside it.
type modelWithACamelCaseFieldBehindAnEmbeddedStruct struct {
	modelWithACamelCaseField
	Version string
}

// modelWithAStringDate carries a date in the type Go reaches for when it is not
// thinking: a string. Nothing about the type says seven digits, so the value is
// whatever the handler formatted, and the wire sweep is what catches the value.
// The name is what catches it here.
type modelWithAStringDate struct {
	PremiereDate string
}

// modelWithAStandardLibraryTime is the same mistake made carefully. time.Time
// is the right type in every other Go program and the wrong one here.
type modelWithAStandardLibraryTime struct {
	DateCreated time.Time
}

// modelWithFractionalTicks is a duration measured in something other than whole
// ticks. behaviours 1.3 puts ticks in storage as well as on the wire "so no
// conversion can be forgotten at a boundary"; a float is that conversion,
// forgotten.
type modelWithFractionalTicks struct {
	RunTimeTicks float64
}

// TestTheCasingSweepCatchesACamelCaseField is the failure proof AC-10's
// *Verified by* line asks for: a sweep that has never failed has proved
// nothing.
//
// It runs the same function the registered models are swept with, over models
// that exist only in this file, and requires the failure to name the field.
func TestTheCasingSweepCatchesACamelCaseField(t *testing.T) {
	cases := []struct {
		name  string
		model reflect.Type
		want  []string
	}{
		{
			name:  "at the top level and nested",
			model: reflect.TypeOf(modelWithACamelCaseField{}),
			want:  []string{"localAddress", "serverName"},
		},
		{
			name:  "through an embedded struct",
			model: reflect.TypeOf(modelWithACamelCaseFieldBehindAnEmbeddedStruct{}),
			want:  []string{"localAddress", "serverName"},
		},
		{
			name:  "in a slice element",
			model: reflect.TypeOf(struct{ Items []modelWithACamelCaseField }{}),
			want:  []string{"localAddress", "serverName"},
		},
		{
			name: "in a dictionary value",
			model: reflect.TypeOf(struct {
				Items map[string]*modelWithACamelCaseField
			}{}),
			want: []string{"localAddress", "serverName"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			found := sweepCasing(c.model)
			if len(found) != len(c.want) {
				t.Fatalf("the casing sweep reported %d findings, want %d:\n%s",
					len(found), len(c.want), strings.Join(found, "\n"))
			}
			for i, want := range c.want {
				if !strings.Contains(found[i], want) {
					t.Errorf("finding %d is %q and does not name %q", i, found[i], want)
				}
			}
		})
	}
}

// TestTheCasingSweepDoesNotFireOnADictionaryKey is the trap conformance L1
// records beside the sweep: a dictionary's keys are data, not schema, and a
// sweep that treated every JSON object key as a property name reported 688 of
// 899 keys as failures in one run of the reference
// `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`.
//
// **This half of the sweep cannot fall into it, and the test is therefore
// weaker than it looks** — said plainly rather than left for the closing audit
// to find. A Go map's key type carries no property names, and a JSON object's
// keys are strings, so even a sweep that walked Key() would find nothing to
// report: the mutation this test is supposed to catch does not exist. It is
// kept as the statement of the rule at this level, and the guard that really
// earns its keep is the wire sweep's dictionaryPointers, which is exercised in
// conformance/sweep_test.go with a body that has a dictionary in it.
func TestTheCasingSweepDoesNotFireOnADictionaryKey(t *testing.T) {
	model := reflect.TypeOf(struct {
		ImageBlurHashes map[string]string
	}{})

	if found := sweepCasing(model); len(found) != 0 {
		t.Errorf("the casing sweep reported %d findings on a dictionary of strings:\n%s",
			len(found), strings.Join(found, "\n"))
	}
}

// TestTheUnitSweepCatchesAFieldThatCannotSpellItsUnit is the unit sweep's
// failure proof at the type level. Its value-level twin — the deliberately
// three-digit date — is in conformance/sweep_test.go, because a value is what
// that half of the rule is about.
func TestTheUnitSweepCatchesAFieldThatCannotSpellItsUnit(t *testing.T) {
	cases := []struct {
		name  string
		model reflect.Type
		want  string
	}{
		{
			name:  "a date held as a string",
			model: reflect.TypeOf(modelWithAStringDate{}),
			want:  "PremiereDate",
		},
		{
			name:  "a date held as a time.Time",
			model: reflect.TypeOf(modelWithAStandardLibraryTime{}),
			want:  "DateCreated",
		},
		{
			name:  "ticks held as a float",
			model: reflect.TypeOf(modelWithFractionalTicks{}),
			want:  "RunTimeTicks",
		},
		{
			name:  "a time.Time under a name that says nothing",
			model: reflect.TypeOf(struct{ Added time.Time }{}),
			want:  "Added",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			found := sweepUnits(c.model)
			if len(found) != 1 {
				t.Fatalf("the unit sweep reported %d findings, want 1:\n%s",
					len(found), strings.Join(found, "\n"))
			}
			if !strings.Contains(found[0], c.want) {
				t.Errorf("the finding is %q and does not name %q", found[0], c.want)
			}
		})
	}
}

// TestTheUnitSweepAcceptsTheUnitTypes is the other side of the proof above: a
// sweep that failed on everything would pass every test written so far.
func TestTheUnitSweepAcceptsTheUnitTypes(t *testing.T) {
	model := reflect.TypeOf(struct {
		PremiereDate units.Time
		EndDate      *units.Time
		RunTimeTicks units.Ticks
		StartTicks   int64
	}{})

	if found := sweepUnits(model); len(found) != 0 {
		t.Errorf("the unit sweep reported %d findings on the unit types themselves:\n%s",
			len(found), strings.Join(found, "\n"))
	}
}

// TestThePascalCaseRuleAcceptsEveryPascalCaseNameOfThePinnedDocument keeps the
// predicate from being either half of the way wrong.
//
// A predicate that is too strict is the dangerous one, because it fails on a
// name that is correct and gets loosened by whoever meets it. The pinned
// document's own names are the corpus that decides: EnableIPv4, UICulture,
// Video3DFormat and Hdr10PlusPresentFlag all have to pass, and a rule spelled
// as "capital, then lower-case letters, repeated" refuses three of them.
//
// **23 of the 1026 names are not PascalCase**, and they are listed rather than
// filtered, because the list is a finding about the document rather than an
// exception to the rule: five are RFC 7807's problem-details members and the
// other eighteen are the plugin repository manifest's, which the reference
// deserialises from a third-party file rather than declaring. None belongs to a
// v1 route. behaviours 1.1 says the reference "serialises every JSON property
// in PascalCase"; over the document that is 1003 of 1026, and the difference is
// recorded here rather than edited into the exported measurement.
func TestThePascalCaseRuleAcceptsEveryPascalCaseNameOfThePinnedDocument(t *testing.T) {
	const path = "../../docs/compatibility/property-names.json"

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var document struct {
		Count int      `json:"count"`
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(document.Names) != document.Count {
		t.Fatalf("%s holds %d names and claims %d", path, len(document.Names), document.Count)
	}

	// The names of the pinned document that this project's own rule refuses.
	// Listed in full so that the count is an assertion and not a tolerance.
	notPascalCase := map[string]bool{
		// RFC 7807 problem details, serialised by the reference's framework
		// rather than by its own naming policy.
		"detail": true, "instance": true, "status": true, "title": true, "type": true,
		// The plugin repository manifest, which is a third-party document the
		// reference reads rather than a response it composes.
		"category": true, "changelog": true, "checksum": true, "description": true,
		"guid": true, "imageUrl": true, "name": true, "overview": true, "owner": true,
		"repositoryName": true, "repositoryUrl": true, "score": true, "sourceUrl": true,
		"subScore": true, "targetAbi": true, "timestamp": true, "version": true,
		"versions": true,
	}

	var refused []string
	for _, name := range document.Names {
		if !isPascalCase(name) {
			refused = append(refused, name)
		}
	}

	if len(refused) != len(notPascalCase) {
		t.Fatalf("the rule refuses %d of the %d names of the pinned document, want %d:\n%s",
			len(refused), len(document.Names), len(notPascalCase), strings.Join(refused, ", "))
	}
	for _, name := range refused {
		if !notPascalCase[name] {
			t.Errorf("the rule refuses %q, which is a PascalCase name of the pinned document", name)
		}
	}
}

// sweepCasing walks a response model's type and reports every property name
// that is not PascalCase.
func sweepCasing(model reflect.Type) []string {
	var found []string
	walkModel(model, "", map[reflect.Type]bool{}, func(where, name string, field reflect.StructField) {
		if !isPascalCase(name) {
			found = append(found, fmt.Sprintf(
				"%s writes the property name %q, which is not PascalCase (behaviours 1.1)", where, name))
		}
	})
	return found
}

// sweepUnits walks a response model's type and reports every field whose type
// cannot spell the unit its name claims.
func sweepUnits(model reflect.Type) []string {
	timeType := reflect.TypeOf(time.Time{})
	unitsTimeType := reflect.TypeOf(units.Time{})

	var found []string
	walkModel(model, "", map[reflect.Type]bool{}, func(where, name string, field reflect.StructField) {
		declared := indirect(field.Type)

		switch {
		case declared == timeType:
			found = append(found, fmt.Sprintf(
				"%s holds %s in a time.Time, which encoding/json writes as RFC 3339 with trailing "+
					"zeros trimmed; a date on this wire is seven fractional digits and a Z, so the "+
					"type is units.Time (behaviours 1.2)", where, name))

		case strings.HasSuffix(name, "Date") && declared != unitsTimeType:
			found = append(found, fmt.Sprintf(
				"%s holds %s in a %s; a field whose name ends in Date is a date and a date is a "+
					"units.Time (conformance L1)", where, name, field.Type))

		case strings.HasSuffix(name, "Ticks") && !isInteger(declared):
			found = append(found, fmt.Sprintf(
				"%s holds %s in a %s; ticks are whole 100-nanosecond intervals and serialise as an "+
					"integer (behaviours 1.3)", where, name, field.Type))
		}
	})
	return found
}

// walkModel calls report once per property a model can put on the wire, with
// the path it was reached by and the name it is written under.
//
// # What it descends into, and what it deliberately does not
//
// Pointers, slices, arrays and dictionary *values* are all containers whose
// contents reach the wire, so the walk goes through them. A dictionary's *keys*
// are not: conformance L1's own measurement is that treating them as property
// names reported 688 of 899 keys as casing failures, because ImageBlurHashes is
// keyed by image tag. Key() is therefore never visited, and there is a test
// that says so.
//
// An anonymous struct field with no name in its tag is *promoted* by
// encoding/json: its members arrive at the top level of the parent object and
// the type's own name reaches the wire not at all. So the walk recurses without
// reporting a name — which is what makes SystemInfo's seven inherited fields
// visible to the sweep rather than one field called PublicSystemInfo (T18).
//
// An interface field — []any, in 001's two empty arrays — has no fields to
// walk. Nothing is reported and nothing is descended into, because a type is
// the wrong place to ask what an interface will hold. Whatever ends up in one
// is swept as a value by the wire sweep instead, which is the second reason the
// two halves are both needed.
func walkModel(t reflect.Type, where string, seen map[reflect.Type]bool, report func(where, name string, field reflect.StructField)) {
	t = indirect(t)

	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		walkModel(t.Elem(), where+"[]", seen, report)
		return
	case reflect.Map:
		// Elem only. See the doc comment.
		walkModel(t.Elem(), where+"{}", seen, report)
		return
	case reflect.Struct:
		// A model that contains itself — a folder holding folders — would
		// otherwise walk forever. The guard is on the type, so each is visited
		// once per sweep.
		if seen[t] {
			return
		}
		seen[t] = true
		defer delete(seen, t)
	default:
		return
	}

	if where == "" {
		where = t.Name()
		if where == "" {
			where = t.String()
		}
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name, ok := wireName(field)
		if !ok {
			continue
		}

		if name == "" {
			// Promoted: the embedded type's members are the parent's members.
			walkModel(field.Type, where, seen, report)
			continue
		}

		report(where, name, field)
		walkModel(field.Type, where+"."+name, seen, report)
	}
}

// wireName answers what a field is written as, and whether it is written at
// all.
//
// The rules are encoding/json's, because encoding/json is what writes the body
// (ADR-0002, internal/wire): a field with `json:"-"` is not written; an
// unexported field is not written unless it is an embedded struct, whose
// members are; and a tag's name, where there is one, replaces the field's own.
//
// The empty string with ok true means "promoted": walk into it, report nothing
// for it.
func wireName(field reflect.StructField) (string, bool) {
	tag := field.Tag.Get("json")
	name, _, _ := strings.Cut(tag, ",")

	if tag == "-" {
		return "", false
	}

	if field.Anonymous && name == "" && indirect(field.Type).Kind() == reflect.Struct {
		// Promoted, whether or not the embedded type is exported: T6 measured
		// encoding/json flattening both.
		return "", true
	}

	if !field.IsExported() {
		return "", false
	}
	if name != "" {
		return name, true
	}
	return field.Name, true
}

// isPascalCase reports whether a property name is spelled the way every
// property name of a v1 response must be (spec 3.0.1, behaviours 1.1).
//
// The rule is deliberately loose in the middle and strict at the front: an
// upper-case letter first, and letters and digits after it. It is not "capital
// then lower-case letters, repeated", because the pinned document's own names
// include EnableIPv4, UICulture, Video3DFormat and Hdr10PlusPresentFlag, and a
// rule that refused those would be loosened by the first person to meet it —
// at which point it would stop refusing localAddress too. The corpus test above
// is what holds the looseness to exactly what the document needs.
func isPascalCase(name string) bool {
	for i, r := range name {
		if i == 0 {
			if !unicode.IsUpper(r) {
				return false
			}
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return name != ""
}

// isInteger reports whether a type serialises as a JSON integer.
func isInteger(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}

// indirect is a type with its pointers removed. An optional field is a pointer
// (ADR-0002), and its optionality says nothing about its name or its unit.
func indirect(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

// registeredOperations is every operation the router is actually built with,
// found by walking it rather than by reading the list that built it.
//
// The mapping back from a method and a pattern to an operationId goes through
// the same table the registration used, which is safe here because what is
// being checked is not the *spelling* of a route — that is T20's question — but
// which rows are served at all.
func registeredOperations(t *testing.T) []string {
	t.Helper()

	table := surface.V1()
	handler := newSystemHandlerFrom(t, httpapi.SystemHandlerConfig{Installations: &fakeInstallations{}})

	routes, err := httpapi.Routes(table, httpapi.Handlers{System: handler})
	if err != nil {
		t.Fatalf("building the routes callback: %v", err)
	}

	router := chi.NewRouter()
	routes(router)

	var operations []string
	err = chi.Walk(router, func(method, pattern string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		endpoint, ok := table.Lookup(method, pattern)
		if !ok {
			return fmt.Errorf("the router serves %s %s and surface.yaml has no such row", method, pattern)
		}
		operations = append(operations, endpoint.Operation)
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	slices.Sort(operations)
	return operations
}
