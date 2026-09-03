package conformance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// This file is the half of the two cross-cutting L1 sweeps (spec 6,
// conformance L1) that walks response *bytes*. Its twin walks Go types and
// lives in internal/httpapi/sweep_test.go.
//
// # Why the sweeps are split, and which half this is
//
// architecture 8 puts "the two reflection sweeps" here. architecture 3 forbids
// this package from importing anything under internal/, and
// tools/check_conformance_imports enforces it. A reflection sweep over this
// project's response models needs those models, so the reflection half cannot
// live here and does not; the import rule is the one honoured, and the reason
// is in doc.go — everything a test here knows is something a client could have
// known.
//
// What is left for this half is not a consolation prize. It is the half that
// reaches the rule conformance L1 corrects itself on: **a date is recognised by
// its value, not by its name.** Of nine date-valued fields observed in the
// reference, three — DateCreated, DateLastMediaAdded and LastPlaybackCheckIn —
// do not end in Date `[probe: tools/probe_wire_format, Jellyfin 10.11.11,
// 2026-09-02]`, so a rule keyed on the field name checks six of the nine. A Go
// type cannot answer "is this value a date". These bytes can.
//
// It is also the only half that sees what a body actually contains rather than
// what its declaration allows: 001's two empty arrays are []any, and an
// interface has no fields to reflect over. Whatever a later feature puts in
// one is swept here or nowhere.
//
// # Where the two halves could leave a gap
//
// A field is swept by neither if it is in no registered response model *and*
// in no body this file requests. The first clause is closed in the other half,
// which checks its registry against the router. **The second is open, and it is
// this file's weakness: sweptResponses below is a hand-written list.** T20 is
// the check that the router serves exactly surface.yaml's implemented rows;
// nothing yet ties this list to that one. Until something does, a feature that
// adds a route adds a row here — and its model's *names* are still swept by the
// other half meanwhile, so what would go unswept is the values, not the schema.
//
// architecture 8 carries the amendment, dated.

// sweptResponse is one request whose body must survive both sweeps.
type sweptResponse struct {
	name   string
	method string
	path   string

	// accept is the Accept header, empty for a client that asks for nothing in
	// particular. Only the two PascalCase profiles appear below: the CamelCase
	// profile writes camelCase property names *by contract* (spec 3.0.2), so
	// sweeping it would be asserting the opposite of what that profile is for.
	// It has its own test at the end of this file, where the sweep firing is
	// the assertion rather than the failure.
	accept string
}

// sweptResponses is every body feature 001 puts on the wire, under each content
// profile whose names are PascalCase.
//
// /System/Info is requested on a fresh installation, where first-time setup is
// outstanding and the route is therefore admitted without a credential
// (spec 3.2) — the only state 001 can reach it in.
var sweptResponses = []sweptResponse{
	{name: "the public system info", method: http.MethodGet, path: publicSystemInfoPath},
	{name: "the public system info, PascalCase", method: http.MethodGet, path: publicSystemInfoPath, accept: pascalCaseProfile},
	{name: "the system info", method: http.MethodGet, path: systemInfoPath},
	{name: "the system info, PascalCase", method: http.MethodGet, path: systemInfoPath, accept: pascalCaseProfile},
	{name: "a ping", method: http.MethodGet, path: pingPath},
	{name: "a ping, PascalCase", method: http.MethodGet, path: pingPath, accept: pascalCaseProfile},
	{name: "a posted ping", method: http.MethodPost, path: pingPath},
	{name: "a posted ping, PascalCase", method: http.MethodPost, path: pingPath, accept: pascalCaseProfile},
}

const (
	pascalCaseProfile = `application/json; profile="PascalCase"`
	camelCaseProfile  = `application/json; profile="CamelCase"`
)

// TestEveryResponseSweepsClean is the two sweeps, run over the wire.
//
// One server, every response 001 sends, both rules. It finds nothing today —
// 001's bodies are strings, booleans, one integer port and two empty arrays —
// which is the state a sweep is meant to be in and also the state in which it
// has proved nothing. The tests below it are what make that state mean
// something.
func TestEveryResponseSweepsClean(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, swept := range sweptResponses {
		t.Run(swept.name, func(t *testing.T) {
			var header http.Header
			if swept.accept != "" {
				header = http.Header{"Accept": {swept.accept}}
			}
			got := server.do(t, swept.method, swept.path, goldenHost, header)

			if got.status != http.StatusOK {
				t.Fatalf("%s %s: status %d, want %d\nbody: %s",
					swept.method, swept.path, got.status, http.StatusOK, got.body)
			}

			for _, found := range sweepBody(t, got.body) {
				t.Errorf("%s %s: %s", swept.method, swept.path, found)
			}
		})
	}
}

// TestTheSweepReachedEveryRouteThisFeatureServes is the coverage half of the
// list above.
//
// A hand-written list of requests is worth exactly as much as its completeness,
// and the failure it invites is a route added and not swept. This cannot check
// the router — that is T20's, and this package has no way to enumerate what is
// registered — so it checks the next best thing: that every path and method
// feature 001 declares appears in the list at least once. A fifth route added
// without a row here still slips past, and that is said plainly in this file's
// opening comment rather than hidden behind a passing test.
func TestTheSweepReachedEveryRouteThisFeatureServes(t *testing.T) {
	t.Parallel()

	wanted := []string{
		http.MethodGet + " " + publicSystemInfoPath,
		http.MethodGet + " " + systemInfoPath,
		http.MethodGet + " " + pingPath,
		http.MethodPost + " " + pingPath,
	}

	swept := map[string]bool{}
	for _, response := range sweptResponses {
		swept[response.method+" "+response.path] = true
	}

	for _, route := range wanted {
		if !swept[route] {
			t.Errorf("no swept response covers %s, so nothing sweeps the values it sends", route)
		}
	}
}

// TestTheCasingSweepCatchesACamelCaseProperty is one half of the failure proof
// the *Verified by* line asks for: a sweep that has never failed has proved
// nothing.
//
// The body is built from a model declared in this file, which is the strong
// form of "it cannot leak into the served surface": a _test.go file is not part
// of any package the server is built from, and this package is imported by
// nothing at all — the server it talks to is a separate process built from
// cmd/atrium. There is no path by which either type below reaches a response.
func TestTheCasingSweepCatchesACamelCaseProperty(t *testing.T) {
	t.Parallel()

	body := marshalTestOnlyModel(t, modelWithACamelCaseProperty{
		ServerName:   "atrium",
		LocalAddress: "http://192.168.1.20:8096",
		Nested:       nestedModelWithACamelCaseProperty{Version: "10.11.11"},
		Items: []nestedModelWithACamelCaseProperty{
			{Version: "10.11.11"},
		},
	})

	// Three findings: one at the top level, one an object deep, and one inside
	// an array element. The third is there because a sweep that walked objects
	// and not arrays would report two and look correct.
	assertSweptExactly(t, body, "/localAddress", "/Nested/version", "/Items/0/version")
}

// TestTheUnitSweepCatchesAThreeDigitDate is the other half, and it is the one
// the corrected rule is really about.
//
// The model carries two dates and neither is caught by its name: DateCreated
// does not end in Date, and Added says nothing at all. Both are caught by their
// **values**. behaviours 1.2 is the rule — seven fractional digits and a Z —
// and the reference's own three- and six-digit values on LastPlayedDate and
// LastActivityDate are why this is a statement about Atrium rather than a
// shared fact: an L3 comparison of those two fields will differ, and this sweep
// is still right about what this server may send.
func TestTheUnitSweepCatchesAThreeDigitDate(t *testing.T) {
	t.Parallel()

	body := marshalTestOnlyModel(t, modelWithABadlySpelledDate{
		DateCreated:  "2025-06-19T00:00:00.000Z",
		Added:        "2025-06-19T00:00:00Z",
		PremiereDate: "2025-06-19T00:00:00.0000000Z",
	})

	assertSweptExactly(t, body, "DateCreated", "Added")
}

// TestTheUnitSweepCatchesFractionalTicks is the tick half of the same rule.
// behaviours 1.3 makes a tick a whole 100-nanosecond interval; a JSON number
// with a fractional part is a conversion somebody forgot at a boundary.
func TestTheUnitSweepCatchesFractionalTicks(t *testing.T) {
	t.Parallel()

	// Written as raw bytes rather than marshalled from a struct, because Go's
	// encoder writes a float64 with a zero fraction as an integer and would
	// hide the very thing under test.
	body := []byte(`{"RunTimeTicks":9000000000.5,"StartTicks":9000000000}`)

	assertSweptExactly(t, body, "RunTimeTicks")
}

// assertSweptExactly runs both sweeps over one body and requires exactly one
// finding per named location, and no others.
//
// The count is asserted as well as the names, because a sweep that reported
// every field would satisfy "it named the bad one" and prove nothing. The names
// are matched without regard to order: encoding/json decodes an object into a
// map, so the order the sweep visits keys in is lexical rather than the
// document's, and asserting on it would be asserting on the decoder.
func assertSweptExactly(t *testing.T, body []byte, want ...string) {
	t.Helper()

	found := sweepBody(t, body)
	if len(found) != len(want) {
		t.Fatalf("the sweeps reported %d findings on %s, want %d:\n%s",
			len(found), body, len(want), strings.Join(found, "\n"))
	}

	report := strings.Join(found, "\n")
	for _, name := range want {
		matches := 0
		for _, finding := range found {
			if strings.Contains(finding, name) {
				matches++
			}
		}
		if matches != 1 {
			t.Errorf("%d of the findings name %q, want 1:\n%s", matches, name, report)
		}
	}
}

// TestADeclaredDictionarysKeysAreNotSweptAsPropertyNames exercises the guard
// conformance L1 records the cost of.
//
// A dictionary's keys are data. Treating them as property names reported **688
// of 899 keys** as casing failures in one run against the reference, because
// ImageBlurHashes is keyed by image tag `[probe: tools/probe_wire_format,
// Jellyfin 10.11.11, 2026-09-02]`. 001 sends no dictionary, so dictionaryPointers
// is empty and the guard would otherwise be code no case reaches — which is the
// same shape as the guards T4 and T15 each deleted after a surviving mutation.
// So the set is stated here instead, and the body carries both a data key and a
// real property name beside it, so that the guard is proven to be narrow rather
// than merely present.
func TestADeclaredDictionarysKeysAreNotSweptAsPropertyNames(t *testing.T) {
	t.Parallel()

	body := []byte(`{"ImageBlurHashes":{"a1b2c3":"W04,","primary":"W04,"},"itemId":"3f9c"}`)

	found := sweepBodyWithDictionaries(t, body, map[string]bool{"/ImageBlurHashes": true})
	if len(found) != 1 {
		t.Fatalf("the casing sweep reported %d findings, want the one that is not a dictionary key:\n%s",
			len(found), strings.Join(found, "\n"))
	}
	if !strings.Contains(found[0], "itemId") {
		t.Errorf("the finding is %q and does not name %q", found[0], "itemId")
	}

	// And the guard is scoped: the same body with nothing declared reports the
	// key as well, which is what makes the declaration load-bearing.
	if undeclared := sweepBodyWithDictionaries(t, body, nil); len(undeclared) != 3 {
		t.Errorf("with no dictionary declared the sweep reported %d findings, want 3 — "+
			"the two data keys and the property name:\n%s",
			len(undeclared), strings.Join(undeclared, "\n"))
	}
}

// TestTheCasingSweepFiresOnTheCamelCaseProfile is the failure proof that needs
// no test-only model at all: it is a body this server really sends.
//
// spec 3.0.2 makes `profile="CamelCase"` a real content profile, and under it
// every property name of /System/Info/Public is camelCase by contract. So the
// sweep run over that response must report a finding for each of the seven —
// which is a live sweep firing on real bytes, and evidence that
// TestEveryResponseSweepsClean passes because the bodies are clean rather than
// because the sweep looked at nothing.
//
// It is also why that profile is absent from sweptResponses: sweeping it as a
// requirement would assert the opposite of what spec 3.0.2 says it is for.
func TestTheCasingSweepFiresOnTheCamelCaseProfile(t *testing.T) {
	t.Parallel()

	server := startServer(t)
	got := server.get(t, publicSystemInfoPath, goldenHost, http.Header{"Accept": {camelCaseProfile}})

	if got.status != http.StatusOK {
		t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusOK, got.body)
	}

	// Seven fields (spec 3.1), and every one of them renamed. Six begin with a
	// lower-case letter; Id becomes id.
	const fields = 7
	found := sweepBody(t, got.body)
	if len(found) != fields {
		t.Fatalf("the casing sweep reported %d findings on the CamelCase profile, want %d:\n%s\nbody: %s",
			len(found), fields, strings.Join(found, "\n"), got.body)
	}
}

// modelWithACamelCaseProperty exists only in this file. Its Go field names are
// PascalCase and read fine in review; the tags are where the mistake is really
// made, and where a body's property names really come from.
type modelWithACamelCaseProperty struct {
	ServerName   string
	LocalAddress string `json:"localAddress"`
	Nested       nestedModelWithACamelCaseProperty
	Items        []nestedModelWithACamelCaseProperty
}

// nestedModelWithACamelCaseProperty puts the second failure one level down, so
// that a sweep which only looked at the top level of a body would pass the test
// above with one finding instead of two.
type nestedModelWithACamelCaseProperty struct {
	Version string `json:"version"`
}

// modelWithABadlySpelledDate carries three dates: one spelled as the wire
// spells it, and two that are not. Neither of the two is named in a way the
// suffix rule would find.
type modelWithABadlySpelledDate struct {
	DateCreated  string
	Added        string
	PremiereDate string
}

// marshalTestOnlyModel serialises one of the models above into the bytes the
// sweep works on. It uses encoding/json directly rather than anything of this
// project's, because this package may not import internal/wire — which is the
// boundary doing its job: the sweep is being fed bytes, not a value.
func marshalTestOnlyModel(t *testing.T, model any) []byte {
	t.Helper()
	body, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("serialising a test-only model: %v", err)
	}
	return body
}

// dictionaryPointers names the places in a response body whose object *keys*
// are data rather than property names, as JSON Pointers to the containing
// object.
//
// It is empty, and it exists because the trap it guards has already been fallen
// into: a sweep that treated every JSON object key as a property name reported
// **688 of 899 keys** as casing failures in one run against the reference,
// because ImageBlurHashes is keyed by image tag (conformance L1)
// `[probe: tools/probe_wire_format, Jellyfin 10.11.11, 2026-09-02]`. No body
// feature 001 sends contains a dictionary, so the correct declaration today is
// an empty one — and the feature that first sends ImageBlurHashes adds
// "/ImageBlurHashes" here rather than discovering the rule from 688 failures.
var dictionaryPointers = map[string]bool{}

// wireDate is the one spelling of a date this server may send: seven fractional
// digits and a Z (behaviours 1.2).
//
// It is written out here rather than borrowed from internal/units, which is the
// import rule again and, here, a benefit: a change to the layout that package
// writes has to be made twice, and the second time is at the wire, where the
// contract is. A sweep that asked internal/units whether internal/units had
// written a date correctly would agree with it by construction.
var wireDate = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{7}Z$`)

// dateShaped is what makes a string a candidate for the rule above: a full ISO
// date, alone or followed by a time.
//
// It is deliberately narrower than "starts with something date-like". A value
// that begins with a date and continues into prose — an album called
// "2001-01-01 Sessions" — must not be swept as a malformed date, and requiring
// either the whole string or a T and a clock is what keeps it out. A date-only
// value is included because it is exactly ten characters and is never a title;
// it is also a real failure, since behaviours 1.2's output form always carries
// a time.
var dateShaped = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}.*)?$`)

// sweepBody runs both cross-cutting sweeps over one response body and returns
// what they found.
//
// The body is decoded with UseNumber, so a number is still the bytes that were
// sent: 9000000000 and 9000000000.0 are the same float64 and a different
// response, and the tick rule is about which of the two arrived.
func sweepBody(t *testing.T, body []byte) []string {
	t.Helper()
	return sweepBodyWithDictionaries(t, body, dictionaryPointers)
}

// sweepBodyWithDictionaries is sweepBody with the declared dictionaries stated
// rather than taken from the package.
//
// The set is a parameter so that the guard can be exercised: 001 declares none,
// and a guard no case reaches is a guard that has proved nothing — which is the
// finding T4 and T15 each arrived at independently.
func sweepBodyWithDictionaries(t *testing.T, body []byte, dictionaries map[string]bool) []string {
	t.Helper()

	// A bare JSON string, which is what /System/Ping answers, decodes to a
	// string with no property name in it. That is not an empty sweep by
	// accident — it is the whole content of that response.
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()

	var document any
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("the body is not JSON: %v\nbody: %s", err, body)
	}

	var found []string
	sweepValue("", "", document, dictionaries, func(finding string) { found = append(found, finding) })
	return found
}

// sweepValue walks one decoded value, applying both rules.
//
// pointer is the JSON Pointer of the value, and name is the property name it
// arrived under — empty at the root and inside an array, because an array's
// elements share their parent's name.
func sweepValue(pointer, name string, value any, dictionaries map[string]bool, report func(string)) {
	switch v := value.(type) {
	case map[string]any:
		for _, key := range sortedObjectKeys(v) {
			child := pointer + "/" + escapePointerToken(key)
			if !dictionaries[pointer] && !isPascalCase(key) {
				report(fmt.Sprintf(
					"%s is the property name %q, which is not PascalCase (behaviours 1.1)",
					child, key))
			}
			sweepValue(child, key, v[key], dictionaries, report)
		}

	case []any:
		for i, element := range v {
			sweepValue(fmt.Sprintf("%s/%d", pointer, i), name, element, dictionaries, report)
		}

	case json.Number:
		if strings.HasSuffix(name, "Ticks") && !isWholeNumber(v.String()) {
			report(fmt.Sprintf(
				"%s is %s; ticks are whole 100-nanosecond intervals and serialise as an integer "+
					"(behaviours 1.3)", pointerOrRoot(pointer), v.String()))
		}

	case string:
		if dateShaped.MatchString(v) && !isWireDate(v) {
			report(fmt.Sprintf(
				"%s is the date %q; a date on this wire carries seven fractional digits and a Z "+
					"(behaviours 1.2)", pointerOrRoot(pointer), v))
		}
	}
}

// isWireDate reports whether a value is a date spelled the way this server must
// spell one.
//
// The shape is checked by the expression and the calendar by the parse: a
// pattern alone accepts 2025-02-30, and a parse alone accepts three fractional
// digits, because Go's fractional layout element matches any number of them.
func isWireDate(s string) bool {
	if !wireDate.MatchString(s) {
		return false
	}
	_, err := time.Parse("2006-01-02T15:04:05.0000000Z", s)
	return err == nil
}

// isWholeNumber reports whether a JSON number arrived as an integer. It is a
// question about the bytes, not about the value: 9000000000.0 parses to a whole
// number and is not one on the wire.
func isWholeNumber(number string) bool {
	return !strings.ContainsAny(number, ".eE")
}

// isPascalCase is the rule of spec 3.0.1: an upper-case letter, then letters
// and digits.
//
// It is a second implementation of the rule the model sweep applies, and that
// is the import boundary rather than an oversight. It is held to the same
// standard by the table beside it: the reference's own names include
// EnableIPv4, UICulture, Video3DFormat and Hdr10PlusPresentFlag, and a rule
// spelled "capital then lower-case letters, repeated" refuses three of them and
// then gets loosened by whoever meets it.
func isPascalCase(name string) bool {
	for i, r := range name {
		if i == 0 {
			if r < 'A' || r > 'Z' {
				return false
			}
			continue
		}
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		isDigit := r >= '0' && r <= '9'
		if !isLetter && !isDigit {
			return false
		}
	}
	return name != ""
}

// TestThePascalCaseRuleHoldsTheAwkwardNames keeps this copy of the rule from
// drifting away from the one the model sweep applies. The names are the ones
// the pinned document contains that a careless rule refuses, and the two
// spellings that must be refused.
func TestThePascalCaseRuleHoldsTheAwkwardNames(t *testing.T) {
	t.Parallel()

	for _, accepted := range []string{
		"EnableIPv4", "UICulture", "Video3DFormat", "Hdr10PlusPresentFlag", "ETag", "Id",
	} {
		if !isPascalCase(accepted) {
			t.Errorf("the rule refuses %q, which is a property name of the pinned document", accepted)
		}
	}

	for _, refused := range []string{
		"localAddress", "uiCulture", "run_time_ticks", "", "3D", "Package Name",
	} {
		if isPascalCase(refused) {
			t.Errorf("the rule accepts %q, which is not PascalCase", refused)
		}
	}
}

// TestTheDateRuleRecognisesADateByItsValue is the correction conformance L1
// makes to its own sentence, written as a test.
//
// The left column is what must be recognised as a date and judged; the right is
// what must not be touched, because a sweep that flagged an ordinary string
// would be turned off by the first person who met it.
func TestTheDateRuleRecognisesADateByItsValue(t *testing.T) {
	t.Parallel()

	for _, wellSpelled := range []string{
		"2025-06-19T00:00:00.0000000Z",
		"0001-01-01T00:00:00.0000000Z", // the unset date the reference sends
	} {
		if !dateShaped.MatchString(wellSpelled) || !isWireDate(wellSpelled) {
			t.Errorf("%q is a date this server may send and the rule does not accept it", wellSpelled)
		}
	}

	for _, badlySpelled := range []string{
		"2025-06-19T00:00:00.000Z",    // three digits, as the reference sends on LastPlayedDate
		"2025-06-19T00:00:00.000000Z", // six, as it sends on LastActivityDate
		"2025-06-19T00:00:00Z",        // none
		"2025-06-19T00:00:00.0000000", // no zone
		"2025-06-19T02:00:00.0000000+02:00",
		"2025-06-19",
		"2025-02-30T00:00:00.0000000Z", // the right shape and not a day
	} {
		if !dateShaped.MatchString(badlySpelled) {
			t.Errorf("%q is a date and the sweep does not recognise it as one", badlySpelled)
			continue
		}
		if isWireDate(badlySpelled) {
			t.Errorf("%q is not a date this server may send and the rule accepts it", badlySpelled)
		}
	}

	for _, notADate := range []string{
		"atrium", "Jellyfin Server", "10.11.11", "/var/lib/atrium",
		"2001-01-01 Sessions", // an album, not a date
		"3f9c1a7e5b2d4e8091a6c3f70d5e2b14",
	} {
		if dateShaped.MatchString(notADate) {
			t.Errorf("the sweep reads %q as a date, and it is a value a library could really hold", notADate)
		}
	}
}

// sortedObjectKeys is the keys of a decoded object in the order they must be
// reported in.
//
// encoding/json decodes an object into a map and the wire order is gone by
// then, so this is the lexical order rather than the document's. That costs
// nothing here — the sweep reports every failing key whatever the order — and
// it is why a *key order* assertion cannot be built on this function.
// system_info_test.go's property-name assertion reads the bytes directly for
// exactly that reason.
func sortedObjectKeys(object map[string]any) []string {
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// pointerOrRoot names a location for a failure message.
func pointerOrRoot(pointer string) string {
	if pointer == "" {
		return "the body"
	}
	return pointer
}

// escapePointerToken is RFC 6901's escaping, so that a property name containing
// a slash does not make a pointer that reads as two.
func escapePointerToken(token string) string {
	token = strings.ReplaceAll(token, "~", "~0")
	return strings.ReplaceAll(token, "/", "~1")
}
