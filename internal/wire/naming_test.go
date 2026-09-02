package wire_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/units"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// The models the table below is written from. They are named types rather than
// anonymous ones so that a row can nest one inside another, which is what
// "property names at every depth" needs to be shown on.
type (
	// leaf is one property name and one dictionary beside it, so that a nested
	// object can be checked for both halves of the rule at once.
	leaf struct {
		DeepValue string
		ImageTags map[string]string
	}

	// identified carries the two names behaviours 1.13 names: the ordinary one
	// and the one the wrong rule gets wrong.
	identified struct {
		Id        string
		UICulture string
	}

	// optional is the shape ADR-0002 gives an absent value: a pointer, never
	// omitempty on a non-pointer.
	optional struct {
		Present *string
		Absent  *string
	}

	// dated is the one model here whose field writes its own JSON.
	dated struct {
		DateCreated units.Time
	}
)

// writeUnder returns the bytes a value reaches the client as under one policy.
//
// Every assertion below is on those bytes (Principle VIII). A property name
// survives a round trip through a decoder in whatever case it arrived, so a
// test that unmarshalled would agree with a server that had converted nothing.
func writeUnder(t *testing.T, v any, profile wire.Profile) string {
	t.Helper()

	recorder := httptest.NewRecorder()
	if err := wire.Write(recorder, http.StatusOK, v, profile); err != nil {
		t.Fatalf("Write(...) = %v, want no error", err)
	}
	return recorder.Body.String()
}

// TestWriteConvertsPropertyNamesAndLeavesDictionaryKeys is spec 3.0.2's rule,
// read as bytes: names at every depth, keys never.
//
// Each row asserts both policies over the same value, which is the other half
// of AC-9 — the two answers carry the same values and differ only in the names
// they are written under.
func TestWriteConvertsPropertyNamesAndLeavesDictionaryKeys(t *testing.T) {
	value := "v"

	cases := []struct {
		name   string
		value  any
		pascal string
		camel  string
	}{
		{
			name:   "the ordinary name, and the one the wrong rule gets wrong",
			value:  identified{Id: "3f9c", UICulture: "en-US"},
			pascal: `{"Id":"3f9c","UICulture":"en-US"}`,
			camel:  `{"id":"3f9c","uiCulture":"en-US"}`,
		},
		{
			// The name behaviours 1.13 calls out as the reason the wrong rule
			// survives a spot check: both rules answer `eTag`.
			name:   "a leading run the two rules agree on",
			value:  struct{ ETag string }{ETag: "W/1"},
			pascal: `{"ETag":"W/1"}`,
			camel:  `{"eTag":"W/1"}`,
		},
		{
			name: "a nested object's names, converted at depth",
			value: struct {
				ServerName string
				Nested     leaf
			}{ServerName: "atrium", Nested: leaf{DeepValue: "d"}},
			pascal: `{"ServerName":"atrium","Nested":{"DeepValue":"d","ImageTags":null}}`,
			camel:  `{"serverName":"atrium","nested":{"deepValue":"d","imageTags":null}}`,
		},
		{
			// The branch the whole design is for. `ProviderIds` is a property
			// name and moves; `Imdb` and `Tmdb` are dictionary keys and do not.
			name: "a map-valued field whose keys come through untouched",
			value: struct{ ProviderIds map[string]string }{
				ProviderIds: map[string]string{"Imdb": "tt1", "Tmdb": "2"},
			},
			pascal: `{"ProviderIds":{"Imdb":"tt1","Tmdb":"2"}}`,
			camel:  `{"providerIds":{"Imdb":"tt1","Tmdb":"2"}}`,
		},
		{
			// The row that says the rule is about where a name came from and
			// not about how it is spelled. These keys are spelled exactly like
			// the property names of the first row and must not move.
			name: "dictionary keys spelled like property names",
			value: struct{ ProviderIds map[string]string }{
				ProviderIds: map[string]string{"Id": "x", "UICulture": "y"},
			},
			pascal: `{"ProviderIds":{"Id":"x","UICulture":"y"}}`,
			camel:  `{"providerIds":{"Id":"x","UICulture":"y"}}`,
		},
		{
			name:   "a dictionary at the top of the body",
			value:  map[string]string{"Id": "x"},
			pascal: `{"Id":"x"}`,
			camel:  `{"Id":"x"}`,
		},
		{
			name: "property names inside a dictionary's values",
			value: map[string]leaf{
				"Primary": {DeepValue: "d", ImageTags: map[string]string{"Backdrop": "b"}},
			},
			pascal: `{"Primary":{"DeepValue":"d","ImageTags":{"Backdrop":"b"}}}`,
			camel:  `{"Primary":{"deepValue":"d","imageTags":{"Backdrop":"b"}}}`,
		},
		{
			name:   "property names inside a list",
			value:  struct{ Items []identified }{Items: []identified{{Id: "a"}, {Id: "b"}}},
			pascal: `{"Items":[{"Id":"a","UICulture":""},{"Id":"b","UICulture":""}]}`,
			camel:  `{"items":[{"id":"a","uiCulture":""},{"id":"b","uiCulture":""}]}`,
		},
		{
			name:   "an optional value that is there and one that is not",
			value:  optional{Present: &value},
			pascal: `{"Present":"v","Absent":null}`,
			camel:  `{"present":"v","absent":null}`,
		},
		{
			name:   "a field that writes its own JSON",
			value:  dated{DateCreated: units.At(time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC))},
			pascal: `{"DateCreated":"2026-09-03T12:00:00.0000000Z"}`,
			camel:  `{"dateCreated":"2026-09-03T12:00:00.0000000Z"}`,
		},
		{
			// The escape pass runs over the renamed document, so it reaches a
			// dictionary key the naming policy deliberately did not touch.
			name: "an escaped dictionary key, escaped but not converted",
			value: struct{ ImageTags map[string]string }{
				ImageTags: map[string]string{"Clé": "é"},
			},
			pascal: `{"ImageTags":{"Cl\u00E9":"\u00E9"}}`,
			camel:  `{"imageTags":{"Cl\u00E9":"\u00E9"}}`,
		},
		{
			name:   "an empty object",
			value:  struct{ Nested struct{} }{},
			pascal: `{"Nested":{}}`,
			camel:  `{"nested":{}}`,
		},
		{
			name: "an empty dictionary and an empty list",
			value: struct {
				ImageTags map[string]string
				Items     []string
			}{ImageTags: map[string]string{}, Items: []string{}},
			pascal: `{"ImageTags":{},"Items":[]}`,
			camel:  `{"imageTags":{},"items":[]}`,
		},
		{
			// A property name that has to be escaped after it is converted, which
			// is what fixes the order of the two passes: renamed first, from the
			// encoder's own bytes, and escaped last, over the finished document.
			// Reverse them and this name reaches the client as raw UTF-8.
			name: "a property name that is not ASCII",
			value: struct {
				Cafe string `json:"Café"`
			}{Cafe: "x"},
			pascal: `{"Caf\u00E9":"x"}`,
			camel:  `{"caf\u00E9":"x"}`,
		},
		{
			name: "a name the tag renames rather than the field",
			value: struct {
				Ignored string `json:"UICulture"`
			}{Ignored: "en-US"},
			pascal: `{"UICulture":"en-US"}`,
			camel:  `{"uiCulture":"en-US"}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := writeUnder(t, c.value, wire.ProfilePlain); got != c.pascal {
				t.Errorf("PascalCase body =\n\t%s\nwant\n\t%s", got, c.pascal)
			}
			if got := writeUnder(t, c.value, wire.ProfileCamel); got != c.camel {
				t.Errorf("camelCase body =\n\t%s\nwant\n\t%s", got, c.camel)
			}
		})
	}
}

// TestWriteConvertsPromotedPropertyNames covers the shape 001's own models are
// most likely to take next: spec 3.2 describes `/System/Info` as a superset of
// `/System/Info/Public`, and the natural way to write that in Go is to embed
// the smaller model in the larger one.
//
// encoding/json flattens an embedded struct into its parent, so the walk has to
// flatten it the same way to know which field a promoted name came from.
func TestWriteConvertsPromotedPropertyNames(t *testing.T) {
	// Public is embedded under an exported name and hidden under an unexported
	// one. encoding/json promotes the exported fields of both, so both have to be
	// flattened here — the second is not an exotic case, it is what an embedded
	// helper type looks like.
	type Public struct {
		Id          string
		ProviderIds map[string]string
	}
	type hidden struct {
		ImageTags map[string]string
	}
	type full struct {
		Public
		hidden
		OperatingSystem string
	}

	value := full{
		Public: Public{Id: "3f9c", ProviderIds: map[string]string{"Imdb": "tt1"}},
		hidden: hidden{ImageTags: map[string]string{"Primary": "p"}},
	}

	// The promoted dictionary is what makes this test about the flattening and
	// not about the two names beside it. A walk that did not flatten would still
	// convert `Id`, because every key of a struct's object is a property name —
	// but it could not say what `ProviderIds` was written from, and the keys
	// under it are the ones that must not move.
	if got, want := writeUnder(t, value, wire.ProfilePlain),
		`{"Id":"3f9c","ProviderIds":{"Imdb":"tt1"},"ImageTags":{"Primary":"p"},`+
			`"OperatingSystem":""}`; got != want {
		t.Errorf("PascalCase body =\n\t%s\nwant\n\t%s", got, want)
	}
	if got, want := writeUnder(t, value, wire.ProfileCamel),
		`{"id":"3f9c","providerIds":{"Imdb":"tt1"},"imageTags":{"Primary":"p"},`+
			`"operatingSystem":""}`; got != want {
		t.Errorf("camelCase body =\n\t%s\nwant\n\t%s", got, want)
	}
}

// TestWriteRefusesAContentProfileItDoesNotHave is the other half of T5's
// argument for leaving NamingCamel undeclared until there was a policy behind
// it, moved up one level to where a caller now chooses.
//
// A Profile this package does not know must not fall through to the plain type:
// that is exactly the answer behaviours 1.13 says leaves a camelCase client
// with an empty object, and it would be invisible. The refusal one level down —
// a Naming marshal does not know — is asserted in the internal test, because
// negotiation is the only thing that produces a Naming now.
func TestWriteRefusesAContentProfileItDoesNotHave(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := wire.Write(recorder, http.StatusOK, identified{Id: "3f9c"}, wire.Profile(7))

	if err == nil {
		t.Fatalf("Write(..., Profile(7)) = nil, want an error")
	}
	if got := recorder.Body.Len(); got != 0 {
		t.Errorf("body = %q, want nothing written", recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want it unset", got)
	}
}

// TestWriteRefusesAnObjectItCannotAccountFor covers the walk's own failure
// mode, which is a refusal rather than a copy.
//
// A dictionary keyed by something other than a string cannot have its keys read
// back to the values they came from, so a value under one that is an interface
// has no type to walk. Nothing in this API has that shape. What the test is
// about is the choice: a body that converted some of its names and not others
// would be a wrong answer nobody could see, and a caller that gets this error
// can still send a refusal, because nothing has been written.
func TestWriteRefusesAnObjectItCannotAccountFor(t *testing.T) {
	recorder := httptest.NewRecorder()

	err := wire.Write(recorder, http.StatusOK,
		map[int]any{1: identified{Id: "3f9c"}}, wire.ProfileCamel)

	if err == nil {
		t.Fatalf("Write(...) = nil, want the walk to refuse a value it cannot type")
	}
	if got := recorder.Body.Len(); got != 0 {
		t.Errorf("body = %q, want nothing written", recorder.Body.String())
	}

	// The same value under the policy that renames nothing is answerable, which
	// is what makes the refusal above about the conversion and not about the
	// value.
	if got, want := writeUnder(t, map[int]any{1: identified{Id: "3f9c"}}, wire.ProfilePlain),
		`{"1":{"Id":"3f9c","UICulture":""}}`; got != want {
		t.Errorf("PascalCase body =\n\t%s\nwant\n\t%s", got, want)
	}
}

// ownJSON writes its own object, the way a custom converter does on the
// reference's side.
type ownJSON struct{}

func (ownJSON) MarshalJSON() ([]byte, error) {
	return []byte(`{"WrittenByHand":1}`), nil
}

// TestWriteLeavesAValueThatWritesItsOwnJSON records the walk's one seam, so
// that it is a decision rather than a surprise.
//
// A type that writes its own JSON writes its own names with it, and this
// package does not rename them. That is the reference's behaviour too: its
// naming policy is applied by the object converter from a type's property
// metadata, and a converter that writes members itself never consults it
// `[source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:34-45,55-58 @ v10.11.11]`.
//
// The consequence for whoever writes the next model: a MarshalJSON that emits
// an object opts that object out of the camelCase profile. units.Time is safe
// because it writes a string; anything that writes members has to write them
// under both policies itself, or not write them.
func TestWriteLeavesAValueThatWritesItsOwnJSON(t *testing.T) {
	value := struct{ Nested ownJSON }{}

	if got, want := writeUnder(t, value, wire.ProfileCamel),
		`{"nested":{"WrittenByHand":1}}`; got != want {
		t.Errorf("camelCase body =\n\t%s\nwant\n\t%s", got, want)
	}
}
