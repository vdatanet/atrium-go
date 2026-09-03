package users

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync/atomic"

	"golang.org/x/crypto/argon2"
)

// The Argon2id parameters, which are constants and not settings.
//
// ADR-0006 fixes them at m = 64 MiB, t = 3, p = 2, a 16-byte salt and a 32-byte
// key, on a table measured on the machine that took the decision rather than on
// a recommendation: at 52.4 ms of this server's time they force a 64 MiB
// footprint on every guess an attacker holding the store makes, against
// bcrypt's 4 KiB and PBKDF2's nothing
// [measurement: golang.org/x/crypto v0.55.0, Go 1.27.0, 2026-09-03].
//
// They are constants because an operator who lowers them silently weakens every
// credential written afterwards, and because raising them needs no setting: a
// record carries its own parameters (Record below), so a raise costs a code
// change and a slow migration measured in logins, not a schema change.
//
// architecture §9 keeps process settings few and says they are not a feature.
// This is one of the things that stays out of them.
const (
	// Argon2idMemoryKiB is m, in kibibytes: 64 MiB.
	Argon2idMemoryKiB uint32 = 64 * 1024
	// Argon2idTime is t, the number of passes. It is the knob that makes a
	// verification slower without making it bigger, which is the property
	// ADR-0006 chose Argon2id over scrypt for.
	Argon2idTime uint32 = 3
	// Argon2idParallelism is p, the number of lanes.
	Argon2idParallelism uint8 = 2
	// Argon2idSaltLength is the salt's length in bytes.
	Argon2idSaltLength int = 16
	// Argon2idKeyLength is the derived key's length in bytes.
	Argon2idKeyLength uint32 = 32
)

// DerivationCeiling is how many Argon2id derivations may run at once.
//
// It is part of the decision and not a tuning detail. ADR-0006 measured the
// scaling and found it exactly linear — n derivations hold n × 64 MiB live
//
//	 4 simultaneous: 102.0 ms wall, peak live heap  256 MiB
//	 8 simultaneous: 186.9 ms wall, peak live heap  512 MiB
//	16 simultaneous: 450.5 ms wall, peak live heap 1024 MiB
//
// [measurement: golang.org/x/crypto/argon2 v0.55.0, Go 1.27.0, 2026-09-03] —
// so the login route is a memory lever an unauthenticated caller can pull, and
// the decoy below makes it worse rather than better: an attacker does not even
// need a valid username to make the server allocate. Four is the whole
// mitigation, and it prices at 256 MiB transient and about 77 verifications a
// second, far above any real login rate for a media server whose sessions
// survive the app closing.
//
// The ceiling is in this package rather than at the handler because it is a
// memory reservation, and a limiter at the handler would not bound the
// provisioning command, which derives with the same parameters (plan §6.4).
const DerivationCeiling = 4

// decoySecretLength is how many random bytes the decoy record is derived from
// (plan §6.4).
const decoySecretLength = 32

// Redaction is what a Plaintext prints as, whatever is asked of it.
const Redaction = "[redacted]"

// ErrMalformedRecord is returned by ParseRecord, and by Verify, for anything
// that is not a PHC Argon2id record this package could have written.
var ErrMalformedRecord = errors.New("not a PHC argon2id password record")

// Plaintext is a password in the clear.
//
// # Why it is a type and not a string
//
// 002 AC-11 requires that a password never appears in any log record at any
// level and never in an error body. A string in a struct satisfies that only
// until somebody logs the struct — and logging a whole struct is the normal
// thing to do with slog, not a mistake somebody has to go out of their way to
// make. So the plaintext is carried in a type that will not print itself: its
// String, GoString and LogValue all return Redaction, which covers %v, %s, %q,
// %#v and every slog handler at once.
//
// GoString is there for the same reason as String rather than for tidiness.
// The field below is unexported, so without it %#v prints
// users.Plaintext{value:"the password"} — the one verb that reaches past a
// Stringer.
//
// # What it does not claim
//
// It does not scrub. A garbage-collected runtime copies, and the copies are not
// reachable to zero; ADR-0006 says so rather than pretending otherwise. The
// mitigations are that the plaintext lives for one request and that it will not
// print itself.
type Plaintext struct {
	value string
}

// These are the compile-time half of the claim above. A type that stopped
// implementing slog.LogValuer would still compile and would start logging the
// password, because slog falls back to formatting whatever it was handed.
var (
	_ slog.LogValuer = Plaintext{}
	_ fmt.Stringer   = Plaintext{}
	_ fmt.GoStringer = Plaintext{}
)

// NewPlaintext wraps a password read from a request body, from standard input,
// or from anywhere else it arrived in the clear.
func NewPlaintext(password string) Plaintext {
	return Plaintext{value: password}
}

// String satisfies fmt.Stringer, and answers %v and %s.
func (p Plaintext) String() string { return Redaction }

// GoString answers %#v, which reaches past String into the struct's fields.
func (p Plaintext) GoString() string { return Redaction }

// LogValue satisfies slog.LogValuer, which every handler resolves before it
// formats.
func (p Plaintext) LogValue() slog.Value { return slog.StringValue(Redaction) }

// IsEmpty reports whether there is no password here at all.
//
// ADR-0006 rule 4: an account with no password is deliberately not equalised,
// because HasPassword is already sent for every account on the unauthenticated
// GET /Users/Public (spec §3.4), so equalising it would protect a fact the
// reference publishes on an open route.
func (p Plaintext) IsEmpty() bool { return p.value == "" }

// Record is a parsed PHC record: the parameters a stored credential was derived
// with, travelling with the credential itself.
//
//	$argon2id$v=19$m=65536,t=3,p=2$<base64 salt>$<base64 key>
//
// # Why the parameters travel
//
// They are read out of the record at verification time and never out of the
// constants above, which is the whole mechanism by which the constants can be
// raised: a credential written at t=3 keeps verifying after the constant
// becomes t=4, with no migration and no schema change. ADR-0003 puts this
// column in the precious half of the store — the half that is migrated forward
// and never rebuilt — and a parameter raise touches that half without being a
// migration.
//
// The reference stores its own hashes the same way, in its own dialect
// ($PBKDF2-SHA512$iterations=210000$<hex salt>$<hex hash>), and uses it to keep
// verifying legacy records written by older versions
// [source: MediaBrowser.Model/Cryptography/PasswordHash.cs:169-209 @ v10.11.11].
// That is a mechanism adopted for what it does, not a format copied: the
// encoding here is PHC's, which every other Argon2 implementation reads, so a
// credential this project wrote is not one only this project can read.
type Record struct {
	// MemoryKiB is m, in kibibytes.
	MemoryKiB uint32
	// Time is t, the number of passes.
	Time uint32
	// Parallelism is p, the number of lanes.
	Parallelism uint8
	// Salt is the per-credential salt.
	Salt []byte
	// Key is the derived key this record was written with.
	Key []byte
}

// String encodes the record in PHC form, with unpadded base64 as the encoding
// requires.
func (r Record) String() string {
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, r.MemoryKiB, r.Time, r.Parallelism,
		base64.RawStdEncoding.EncodeToString(r.Salt),
		base64.RawStdEncoding.EncodeToString(r.Key),
	)
}

// belowCurrentConstants reports whether this record was derived with anything
// weaker than the constants above, on any axis.
//
// It is "below on any axis" rather than "different from", because the direction
// the constants move is up: a raise must rehash, and a record that is somehow
// stronger than the current constants must not be quietly weakened to match
// them.
func (r Record) belowCurrentConstants() bool {
	return r.MemoryKiB < Argon2idMemoryKiB ||
		r.Time < Argon2idTime ||
		r.Parallelism < Argon2idParallelism ||
		len(r.Salt) < Argon2idSaltLength ||
		uint32(len(r.Key)) < Argon2idKeyLength
}

// ParseRecord reads a stored credential.
//
// It is strict about every field, because a record that parsed loosely would be
// a record whose parameters could be read as something other than what wrote
// it, and the parameters are the security property here rather than metadata.
func ParseRecord(s string) (Record, error) {
	// The leading $ makes the first field empty, which is what the encoding
	// says and what makes a record without it malformed rather than shifted.
	fields := strings.Split(s, "$")
	if len(fields) != 6 || fields[0] != "" {
		return Record{}, fmt.Errorf("%w: expected six $-separated fields, found %d", ErrMalformedRecord, len(fields))
	}
	if fields[1] != "argon2id" {
		return Record{}, fmt.Errorf("%w: algorithm is %q, not argon2id", ErrMalformedRecord, fields[1])
	}
	version, err := namedNumber(fields[2], "v")
	if err != nil {
		return Record{}, err
	}
	if version != argon2.Version {
		return Record{}, fmt.Errorf("%w: version is %d, not %d", ErrMalformedRecord, version, argon2.Version)
	}

	parameters := strings.Split(fields[3], ",")
	if len(parameters) != 3 {
		return Record{}, fmt.Errorf("%w: expected three parameters, found %d", ErrMalformedRecord, len(parameters))
	}
	memory, err := namedNumber(parameters[0], "m")
	if err != nil {
		return Record{}, err
	}
	passes, err := namedNumber(parameters[1], "t")
	if err != nil {
		return Record{}, err
	}
	lanes, err := namedNumber(parameters[2], "p")
	if err != nil {
		return Record{}, err
	}
	// The lower bounds are not pedantry: golang.org/x/crypto/argon2 panics on
	// zero passes or zero lanes, so a record that parsed with either would
	// turn a corrupt row in the store into a crash on the login path.
	if memory < 1 || passes < 1 || lanes < 1 {
		return Record{}, fmt.Errorf("%w: m, t and p must all be at least 1", ErrMalformedRecord)
	}
	if memory > int64(^uint32(0)) || passes > int64(^uint32(0)) || lanes > 255 {
		return Record{}, fmt.Errorf("%w: a parameter is out of range", ErrMalformedRecord)
	}

	salt, err := base64.RawStdEncoding.DecodeString(fields[4])
	if err != nil {
		return Record{}, fmt.Errorf("%w: the salt is not unpadded base64", ErrMalformedRecord)
	}
	key, err := base64.RawStdEncoding.DecodeString(fields[5])
	if err != nil {
		return Record{}, fmt.Errorf("%w: the key is not unpadded base64", ErrMalformedRecord)
	}
	if len(salt) == 0 || len(key) == 0 {
		return Record{}, fmt.Errorf("%w: the salt and the key may not be empty", ErrMalformedRecord)
	}

	return Record{
		MemoryKiB:   uint32(memory),
		Time:        uint32(passes),
		Parallelism: uint8(lanes),
		Salt:        salt,
		Key:         key,
	}, nil
}

// namedNumber reads one "name=number" field, and refuses a different name.
func namedNumber(field, name string) (int64, error) {
	prefix := name + "="
	if !strings.HasPrefix(field, prefix) {
		return 0, fmt.Errorf("%w: expected %q, found %q", ErrMalformedRecord, prefix, field)
	}
	number, err := strconv.ParseInt(strings.TrimPrefix(field, prefix), 10, 64)
	if err != nil || number < 0 {
		return 0, fmt.Errorf("%w: %q is not a non-negative number", ErrMalformedRecord, field)
	}
	return number, nil
}

// Derive returns the PHC record for a password, using the current constants.
//
// It blocks while DerivationCeiling derivations are already in flight. It never
// refuses: a 503 on POST /Users/AuthenticateByName is a status the reference
// does not send there, and Principle I binds that even though it does not bind
// the choice of algorithm. Latency is not a wire delta and a new status code
// is, so a flood makes Atrium slow rather than makes Atrium wrong.
func Derive(password Plaintext) (string, error) {
	salt := make([]byte, Argon2idSaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generating a password salt: %w", err)
	}
	record := Record{
		MemoryKiB:   Argon2idMemoryKiB,
		Time:        Argon2idTime,
		Parallelism: Argon2idParallelism,
		Salt:        salt,
	}
	record.Key = derive(password, record, Argon2idKeyLength)
	return record.String(), nil
}

// Verify reports whether a password derives the key a stored record holds, and
// whether that record was written with parameters below the current constants.
//
// needsRehash is only meaningful when ok: ADR-0006 makes a successful login the
// only moment the plaintext exists, so it is the only moment a rehash is
// possible, and the caller re-derives inside the same request (plan §6.4).
//
// err is non-nil only for a record this package could not have written; a wrong
// password is (false, _, nil) and not an error.
func Verify(record string, password Plaintext) (ok bool, needsRehash bool, err error) {
	stored, err := ParseRecord(record)
	if err != nil {
		return false, false, err
	}
	derived := derive(password, stored, uint32(len(stored.Key)))
	// crypto/subtle and not bytes.Equal. The reference compares with
	// SequenceEqual, which short-circuits
	// [source: Emby.Server.Implementations/Cryptography/CryptographyProvider.cs:42 @ v10.11.11];
	// diverging here is free, because the comparison is not a byte on the
	// wire, and it is recorded so that nobody later reads the reference's
	// code and assumes this project matched it.
	ok = subtle.ConstantTimeCompare(derived, stored.Key) == 1
	return ok, stored.belowCurrentConstants(), nil
}

// DecoyRecord is the record an unknown username is verified against.
//
// ADR-0006 rule 1: without it the unknown-username 401 returns in microseconds
// and the wrong-password 401 returns in 52 ms, and two refusals spec §3.3 made
// byte-identical become distinguishable with a stopwatch.
//
// It is derived once at start rather than lazily, so that the first
// unknown-username request is not the one that pays for it — which would be an
// oracle in the first second of every process (plan §6.4).
func DecoyRecord() string { return decoyRecord }

var decoyRecord string

// init derives the decoy from decoySecretLength random bytes with the current
// constants.
//
// Random and not a literal, and *derived* rather than a stored string, because
// of ADR-0006 rule 2: a decoy pinned to old parameters becomes its own oracle
// the moment the constants are raised, since the account that does not exist
// would then answer faster than one whose record has been rehashed. The test
// that holds this is the one that compares this record's own parameters against
// the constants above.
//
// It panics rather than degrading. A process that could not derive its decoy
// would serve an unknown-username refusal in microseconds, which is exactly the
// disclosure the decoy exists to close, and it would do so silently.
func init() {
	secret := make([]byte, decoySecretLength)
	if _, err := rand.Read(secret); err != nil {
		panic("users: deriving the password decoy: " + err.Error())
	}
	record, err := Derive(NewPlaintext(string(secret)))
	if err != nil {
		panic("users: deriving the password decoy: " + err.Error())
	}
	decoyRecord = record
}

// derivationSlots is the ceiling: a buffered channel of DerivationCeiling
// tokens, acquired around the derivation and nothing else.
var derivationSlots = make(chan struct{}, DerivationCeiling)

var (
	derivationsCompleted atomic.Uint64
	derivationsInFlight  atomic.Int64
	derivationsPeak      atomic.Int64
)

// DerivationStats is what the process has spent on Argon2id so far.
//
// It is exported because ADR-0006's equalisation is a claim about *how many*
// derivations a path runs — exactly one for a username matching no account,
// none at all for a disabled or locked-out one — and a claim of that shape is
// checkable only against a count. plan §8.1 makes that count the mechanism half
// of the one check ADR-0006's argument stands on.
type DerivationStats struct {
	// Completed is how many derivations have finished.
	Completed uint64
	// InFlight is how many are inside the ceiling right now.
	InFlight int
	// PeakInFlight is the high-water mark of InFlight. It can never exceed
	// DerivationCeiling, and that is the assertion the ceiling is worth.
	PeakInFlight int
}

// Derivations reads the counters above. They are a snapshot and not a
// transaction: each field is consistent on its own.
func Derivations() DerivationStats {
	return DerivationStats{
		Completed:    derivationsCompleted.Load(),
		InFlight:     int(derivationsInFlight.Load()),
		PeakInFlight: int(derivationsPeak.Load()),
	}
}

// derive is the only place in this package that calls Argon2id, which is what
// makes the ceiling and the counters above true of every derivation rather than
// of the ones somebody remembered to route through them.
func derive(password Plaintext, record Record, keyLength uint32) []byte {
	derivationSlots <- struct{}{}
	inFlight := derivationsInFlight.Add(1)
	for {
		peak := derivationsPeak.Load()
		if inFlight <= peak || derivationsPeak.CompareAndSwap(peak, inFlight) {
			break
		}
	}
	defer func() {
		derivationsInFlight.Add(-1)
		derivationsCompleted.Add(1)
		<-derivationSlots
	}()

	return argon2.IDKey([]byte(password.value), record.Salt,
		record.Time, record.MemoryKiB, record.Parallelism, keyLength)
}
