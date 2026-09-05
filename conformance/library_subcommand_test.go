package conformance_test

// The library subcommand, against the built binary — which is the only place
// `cmd/atrium`'s dispatch is observable at all.
//
// 003 T14 adds a second arm to a dispatch that branches on the first argument,
// and **the regression that shape is most likely to introduce is to the other
// two branches**: a server started as `atrium --data-dir …` and an account
// created with `atrium user add`. 002 T7 paid for that once. It is cheaper to
// re-assert here than to rediscover, and no Go test in `internal/` can assert
// it, because nothing may import `main`.
//
// What this file is not: 003's conformance surface. That is 003 T18's, and it is
// bounded by 003 registering no route at all.

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The library subcommand's own words, spelled here rather than imported, for
// the reason the account subcommand's are (provisioning_test.go): this package
// may import nothing of ours, so what is below is the command line an operator
// types.
const (
	libraryCommand     = "library"
	libraryAddCommand  = "add"
	libraryListCommand = "list"
)

// The three arms of the dispatch, in one test, because the assertion is that
// adding the third left the other two alone.
//
// The order matters: the library verb runs first, on the same installation the
// account is then created in, so a build whose new arm consumed the argument
// vector or left the store in a state `user add` could not open would fail on
// the line after.
func TestTheLibraryArmDidNotTakeTheServerOrTheAccountArmWithIt(t *testing.T) {
	t.Parallel()

	data := t.TempDir()
	root := filepath.Join(t.TempDir(), "films")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("making %s: %v", root, err)
	}
	if err := os.WriteFile(filepath.Join(root, "The Matrix (1999).mkv"), []byte("x"), 0o644); err != nil {
		t.Fatalf("writing a film: %v", err)
	}

	// `atrium library add` — the arm this task adds. Before this change the
	// binary answered it by falling through to the server, where the
	// configuration refuses it as an unexpected argument.
	output, err := runBinary(t, "", libraryCommand, libraryAddCommand,
		"--data-dir", data, "--name", "Films", "--type", "movies", "--root", root)
	if err != nil {
		t.Fatalf("atrium %s %s: %v\n%s", libraryCommand, libraryAddCommand, err, output)
	}

	listed, err := runBinary(t, "", libraryCommand, libraryListCommand,
		"--data-dir", data, "--format", "json")
	if err != nil {
		t.Fatalf("atrium %s %s: %v\n%s", libraryCommand, libraryListCommand, err, listed)
	}
	if !strings.Contains(listed, `"name":"Films"`) {
		t.Errorf("the library is not in the list the binary printed:\n%s", listed)
	}

	// `atrium user add` still works, on the same installation.
	if output, err := runBinary(t, "hunter2\n", userCommand, userAddCommand,
		"--data-dir", data, "--name", "Ada"); err != nil {
		t.Fatalf("atrium %s %s: %v\n%s", userCommand, userAddCommand, err, output)
	}

	// And a first argument that is not a subcommand still starts the server:
	// `atrium --data-dir …` has to keep serving, which is what every other test
	// in this package depends on and what this line states outright.
	server := startServer(t)
	response := server.get(t, pingPath, goldenHost, nil)
	if response.status != http.StatusOK {
		t.Errorf("GET %s answered %d, want %d: the server arm of the dispatch",
			pingPath, response.status, http.StatusOK)
	}
}
