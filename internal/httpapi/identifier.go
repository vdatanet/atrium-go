package httpapi

// The shape every identifier this API carries is written in: 32 hexadecimal
// characters with no dashes, which is a .NET Guid in its "N" format
// (behaviours 1.4) [prior-probe: Jellyfin 10.11.11, 2026-06-13].
const identifierDigits = 32

// emptyIdentifier is the all-zero identifier — .NET's Guid.Empty, written the
// way every other identifier is.
//
// It is well formed and it is nobody's, and the reference does **not** answer
// it with the 404 it answers an ordinary unknown identifier with: its account
// lookup refuses an empty Guid before it queries anything
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:123-133 @ v10.11.11],
// the ArgumentException that raises is mapped to 400, and the body is
// behaviours 1.11's twenty-five bytes under text/plain
// [source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-136 @ v10.11.11].
// The same request was measured on another route that resolves an identifier
// and answered exactly that (009 spec 3.8's
// identifier table, 2026-09-01), which is what makes the reading above a
// reading of one shape rather than a guess at one.
const emptyIdentifier = "00000000000000000000000000000000"

// canonicalIdentifier reports the spelling this server stores an identifier in,
// and whether what a client wrote is an identifier at all.
//
// # What is accepted, and the delta that is knowingly kept
//
// Thirty-two hexadecimal characters, in either case, folded to the lower case
// every identifier this server derives is written in (users.DeriveID,
// sessions.DeriveID). Upper case is accepted because the reference's binder
// parses the segment as a Guid and answers an upper-case spelling with the
// object rather than with a refusal — measured on another route and recorded
// in 009 spec 3.8's identifier table,
// 2026-09-01.
//
// That same measured row records two further spellings the .NET binder
// accepts and this function does not: the dashed 8-4-4-4-12 form and the
// braced form around it. **Refusing them is a delta**, in the direction
// behaviours 3.0.3 calls the dangerous one — a request that succeeds against
// every Jellyfin there is meets a 400 here. It is kept rather than closed
// because the row is a measurement of a *playlist* identifier and reading it
// onto this route is a reading; because no identifier this API ever emits is
// written either way, so no conforming client can reach it; and because
// canonicalising a spelling this server never produces is a rule nobody has
// asked a running reference for. It is asserted as a test rather than left in
// this comment — TestADashedIdentifierIsRefusedHereAndTheReferencesBinderParsesIt
// — so that the day the probe lands, a failing test names the behaviour that
// moved instead of somebody rediscovering it. The register at T23 is owed a
// row.
func canonicalIdentifier(value string) (string, bool) {
	if len(value) != identifierDigits {
		return "", false
	}
	folded := make([]byte, identifierDigits)
	for i := 0; i < identifierDigits; i++ {
		c := value[i]
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f':
			folded[i] = c
		case c >= 'A' && c <= 'F':
			folded[i] = c + ('a' - 'A')
		default:
			return "", false
		}
	}
	return string(folded), true
}
