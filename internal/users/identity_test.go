package users_test

import (
	"regexp"
	"testing"

	"github.com/vdatanet/atrium-go/internal/users"
)

// identifier is behaviours 1.4's shape: 32 lowercase hexadecimal characters,
// which is a .NET Guid written without dashes.
var identifier = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestADerivedIdentifierIsThirtyTwoLowercaseHex(t *testing.T) {
	for _, name := range []string{"Ada", "", "ADA", "a name with spaces", "Ünïcøde", "administrator"} {
		if got := users.DeriveID(name); !identifier.MatchString(got) {
			t.Errorf("DeriveID(%q) = %q, which is not 32 lowercase hex", name, got)
		}
	}
}

// The identifier follows the *folded* name, which is what makes it the same on
// two installations that spell one account differently in case.
//
// It is also what stops the derivation from disagreeing with the column the
// login reads: username_folded is the only column an authentication looks at
// (spec 3.3), so an identifier derived from the name as typed would be an
// identifier for a spelling the store does not key on.
func TestADerivedIdentifierFollowsTheFold(t *testing.T) {
	if users.DeriveID("Ada") != users.DeriveID(users.Fold("Ada")) {
		t.Error("DeriveID does not fold the name it is given")
	}
	if users.DeriveID("ADA") != users.DeriveID("ada") {
		t.Error("ADA and ada derive different identifiers")
	}
}

// Different names derive different identifiers, which is the property the
// derivation has to have and the one a constant would not.
func TestDifferentNamesDeriveDifferentIdentifiers(t *testing.T) {
	seen := map[string]string{}
	for _, name := range []string{"Ada", "Bob", "Cyd", "administrator", "restricted", "playback-denied"} {
		id := users.DeriveID(name)
		if previous, taken := seen[id]; taken {
			t.Errorf("%q and %q both derive %s", previous, name, id)
		}
		seen[id] = name
	}
}

// The derivation is stable across runs, which is Principle VII's whole point
// here: a golden body that names a user must not name one particular run.
//
// The literal is what makes this an assertion rather than a tautology. It was
// recorded from this implementation, so it does not prove the derivation is
// *right* — nothing can, since no reference derives an identifier this way —
// but it fails the day the input, the digest or the truncation changes, which
// is the day every recorded body naming a user silently stops matching.
func TestTheDerivationIsPinnedSoThatAGoldenCanNameAUser(t *testing.T) {
	const ada = "fdee430d40bd57deeac186cd9790033d"
	if got := users.DeriveID("Ada"); got != ada {
		t.Errorf("DeriveID(\"Ada\") = %q, want %q: every golden naming a user has moved", got, ada)
	}
}
