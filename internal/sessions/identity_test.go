package sessions_test

import (
	"regexp"
	"testing"

	"github.com/vdatanet/atrium-go/internal/sessions"
)

// identifier is behaviours 1.4's shape: 32 lowercase hexadecimal characters,
// which is a .NET Guid written without dashes.
var identifier = regexp.MustCompile(`^[0-9a-f]{32}$`)

func TestADerivedSessionIdentifierIsThirtyTwoLowercaseHex(t *testing.T) {
	pairs := [][2]string{
		{"Embeat", "living-room"},
		{"", ""},
		{"Atrium TV", ""},
		{"", "living-room"},
		{"Jellyfin Web", "8ba0a1f4-2f1e-4a5b-9a2c-0f2b3c4d5e6f"},
		{"Ünïcøde Client", "dévice"},
	}
	for _, pair := range pairs {
		if got := sessions.DeriveID(pair[0], pair[1]); !identifier.MatchString(got) {
			t.Errorf("DeriveID(%q, %q) = %q, which is not 32 lowercase hex", pair[0], pair[1], got)
		}
	}
}

// The derivation is stable across processes, which is Principle VII's point
// here: re-authenticating from one device has to land on the session row that
// device already has, and a golden body carrying a session must not be a golden
// naming one particular run.
//
// The literal is what makes this an assertion rather than a tautology. It was
// recorded from this implementation, so it does not prove the derivation is
// *right* — nothing can, since the derivation deliberately answers something
// the reference does not (below) — but it fails the day the inputs, the
// separator, the digest or the truncation change, which is the day every
// recorded body carrying a session identifier silently stops matching.
func TestTheDerivationIsPinnedSoThatAGoldenCanCarryASession(t *testing.T) {
	const embeatLivingRoom = "b424c9eda88740d983e68fc8cced53d7"
	if got := sessions.DeriveID("Embeat", "living-room"); got != embeatLivingRoom {
		t.Errorf("DeriveID(\"Embeat\", \"living-room\") = %q, want %q: every golden carrying a session has moved", got, embeatLivingRoom)
	}
}

// Determinism and distinctness are two properties, and the pin above proves
// only the first: a DeriveID that returned a constant would pass it, because a
// constant is stable across runs. This is the other half.
func TestDifferentClientsAndDevicesDeriveDifferentIdentifiers(t *testing.T) {
	pairs := [][2]string{
		{"Embeat", "living-room"},
		{"Embeat", "phone"},
		{"Atrium TV", "living-room"},
		{"Atrium TV", "phone"},
		{"", "living-room"},
		{"Embeat", ""},
	}
	seen := map[string][2]string{}
	for _, pair := range pairs {
		id := sessions.DeriveID(pair[0], pair[1])
		if previous, taken := seen[id]; taken {
			t.Errorf("(%q, %q) and (%q, %q) both derive %s", previous[0], previous[1], pair[0], pair[1], id)
		}
		seen[id] = pair
	}
}

// The device identifier is matched exactly here, unlike the deviceId *query
// parameter*, which spec 3.8 matches without regard to case. Two different
// spellings of one device are two sessions, which is what the reference's own
// key does too — it concatenates the strings it was handed.
func TestTheDerivationIsCaseSensitiveInBothInputs(t *testing.T) {
	if sessions.DeriveID("Embeat", "living-room") == sessions.DeriveID("embeat", "living-room") {
		t.Error("the client name is folded, and it must not be")
	}
	if sessions.DeriveID("Embeat", "living-room") == sessions.DeriveID("Embeat", "LIVING-ROOM") {
		t.Error("the device identifier is folded, and it must not be")
	}
}

// This is the divergence, asserted rather than assumed.
//
// 002 plan 6.5 argues deliberately against copying the reference's
// MD5(Client + DeviceId)
// [source: Emby.Server.Implementations/Session/SessionManager.cs:486-487,554 @ v10.11.11],
// and the reason this test can carry is that a plain concatenation cannot see
// where its two inputs join. The first assertion is of the reference's rule —
// "ab" + "c" and "a" + "bc" really are one string, so any digest of that string
// is one identifier — and the second is that this derivation answers two.
//
// Asserting the *difference* rather than an expected value is what makes this a
// decision rather than an accident: a later change that "fixed" DeriveID to
// agree with the reference byte for byte would fail here, loudly, and the
// person making it would meet plan 6.5's three reasons before they could
// proceed. allowlist.yaml already declares POST /Users/AuthenticateByName
// /SessionInfo/Id a derived-identifier difference, and AGENTS.md 3 makes a
// conformance assertion a declared inequality where a declared difference that
// has gone away fails too — so agreeing here would be a change to three paired
// artefacts and not a tidy-up.
func TestTheReferencesOwnConcatenationCollidesWhereThisDerivationDoesNot(t *testing.T) {
	if "ab"+"c" != "a"+"bc" {
		t.Fatal("the premise of this test is false, which cannot happen")
	}
	if sessions.DeriveID("ab", "c") == sessions.DeriveID("a", "bc") {
		t.Error(`DeriveID("ab", "c") equals DeriveID("a", "bc"): the derivation has inherited the reference's collision`)
	}
}
