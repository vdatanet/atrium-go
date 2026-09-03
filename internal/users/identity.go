package users

import (
	"crypto/sha256"
	"encoding/hex"
)

// identifierBytes is how much of the digest an identifier keeps: 16 bytes,
// which is 32 characters of hexadecimal and exactly the shape behaviours 1.4
// records for every identifier this API carries — a .NET Guid written without
// dashes [source: src/Jellyfin.Extensions/Json/JsonDefaults.cs:37 @ v10.11.11].
//
// Truncating a 32-byte digest to 16 is not a weakening of anything: the
// identifier is a name, not a secret, and what it has to be is stable and
// collision-free over the usernames one installation holds. Two folded
// usernames colliding here would need a 128-bit birthday collision, and the
// database would refuse the second row anyway, because username_folded is
// unique.
const identifierBytes = 16

// DeriveID is the identifier of the account named username: 32 lowercase hex,
// the first 16 bytes of SHA-256 over the *folded* name.
//
// Derived rather than random, which is Principle VII and plan 6.9. It buys
// something concrete rather than a principle: an installation provisioned twice
// with the same names has the same identifiers, so a golden body that names a
// user is not a golden that names one particular run. A random identifier would
// make every user-shaped response unrecordable without a fixture that stated
// the identifier the way conformance/ already has to state the installation's.
//
// It folds the name itself rather than taking a folded one, because the two
// callers that need an identifier — provisioning, and anything that ever looks
// one up by name — would otherwise each have to remember to fold first, and a
// caller that forgot would derive an identifier for a name the store cannot
// find. One function, one answer.
//
// # The cost, stated
//
// **Renaming an account would change its identifier**, which is behaviours 1.4's
// library-root trap in miniature: every client-side reference to the account —
// a favourite, a resume position, a cached user object — is keyed on the
// identifier, and a rename would silently orphan all of them. v1 has no rename,
// so the cost is not payable today (plan 6.9). The feature that adds one
// inherits this line and has to decide what it does about it before it ships.
func DeriveID(username string) string {
	digest := sha256.Sum256([]byte(Fold(username)))
	return hex.EncodeToString(digest[:identifierBytes])
}
