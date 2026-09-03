package conformance_test

import (
	"net/http"
	"testing"
)

// The three declared content types, as spec 3.0.2 and AC-9 describe them, on
// the two responses in this feature that have property names to convert.
//
// # Why this file exists
//
// AC-9 is proven in full in internal/wire, over a model declared in a test
// file, with a ResponseRecorder — and that proof is the strong one: it compares
// the plain and PascalCase bodies with each other as well as against a written
// expectation, it puts a dictionary beside a property so the two conversions
// can be told apart, and it goes to depth. Nothing here repeats it.
//
// What it does not reach is the criterion's subject. AC-9 is about *requests*
// to this feature's endpoints, and internal/wire has no endpoints; the handler
// tests that connect the two are recorder tests in internal/httpapi.
//
// **Measured, at the closing audit (T21):** making /System/Info/Public write
// with a constant wire.ProfilePlain instead of the negotiated profile left
// every test in this package green except TestTheCasingSweepFiresOnTheCamelCase
// Profile — which is the casing sweep's own failure proof, is named for AC-10's
// machinery, and asserts a count of findings rather than anything AC-9 says.
// Nothing at this boundary compared the two PascalCase answers, and nothing
// checked the echo on either /System/Info route.
// [measurement: mutation of internal/httpapi.SystemHandler.PublicInfo,
// 2026-09-03]
//
// # The Host is stated, for the same reason the golden states it
//
// LocalAddress is derived from the request (spec 3.4 tier 2), so two requests
// that differed in Host would differ in one field for a reason that has nothing
// to do with the profile.
const (
	plainType         = "application/json"
	plainContentType  = "application/json; charset=utf-8"
	pascalContentType = `application/json; profile="PascalCase"; charset=utf-8`
	camelContentType  = `application/json; profile="CamelCase"; charset=utf-8`
)

// profiledRoutes are the responses with property names in them. /System/Ping is
// a bare JSON string and has none, which is why its own echo test
// (TestPingEchoesTheProfileItWasAskedFor) can compare one golden across all
// three profiles and this one cannot.
var profiledRoutes = []string{publicSystemInfoPath, systemInfoPath}

// AC-9's first clause: "three names for two behaviours".
//
// Byte-identical bodies, and content types that are not identical — the second
// half matters as much as the first, because a server that ignored the profile
// entirely would satisfy the first half perfectly.
func TestThePlainTypeAndThePascalCaseProfileAnswerOneBodyUnderTwoContentTypes(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	for _, path := range profiledRoutes {
		t.Run(path, func(t *testing.T) {
			plain := server.get(t, path, goldenHost, http.Header{"Accept": {plainType}})
			pascal := server.get(t, path, goldenHost, http.Header{"Accept": {pascalCaseProfile}})

			for _, got := range []*response{plain, pascal} {
				if got.status != http.StatusOK {
					t.Fatalf("%s: status %d, want 200\n%s", path, got.status, got.body)
				}
			}

			if string(plain.body) != string(pascal.body) {
				t.Errorf("%s: the plain type and the PascalCase profile differ:\n plain %s\npascal %s",
					path, plain.body, pascal.body)
			}
			if got := plain.header.Get("Content-Type"); got != plainContentType {
				t.Errorf("%s under %q: Content-Type: got %q, want %q", path, plainType, got, plainContentType)
			}
			if got := pascal.header.Get("Content-Type"); got != pascalContentType {
				t.Errorf("%s under %q: Content-Type: got %q, want %q", path, pascalCaseProfile, got, pascalContentType)
			}
		})
	}
}

// AC-9's second clause: the same values, under camelCase property names, and a
// content type that says so.
//
// # What is asserted, and what is deliberately not
//
// The **conversion rule** is not asserted here. It is .NET's own — a leading
// run of capitals lowers all but the last of it — it is measured against the
// pinned document, and it lives in internal/wire with the tests that prove it.
// Re-deriving it in this package to normalise a name before comparing would be
// a second implementation of the rule, and a test that used it would pass on a
// server that had the same rule wrong in the same way.
//
// So the seven names of spec 3.1 are written out as literals, which is what the
// PascalCase key-order test beside this one does too, and the **values** are
// compared position by position as the raw bytes they arrived as: an eighth
// field, a dropped field, a reordering and a changed value are each a failure,
// and none of them needs a converter to see.
func TestTheCamelCaseProfileAnswersSpecThreeOnesValuesUnderConvertedNames(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	plain := server.get(t, publicSystemInfoPath, goldenHost, http.Header{"Accept": {plainType}})
	camel := server.get(t, publicSystemInfoPath, goldenHost, http.Header{"Accept": {camelCaseProfile}})

	for _, got := range []*response{plain, camel} {
		if got.status != http.StatusOK {
			t.Fatalf("%s: status %d, want 200\n%s", publicSystemInfoPath, got.status, got.body)
		}
	}

	if got := camel.header.Get("Content-Type"); got != camelContentType {
		t.Errorf("Content-Type: got %q, want %q", got, camelContentType)
	}

	want := []string{
		"localAddress",
		"serverName",
		"version",
		"productName",
		"operatingSystem",
		"id",
		"startupWizardCompleted",
	}
	names := propertyNames(t, camel.body)
	if !equalStrings(names, want) {
		t.Fatalf("property names under %q:\n got %v\nwant %v", camelCaseProfile, names, want)
	}

	// The bodies must not be the same bytes — the names moved — and every
	// value must be.
	if string(plain.body) == string(camel.body) {
		t.Errorf("the CamelCase profile answered the plain body unchanged: %s", camel.body)
	}

	plainNames := propertyNames(t, plain.body)
	plainFields := rawFields(t, plain.body)
	camelFields := rawFields(t, camel.body)
	if len(plainNames) != len(names) {
		t.Fatalf("the plain body carries %d fields and the camelCase one %d", len(plainNames), len(names))
	}
	for i, name := range plainNames {
		if string(camelFields[names[i]]) != string(plainFields[name]) {
			t.Errorf("%s/%s: the plain body says %s and the camelCase one %s",
				name, names[i], plainFields[name], camelFields[names[i]])
		}
	}
}

// The same clause on /System/Info, which is a superset of the body above and
// whose twenty-six names are not written out a second time.
//
// The names it does not restate are checked structurally instead: every name in
// the camelCase body begins with a lower-case letter where the PascalCase one
// begins with a capital, in the same order, and every value is the same bytes.
// That is weaker than the literal list above — it would not catch a wrong
// conversion of a *later* letter — and the reason it is enough here is that the
// route above is the one whose names are contract, while this one's are its
// superset and the conversion is the same code path.
func TestTheCamelCaseProfileConvertsTheSupersetsNamesAndNotItsValues(t *testing.T) {
	t.Parallel()

	server := startServer(t)

	plain := server.get(t, systemInfoPath, goldenHost, http.Header{"Accept": {plainType}})
	camel := server.get(t, systemInfoPath, goldenHost, http.Header{"Accept": {camelCaseProfile}})

	for _, got := range []*response{plain, camel} {
		if got.status != http.StatusOK {
			t.Fatalf("%s: status %d, want 200\n%s", systemInfoPath, got.status, got.body)
		}
	}

	if got := camel.header.Get("Content-Type"); got != camelContentType {
		t.Errorf("Content-Type: got %q, want %q", got, camelContentType)
	}

	plainNames := propertyNames(t, plain.body)
	camelNames := propertyNames(t, camel.body)
	if len(plainNames) != len(camelNames) {
		t.Fatalf("the plain body carries %d fields and the camelCase one %d:\n%v\n%v",
			len(plainNames), len(camelNames), plainNames, camelNames)
	}

	plainFields := rawFields(t, plain.body)
	camelFields := rawFields(t, camel.body)
	for i, name := range plainNames {
		converted := camelNames[i]
		if name[0] < 'A' || name[0] > 'Z' {
			t.Errorf("%s is not PascalCase in the plain body, which behaviours 1.1 forbids", name)
		}
		if converted[0] < 'a' || converted[0] > 'z' {
			t.Errorf("%s answered %q under the CamelCase profile, which is not a converted name", name, converted)
		}
		if name[1:] != converted[1:] {
			t.Errorf("%s answered %q under the CamelCase profile; only the leading run may move", name, converted)
		}
		if string(camelFields[converted]) != string(plainFields[name]) {
			t.Errorf("%s/%s: the plain body says %s and the camelCase one %s",
				name, converted, plainFields[name], camelFields[converted])
		}
	}
}
