package conformance_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// atriumPackage is the binary these tests exercise, named by its import path so
// that this file holds no relative path into the repository's layout.
//
// It is a *string*, not an import: this package may not import anything under
// internal/, and the server it talks to therefore has to be a process rather
// than a value (see doc.go).
const atriumPackage = "github.com/vdatanet/atrium-go/cmd/atrium"

// atriumBinary is the built server, shared by every test in the package. One
// build, many servers.
var atriumBinary string

// startTimeout bounds how long a server may take to report that it is
// listening. It is generous because the first start of a fresh installation
// creates an identity file and applies every migration; it exists so that a
// server which will never listen fails the run instead of hanging it.
const startTimeout = 30 * time.Second

// stopTimeout bounds the drain after SIGTERM. The server's own shutdown budget
// is smaller, so exceeding this means it did not stop at all.
const stopTimeout = 30 * time.Second

func TestMain(m *testing.M) {
	code, err := run(m)
	if err != nil {
		fmt.Fprintln(os.Stderr, "conformance:", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(m *testing.M) (int, error) {
	directory, err := os.MkdirTemp("", "atrium-conformance-")
	if err != nil {
		return 0, fmt.Errorf("making a directory for the binary: %w", err)
	}
	defer os.RemoveAll(directory)

	atriumBinary = filepath.Join(directory, "atrium")
	build := exec.Command("go", "build", "-o", atriumBinary, atriumPackage)
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return 0, fmt.Errorf("building %s: %w", atriumPackage, err)
	}

	return m.Run(), nil
}

// server is one running Atrium, and the address it answers on.
type server struct {
	// baseURL is http://127.0.0.1:<port>, with no trailing slash.
	baseURL string

	// dataDirectory is the installation's own directory. A test may look in it
	// — the operator can — but nothing here reads a value out of it to assert
	// on: what the server holds is only interesting through the wire.
	dataDirectory string

	// seeded names everything that was in the data directory when the server
	// started, which is what the options put there and nothing else.
	//
	// It exists so that "before any user exists and before any library is
	// configured" (AC-2, AC-3) is an assertion rather than a claim: a test that
	// means an empty installation asserts this is empty, and stops being that
	// test the day somebody gives it an option.
	seeded []string

	log *log
}

// serverOption prepares the data directory before the server is started, which
// is the only way this package can put an installation into a chosen state: it
// cannot reach into the store, so it arranges what is on disk and starts the
// server on it.
type serverOption func(t *testing.T, dataDirectory string)

// withInstallationIdentity writes the identity file the server will read
// instead of generating one.
//
// It is what makes a byte-compared golden possible at all: the identity is 16
// cryptographically random bytes on a first start, so a body carrying it can
// only be held still by stating it. The file name is spelled here rather than
// imported, which is the boundary doing its job — and if it is ever renamed,
// this test fails loudly with a generated identity in the body rather than
// passing for the wrong reason.
func withInstallationIdentity(id string) serverOption {
	return func(t *testing.T, dataDirectory string) {
		t.Helper()
		path := filepath.Join(dataDirectory, "installation-id")
		if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
			t.Fatalf("writing the installation identity: %v", err)
		}
	}
}

// startServer starts one Atrium on an empty data directory and returns once it
// is listening. It is stopped, and its directory removed, when the test ends.
//
// The port is 0, so the operating system chooses one and no two tests can
// collide over it. The address is read back out of the server's own log, which
// is the only place a caller that did not choose the port can learn it.
func startServer(t *testing.T, options ...serverOption) *server {
	t.Helper()

	// t.TempDir() already exists, which is what EnsureDataDirectory needs: it
	// creates the final component only and refuses a missing parent.
	dataDirectory := t.TempDir()
	for _, option := range options {
		option(t, dataDirectory)
	}

	seeded := directoryContents(t, dataDirectory)

	command := exec.Command(atriumBinary,
		"--data-dir", dataDirectory,
		"--bind-address", "127.0.0.1:0",
		"--log-level", "info",
	)
	// A developer's own ATRIUM_* variables must not reach a test server: every
	// one of them is a configuration fallback, and a test that answers
	// differently on one machine has measured that machine.
	command.Env = withoutAtriumEnvironment(os.Environ())

	stderr, err := command.StderrPipe()
	if err != nil {
		t.Fatalf("opening the server's log: %v", err)
	}

	collected := &log{}
	listening := collected.collect(stderr)

	if err := command.Start(); err != nil {
		t.Fatalf("starting %s: %v", atriumBinary, err)
	}

	stopped := make(chan error, 1)
	t.Cleanup(func() {
		if err := command.Process.Signal(syscall.SIGTERM); err != nil {
			t.Errorf("signalling the server to stop: %v", err)
		}
		select {
		case err := <-stopped:
			if err != nil {
				t.Errorf("the server exited with an error: %v\n%s", err, collected.text())
			}
		case <-time.After(stopTimeout):
			_ = command.Process.Kill()
			t.Errorf("the server did not stop within %s\n%s", stopTimeout, collected.text())
		}
	})
	go func() { stopped <- command.Wait() }()

	select {
	case address := <-listening:
		return &server{baseURL: "http://" + address, dataDirectory: dataDirectory, seeded: seeded, log: collected}
	case err := <-stopped:
		t.Fatalf("the server stopped before it listened (%v):\n%s", err, collected.text())
	case <-time.After(startTimeout):
		t.Fatalf("the server did not listen within %s:\n%s", startTimeout, collected.text())
	}
	return nil
}

// directoryContents names what is in a directory, sorted, so that a test can
// state what an installation started with.
func directoryContents(t *testing.T, directory string) []string {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatalf("reading %s: %v", directory, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

// get issues one GET to the running server.
//
// host, when it is not empty, becomes the request's Host header — which is what
// spec 3.4's second tier answers with, and therefore what makes LocalAddress
// something a test can state rather than something it has to accept. The URL
// still names the loopback address the server is actually listening on, so the
// request goes where it was sent and the server is told what it was called.
func (s *server) get(t *testing.T, path, host string, header http.Header) *response {
	t.Helper()
	return s.do(t, http.MethodGet, path, host, header)
}

// do issues one request of any method.
//
// It exists because /System/Ping is answered on two methods and spec 3.3 gives
// them one response, so "both methods answer the same bytes" has to be a thing
// a test can send rather than a thing it can only assert about GET. The body is
// always empty: no route in 001 takes one, and POST /System/Ping is a request
// with no parameters at all.
func (s *server) do(t *testing.T, method, path, host string, header http.Header) *response {
	t.Helper()
	return s.send(t, method, path, host, header, nil)
}

// send is do with a request body.
//
// It exists because 002's three writes are the first routes in v1 that read
// one, and a helper that could not carry a body would mean a second way of
// issuing a request in this package. The body is bytes rather than a reader
// for the reason a response's is: what a test states is what goes on the wire.
func (s *server) send(t *testing.T, method, path, host string, header http.Header, body []byte) *response {
	t.Helper()

	var payload io.Reader
	if body != nil {
		payload = bytes.NewReader(body)
	}

	request, err := http.NewRequest(method, s.baseURL+path, payload)
	if err != nil {
		t.Fatalf("building a %s request for %s: %v", method, path, err)
	}
	if host != "" {
		request.Host = host
	}
	for name, values := range header {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}

	// A client of its own, so that nothing is shared with another test and
	// nothing is left connected when this one ends.
	client := &http.Client{Timeout: 30 * time.Second}
	defer client.CloseIdleConnections()

	got, err := client.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", method, path, err, s.log.text())
	}
	defer got.Body.Close()

	answered, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("reading the body of %s %s: %v", method, path, err)
	}
	return &response{status: got.StatusCode, header: got.Header, body: answered}
}

// response is one answer, with its body kept as bytes.
//
// Principle VIII: casing, null-versus-absent and numeric type are all invisible
// once a body is parsed, so what a test is handed is what came off the socket.
type response struct {
	status int
	header http.Header
	body   []byte
}

// log collects a server's standard error and answers when it says it is
// listening.
type log struct {
	mu    sync.Mutex
	lines []string
}

// listeningLine matches the server's own "listening" record and captures the
// address it bound. slog's text handler writes key=value pairs, and an address
// carries no space, so it is unquoted.
var listeningLine = regexp.MustCompile(`msg=listening.*?address="?([^" ]+)`)

// collect reads the server's log in the background and signals the address the
// first time the server says it is listening.
func (l *log) collect(r io.Reader) <-chan string {
	listening := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			line := scanner.Text()
			l.mu.Lock()
			l.lines = append(l.lines, line)
			l.mu.Unlock()
			if match := listeningLine.FindStringSubmatch(line); match != nil {
				select {
				case listening <- match[1]:
				default:
				}
			}
		}
	}()
	return listening
}

// text is everything the server has logged so far, for a failure message.
func (l *log) text() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.lines, "\n")
}

// withoutAtriumEnvironment removes this project's own configuration fallbacks
// from an environment.
func withoutAtriumEnvironment(environment []string) []string {
	kept := make([]string, 0, len(environment))
	for _, variable := range environment {
		if strings.HasPrefix(variable, "ATRIUM_") {
			continue
		}
		kept = append(kept, variable)
	}
	return kept
}
