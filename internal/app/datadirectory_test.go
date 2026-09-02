package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/store/sqlite"
	"github.com/vdatanet/atrium-go/internal/system"
)

// TestEnsureDataDirectoryCreatesTheDirectory. A first start into a directory
// that is not there is the ordinary case — an install, a container with an
// empty volume — and requiring a mkdir first would make the very first thing an
// operator does a step nothing told them about.
func TestEnsureDataDirectoryCreatesTheDirectory(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "atrium")

	if err := EnsureDataDirectory(directory); err != nil {
		t.Fatalf("EnsureDataDirectory returned %v", err)
	}

	info, err := os.Stat(directory)
	if err != nil {
		t.Fatalf("the directory was not created: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("the path exists and is not a directory")
	}
	if mode := info.Mode().Perm(); mode != dataDirectoryMode.Perm() {
		t.Errorf("the directory is %o, want %o — it holds credentials and tokens",
			mode, dataDirectoryMode.Perm())
	}
}

// TestEnsureDataDirectoryAcceptsOneThatIsAlreadyThere. Every start after the
// first, and every start with an operator-prepared directory.
func TestEnsureDataDirectoryAcceptsOneThatIsAlreadyThere(t *testing.T) {
	directory := t.TempDir()
	marker := filepath.Join(directory, "already-here")
	if err := os.WriteFile(marker, nil, 0o600); err != nil {
		t.Fatalf("writing the marker: %v", err)
	}

	if err := EnsureDataDirectory(directory); err != nil {
		t.Fatalf("EnsureDataDirectory returned %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("the existing contents did not survive: %v", err)
	}
}

// TestEnsureDataDirectoryRefusesAMissingParent is the reason it creates one
// component and not a path.
//
// A server that invents every directory it was pointed at answers a mistyped
// --data-dir with an empty installation that looks exactly like a fresh one:
// no users, no libraries, a new server identity, and every client asked to log
// in again — while the real data sits untouched under the path that was meant.
// A start that stops and names the parent it could not find is the cheap
// version of that mistake.
func TestEnsureDataDirectoryRefusesAMissingParent(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "lbi")
	directory := filepath.Join(parent, "atrium")

	err := EnsureDataDirectory(directory)
	if err == nil {
		t.Fatal("EnsureDataDirectory created a path whose parent does not exist, want a refusal")
	}
	if !strings.Contains(err.Error(), parent) {
		t.Errorf("error %q does not name the parent that is missing", err)
	}
	if _, statErr := os.Stat(directory); statErr == nil {
		t.Error("the directory was created anyway")
	}
}

// TestEnsureDataDirectoryRefusesAFile covers the path that exists and is not a
// directory, which Mkdir reports as "already exists" and nothing else.
func TestEnsureDataDirectoryRefusesAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "atrium")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	err := EnsureDataDirectory(path)
	if err == nil {
		t.Fatal("EnsureDataDirectory accepted a file, want a refusal")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error %q does not say the path is not a directory", err)
	}
}

// TestRunPreparesTheDataDirectoryAndTheStore is the wiring: one start, into a
// directory that does not exist yet, leaves an installation identity and a
// migrated database behind.
//
// The context is cancelled before Run is called, so the process does its whole
// start and its whole stop with nothing in between.
func TestRunPreparesTheDataDirectoryAndTheStore(t *testing.T) {
	directory := filepath.Join(t.TempDir(), "atrium")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr bytes.Buffer
	if err := Run(ctx, []string{
		"--" + flagDataDirectory, directory,
		"--" + flagBindAddress, "127.0.0.1:0",
	}, nil, &stderr); err != nil {
		t.Fatalf("Run returned %v, want nil:\n%s", err, stderr.String())
	}

	for _, name := range []string{system.InstallationIDFile, sqlite.DatabaseFile} {
		if _, err := os.Stat(filepath.Join(directory, name)); err != nil {
			t.Errorf("a start left no %s behind: %v", name, err)
		}
	}

	// And the database it left is migrated, read by opening it again the way a
	// second start would.
	store, err := sqlite.Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("reopening the store returned %v", err)
	}
	defer store.Close()

	if applied := store.AppliedMigrations(sqlite.Precious); len(applied) != 0 {
		t.Errorf("reopening applied %v, want nothing — the start had already migrated", applied)
	}
	installation, err := store.Installation(context.Background())
	if err != nil {
		t.Fatalf("Installation returned %v", err)
	}
	if installation.Name != "atrium" {
		t.Errorf("ServerName is %q after a first start, want %q (spec 3.1)", installation.Name, "atrium")
	}
}

// TestRunRefusesToStartWhenTheStoreCannotBeOpened is plan 7's row. The store is
// opened before the listener for the same reason the identity is read before
// it: a refusal has to be about the thing that is wrong, and a bad port must
// not get the chance to be the reported error instead.
func TestRunRefusesToStartWhenTheStoreCannotBeOpened(t *testing.T) {
	directory := t.TempDir()

	// A directory where the database file belongs. SQLite cannot open it, and
	// nothing this process does can make it able to.
	if err := os.Mkdir(filepath.Join(directory, sqlite.DatabaseFile), 0o700); err != nil {
		t.Fatalf("putting a directory in the database's place: %v", err)
	}

	var stderr bytes.Buffer
	err := Run(context.Background(), []string{
		"--" + flagDataDirectory, directory,
		"--" + flagBindAddress, "127.0.0.1:0",
	}, nil, &stderr)

	if err == nil {
		t.Fatal("Run returned nil with an unopenable store, want a refusal to start")
	}
	if !strings.Contains(err.Error(), sqlite.DatabaseFile) {
		t.Errorf("error %q does not name the database", err)
	}
}
