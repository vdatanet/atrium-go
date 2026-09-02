package system

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// InstallationIDFile is the name, within the data directory, of the file that
// carries this installation's identity.
//
// It is a file and not a row in the store, and AC-4 is the whole reason: the
// identity has to be "identical across a restart and across a rebuild of the
// store from empty" (spec 5, AC-4), which a row cannot be. Deriving it from the
// data directory's path would satisfy both clauses and make moving the
// directory a silent re-authentication of every client — behaviours 1.4's
// library-root trap at the level of the whole server. So it is persisted beside
// the store rather than in it: a store rebuild does not touch it, and moving
// the data directory carries it along (plan 4).
const InstallationIDFile = "installation-id"

// The identity is 32 lowercase hexadecimal characters, which is what the
// reference serialises a GUID as and therefore what a client is prepared to
// read back: .NET's "N" format, 32 hex characters and no dashes
// (behaviours 1.4). Sixteen random bytes is that many characters and is also
// the width of the value the reference is rendering.
const (
	installationIDBytes  = 16
	installationIDLength = 2 * installationIDBytes
)

// ErrMalformedInstallationID is returned when the file exists and does not hold
// 32 lowercase hexadecimal characters. It is a distinct error because refusing
// the start on it is a decision rather than an accident: generating a fresh
// identity instead would make every client treat the server as new and
// re-authenticate — a silent, expensive failure where a refusal to start is a
// loud, cheap one (plan 7).
var ErrMalformedInstallationID = errors.New("not 32 lowercase hexadecimal characters")

// InstallationID returns the identity of the installation whose data lives in
// dataDirectory, creating it on a first start and reading it on every start
// after that. The value it returns is what /System/Info/Public reports as Id
// (spec 3.1).
//
// It does not create dataDirectory. The directory is the one thing an operator
// names, and 001 is "testable with nothing on disk but a data directory"
// (plan 8) — a server that invents the directory it was pointed at would hide a
// mistyped path as an empty installation.
//
// Every error names the file, because every one of them is answered by an
// operator looking at it.
func InstallationID(dataDirectory string) (string, error) {
	path := filepath.Join(dataDirectory, InstallationIDFile)

	// Two passes at most. The second exists for one case only: another start
	// created the file between this one's read and its own create, and the
	// loser of that race reads what the winner wrote instead of failing.
	for range 2 {
		id, err := readInstallationID(path)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}

		id, err = createInstallationID(path)
		if err == nil {
			return id, nil
		}
		if !errors.Is(err, fs.ErrExist) {
			return "", err
		}
	}
	return "", fmt.Errorf("%s: appeared and disappeared while it was being created", path)
}

// readInstallationID reads the file and returns the identity it holds.
//
// Surrounding whitespace is not part of the value: the file is written as a
// single line, and an operator who has opened it in an editor gets the trailing
// newline back whether they wanted one or not. What is inside the line is
// judged exactly — an uppercase digit is refused rather than folded, because
// the identity is echoed verbatim into a response and behaviours 1.4 measured
// the reference's spelling as lowercase.
func readInstallationID(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		return "", fmt.Errorf("%s: %w", path, err)
	}

	id := strings.TrimSpace(string(content))
	if !isInstallationID(id) {
		return "", fmt.Errorf("%s: %w", path, ErrMalformedInstallationID)
	}
	return id, nil
}

// createInstallationID writes a new identity, and reports fs.ErrExist when
// another start got there first.
//
// The file appears with its content already in it, and that is not a detail:
// creating it with O_EXCL and writing afterwards leaves a window in which the
// file exists and is empty, and a second start that reads in that window
// refuses to boot on an identity it watched being written. Sixty-four rounds of
// eight concurrent starts hit that window on round five, so the identity is
// written to a temporary file in the same directory and published with a hard
// link — which fails with fs.ErrExist exactly as O_EXCL would, and never
// exposes a half-written file. The one guarantee plan 4 asks of O_EXCL, that
// "two starts cannot race", is the one this keeps.
func createInstallationID(path string) (string, error) {
	id, err := newInstallationID()
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	directory := filepath.Dir(path)
	// os.CreateTemp makes the file 0600, which is the mode the identity keeps:
	// nothing but this process writes it. It is not a secret — it is served
	// unauthenticated as Id (spec 3.1) — so the mode is about ownership of the
	// data directory and not about concealment.
	temporary, err := os.CreateTemp(directory, InstallationIDFile+".*")
	if err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	defer os.Remove(temporary.Name())

	if err := writeAndSync(temporary, id+"\n"); err != nil {
		temporary.Close()
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if err := temporary.Close(); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}

	if err := os.Link(temporary.Name(), path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", err
		}
		return "", fmt.Errorf("%s: %w", path, err)
	}
	if err := syncDirectory(directory); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return id, nil
}

// syncDirectory puts the new name on the disk, not only the bytes behind it. An
// identity written once in the life of an installation and read on every start
// after that has exactly one moment at which a power cut can lose it, and this
// is the second half of it.
func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	if err := handle.Sync(); err != nil {
		handle.Close()
		return err
	}
	return handle.Close()
}

// writeAndSync puts the line on the disk rather than in a cache.
func writeAndSync(file *os.File, line string) error {
	if _, err := file.WriteString(line); err != nil {
		return err
	}
	return file.Sync()
}

// newInstallationID generates an identity from 16 cryptographically random
// bytes. crypto/rand and not math/rand: two installations imaged from the same
// machine, or started in the same second, must not share an identity, and a
// seeded generator is exactly how they would.
func newInstallationID() (string, error) {
	var bytes [installationIDBytes]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generating an installation identity: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}

// isInstallationID reports whether s is exactly 32 lowercase hexadecimal
// characters. hex.DecodeString is not the check: it accepts uppercase, and the
// value goes out on the wire as it came in.
func isInstallationID(s string) bool {
	if len(s) != installationIDLength {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
