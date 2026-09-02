package surface_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// TestTheTableCarriesEveryRowOfTheDocument is the count the task is written
// against: 59 endpoints, and the levels surface.yaml declares. The numbers are
// read off the document itself
// `[source: docs/compatibility/surface.yaml]` and CLAUDE.md's own summary of
// it — "1 x L1, 50 x L2, 8 x L3" — so a row added or removed without a spec
// saying so fails here.
//
// 59 rows share 51 paths, because a path served by two methods is two rows and
// one path. That difference is the whole reason `Allow` is computed from the
// path rather than from the matched route (spec 3.6).
func TestTheTableCarriesEveryRowOfTheDocument(t *testing.T) {
	table := surface.V1()

	if got, want := table.Len(), 59; got != want {
		t.Errorf("Len() = %d, want %d", got, want)
	}
	if got, want := len(table.Endpoints()), 59; got != want {
		t.Errorf("len(Endpoints()) = %d, want %d", got, want)
	}
	if got, want := len(table.Paths()), 51; got != want {
		t.Errorf("len(Paths()) = %d, want %d", got, want)
	}

	levels := map[surface.Level]int{}
	features := map[string]int{}
	for _, e := range table.Endpoints() {
		levels[e.Level]++
		features[e.Feature]++
	}
	wantLevels := map[surface.Level]int{surface.L1: 1, surface.L2: 50, surface.L3: 8}
	if !reflect.DeepEqual(levels, wantLevels) {
		t.Errorf("levels = %v, want %v", levels, wantLevels)
	}
	wantFeatures := map[string]int{
		"001": 4, "002": 7, "004": 1, "005": 17,
		"006": 2, "007": 7, "008": 11, "009": 7, "011": 3,
	}
	if !reflect.DeepEqual(features, wantFeatures) {
		t.Errorf("rows per feature = %v, want %v", features, wantFeatures)
	}

	if got, want := table.Reference().OpenAPIVersion, "10.11.11"; got != want {
		t.Errorf("Reference().OpenAPIVersion = %q, want %q", got, want)
	}
	if got, want := table.Reference().SourceTag, "v10.11.11"; got != want {
		t.Errorf("Reference().SourceTag = %q, want %q", got, want)
	}
}

// TestFeature001OwnsFourRows names them, because the four routes this feature
// serves are the ones T11 will register and T20 will assert are exactly what
// the server exposes. Each field is compared, not only the operation: a row
// whose level quietly dropped from L3 to L2 would be a promise this repository
// stopped keeping without anything failing.
func TestFeature001OwnsFourRows(t *testing.T) {
	want := []surface.Endpoint{
		{
			Path:      "/System/Info/Public",
			Method:    "GET",
			Operation: "GetPublicSystemInfo",
			Consumers: []string{"music-client", "video-client"},
			Feature:   "001",
			Level:     surface.L3,
		},
		{
			Path:      "/System/Info",
			Method:    "GET",
			Operation: "GetSystemInfo",
			Consumers: []string{"music-client"},
			Feature:   "001",
			Level:     surface.L2,
		},
		{
			Path:      "/System/Ping",
			Method:    "GET",
			Operation: "GetPingSystem",
			Feature:   "001",
			Level:     surface.L2,
		},
		{
			Path:      "/System/Ping",
			Method:    "POST",
			Operation: "PostPingSystem",
			Feature:   "001",
			Level:     surface.L2,
		},
	}

	if got := surface.V1().ForFeature("001"); !reflect.DeepEqual(got, want) {
		t.Errorf("ForFeature(\"001\") =\n%+v\nwant\n%+v", got, want)
	}

	for _, e := range want {
		got, ok := surface.V1().Lookup(e.Method, e.Path)
		if !ok {
			t.Fatalf("Lookup(%q, %q) found nothing", e.Method, e.Path)
		}
		if !reflect.DeepEqual(got, e) {
			t.Errorf("Lookup(%q, %q) = %+v, want %+v", e.Method, e.Path, got, e)
		}
	}
}

// TestMethodsAreEveryMethodThePathHas is the shape T11 needs: spec 3.6 says
// `Allow` lists every method the *path* has, "which is not the same as every
// method the matched route has". /System/Ping is the case that tells the two
// apart, and the order is the header's own — alphabetical.
func TestMethodsAreEveryMethodThePathHas(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{path: "/System/Ping", want: []string{"GET", "POST"}},
		{path: "/System/Info", want: []string{"GET"}},
		{path: "/System/Info/Public", want: []string{"GET"}},
		{path: "/Items/{itemId}", want: []string{"DELETE", "GET", "POST"}},
		{path: "/Playlists/{playlistId}/Items", want: []string{"DELETE", "GET", "POST"}},

		// The lookup is exact on purpose: folding a client's spelling is
		// canonicalisation's job (T9), and answering here would hide from that
		// middleware the fact that it had not run.
		{path: "/system/ping", want: nil},
		{path: "/System/Ping/", want: nil},
		{path: "/Nothing/Here", want: nil},
	}

	for _, c := range cases {
		t.Run(c.path, func(t *testing.T) {
			if got := surface.V1().Methods(c.path); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Methods(%q) = %v, want %v", c.path, got, c.want)
			}
		})
	}
}

// TestEveryPathIsReachableFromMethods checks the two indexes agree: every path
// Paths() reports has at least one method, and every row's method is among the
// methods of its own path. The table is the single source three consumers read
// (plan 3), so an index that disagreed with the rows would give two of them
// different answers.
func TestEveryPathIsReachableFromMethods(t *testing.T) {
	table := surface.V1()

	seen := map[string]bool{}
	for _, path := range table.Paths() {
		if seen[path] {
			t.Errorf("Paths() reports %q twice", path)
		}
		seen[path] = true
		if len(table.Methods(path)) == 0 {
			t.Errorf("Methods(%q) is empty, but the path is in Paths()", path)
		}
	}

	for _, e := range table.Endpoints() {
		if !seen[e.Path] {
			t.Errorf("%s %s has no entry in Paths()", e.Method, e.Path)
		}
		found := false
		for _, m := range table.Methods(e.Path) {
			found = found || m == e.Method
		}
		if !found {
			t.Errorf("Methods(%q) does not contain %s", e.Path, e.Method)
		}
	}
}

// TestTheTableHandsOutCopies keeps the three consumers from being able to
// change what the other two read. It is cheap to get wrong: a slice returned
// straight out of the map is the map's own backing array.
func TestTheTableHandsOutCopies(t *testing.T) {
	table := surface.V1()

	methods := table.Methods("/System/Ping")
	methods[0] = "PATCH"
	if got := table.Methods("/System/Ping"); got[0] != "GET" {
		t.Errorf("Methods is not a copy: after writing to it the table says %v", got)
	}

	public, _ := table.Lookup("GET", "/System/Info/Public")
	public.Consumers[0] = "nobody"
	again, _ := table.Lookup("GET", "/System/Info/Public")
	if again.Consumers[0] != "music-client" {
		t.Errorf("Endpoint.Consumers is not a copy: the table now says %v", again.Consumers)
	}

	paths := table.Paths()
	paths[0] = "/elsewhere"
	if got := table.Paths(); got[0] == "/elsewhere" {
		t.Error("Paths is not a copy")
	}
}

// TestTheBaselineFixtureLoads is what makes the two rejection tests below mean
// anything. Both fixtures are this one with a single line changed, so a
// refusal is evidence about that line only if this one is accepted.
func TestTheBaselineFixtureLoads(t *testing.T) {
	table := loadFixture(t, "valid.yaml")

	if got, want := table.Len(), 2; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}
	if got, want := table.Methods("/System/Ping"), []string{"GET", "POST"}; !reflect.DeepEqual(got, want) {
		t.Errorf("Methods = %v, want %v", got, want)
	}
	if got := table.Endpoints()[0].Consumers; got != nil {
		t.Errorf("an empty consumers list loaded as %v, want nil", got)
	}
	if got, want := table.Endpoints()[1].Consumers, []string{"music-client"}; !reflect.DeepEqual(got, want) {
		t.Errorf("consumers = %v, want %v", got, want)
	}
}

// TestLoadRefusesADuplicateRouteAndAnUnknownLevel is the task's rejection
// pair. A route declared twice is two handlers for one request with no rule
// for choosing, and a level nobody defined is a row with no proof obligation —
// both are documents this loader must not hand back as if they were tables.
func TestLoadRefusesADuplicateRouteAndAnUnknownLevel(t *testing.T) {
	cases := []struct {
		fixture string
		want    string
	}{
		{fixture: "duplicate-route.yaml", want: "GET /System/Ping is already declared at line 11"},
		{fixture: "unknown-level.yaml", want: `unknown level "L9"`},
	}

	for _, c := range cases {
		t.Run(c.fixture, func(t *testing.T) {
			_, err := surface.Load(readFixture(t, c.fixture))
			if err == nil {
				t.Fatalf("Load(%s) returned no error", c.fixture)
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("Load(%s) = %q, want it to contain %q", c.fixture, err, c.want)
			}
		})
	}
}

// TestLoadRefusesADocumentItCannotRead covers the strictness the package
// comment claims. Every case is the baseline fixture's shape with one thing
// wrong, and every one of them is a document a general YAML parser would
// happily hand back — which is the argument for reading it here (Load's
// comment).
func TestLoadRefusesADocumentItCannotRead(t *testing.T) {
	const head = "reference:\n" +
		"  jellyfin_openapi_version: \"10.11.11\"\n" +
		"  jellyfin_source_tag: \"v10.11.11\"\n" +
		"endpoints:\n"
	const row = "  - path: \"/System/Ping\"\n" +
		"    method: GET\n" +
		"    operation: GetPingSystem\n" +
		"    consumers: []\n" +
		"    feature: \"001\"\n" +
		"    level: L2\n"

	cases := []struct {
		name     string
		document string
		want     string
	}{
		{name: "the baseline of this table", document: head + row, want: ""},
		{name: "no reference block", document: "endpoints:\n" + row, want: "no reference: block"},
		{name: "no endpoints", document: head, want: "declares no endpoints"},
		{name: "an unknown top-level key", document: head + row + "extras:\n", want: "unknown top-level key"},
		{name: "a key nobody consumes", document: head + row + "    notes: whatever\n", want: `unknown key "notes"`},
		{name: "a missing key", document: head + strings.Replace(row, "    level: L2\n", "", 1), want: "has no level:"},
		{name: "a repeated key", document: head + row + "    level: L2\n", want: `key "level" appears twice`},
		{name: "a tab", document: head + "\t" + row, want: "a tab is not an indent"},
		{name: "an odd indent", document: head + row + "   level: L2\n", want: "unexpected indent of 3"},
		{name: "a row that is not a list item", document: head + strings.Replace(row, "  - path", "  path", 1), want: `expected a new endpoint`},
		{name: "no space after the colon", document: head + strings.Replace(row, "level: L2", "level:L2", 1), want: "expected a space"},
		{name: "an empty value", document: head + strings.Replace(row, `path: "/System/Ping"`, `path: ""`, 1), want: "path is empty"},
		{name: "a path with no leading slash", document: head + strings.Replace(row, `"/System/Ping"`, `"System/Ping"`, 1), want: "not a canonical spelling"},
		{name: "a path with a trailing slash", document: head + strings.Replace(row, `"/System/Ping"`, `"/System/Ping/"`, 1), want: "not a canonical spelling"},
		{name: "a lower-case method", document: head + strings.Replace(row, "method: GET", "method: get", 1), want: "not an upper-case token"},
		{name: "a feature that is not a directory number", document: head + strings.Replace(row, `feature: "001"`, `feature: "1"`, 1), want: "three-digit"},
		{name: "a consumers list that is not a list", document: head + strings.Replace(row, "consumers: []", "consumers: music-client", 1), want: "expected a list"},
		{
			name:     "two paths differing only in casing",
			document: head + row + strings.NewReplacer("/System/Ping", "/System/ping", "GetPingSystem", "GetPingSystemAgain").Replace(row),
			want:     "differs only in casing",
		},
		{
			name:     "one operation on two routes",
			document: head + row + strings.Replace(row, "method: GET", "method: POST", 1),
			want:     "operation GetPingSystem is already declared",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := surface.Load([]byte(c.document))
			switch {
			case c.want == "" && err != nil:
				t.Fatalf("Load returned %v, want it to load", err)
			case c.want == "":
				return
			case err == nil:
				t.Fatalf("Load returned no error, want one containing %q", c.want)
			case !strings.Contains(err.Error(), c.want):
				t.Errorf("Load = %q, want it to contain %q", err, c.want)
			}
		})
	}
}

func loadFixture(t *testing.T, name string) *surface.Table {
	t.Helper()
	table, err := surface.Load(readFixture(t, name))
	if err != nil {
		t.Fatalf("Load(%s): %v", name, err)
	}
	return table
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	return data
}
