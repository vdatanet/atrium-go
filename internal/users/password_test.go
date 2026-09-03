package users_test

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/vdatanet/atrium-go/internal/users"
)

// thePassword is what every derivation below is over. It is 28 bytes, the
// length ADR-0006's own benchmark timed.
const thePassword = "correct horse battery staple"

func TestADerivedRecordVerifiesAndAWrongPasswordDoesNot(t *testing.T) {
	record, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}

	ok, _, err := users.Verify(record, users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Verify with the right password: %v", err)
	}
	if !ok {
		t.Error("the password that was derived does not verify against its own record")
	}

	ok, _, err = users.Verify(record, users.NewPlaintext(thePassword+"!"))
	if err != nil {
		t.Fatalf("Verify with the wrong password: %v", err)
	}
	if ok {
		t.Error("a wrong password verified")
	}
}

// TestTwoDerivationsOfOnePasswordDifferBecauseTheSaltIsPerCredential is the
// property every other test here would pass without: a fixed salt round-trips,
// verifies, rehashes and redacts exactly like a random one, and turns the store
// into a rainbow table's index.
func TestTwoDerivationsOfOnePasswordDifferBecauseTheSaltIsPerCredential(t *testing.T) {
	first, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	second, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if first == second {
		t.Fatal("two derivations of one password produced the same record, so the salt is not per-credential")
	}

	// Both still verify, which is the half a plain inequality does not make:
	// the salt travels inside the record rather than beside it.
	for _, record := range []string{first, second} {
		ok, _, err := users.Verify(record, users.NewPlaintext(thePassword))
		if err != nil || !ok {
			t.Errorf("a record does not verify its own password (ok %v, err %v)", ok, err)
		}
	}
}

func TestADerivedRecordCarriesTheCurrentConstants(t *testing.T) {
	record, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if !strings.HasPrefix(record, "$argon2id$v=19$m=65536,t=3,p=2$") {
		t.Errorf("a record does not carry ADR-0006's PHC prefix: %q", record)
	}
	assertParametersAreTheCurrentConstants(t, record, "a freshly derived record")

	// Unpadded base64, which is what the PHC encoding says. A padded record
	// would round-trip through this package and be unreadable everywhere
	// else, which is the whole argument for PHC over a private format.
	fields := strings.Split(record, "$")
	if len(fields) != 6 {
		t.Fatalf("a record does not have six $-separated fields: %q", record)
	}
	for _, field := range fields[4:] {
		if strings.Contains(field, "=") {
			t.Errorf("a record's base64 is padded: %q", field)
		}
	}
}

// TestTheDecoysOwnParametersEqualTheCurrentConstants is ADR-0006's rule 2
// written as a test.
//
// The decoy is what a username matching no account is verified against, so that
// a refusal for an account that does not exist costs what a refusal for a wrong
// password costs. A decoy pinned to old parameters is its own timing oracle:
// the moment the constants are raised, the account that does not exist answers
// faster than one whose record has been rehashed, and the disclosure the decoy
// closed reopens on the axis nobody is watching.
//
// This is the assertion that fails on the day somebody raises the constants
// without the decoy following. Nothing else in the suite would notice — every
// other test here derives *with* the constants and would go on passing.
func TestTheDecoysOwnParametersEqualTheCurrentConstants(t *testing.T) {
	assertParametersAreTheCurrentConstants(t, users.DecoyRecord(), "the decoy")
}

// TestTheDecoyIsDerivedFromASecretNobodyCanGuess is the other half of rule 2:
// the decoy is derived at start from random bytes and is never a literal.
func TestTheDecoyIsDerivedFromASecretNobodyCanGuess(t *testing.T) {
	decoy := users.DecoyRecord()
	if decoy == "" {
		t.Fatal("there is no decoy record")
	}
	// The empty password is the one guess worth writing down: it is what a
	// decoy derived from a zero-valued or forgotten secret answers to, and it
	// is what an unauthenticated caller sends first.
	ok, _, err := users.Verify(decoy, users.NewPlaintext(""))
	if err != nil {
		t.Fatalf("Verify against the decoy: %v", err)
	}
	if ok {
		t.Error("the empty password verifies against the decoy, so the decoy is not derived from a random secret")
	}
}

func assertParametersAreTheCurrentConstants(t *testing.T, record, what string) {
	t.Helper()
	parsed, err := users.ParseRecord(record)
	if err != nil {
		t.Fatalf("ParseRecord(%s): %v", what, err)
	}
	if parsed.MemoryKiB != users.Argon2idMemoryKiB {
		t.Errorf("%s carries m=%d, the constant is %d", what, parsed.MemoryKiB, users.Argon2idMemoryKiB)
	}
	if parsed.Time != users.Argon2idTime {
		t.Errorf("%s carries t=%d, the constant is %d", what, parsed.Time, users.Argon2idTime)
	}
	if parsed.Parallelism != users.Argon2idParallelism {
		t.Errorf("%s carries p=%d, the constant is %d", what, parsed.Parallelism, users.Argon2idParallelism)
	}
	if len(parsed.Salt) != users.Argon2idSaltLength {
		t.Errorf("%s carries a %d-byte salt, the constant is %d", what, len(parsed.Salt), users.Argon2idSaltLength)
	}
	if uint32(len(parsed.Key)) != users.Argon2idKeyLength {
		t.Errorf("%s carries a %d-byte key, the constant is %d", what, len(parsed.Key), users.Argon2idKeyLength)
	}
}

// TestARecordBelowTheConstantsNeedsARehashAndOneAtThemDoesNot is the mechanism
// that lets the constants be raised at all: an existing credential keeps
// verifying, and says so, rather than needing a migration.
//
// Each row lowers exactly one axis. A row that lowered all five at once would
// pass on a build that compared only one of them, and the axis a raise moves
// first is not knowable from here.
func TestARecordBelowTheConstantsNeedsARehashAndOneAtThemDoesNot(t *testing.T) {
	current, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	ok, needsRehash, err := users.Verify(current, users.NewPlaintext(thePassword))
	if err != nil || !ok {
		t.Fatalf("a record at the current constants does not verify (ok %v, err %v)", ok, err)
	}
	if needsRehash {
		t.Error("a record derived at the current constants reports needsRehash")
	}

	for _, weaker := range []struct {
		name   string
		record users.Record
	}{
		{"less memory", recordAt(users.Argon2idMemoryKiB/2, users.Argon2idTime, users.Argon2idParallelism, users.Argon2idSaltLength, users.Argon2idKeyLength)},
		{"fewer passes", recordAt(users.Argon2idMemoryKiB, users.Argon2idTime-1, users.Argon2idParallelism, users.Argon2idSaltLength, users.Argon2idKeyLength)},
		{"fewer lanes", recordAt(users.Argon2idMemoryKiB, users.Argon2idTime, users.Argon2idParallelism-1, users.Argon2idSaltLength, users.Argon2idKeyLength)},
		{"a shorter salt", recordAt(users.Argon2idMemoryKiB, users.Argon2idTime, users.Argon2idParallelism, users.Argon2idSaltLength-1, users.Argon2idKeyLength)},
		{"a shorter key", recordAt(users.Argon2idMemoryKiB, users.Argon2idTime, users.Argon2idParallelism, users.Argon2idSaltLength, users.Argon2idKeyLength-1)},
	} {
		ok, needsRehash, err := users.Verify(weaker.record.String(), users.NewPlaintext(thePassword))
		if err != nil {
			t.Errorf("%s: Verify: %v", weaker.name, err)
			continue
		}
		if !ok {
			t.Errorf("%s: the record does not verify its own password, so needsRehash would be read off a failure", weaker.name)
			continue
		}
		if !needsRehash {
			t.Errorf("%s: a record below the current constants does not report needsRehash", weaker.name)
		}
	}
}

// recordAt builds a working record at parameters this package would never
// choose, by deriving the key with the same library the package uses.
//
// Reaching for golang.org/x/crypto/argon2 here is deliberate: Derive only ever
// writes the constants, so there is no way through the package's own API to
// produce the older record a rehash exists for. It also makes the encoding an
// assertion — the parameters this test wrote into the string are the parameters
// Verify must have fed to the library, or the key would not match.
func recordAt(memory, passes uint32, lanes uint8, saltLength int, keyLength uint32) users.Record {
	salt := bytes.Repeat([]byte{0x5a}, saltLength)
	return users.Record{
		MemoryKiB:   memory,
		Time:        passes,
		Parallelism: lanes,
		Salt:        salt,
		Key:         argon2.IDKey([]byte(thePassword), salt, passes, memory, lanes, keyLength),
	}
}

// TestTheStoredKeyIsComparedToItsLastByte is the comparison's assertion.
//
// A record whose stored key differs from the derived one in its *last* byte
// must fail exactly as one differing in its first does. What that catches is a
// comparison that stops early on a match-so-far — a prefix comparison, a
// truncated length, a loop that forgot its last index — any of which lets a
// near-miss authenticate.
//
// What it does not catch is worth saying, because the name would otherwise
// promise more than the code delivers: bytes.Equal passes this test too, and no
// unit test in Go can observe the timing difference between it and
// crypto/subtle.ConstantTimeCompare. The reason the code uses subtle is
// ADR-0006 rule 3 and the reading of the code, not this assertion.
func TestTheStoredKeyIsComparedToItsLastByte(t *testing.T) {
	record, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	parsed, err := users.ParseRecord(record)
	if err != nil {
		t.Fatalf("ParseRecord: %v", err)
	}

	for _, position := range []struct {
		name  string
		index int
	}{
		{"the first byte", 0},
		{"the last byte", len(parsed.Key) - 1},
	} {
		flipped := bytes.Clone(parsed.Key)
		flipped[position.index] ^= 0xff
		altered := parsed
		altered.Key = flipped

		ok, _, err := users.Verify(altered.String(), users.NewPlaintext(thePassword))
		if err != nil {
			t.Errorf("%s: Verify: %v", position.name, err)
			continue
		}
		if ok {
			t.Errorf("a record whose key differs in %s verified", position.name)
		}
	}

	// A truncated key is the same failure by another route, and it is the one
	// crypto/subtle answers by length before it answers by content.
	truncated := parsed
	truncated.Key = parsed.Key[:len(parsed.Key)-1]
	ok, _, err := users.Verify(truncated.String(), users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Verify against a truncated key: %v", err)
	}
	if ok {
		t.Error("a record whose key is a prefix of the derived one verified")
	}
}

// TestFiveConcurrentDerivationsStayUnderTheCeilingAndNoneIsRefused asserts both
// halves of ADR-0006's ceiling, because a build that gets the decision wrong
// satisfies either one alone.
//
// The ceiling holds: four × 64 MiB is the reservation the deployment shape has
// to afford, the scaling was measured and is exactly linear, so a fifth in
// flight is 64 MiB nobody budgeted for.
//
// And nothing is refused: ADR-0006 chose queueing over a 503 deliberately,
// because a 503 on POST /Users/AuthenticateByName is a status the reference
// does not send there and Principle I binds the bytes even where it does not
// bind the algorithm. Latency is not a wire delta; a status code is. A limiter
// that refused would pass the first half of this test and be wrong on the wire.
func TestFiveConcurrentDerivationsStayUnderTheCeilingAndNoneIsRefused(t *testing.T) {
	const concurrent = users.DerivationCeiling + 1

	// The number itself, because every other assertion here is written in
	// terms of the constant and would therefore follow it anywhere. Four is
	// what ADR-0006 priced: 256 MiB transient and about 77 verifications a
	// second, on a host that is also running SQLite and ffmpeg. Sixteen is a
	// gibibyte, measured and linear. Moving this line is a decision that needs
	// a record superseding ADR-0006, not a tuning commit.
	if users.DerivationCeiling != 4 {
		t.Errorf("the ceiling is %d; ADR-0006 decided 4, and priced it at %d MiB transient",
			users.DerivationCeiling, users.DerivationCeiling*int(users.Argon2idMemoryKiB)/1024)
	}

	// The high-water mark this test observes for itself. The package's own
	// PeakInFlight is process-wide and monotonic, so it answers "was the
	// ceiling ever breached"; this one answers "did these five actually
	// overlap", without which the first question is vacuous.
	observed := make(chan int, 1)
	done := make(chan struct{})
	go func() {
		peak := 0
		for {
			select {
			case <-done:
				observed <- peak
				return
			default:
			}
			if inFlight := users.Derivations().InFlight; inFlight > peak {
				peak = inFlight
			}
			time.Sleep(200 * time.Microsecond)
		}
	}()

	passwordFor := func(i int) string { return fmt.Sprintf("%s %d", thePassword, i) }

	records := make([]string, concurrent)
	errs := make([]error, concurrent)
	var start sync.WaitGroup
	var finished sync.WaitGroup
	start.Add(1)
	for i := range concurrent {
		finished.Add(1)
		go func() {
			defer finished.Done()
			start.Wait()
			records[i], errs[i] = users.Derive(users.NewPlaintext(passwordFor(i)))
		}()
	}
	start.Done()
	finished.Wait()
	close(done)
	observedPeak := <-observed

	// None is refused: every one of the five returns a record that works.
	for i := range concurrent {
		if errs[i] != nil {
			t.Errorf("derivation %d was refused: %v", i, errs[i])
			continue
		}
		ok, _, err := users.Verify(records[i], users.NewPlaintext(passwordFor(i)))
		if err != nil || !ok {
			t.Errorf("derivation %d did not produce a working record (ok %v, err %v)", i, ok, err)
		}
	}

	// The ceiling holds, process-wide and for the whole run — the counter is a
	// high-water mark, so a breach in any earlier test is still visible here.
	if peak := users.Derivations().PeakInFlight; peak > users.DerivationCeiling {
		t.Errorf("%d derivations were in flight at once, the ceiling is %d", peak, users.DerivationCeiling)
	}

	// And the five really did contend. Without this, the assertion above is
	// satisfied by a build that ran them one after another, which proves
	// nothing at all about a ceiling.
	if observedPeak < 2 {
		t.Errorf("the %d derivations never overlapped (peak in flight %d), so the ceiling assertion above proved nothing",
			concurrent, observedPeak)
	}
	if observedPeak > users.DerivationCeiling {
		t.Errorf("this test observed %d derivations in flight, the ceiling is %d", observedPeak, users.DerivationCeiling)
	}
}

// TestAPlaintextRedactsItselfThroughEveryVerbAndThroughSlog is 002 AC-11's
// mechanism: a password never appears in any log record at any level and never
// in an error body.
//
// The assertions are over the ways a password actually escapes — somebody
// formats the value, somebody formats the struct that holds it, or somebody
// hands either to slog — because the criterion is about what a careless caller
// produces, not about what a careful one avoids.
func TestAPlaintextRedactsItselfThroughEveryVerbAndThroughSlog(t *testing.T) {
	const secret = "hunter2-and-a-tail-nobody-would-choose"
	password := users.NewPlaintext(secret)

	// A struct holding one, because "never stored in a field of anything that
	// is logged whole" is a rule about callers, and this is what happens the
	// first time one forgets it.
	holder := struct {
		Username string
		Pw       users.Plaintext
	}{Username: "alice", Pw: password}

	for _, formatted := range []struct {
		what   string
		actual string
	}{
		{"%v", fmt.Sprintf("%v", password)},
		{"%s", fmt.Sprintf("%s", password)},
		{"%q", fmt.Sprintf("%q", password)},
		{"%#v", fmt.Sprintf("%#v", password)},
		{"%v of a struct holding one", fmt.Sprintf("%v", holder)},
		{"%+v of a struct holding one", fmt.Sprintf("%+v", holder)},
		{"%#v of a struct holding one", fmt.Sprintf("%#v", holder)},
		{"String()", password.String()},
		{"an error wrapping one", fmt.Errorf("authenticating %s: %v", holder.Username, password).Error()},
	} {
		if strings.Contains(formatted.actual, secret) {
			t.Errorf("%s leaked the password: %s", formatted.what, formatted.actual)
		}
		if !strings.Contains(formatted.actual, users.Redaction) {
			t.Errorf("%s did not carry the redaction: %s", formatted.what, formatted.actual)
		}
	}

	// Through a real slog handler, at every level, as an attribute, inside a
	// group and inside a struct — slog resolves a LogValuer wherever it finds
	// one, and the claim is that there is nowhere it does not.
	for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		var buffer bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buffer, &slog.HandlerOptions{Level: slog.LevelDebug}))
		logger.Log(t.Context(), level, "authenticating",
			"pw", password,
			"holder", holder,
			slog.Group("request", "pw", password),
		)
		if strings.Contains(buffer.String(), secret) {
			t.Errorf("slog at %s leaked the password: %s", level, buffer.String())
		}
		if !strings.Contains(buffer.String(), users.Redaction) {
			t.Errorf("slog at %s did not carry the redaction: %s", level, buffer.String())
		}
	}
}

func TestAMalformedRecordIsRefusedRatherThanVerified(t *testing.T) {
	valid, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	fields := strings.Split(valid, "$")

	for _, malformed := range []struct {
		name   string
		record string
	}{
		{"empty", ""},
		{"not a record at all", "hunter2"},
		{"no leading dollar", strings.TrimPrefix(valid, "$")},
		{"a different algorithm", "$argon2i$" + strings.Join(fields[2:], "$")},
		{"a different version", "$argon2id$v=16$" + strings.Join(fields[3:], "$")},
		{"two parameters", "$argon2id$v=19$m=65536,t=3$" + strings.Join(fields[4:], "$")},
		{"the parameters in another order", "$argon2id$v=19$t=3,m=65536,p=2$" + strings.Join(fields[4:], "$")},
		{"zero passes", "$argon2id$v=19$m=65536,t=0,p=2$" + strings.Join(fields[4:], "$")},
		{"zero lanes", "$argon2id$v=19$m=65536,t=3,p=0$" + strings.Join(fields[4:], "$")},
		{"a negative parameter", "$argon2id$v=19$m=65536,t=-1,p=2$" + strings.Join(fields[4:], "$")},
		{"padded base64", strings.Join(fields[:4], "$") + "$" +
			base64.StdEncoding.EncodeToString([]byte("0123456789abcdef")) + "$" + fields[5]},
		{"an empty key", strings.Join(fields[:5], "$") + "$"},
		{"one field too many", valid + "$extra"},
	} {
		if _, err := users.ParseRecord(malformed.record); !errors.Is(err, users.ErrMalformedRecord) {
			t.Errorf("ParseRecord(%s) returned %v, expected ErrMalformedRecord", malformed.name, err)
		}
		ok, _, err := users.Verify(malformed.record, users.NewPlaintext(thePassword))
		if ok {
			t.Errorf("Verify against %s reported success", malformed.name)
		}
		if !errors.Is(err, users.ErrMalformedRecord) {
			t.Errorf("Verify against %s returned %v, expected ErrMalformedRecord", malformed.name, err)
		}
	}
}

func TestARecordRoundTripsThroughItsOwnEncoding(t *testing.T) {
	record, err := users.Derive(users.NewPlaintext(thePassword))
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	parsed, err := users.ParseRecord(record)
	if err != nil {
		t.Fatalf("ParseRecord: %v", err)
	}
	if parsed.String() != record {
		t.Errorf("a parsed record re-encodes as %q, not %q", parsed.String(), record)
	}
}
