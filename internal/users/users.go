// Package users holds the account domain: the policy, the configuration, the
// rule that decides what a stored document means when it does not carry every
// property, and the credential — ADR-0006's Argon2id record, the ceiling on how
// many derivations run at once, and the plaintext type that will not print
// itself.
//
// # Where it sits
//
// It is Domain (architecture 2), so it may import Ports and the unit types and
// nothing else of ours. In particular it does not import internal/wire, which
// is the Edge's serialiser: a model is a fact about what this server holds, and
// the profile a response is written under is a fact about one request.
//
// That is also why the two document encoders below call encoding/json
// directly rather than going through internal/wire. A stored document is not a
// response — it is negotiated with nobody, it is never escaped for a client's
// decoder, and it carries no content type. What the two encodings *share* is
// the struct, and the struct is where the thing that matters lives: Go writes a
// struct's fields in declaration order, so one declaration fixes both the order
// of the stored document and the key order of the body
// POST /Users/AuthenticateByName sends, which is this feature's one L3 row and
// therefore a byte comparison (plan 4).
//
// # The one rule this package exists to state
//
// **A stored document decodes onto the reference's defaults, never onto the Go
// zero value.** Every decode starts from DefaultPolicy or
// DefaultConfiguration and unmarshals the document over it (plan 4).
//
// It is written as a package rule rather than left to each caller because the
// failure is silent and total: the reference's UserPolicy constructor sets
// thirteen booleans to true and LoginAttemptsBeforeLockout to -1
// [source: MediaBrowser.Model/Users/UserPolicy.cs:16-68 @ v10.11.11], so a
// document written by an older build — one that predates a property — decodes
// onto Policy{} with EnableMediaPlayback false and LoginAttemptsBeforeLockout
// 0. The first locks every account out of playback; the second is not "no
// limit" but the sentinel spec 3.3 reads as *lock after three failures*
// (plan 6.7). Neither is visible to a test that round-trips a document holding
// every property, which is the test anybody writes first.
package users
