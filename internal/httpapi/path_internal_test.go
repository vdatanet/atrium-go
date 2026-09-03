package httpapi

import (
	"strings"
	"testing"
)

// TestAPathTheFoldMapCannotDescribeIsRefused covers the builder's refusals.
//
// They are reachable only from here, because surface.Load refuses most of
// these before a table can carry one — which is why newPathFolder takes the
// paths rather than the table. A refusal nothing can reach is a refusal nobody
// has checked.
func TestAPathTheFoldMapCannotDescribeIsRefused(t *testing.T) {
	rows := []struct {
		name  string
		paths []string
		want  string
	}{
		{"no leading slash", []string{"System/Ping"}, "does not begin with a slash"},
		{"an empty segment", []string{"/System//Ping"}, "empty segment"},
		{"a parameter that is never closed", []string{"/Items/{itemId"}, "never closes"},
		{"a brace that closes nothing", []string{"/Items/itemId}"}, "never opened"},
		{"a brace that closes nothing, after a literal run", []string{"/Items/a}b{c}"}, "never opened"},
		{"a parameter with no name", []string{"/Items/{}"}, "does not name its parameter"},
		{"a parameter opened twice", []string{"/Items/{{itemId}"}, "does not name its parameter"},
		{"two parameters side by side", []string{"/Items/{itemId}{imageType}"}, "side by side"},
		{"two paths differing only in casing", []string{"/System/Ping", "/system/PING"}, "fold together"},
		{"two paths differing only in a parameter's name", []string{"/Items/{itemId}", "/Items/{userId}"}, "fold together"},
		{"two paths differing only in a parameter's name inside a segment", []string{"/Audio/{a}/stream.{b}", "/Audio/{c}/STREAM.{d}"}, "fold together"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			_, err := newPathFolder(row.paths)
			if err == nil {
				t.Fatalf("newPathFolder(%q) = nil error, want one mentioning %q", row.paths, row.want)
			}
			if !strings.Contains(err.Error(), row.want) {
				t.Errorf("newPathFolder(%q) error = %q, want it to mention %q", row.paths, err, row.want)
			}
		})
	}
}

// TestAPathThatOnlyLooksLikeACollisionIsAccepted is the other side of the
// refusals above: a rule that refused these would refuse the shipped table.
func TestAPathThatOnlyLooksLikeACollisionIsAccepted(t *testing.T) {
	paths := []string{
		"/Items/Filters",          // a literal a parameter would also accept
		"/Items/{itemId}",         //
		"/Items/{itemId}/Similar", // one segment longer
		"/Audio/{itemId}/stream",  // the same shape under a different literal
		"/Videos/{itemId}/stream",

		// The same literal runs in the same order, and a different pattern:
		// one takes the parameter before the dot and one after. They are only
		// told apart by a key that says where the parameters are, which is
		// what parameterMark is for — a key built from the literals alone
		// would refuse this pair.
		"/A/{name}.",
		"/A/.{extension}",
	}
	if _, err := newPathFolder(paths); err != nil {
		t.Fatalf("newPathFolder(%q) = %v, want it to build", paths, err)
	}
}

// TestALiteralPathIsLookedUpBeforeAParametrisedOne fixes the precedence the
// doc comment states, over paths chosen so that both do match.
func TestALiteralPathIsLookedUpBeforeAParametrisedOne(t *testing.T) {
	folder, err := newPathFolder([]string{"/Items/{itemId}", "/Items/Filters"})
	if err != nil {
		t.Fatalf("newPathFolder: %v", err)
	}

	// /Items/{itemId} comes first in the list, so registration order alone
	// would answer /Items/{itemId} here.
	got, ok := folder.Canonicalise("/items/filters")
	if !ok {
		t.Fatalf("Canonicalise refused /items/filters")
	}
	if got != "/Items/Filters" {
		t.Errorf("Canonicalise(/items/filters) = %q, want %q", got, "/Items/Filters")
	}
}

// TestAParameterMustMatchAtLeastOneByte pins the rule matchSegment's comment
// states: a parameter that matched nothing would let /Audio/x/.mp4 stand in
// for a stream with no name.
func TestAParameterMustMatchAtLeastOneByte(t *testing.T) {
	folder, err := newPathFolder([]string{
		"/Audio/{itemId}/stream.{container}",
		"/Videos/{itemId}/hls1/{playlistId}/{segmentId}.{container}",
	})
	if err != nil {
		t.Fatalf("newPathFolder: %v", err)
	}

	rows := []struct {
		name string
		sent string
		want string
	}{
		{"the route it is all about", "/audio/x/STREAM.MP4", "/Audio/x/stream.MP4"},
		{"an empty parameter at the end of a segment", "/audio/x/stream.", "/audio/x/stream."},
		{"an empty segment where a parameter is declared", "/audio//stream.mp4", "/audio//stream.mp4"},

		// The segment is {segmentId}.{container}, so the first parameter ends
		// at the dot. Were it allowed to end before it, an empty segmentId
		// would match and .ts would be a segment with no name.
		{"the same shape in the middle of a segment", "/videos/x/HLS1/p/SeG.ts", "/Videos/x/hls1/p/SeG.ts"},
		{"an empty parameter in the middle of a segment", "/videos/x/hls1/p/.ts", "/videos/x/hls1/p/.ts"},
	}

	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			got, ok := folder.Canonicalise(row.sent)
			if !ok {
				t.Fatalf("Canonicalise(%q) refused the path; only a doubled trailing slash is refused here", row.sent)
			}
			if got != row.want {
				t.Errorf("Canonicalise(%q) = %q, want %q", row.sent, got, row.want)
			}
		})
	}
}
