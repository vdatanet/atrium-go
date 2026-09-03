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
