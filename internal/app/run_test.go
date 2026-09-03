package app

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/build"
	"github.com/vdatanet/atrium-go/internal/httpapi"
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

// TestRunServesThePipelineAndOpensTheGate is the wiring half of T14: the
// binary serves the assembled pipeline, and it opens the readiness gate once
// the start has finished.
//
// Both halves matter and each fails in a way the other cannot see. A binary
// that served something other than the pipeline would answer this request
// without X-Response-Time-ms or Server; a binary that assembled the pipeline
// and never called MarkReady would answer 503 to every request for the life of
// the process, which is the failure a gate that is shut when it is built buys
// in exchange for closing the starting window (plan 6.8).
//
// The address comes off the log because that is the only place a running Run
// publishes it. Port 0 is asked for deliberately: a fixed port would make this
// test fail on a machine that happens to be using it.
func TestRunServesThePipelineAndOpensTheGate(t *testing.T) {
	stderr := &synchronisedBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stopped := make(chan error, 1)
	go func() {
		stopped <- Run(ctx, []string{
			"--" + flagDataDirectory, t.TempDir(),
			"--" + flagBindAddress, "127.0.0.1:0",
		}, nil, stderr)
	}()

	address := waitForListeningAddress(t, stderr)

	client := &http.Client{Transport: &http.Transport{}, Timeout: 10 * time.Second}
	defer client.CloseIdleConnections()

	response, err := client.Get("http://" + address + "/System/Ping")
	if err != nil {
		t.Fatalf("issuing a request to the running binary: %v", err)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	if response.StatusCode == http.StatusServiceUnavailable {
		t.Errorf("status = 503 with %s = %q: the gate was assembled and never opened",
			httpapi.MessageHeader, response.Header.Get(httpapi.MessageHeader))
	}
	if got := response.Header.Values(httpapi.ResponseTimeHeader); len(got) != 1 {
		t.Errorf("%s = %v, want exactly one field line — the binary is not serving the pipeline",
			httpapi.ResponseTimeHeader, got)
	}
	if got, want := response.Header.Values(httpapi.ServerHeaderName), "Atrium/"+build.Version(); len(got) != 1 || got[0] != want {
		t.Errorf("%s = %v, want exactly [%q]", httpapi.ServerHeaderName, got, want)
	}

	cancel()
	select {
	case err := <-stopped:
		if err != nil {
			t.Errorf("Run returned %v, want nil", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return within 10s of the stop")
	}
}

// listeningAddress reads the address off the line Serve writes when it starts
// accepting. slog's text handler quotes a value only where it has to, and an
// address needs no quoting.
var listeningAddress = regexp.MustCompile(`msg=listening address=(\S+)`)

func waitForListeningAddress(t *testing.T, log *synchronisedBuffer) string {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for {
		if match := listeningAddress.FindStringSubmatch(log.String()); match != nil {
			return match[1]
		}
		if time.Now().After(deadline) {
			t.Fatalf("no listening line within 10s:\n%s", log.String())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// synchronisedBuffer is a log this test can read while Run is still writing to
// it. bytes.Buffer is not safe for that, and the race detector says so.
type synchronisedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronisedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *synchronisedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
