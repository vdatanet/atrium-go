package httpapi

import (
	"net/http"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// ping is a route the v1 table has, so that the rows below fail on the name
// under test rather than on the route.
var ping = Route{Method: http.MethodGet, Path: "/System/Ping"}

// TestADeclaredNameNoRequestCouldMatchIsRefused covers checkQueryName.
//
// Every one of these declarations would compile, load and fold nothing — and
// the failure it hides is exactly the one this stage exists to prevent, where
// the client's spelling is ignored and the answer is wrong with a 200 on it.
func TestADeclaredNameNoRequestCouldMatchIsRefused(t *testing.T) {
	rows := []struct {
		name     string
		spelling string
		want     string
	}{
		{"nothing at all", "", "empty query parameter name"},
		{"a name carrying the fragment separator", "a&b", `carries "&"`},
		{"a name carrying the value separator", "a=b", `carries "="`},
		{"a name written percent-encoded", "%4Cimit", `carries "%"`},
		{"a name carrying a plus", "a+b", `carries "+"`},
		{"a name carrying a hash", "a#b", `carries "#"`},
		{"a name carrying a space", "start index", "outside printable ASCII"},
		{"a name carrying a tab", "start\tindex", "outside printable ASCII"},
		{"a name that is not ASCII", "startÍndex", "outside printable ASCII"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, err := NewQueryFolder(surface.V1(), QuerySpellings{ping: {row.spelling}})
			if err == nil {
				t.Fatalf("NewQueryFolder accepted the declared name %q", row.spelling)
			}
			if !strings.Contains(err.Error(), row.want) {
				t.Errorf("NewQueryFolder error = %q, want it to mention %q", err, row.want)
			}
		})
	}
}

// TestADeclaredNameThatOnlyLooksIrregularIsAccepted is the other side of the
// refusals above, and it is here because a check of this shape can only ever
// get stricter: no refusal test can tell a rule that refuses too much from one
// that refuses exactly enough, so what sees it is a test for what must be
// accepted. The names are the shapes a real parameter takes — camelCase,
// PascalCase, the header-shaped one a token is passed under, and the bracketed
// and dotted spellings a nested filter uses.
func TestADeclaredNameThatOnlyLooksIrregularIsAccepted(t *testing.T) {
	names := []string{
		"limit",
		"StartIndex",
		"sortBy",
		"api_key",
		"X-Emby-Token",
		"filter[genre]",
		"profile.audio",
		"enableImages2",
		"~tilde",
		"*",
	}

	folder, err := NewQueryFolder(surface.V1(), QuerySpellings{ping: names})
	if err != nil {
		t.Fatalf("NewQueryFolder(%q) = %v, want it to build", names, err)
	}
	for _, name := range names {
		sent := strings.ToUpper(name) + "=1"
		if got := folder.Canonicalise(ping.Method, ping.Path, sent); got != name+"=1" {
			t.Errorf("Canonicalise(%q) = %q, want %q", sent, got, name+"=1")
		}
	}
}

// TestTwoDeclaredNamesThatFoldTogetherAreRefused is the query counterpart of
// the path folder's refusal of two paths that fold together: a request
// spelling LIMIT could be either, and choosing between them would be a coin
// toss rather than a rule.
func TestTwoDeclaredNamesThatFoldTogetherAreRefused(t *testing.T) {
	_, err := NewQueryFolder(surface.V1(), QuerySpellings{ping: {"Limit", "limit"}})
	if err == nil {
		t.Fatal("NewQueryFolder accepted two names that fold together")
	}
	if !strings.Contains(err.Error(), "fold together") {
		t.Errorf("NewQueryFolder error = %q, want it to say the two fold together", err)
	}
}

// TestTheSameNameOnTwoRoutesIsNotAClash guards the scope of the refusal
// above: names are declared per route, so two routes spelling the same
// parameter differently is ordinary rather than ambiguous.
func TestTheSameNameOnTwoRoutesIsNotAClash(t *testing.T) {
	spellings := QuerySpellings{
		ping: {"limit"},
		{Method: http.MethodGet, Path: "/Items/Filters"}: {"Limit"},
	}

	folder, err := NewQueryFolder(surface.V1(), spellings)
	if err != nil {
		t.Fatalf("NewQueryFolder = %v, want it to build", err)
	}
	if got := folder.Canonicalise(http.MethodGet, "/System/Ping", "LIMIT=1"); got != "limit=1" {
		t.Errorf("on /System/Ping: Canonicalise = %q, want %q", got, "limit=1")
	}
	if got := folder.Canonicalise(http.MethodGet, "/Items/Filters", "LIMIT=1"); got != "Limit=1" {
		t.Errorf("on /Items/Filters: Canonicalise = %q, want %q", got, "Limit=1")
	}
}

// TestARouteIsFoundByItsPatternAndNotByOneRequest pins the lookup query
// canonicalisation borrows from the path fold: a request never carries the
// pattern that matched it, and the declarations are keyed by the pattern.
func TestARouteIsFoundByItsPatternAndNotByOneRequest(t *testing.T) {
	folder, err := newPathFolder([]string{
		"/Items/Filters",
		"/Items/{itemId}",
		"/Audio/{itemId}/stream.{container}",
	})
	if err != nil {
		t.Fatalf("newPathFolder: %v", err)
	}

	rows := []struct {
		sent  string
		want  string
		found bool
	}{
		{"/Items/Filters", "/Items/Filters", true},
		{"/items/filters", "/Items/Filters", true},
		{"/Items/AbC", "/Items/{itemId}", true},
		{"/ITEMS/AbC", "/Items/{itemId}", true},
		{"/audio/AbC/STREAM.MP4", "/Audio/{itemId}/stream.{container}", true},
		{"/Nope", "", false},
		{"/Items/AbC/Extra", "", false},
	}

	for _, row := range rows {
		got, found := folder.pattern(row.sent)
		if found != row.found {
			t.Errorf("pattern(%q) found = %t, want %t", row.sent, found, row.found)
			continue
		}
		if got != row.want {
			t.Errorf("pattern(%q) = %q, want %q", row.sent, got, row.want)
		}
	}
}

// TestTheRefusalNamesTheSameFaultWhateverTheMapOrder is the determinism
// sortedRoutes exists for. Ranging over the declarations directly would make
// which of two faults is reported depend on Go's map iteration order, and a
// build that refuses for a different reason each time is a build nobody can
// act on.
func TestTheRefusalNamesTheSameFaultWhateverTheMapOrder(t *testing.T) {
	spellings := QuerySpellings{
		{Method: http.MethodGet, Path: "/Zzz"}: {"limit"},
		{Method: http.MethodGet, Path: "/Aaa"}: {"limit"},
		{Method: http.MethodGet, Path: "/Mmm"}: {"limit"},
	}

	var first string
	for attempt := range 32 {
		_, err := NewQueryFolder(surface.V1(), spellings)
		if err == nil {
			t.Fatal("NewQueryFolder accepted three routes the table does not have")
		}
		if attempt == 0 {
			first = err.Error()
			continue
		}
		if err.Error() != first {
			t.Fatalf("attempt %d reported %q, where the first reported %q", attempt, err, first)
		}
	}
	if !strings.Contains(first, "/Aaa") {
		t.Errorf("the refusal named %q, want the first fault in the routes' own order, which is /Aaa", first)
	}
}
