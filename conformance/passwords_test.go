package conformance_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// 002 AC-11 against the running binary: **a password never appears in any log
// record at any level, and never in an error body.**
//
// # Why this file exists, and what it corrects
//
// AC-11 was proven, until 002's closing audit, by
// `TestAPlaintextRedactsItselfThroughEveryVerbAndThroughSlog` in
// `internal/users` — a thorough, correct test of the **mechanism**
// `users.Plaintext`: every formatting verb, a struct holding one, an error
// wrapping one, and a real `slog` handler at four levels. It proves that a
// caller who reaches for the redacting type cannot leak through it.
//
// It says nothing about the criterion, because the criterion is about a
// **request**. The login route reads its body into a struct whose `Pw` is an
// ordinary `string` — it has to be, that is what `encoding/json` fills — and
// `users.NewPlaintext` is called on it one line later. **Between those two
// lines the password is a plain string in a struct nothing forbids logging**,
// and a single `slog.Info("authenticating", "body", body)` at that point
// printed the password to the server's standard error and left the entire
// suite green `[measurement: mutation of internal/httpapi.UsersHandler.AuthenticateByName,
// Go 1.27.0, 2026-09-03]`. That is 001's own closing finding — a criterion
// about a request proven about the mechanism that serves it — one feature on.
//
// # The three controls, because an assertion of absence is the easiest green
//
// A search that finds nothing proves nothing until it is shown that it could
// have found something. This test therefore asserts three things it expects to
// be **present**, beside the one it expects to be absent:
//
//  1. **The server is running at `debug`.** AC-11 says *at any level*, and a
//     fixture pinned to `info` cannot fail on a `Debug` line carrying a
//     password. The debug-only "store opened" record is what proves the level
//     really moved.
//  2. **The log was collected.** The harness reads standard error in a
//     goroutine; a run in which nothing was read would satisfy the absence
//     trivially. The startup records prove it was.
//  3. **The body search works.** The account's own name is a marker of exactly
//     the same shape as the password — an unusual ASCII string, in the same
//     request — that the server is *supposed* to disclose. Finding it in
//     `/Users/Public` proves the search is looking where the password would be
//     if it travelled.
func TestAPasswordReachesNoLogRecordAndNoResponseBody(t *testing.T) {
	t.Parallel()

	const (
		// Distinctive on purpose. A password like "hunter2" can occur in a log
		// or a body for some other reason, and then a failure here would be
		// ambiguous and a pass would be luck.
		secret = "correct-horse-battery-staple-Q7vN2xZ"

		// The control marker: the same shape of string, in the same requests,
		// which the server *does* disclose (spec 3.5's `Name`).
		account = "audible-marker-K3pL9wR"

		device = "device-passwords"
	)

	server := startServer(t,
		// The whole point of the file (control 1).
		withLogLevel("debug"),
		// `--hidden=false` because a bare `user add` is hidden and control 3
		// reads the account's name off /Users/Public.
		withProvisionedAccount(account, secret+"\n", "--hidden=false"),
	)

	// Every request in this feature that carries a password, including the
	// three that are refused. A refusal is where a careless implementation
	// leaks: the successful path has a body to build and the refused one has
	// only a message, and a message is where somebody puts "wrong password for
	// %q: %s".
	for _, sent := range []struct {
		what   string
		body   []byte
		status int
	}{
		{"a successful authentication", loginBody(account, secret), http.StatusOK},
		{"a wrong password on an enabled account",
			loginBody(account, secret+"-wrong"), http.StatusUnauthorized},
		{"the same password against a username nobody has",
			loginBody("nobody-"+account, secret), http.StatusUnauthorized},

		// The one request where the refusal quotes what it could not read:
		// the body is the password itself, so a deserialiser that echoed its
		// input would put it on the wire. behaviours 1.11's validation shape
		// carries the parser's own words under `"$"`.
		{"a body that is not a JSON document, and is the password",
			[]byte(secret), http.StatusBadRequest},

		// And the same, as valid JSON whose Pw cannot bind — a second
		// deserialiser message, on a different failure.
		{"a body whose Pw is not a string",
			[]byte(`{"Username":"` + account + `","Pw":["` + secret + `"]}`), http.StatusBadRequest},
	} {
		t.Run(sent.what, func(t *testing.T) {
			got := server.send(t, http.MethodPost, authenticateByNamePath, goldenHost,
				http.Header{"Authorization": {clientIdentification(device, "")}}, sent.body)
			if got.status != sent.status {
				t.Errorf("%s answered %d, want %d\nbody: %s", sent.what, got.status, sent.status, got.body)
			}
			if strings.Contains(string(got.body), secret) {
				t.Errorf("%s answered a body carrying the password:\n%s", sent.what, got.body)
			}
		})
	}

	// Control 3, and the assertion that the search above is looking somewhere a
	// string can actually be found.
	t.Run("the search finds a marker the server does disclose", func(t *testing.T) {
		public := server.get(t, publicUsersPath, goldenHost, nil)
		if public.status != http.StatusOK {
			t.Fatalf("%s answered %d, want 200\nbody: %s", publicUsersPath, public.status, public.body)
		}
		if !strings.Contains(string(public.body), account) {
			t.Fatalf("%s does not carry the account name, so the substring search over the "+
				"bodies above proved nothing:\n%s", publicUsersPath, public.body)
		}
		if strings.Contains(string(public.body), secret) {
			t.Errorf("%s carries the password:\n%s", publicUsersPath, public.body)
		}
	})

	t.Run("no log record at any level carries the password", func(t *testing.T) {
		// The collector reads standard error in a goroutine, so a line written
		// while a request above was being served may not have been read yet.
		// Without this the absence below would be an absence from lines this
		// test never saw.
		server.log.settled(logQuietPeriod, logSettleCeiling)
		recorded := server.log.text()

		// Control 2: something was collected at all.
		if !strings.Contains(recorded, "msg=ready") {
			t.Fatalf("the server's log carries no startup record, so nothing was collected "+
				"and the absence below asserts nothing:\n%s", recorded)
		}
		// Control 1: and it was collected at debug, which is the level AC-11's
		// "at any level" is really about — everything above debug is emitted
		// whatever this flag says.
		if !strings.Contains(recorded, `level=DEBUG`) {
			t.Fatalf("the server's log carries no DEBUG record, so it is not running at debug "+
				"and a Debug line carrying a password would never reach this test:\n%s", recorded)
		}

		for _, line := range strings.Split(recorded, "\n") {
			if strings.Contains(line, secret) {
				t.Errorf("a log record carries the password: %s", line)
			}
		}
	})
}

// How long the log has to stay quiet before it counts as settled, and how long
// this package will wait for that.
//
// The quiet period is generous because it is paid once per test that reads the
// log — one, today — and a period too short turns the assertion above back into
// the thing it exists to prevent.
const (
	logQuietPeriod   = 200 * time.Millisecond
	logSettleCeiling = 3 * time.Second
)
