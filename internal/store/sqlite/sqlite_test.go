package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openForTest opens a store in a fresh directory and closes it when the test
// ends. Every test here starts from nothing on disk, which is also what plan 8
// says 001 is testable with.
func openForTest(t *testing.T) *Store {
	t.Helper()
	return openIn(t, t.TempDir())
}

func openIn(t *testing.T, directory string) *Store {
	t.Helper()
	store, err := Open(context.Background(), directory)
	if err != nil {
		t.Fatalf("Open(%s) returned %v, want a store", directory, err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close returned %v", err)
		}
	})
	return store
}

// TestOpenPutsTheDatabaseInTheDataDirectory pins the one thing outside this
// package that can observe where the store lives: the file an operator backs up
// and the file a store rebuild deletes.
func TestOpenPutsTheDatabaseInTheDataDirectory(t *testing.T) {
	directory := t.TempDir()
	store := openIn(t, directory)

	want := filepath.Join(directory, DatabaseFile)
	if store.Path() != want {
		t.Errorf("Path is %s, want %s", store.Path(), want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("the database is not on disk: %v", err)
	}
}

// TestOpenRefusesAMissingDataDirectory is plan 7's "store unopenable" row, and
// the reason Open checks the directory before it hands the path to the driver:
// SQLite answers a missing directory, a permission problem and a corrupt header
// with the same sentence.
func TestOpenRefusesAMissingDataDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nowhere")

	store, err := Open(context.Background(), missing)
	if err == nil {
		store.Close()
		t.Fatal("Open returned a store for a directory that is not there, want an error")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("error %q does not name %s", err, missing)
	}
}

// TestOpenRefusesADataDirectoryThatIsAFile covers the other half of the same
// check: a path that exists and is not a directory.
func TestOpenRefusesADataDirectoryThatIsAFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a-file")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	store, err := Open(context.Background(), path)
	if err == nil {
		store.Close()
		t.Fatal("Open returned a store for a file, want an error")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error %q does not say the path is not a directory", err)
	}
}

// TestTheStoreOpensWithThePragmasTheRecordNames asserts every connection
// setting ADR-0003 decided, on a connection database/sql handed out rather than
// on one this test configured.
//
// It is worth a test because all four are silent when they are wrong: foreign
// keys off is a schema whose constraints are comments, and a journal mode that
// is not WAL is a reader blocked behind every scan — neither of which fails
// anything until a feature that depends on it is written.
func TestTheStoreOpensWithThePragmasTheRecordNames(t *testing.T) {
	store := openForTest(t)

	for _, c := range []struct {
		pragma string
		want   string
		why    string
	}{
		{"journal_mode", "wal", "a reader must not block behind a scan"},
		{"synchronous", "1", "NORMAL: the log is not fsynced on every commit"},
		{"foreign_keys", "1", "SQLite leaves them off by default"},
		{"busy_timeout", busyTimeoutMilliseconds, "wait for another process's write lock"},
	} {
		var got string
		if err := store.writer.QueryRow("PRAGMA " + c.pragma).Scan(&got); err != nil {
			t.Errorf("PRAGMA %s: %v", c.pragma, err)
			continue
		}
		if got != c.want {
			t.Errorf("the writer's %s is %q, want %q — %s", c.pragma, got, c.want, c.why)
		}
	}
}

// TestTheReaderPoolRefusesAWrite is what makes "one writer handle and a pool of
// readers" (ADR-0003) a property of the database rather than a convention.
//
// Without query_only the reading pool is an ordinary handle, every reader is a
// potential second writer, and the first one to try it finds out at run time
// under a lock somebody else holds.
func TestTheReaderPoolRefusesAWrite(t *testing.T) {
	store := openForTest(t)

	if _, err := store.reader.Exec(`UPDATE installation SET server_name = 'through the readers' WHERE id = 1`); err == nil {
		t.Fatal("the reading pool accepted a write, want it refused")
	}

	// And the row is untouched, which is the part that matters: an error that
	// arrived after the write would be worse than no error at all.
	var name string
	if err := store.reader.QueryRow(`SELECT server_name FROM installation WHERE id = 1`).Scan(&name); err != nil {
		t.Fatalf("reading the name back: %v", err)
	}
	if name != "atrium" {
		t.Errorf("server_name is %q after a refused write, want it unchanged", name)
	}
}

// TestCloseIsReportedOnceAndReleasesBoth asserts the handles are actually
// released: a handle left open holds the write-ahead log open with it.
func TestCloseIsReportedOnceAndReleasesBoth(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir())
	if err != nil {
		t.Fatalf("Open returned %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close returned %v", err)
	}

	// database/sql reports a closed handle with an unexported error, so the
	// assertion is that both refuse rather than which sentinel they refuse
	// with.
	if err := store.writer.PingContext(context.Background()); err == nil {
		t.Error("the writer still answers after Close, want it released")
	}
	if err := store.reader.PingContext(context.Background()); err == nil {
		t.Error("the reader still answers after Close, want it released")
	}
}
