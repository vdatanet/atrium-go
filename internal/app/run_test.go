package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/system"
)

// TestRunRefusesToStartOnAMalformedInstallationID is the wiring half of the
// refusal: internal/system decides that the file is unusable, and this asserts
// that the process stops rather than carrying on with a new identity (plan 7).
//
// It is here and not beside the identity because "refuse to start" is a claim
// about the entry layer. The bind address would be refused too, so the test
// also says the identity is read before anything is listened on: an unusable
// port never gets the chance to be the reported error.
func TestRunRefusesToStartOnAMalformedInstallationID(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, system.InstallationIDFile)
	if err := os.WriteFile(path, []byte("nonsense"), 0o600); err != nil {
		t.Fatalf("writing the identity file: %v", err)
	}

	var stderr bytes.Buffer
	err := Run(context.Background(), []string{
		"--" + flagDataDirectory, directory,
		"--" + flagBindAddress, "127.0.0.1:0",
	}, nil, &stderr)

	if err == nil {
		t.Fatal("Run returned nil, want a refusal to start")
	}
	if !errors.Is(err, system.ErrMalformedInstallationID) {
		t.Errorf("Run returned %v, want it to be ErrMalformedInstallationID", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name %s", err, path)
	}
}

// TestRunReportsTheInstallationIDItStartedWith pins the identity into the
// startup line. An installation's identity is what a client keys its session
// on, and a report that two runs disagreed is unreadable without knowing which
// installation each run was.
//
// The context is cancelled before Run is called, so the process does its whole
// start and then its whole stop with nothing in between. That is the cheapest
// way to read what a start writes.
func TestRunReportsTheInstallationIDItStartedWith(t *testing.T) {
	directory := t.TempDir()
	id, err := system.InstallationID(directory)
	if err != nil {
		t.Fatalf("creating the identity: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var stderr bytes.Buffer
	if err := Run(ctx, []string{
		"--" + flagDataDirectory, directory,
		"--" + flagBindAddress, "127.0.0.1:0",
	}, nil, &stderr); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}

	if log := stderr.String(); !strings.Contains(log, id) {
		t.Errorf("the startup log does not report the installation id %q:\n%s", id, log)
	}
}
