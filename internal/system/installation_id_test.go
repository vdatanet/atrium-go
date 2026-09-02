package system

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestInstallationIDIsCreatedOnAFreshDataDirectory is the first start: nothing
// on disk but the directory, and the identity AC-4 asks for comes back as 32
// lowercase hexadecimal characters.
//
// It also asserts what landed in the file, because the file is the contract
// with the next start and with an operator restoring a backup — a value held
// only in memory would pass a test that never restarted.
func TestInstallationIDIsCreatedOnAFreshDataDirectory(t *testing.T) {
	directory := t.TempDir()

	id, err := InstallationID(directory)
	if err != nil {
		t.Fatalf("InstallationID on a fresh directory: %v", err)
	}
	if !isInstallationID(id) {
		t.Errorf("id = %q, want 32 lowercase hexadecimal characters", id)
	}

	content, err := os.ReadFile(filepath.Join(directory, InstallationIDFile))
	if err != nil {
		t.Fatalf("reading %s: %v", InstallationIDFile, err)
	}
	if got := strings.TrimSpace(string(content)); got != id {
		t.Errorf("%s holds %q, want the returned id %q", InstallationIDFile, got, id)
	}
}

// TestInstallationIDIsTheSameOnASecondStart is AC-4's first clause: the
// identity is generated once and never changes, so a restart is a read.
func TestInstallationIDIsTheSameOnASecondStart(t *testing.T) {
	directory := t.TempDir()

	first, err := InstallationID(directory)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}
	second, err := InstallationID(directory)
	if err != nil {
		t.Fatalf("second start: %v", err)
	}

	if second != first {
		t.Errorf("second start returned %q, want the first start's %q: a client would re-authenticate", second, first)
	}
}

// TestInstallationIDSurvivesARebuildOfTheStore is AC-4's second clause, and the
// whole reason the identity is a file rather than a row: "identical across a
// restart and across a rebuild of the store from empty". A client's session has
// to survive an operator rebuilding a corrupted database.
//
// The rebuild is performed as the removal of everything in the data directory
// except the identity file. The store's own file names are 003's — T3's — to
// choose, and naming them here would make this test pass for the wrong reason
// on the day one of them changes. Deleting everything else is the stronger
// statement anyway: whatever the store turns out to be, the identity does not
// depend on it.
func TestInstallationIDSurvivesARebuildOfTheStore(t *testing.T) {
	directory := t.TempDir()

	before, err := InstallationID(directory)
	if err != nil {
		t.Fatalf("first start: %v", err)
	}

	// Stand in for whatever the store writes beside the identity.
	for _, name := range []string{"atrium.db", "atrium.db-wal", "atrium.db-shm", "cache"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("store"), 0o600); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("reading the data directory: %v", err)
	}
	removed := 0
	for _, entry := range entries {
		if entry.Name() == InstallationIDFile {
			continue
		}
		if err := os.RemoveAll(filepath.Join(directory, entry.Name())); err != nil {
			t.Fatalf("removing %s: %v", entry.Name(), err)
		}
		removed++
	}
	if removed == 0 {
		t.Fatal("nothing was removed: the rebuild this test claims to perform did not happen")
	}

	after, err := InstallationID(directory)
	if err != nil {
		t.Fatalf("start after the store was rebuilt: %v", err)
	}
	if after != before {
		t.Errorf("after a store rebuild the id is %q, want the earlier %q (AC-4)", after, before)
	}
}

// TestAMalformedInstallationIDRefusesTheStart is plan 7's deliberate refusal.
// Generating a fresh identity over a file that cannot be read would make every
// client treat the server as new; the refusal is the loud, cheap failure.
//
// Every case asserts the error names the file, because the operator's next move
// is to look at it, and an error that says only "malformed" leaves them
// searching a data directory for which file it meant.
func TestAMalformedInstallationIDRefusesTheStart(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"nonsense", "nonsense"},
		{"empty", ""},
		{"whitespace only", "\n\n"},
		{"uppercase hexadecimal", "3F9C1A7E5B2D4E8091A6C3F70D5E2B14"},
		{"one character short", "3f9c1a7e5b2d4e8091a6c3f70d5e2b1"},
		{"one character long", "3f9c1a7e5b2d4e8091a6c3f70d5e2b141"},
		{"hexadecimal with a dash", "3f9c1a7e-5b2d-4e80-91a6-c3f70d5e2b14"},
		{"two lines", "3f9c1a7e5b2d4e8091a6c3f70d5e2b14\n3f9c1a7e5b2d4e8091a6c3f70d5e2b14\n"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, InstallationIDFile)
			if err := os.WriteFile(path, []byte(c.content), 0o600); err != nil {
				t.Fatalf("writing the file: %v", err)
			}

			id, err := InstallationID(directory)
			if err == nil {
				t.Fatalf("InstallationID returned %q and no error, want a refusal", id)
			}
			if !errors.Is(err, ErrMalformedInstallationID) {
				t.Errorf("error is %v, want it to be ErrMalformedInstallationID", err)
			}
			if !strings.Contains(err.Error(), path) {
				t.Errorf("error %q does not name %s", err, path)
			}

			if after, readErr := os.ReadFile(path); readErr != nil || string(after) != c.content {
				t.Errorf("the file was changed to %q (err %v), want the refusal to leave it alone", after, readErr)
			}
		})
	}
}

// TestAnUnreadableInstallationIDRefusesTheStart is the other half of the same
// refusal: the file is there and cannot be read. It is a separate test because
// it is refused for a different reason and must not be reported as malformed.
func TestAnUnreadableInstallationIDRefusesTheStart(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads a mode 0 file, so this test cannot make one unreadable")
	}

	directory := t.TempDir()
	path := filepath.Join(directory, InstallationIDFile)
	if err := os.WriteFile(path, []byte("3f9c1a7e5b2d4e8091a6c3f70d5e2b14\n"), 0o000); err != nil {
		t.Fatalf("writing the file: %v", err)
	}

	id, err := InstallationID(directory)
	if err == nil {
		t.Fatalf("InstallationID returned %q and no error, want a refusal", id)
	}
	if errors.Is(err, ErrMalformedInstallationID) {
		t.Errorf("error is %v, want an unreadable file not to be reported as malformed", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name %s", err, path)
	}
}

// TestConcurrentStartsAgreeOnOneInstallationID exercises the reason the create
// is O_EXCL. Two processes pointed at one data directory must not each believe
// they generated the identity; the loser of the race reads the winner's value
// rather than failing or overwriting it.
//
// Goroutines are not processes, but the file operations are the same ones, and
// this is the only way the retry branch is reached at all.
func TestConcurrentStartsAgreeOnOneInstallationID(t *testing.T) {
	const (
		rounds = 64
		starts = 8
	)

	for round := range rounds {
		directory := t.TempDir()

		ids := make([]string, starts)
		errs := make([]error, starts)
		begin := make(chan struct{})
		var done sync.WaitGroup
		for i := range starts {
			done.Add(1)
			go func() {
				defer done.Done()
				<-begin
				ids[i], errs[i] = InstallationID(directory)
			}()
		}
		close(begin)
		done.Wait()

		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d, start %d: %v", round, i, err)
			}
		}
		for i, id := range ids {
			if id != ids[0] {
				t.Fatalf("round %d: start %d returned %q, want the same %q as start 0", round, i, id, ids[0])
			}
		}
	}
}
