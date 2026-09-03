//go:build unix

package app

import (
	"context"
	"io"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestServeStopsOnSIGTERM sends the process the signal a service manager sends
// and asserts the whole chain reacts: the handler ShutdownContext installed,
// the context it cancels, the drain, and a Serve that returns.
//
// Signalling this test's own process is what makes it a test of the signal
// rather than of a context: the request issued first proves the handler is
// installed before the signal is raised, since Serve only reaches the point of
// answering after ShutdownContext has returned.
//
// Unix only. Sending SIGTERM is not a thing a Windows process does to itself.
func TestServeStopsOnSIGTERM(t *testing.T) {
	ctx, stop := ShutdownContext(context.Background())
	defer stop()

	// Counted after the handler is installed: os/signal's receiving goroutine
	// is started by the first Notify in a process and never exits, so it is
	// part of the baseline rather than a stray.
	before := runtime.NumGoroutine()

	server := newTestServer(t, testPipeline(t))
	stopped := serve(t, ctx, server)

	client := newTestClient(t)
	response, err := client.Get(baseURL(server) + "/System/Ping")
	if err != nil {
		t.Fatalf("issuing a request: %v", err)
	}
	io.Copy(io.Discard, response.Body)
	response.Body.Close()

	if err := syscall.Kill(syscall.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("signalling this process: %v", err)
	}

	if err := waitFor(t, stopped, 10*time.Second); err != nil {
		t.Errorf("Serve returned %v, want nil", err)
	}

	// The signal handler is released before the goroutines are counted,
	// because it is one of them.
	stop()
	client.CloseIdleConnections()
	assertNoStrayGoroutines(t, before)
}
