package sessions

import (
	"crypto/sha256"
	"encoding/hex"
)

// identifierBytes is how much of the digest an identifier keeps: 16 bytes,
// which is 32 characters of hexadecimal and exactly the shape behaviours 1.4
// records for every identifier this API carries — a .NET Guid written without
// dashes [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:37 @ v10.11.11].
const identifierBytes = 16

// separator is the byte between the two inputs, and it is the whole of what
// makes this derivation different from the reference's. See DeriveID.
const separator = 0x00

// DeriveID is the identifier of the session a client named client opened on the
// device named deviceID: 32 lowercase hex, the first 16 bytes of SHA-256 over
// the client name, a NUL, and the device identifier (002 plan 6.5).
//
// Derived rather than random, which is Principle VII: re-authenticating from
// the same device has to land on the same session row, and a random identifier
// would make that a lookup rather than an arithmetic fact. The key is
// (Client, DeviceId) and not the (user, device, client) triple spec 3.8
// describes, because the reference keys a session on the client and the device
// alone and updates its user when somebody else logs in there — 002 plan 6.5
// argues it, and ports.Session records it.
//
// # It deliberately disagrees with the reference, and that is the point
//
// The reference computes MD5(Client + DeviceId)
// [source: Emby.Server.Implementations/Session/SessionManager.cs:486-487,554 @ v10.11.11],
// which is both possible and free to reproduce byte for byte — which is exactly
// why not doing it has to be argued rather than assumed. 002 plan 6.5 gives
// three reasons, and the one this function can hold in its own shape is the
// third: a plain concatenation cannot see where its two inputs join, so
// ("ab", "c") and ("a", "bc") are one session there. The NUL is what separates
// them here.
//
// The other two reasons are about the project rather than about the collision.
// allowlist.yaml already declares POST /Users/AuthenticateByName /SessionInfo/Id
// a derived-identifier difference, and AGENTS.md 3 makes a conformance
// assertion a declared inequality where a declared difference that has gone
// away fails too — so agreeing with the reference here would be a change to
// three paired artefacts and not a simplification. And architecture 6 settles
// the general question: reproducing the reference's exact identifier bytes is
// not a goal and never was; reproducing its stability is.
//
// The divergence is asserted rather than assumed:
// TestTheReferencesOwnConcatenationCollidesWhereThisDerivationDoesNot in
// identity_test.go is the test that fails the day somebody "fixes" this to
// match.
func DeriveID(client, deviceID string) string {
	digest := sha256.New()
	digest.Write([]byte(client))
	digest.Write([]byte{separator})
	digest.Write([]byte(deviceID))
	return hex.EncodeToString(digest.Sum(nil)[:identifierBytes])
}

// TokenDigest is what the store holds instead of the token itself: the
// unsalted SHA-256 of the token, whole, in lowercase hex (002 plan 4).
//
// One function, called by both sides of the token's life — the login that
// mints one and writes its digest, and the authenticator that resolves a
// presented one — because two spellings of a digest are two digests, and the
// symptom of a disagreement is every credential this server issues failing to
// authenticate with no error anywhere.
//
// # Why the token is not stored
//
// ADR-0006's threat model is the store file leaking, and a leaked table of
// live bearer tokens is that leak with the hashing skipped: the file alone
// would let anybody hold every logged-in client's credential.
//
// # Why it is unsalted, and why that is not the password rule being ignored
//
// A password is low-entropy and chosen by a person, which is what ADR-0006
// spends 52 ms and a memory reservation on. A token is 128 bits from
// crypto/rand (002 plan 6.5), so a salt would defend against precomputation
// over a space nobody can precompute, while costing the primary-key lookup
// that makes a per-request check one indexed read. This is invisible on the
// wire — no response carries a stored token — so it is an engineering choice
// in ADR-0006's own sense and not a delta.
//
// # It is not truncated, where DeriveID is
//
// DeriveID keeps 16 bytes because its output is an *identifier* and behaviours
// 1.4 makes an identifier 32 hex characters. This output is a lookup key that
// no response carries, so there is nothing to match and no reason to throw
// half of it away.
func TokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
