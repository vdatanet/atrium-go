package ports_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/vdatanet/atrium-go/internal/ports"
)

// The refusal 003 §3.6 asks for is an absence, and this is the file that
// asserts an absence.
//
// A library's collection type and its case sensitivity are frozen at creation.
// Changing either re-derives every identifier under the library and nothing
// stores the old ones to undo with, so the specification refuses the change —
// and the way this design refuses it is that **there is nothing to call**.
//
// That is why the assertions below are over the interface and not over the SQL,
// and why none of them calls anything and expects an error. A test that asserted
// an error would be asserting a refusal this design does not implement: it would
// pass over a build with a `SetCollectionType` that returned one, which is a
// method that can be reached, logged, retried and eventually worked around, and
// each of those is a path to the rewrite that has no undo. What this design
// offers instead is a caller that does not compile, and the observable form of
// that is the shape of the method set.
//
// The package is `ports_test` rather than `ports` for the same reason: the
// surface under test is the one another package can see.

// libraryStoreMethods is `ports.LibraryStore`'s whole method set, spelled out.
//
// It is a golden list and it is meant to be annoying. Adding a method to
// LibraryStore fails this test, and the failure is the point: 003 §3.6 makes
// three of a library's six fields unwritable after creation, so a new method on
// this interface is a question about whether it writes one of them, asked at
// the moment somebody could still answer it. Updating the list is the correct
// fix *after* that question has been answered, and never before.
//
// The signatures are here and not only the names, because the frozen columns
// travel in parameters: `RenameLibrary(ctx, id, name, folded string)` growing a
// fourth string would be a collection type in every way that matters and would
// keep its name.
var libraryStoreMethods = map[string]string{
	"CreateLibrary":       "func(context.Context, ports.Library) error",
	"Libraries":           "func(context.Context) ([]ports.Library, error)",
	"LibraryByFoldedName": "func(context.Context, string) (ports.Library, bool, error)",
	"RenameLibrary":       "func(context.Context, string, string, string) error",
	"ReplaceRoots":        "func(context.Context, string, []string) error",
	"RemoveLibrary":       "func(context.Context, string) error",
}

// TestTheLibraryStoreOffersExactlyTheSixMethodsAndNoMore is the golden half.
func TestTheLibraryStoreOffersExactlyTheSixMethodsAndNoMore(t *testing.T) {
	declared := methodSignatures(reflect.TypeOf((*ports.LibraryStore)(nil)).Elem())

	for name, want := range libraryStoreMethods {
		got, found := declared[name]
		if !found {
			t.Errorf("LibraryStore no longer declares %s", name)
			continue
		}
		if got != want {
			t.Errorf("LibraryStore.%s is %s, want %s — a parameter moved, and the frozen "+
				"columns of 003 §3.6 travel in parameters", name, got, want)
		}
	}
	for name, got := range declared {
		if _, expected := libraryStoreMethods[name]; !expected {
			t.Errorf("LibraryStore declares %s %s, which this list does not have. "+
				"003 §3.6 freezes a library's collection type and its case sensitivity at "+
				"creation and refuses a change by offering no way to make one: if the new "+
				"method writes either, it is the refusal being undone, and if it does not, "+
				"add it here and say so", name, got)
		}
	}
}

// TestNoMethodOfTheLibraryStoreCanCarryAFrozenColumn is the half that survives
// somebody updating the golden list above without reading it.
//
// It asserts over the parameter *types* rather than over the method set, and it
// catches exactly what a type can catch:
//
//   - `case_sensitive` is a Go `bool`, and it is the only bool anywhere in a
//     [ports.Library]. No method may take one. `SetCaseSensitive(ctx, id string,
//     on bool)` fails here whatever it is called.
//   - a whole [ports.Library] carries all three frozen fields at once, so a
//     method taking one can rewrite every one of them. Exactly one method may,
//     and it is the one that creates the library.
//
// What it cannot catch is honest to state: `collection_type` is a bare `string`,
// so `SetCollectionType(ctx, id, collectionType string)` is type-identical to a
// rename that lost an argument, and reflection over an interface does not carry
// parameter names. The golden list above is what catches that one, which is why
// both tests are here and neither is redundant.
func TestNoMethodOfTheLibraryStoreCanCarryAFrozenColumn(t *testing.T) {
	libraryStore := reflect.TypeOf((*ports.LibraryStore)(nil)).Elem()
	library := reflect.TypeOf(ports.Library{})

	for i := range libraryStore.NumMethod() {
		method := libraryStore.Method(i)
		for parameter := range method.Type.NumIn() {
			switch in := method.Type.In(parameter); {
			case in.Kind() == reflect.Bool:
				t.Errorf("LibraryStore.%s takes a bool. `case_sensitive` is the only bool a "+
					"library has, and 003 §3.6 freezes it at creation: changing it rewrites "+
					"every identifier under the library and nothing stores the old ones",
					method.Name)
			case in == library && method.Name != "CreateLibrary":
				t.Errorf("LibraryStore.%s takes a whole Library, which carries all three of "+
					"003 §3.6's frozen fields. Only CreateLibrary may", method.Name)
			}
		}
	}
}

// TestALibraryCarriesExactlyOneBoolean is the premise the test above rests on,
// asserted rather than assumed.
//
// "No method takes a bool" only refuses a case-sensitivity setter while
// `CaseSensitive` is the one boolean a library has. A second bool field added to
// the record would make the rule catch a setter for the wrong reason and, worse,
// would make a reader believe the rule still says what it said.
func TestALibraryCarriesExactlyOneBoolean(t *testing.T) {
	library := reflect.TypeOf(ports.Library{})

	var booleans []string
	for i := range library.NumField() {
		if library.Field(i).Type.Kind() == reflect.Bool {
			booleans = append(booleans, library.Field(i).Name)
		}
	}
	if !slices.Equal(booleans, []string{"CaseSensitive"}) {
		t.Errorf("a Library's boolean fields are %v, want exactly [CaseSensitive] — "+
			"TestNoMethodOfTheLibraryStoreCanCarryAFrozenColumn refuses a bool parameter on "+
			"the strength of there being only this one", booleans)
	}
}

// methodSignatures reads an interface's method set as name to signature.
func methodSignatures(declared reflect.Type) map[string]string {
	signatures := map[string]string{}
	for i := range declared.NumMethod() {
		method := declared.Method(i)
		signatures[method.Name] = method.Type.String()
	}
	return signatures
}
