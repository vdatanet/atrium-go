package app

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/httpapi"
	"github.com/vdatanet/atrium-go/internal/surface"
)

// TestServeAnswersARequestAndStopsWhenItsContextIsCancelled is the lifecycle
// this task exists for: a server on a port the operating system chose, one
// request answered, a stop that returns before a deadline, and nothing left
// running afterwards.
//
// The handler is the one the binary actually serves, so the request also says
// what "serves nothing" means: 404, empty body, no Content-Type — the shape
// behaviours 1.11 measured for a path matching no route.
func TestServeAnswersARequestAndStopsWhenItsContextIsCancelled(t *testing.T) {
	before := runtime.NumGoroutine()

	server := newTestServer(t, testPipeline(t))
	ctx, cancel := context.WithCancel(context.Background())
	stopped := serve(t, ctx, server)

	client := newTestClient(t)
	response, err := client.Get(baseURL(server) + "/System/Ping")
	if err != nil {
		t.Fatalf("issuing a request: %v", err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("reading the body: %v", err)
	}
	response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d: nothing is routed yet", response.StatusCode, http.StatusNotFound)
	}
	if len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}
	if got, ok := response.Header["Content-Type"]; ok {
		t.Errorf("Content-Type = %q, want the header absent", got)
	}

	cancel()
	if err := waitFor(t, stopped, 5*time.Second); err != nil {
		t.Errorf("Serve returned %v, want nil", err)
	}

	client.CloseIdleConnections()
	assertNoStrayGoroutines(t, before)
}

// TestServeDrainsARequestThatIsAlreadyInFlight holds the half of a graceful
// stop that a plain listener close would not: a request that has been accepted
// is answered, not severed.
func TestServeDrainsARequestThatIsAlreadyInFlight(t *testing.T) {
	before := runtime.NumGoroutine()

	entered := make(chan struct{})
	release := make(chan struct{})
	server := newTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		io.WriteString(w, "drained")
	}))

	ctx, cancel := context.WithCancel(context.Background())
	stopped := serve(t, ctx, server)

	client := newTestClient(t)
	answered := make(chan string, 1)
	failed := make(chan error, 1)
	go func() {
		response, err := client.Get(baseURL(server) + "/")
		if err != nil {
			failed <- err
			return
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			failed <- err
			return
		}
		answered <- string(body)
	}()

	select {
	case <-entered:
	case err := <-failed:
		t.Fatalf("the request failed before the handler ran: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("the handler was never reached")
	}

	// The stop is asked for while the request is inside the handler, and the
	// handler is held there long enough that a stop which severed connections
	// instead of draining them would be visible rather than a race.
	cancel()
	time.Sleep(200 * time.Millisecond)
	close(release)

	select {
	case body := <-answered:
		if body != "drained" {
			t.Errorf("body = %q, want %q", body, "drained")
		}
	case err := <-failed:
		t.Fatalf("a request in flight was cut off by the stop: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("the request in flight was never answered")
	}

	if err := waitFor(t, stopped, 5*time.Second); err != nil {
		t.Errorf("Serve returned %v, want nil", err)
	}

	client.CloseIdleConnections()
	assertNoStrayGoroutines(t, before)
}

// TestServeGivesUpOnADrainThatOutlastsItsDeadline proves the bound is real. A
// stop nobody can rely on finishing is the shape that ends with a process
// killed mid-write, so the timeout has to be observable rather than assumed.
func TestServeGivesUpOnADrainThatOutlastsItsDeadline(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := newTestServer(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(entered)
		<-release
	}))
	server.drain = 50 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	stopped := serve(t, ctx, server)

	client := newTestClient(t)
	go func() {
		if response, err := client.Get(baseURL(server) + "/"); err == nil {
			response.Body.Close()
		}
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the handler was never reached")
	}
	cancel()

	err := waitFor(t, stopped, 5*time.Second)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve returned %v, want an error wrapping %v", err, context.DeadlineExceeded)
	}

	close(release)
	server.http.Close()
	client.CloseIdleConnections()
}

// TestNewServerRefusesAnAddressItCannotBind keeps the failure at the flag it
// came from. Binding in NewServer rather than in Serve is what makes that
// possible: the process learns the address is taken before it claims to be up.
func TestNewServerRefusesAnAddressItCannotBind(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("taking an address: %v", err)
	}
	defer taken.Close()

	cfg := testConfig(t)
	cfg.BindAddress = taken.Addr().String()

	server, err := NewServer(cfg, testLogger(), testPipeline(t))
	if err == nil {
		server.http.Close()
		t.Fatal("NewServer bound an address that was already taken")
	}
	if !strings.Contains(err.Error(), cfg.BindAddress) {
		t.Errorf("error %q does not name the address it failed to bind", err)
	}
}

// TestAddrIsWhereTheServerActuallyListens is the property a test needs to send
// a request to a server that asked for port 0.
func TestAddrIsWhereTheServerActuallyListens(t *testing.T) {
	server := newTestServer(t, testPipeline(t))
	defer server.http.Close()

	address, ok := server.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("Addr() = %T, want a *net.TCPAddr", server.Addr())
	}
	if address.Port == 0 {
		t.Error("Addr() reports port 0, which is the port that was asked for and not the one that was granted")
	}
}

// testPipeline is the pipeline the binary serves, with the gate already open —
// which is the state Run leaves it in once the start has finished.
//
// Nothing is routed yet, so every path the table names is answered by the
// router's own refusal: 404, empty body, no Content-Type (behaviours 1.11).
// That is the same shape the placeholder handler this replaced answered, which
// is why the lifecycle assertions below did not have to change.
func testPipeline(t *testing.T) *httpapi.Pipeline {
	t.Helper()
	pipeline, err := httpapi.NewPipeline(surface.V1(), httpapi.V1QuerySpellings(), nil)
	if err != nil {
		t.Fatalf("assembling the pipeline: %v", err)
	}
	pipeline.Gate().MarkReady()
	return pipeline
}

func testConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		BindAddress:     "127.0.0.1:0",
		DataDirectory:   t.TempDir(),
		LogLevel:        slog.LevelError,
		ShutdownTimeout: 5 * time.Second,
	}
}

func testLogger() *slog.Logger {
	return NewLogger(io.Discard, slog.LevelError)
}

func newTestServer(t *testing.T, handler http.Handler) *Server {
	t.Helper()
	server, err := NewServer(testConfig(t), testLogger(), handler)
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return server
}

// serve runs the server and reports what it returned. The channel is buffered
// so that a test which gives up waiting does not strand the goroutine.
func serve(t *testing.T, ctx context.Context, server *Server) <-chan error {
	t.Helper()
	stopped := make(chan error, 1)
	go func() { stopped <- server.Serve(ctx) }()
	return stopped
}

func baseURL(server *Server) string {
	return "http://" + server.Addr().String()
}

// newTestClient gives each test its own connection pool, so that one test's
// kept-alive connection is not another test's stray goroutine.
func newTestClient(t *testing.T) *http.Client {
	t.Helper()
	client := &http.Client{Transport: &http.Transport{}, Timeout: 10 * time.Second}
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func waitFor(t *testing.T, stopped <-chan error, deadline time.Duration) error {
	t.Helper()
	select {
	case err := <-stopped:
		return err
	case <-time.After(deadline):
		t.Fatalf("Serve did not return within %s of the stop", deadline)
		return nil
	}
}

// assertNoStrayGoroutines is the "no goroutine left running" half of this
// task's check. It polls rather than sampling once, because the goroutines a
// connection is made of are torn down by the runtime a moment after the stop
// they were told about.
func assertNoStrayGoroutines(t *testing.T, before int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		now := runtime.NumGoroutine()
		if now <= before {
			return
		}
		if time.Now().After(deadline) {
			stacks := make([]byte, 1<<16)
			stacks = stacks[:runtime.Stack(stacks, true)]
			t.Fatalf("%d goroutines are running after the stop, against %d before it:\n%s",
				now, before, stacks)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
