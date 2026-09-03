package httpapi

import (
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/surface"
)

// TestNewRefusalsRefusesATableItCannotFold is the constructor's only failure,
// and it is reachable rather than defensive: two parametrised paths that
// accept the same requests load into a table happily — they are distinct
// strings with distinct operations — and only the fold map notices.
//
// A stage that cannot be built returns an error rather than panicking, so this
// is a failure to start (plan 7).
func TestNewRefusalsRefusesATableItCannotFold(t *testing.T) {
	document := `reference:
  jellyfin_openapi_version: "10.11.11"
  jellyfin_source_tag: "v10.11.11"

endpoints:
  - path: "/Items/{itemId}"
    method: GET
    operation: GetItem
    consumers: []
    feature: "005"
    level: L2
  - path: "/Items/{id}"
    method: POST
    operation: UpdateItem
    consumers: []
    feature: "009"
    level: L2
`
	table, err := surface.Load([]byte(document))
	if err != nil {
		t.Fatalf("the fixture document does not load, so it proves nothing about the folder: %v", err)
	}

	if _, err := NewRefusals(table); err == nil {
		t.Fatal("NewRefusals accepted a table whose two paths fold together; there is no rule for choosing between them")
	} else if !strings.Contains(err.Error(), "fold together") {
		t.Errorf("NewRefusals error = %v, want it to name the ambiguity", err)
	}
}

// TestAllowIsJoinedOnceAtConstruction asserts the shape of the built value
// rather than only what a request sees, because the join is the ordering
// behaviours 1.11 measured and a map is where it is kept.
func TestAllowIsJoinedOnceAtConstruction(t *testing.T) {
	refusals, err := NewRefusals(surface.V1())
	if err != nil {
		t.Fatalf("building the refusal stage over the v1 table: %v", err)
	}

	if got := refusals.allow["/System/Ping"]; got != "GET, POST" {
		t.Errorf("allow[/System/Ping] = %q, want %q", got, "GET, POST")
	}
	// behaviours 1.11's own measured pair, where alphabetical order and
	// registration order differ
	// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28].
	if got := refusals.allow["/UserFavoriteItems/{itemId}"]; got != "DELETE, POST" {
		t.Errorf("allow[/UserFavoriteItems/{itemId}] = %q, want %q", got, "DELETE, POST")
	}
	if len(refusals.allow) != len(surface.V1().Paths()) {
		t.Errorf("allow holds %d paths, want one per path in the table (%d)", len(refusals.allow), len(surface.V1().Paths()))
	}
}
