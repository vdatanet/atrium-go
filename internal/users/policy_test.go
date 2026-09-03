package users_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/users"
	"github.com/vdatanet/atrium-go/internal/wire"
)

// policyOrder is the reference's declaration order, transcribed from the
// reference and not from this project's struct
// [source: MediaBrowser.Model/Users/UserPolicy.cs:74-197 @ v10.11.11].
//
// It is written out rather than derived by reflection over users.Policy on
// purpose: a list derived from the model would agree with the model whatever
// the model said, and the question this list answers is whether the model
// agrees with the reference.
var policyOrder = []string{
	"IsAdministrator",
	"IsHidden",
	"EnableCollectionManagement",
	"EnableSubtitleManagement",
	"EnableLyricManagement",
	"IsDisabled",
	"MaxParentalRating",
	"MaxParentalSubRating",
	"BlockedTags",
	"AllowedTags",
	"EnableUserPreferenceAccess",
	"AccessSchedules",
	"BlockUnratedItems",
	"EnableRemoteControlOfOtherUsers",
	"EnableSharedDeviceControl",
	"EnableRemoteAccess",
	"EnableLiveTvManagement",
	"EnableLiveTvAccess",
	"EnableMediaPlayback",
	"EnableAudioPlaybackTranscoding",
	"EnableVideoPlaybackTranscoding",
	"EnablePlaybackRemuxing",
	"ForceRemoteSourceTranscoding",
	"EnableContentDeletion",
	"EnableContentDeletionFromFolders",
	"EnableContentDownloading",
	"EnableSyncTranscoding",
	"EnableMediaConversion",
	"EnabledDevices",
	"EnableAllDevices",
	"EnabledChannels",
	"EnableAllChannels",
	"EnabledFolders",
	"EnableAllFolders",
	"InvalidLoginAttemptCount",
	"LoginAttemptsBeforeLockout",
	"MaxActiveSessions",
	"EnablePublicSharing",
	"BlockedMediaFolders",
	"BlockedChannels",
	"RemoteClientBitrateLimit",
	"AuthenticationProviderId",
	"PasswordResetProviderId",
	"SyncPlayAccess",
}

// theTwoThatDoNotTravel are the members behaviours 1.7 drops from every body a
// default account produces: nullable integers, null unless an operator has
// restricted the account
// [source: MediaBrowser.Model/Users/UserPolicy.cs:112-114 @ v10.11.11].
var theTwoThatDoNotTravel = []string{"MaxParentalRating", "MaxParentalSubRating"}

// TestAStoredPolicyDocumentDecodesOntoTheReferencesDefaults is the test this
// whole task is named for, and it is the one that fails when a later change
// replaces DefaultPolicy() with Policy{} because "the round-trip still passes".
//
// The document holds **one** property. Everything else has to come from
// somewhere, and Go's answer — false for a bool, 0 for an int — is wrong in
// both directions at once: EnableMediaPlayback false locks every account out of
// playback, and LoginAttemptsBeforeLockout 0 is not "no limit" but the
// reference's sentinel for *lock after three failures*
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:816-821 @
// v10.11.11], where -1 is what a default account carries
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
//
// The zero value is asserted alongside, so that the test states what it is
// guarding against rather than only what it wants. If Go's default ever
// coincided with the reference's, this test would be passing for no reason and
// the assertion below is what would say so.
func TestAStoredPolicyDocumentDecodesOntoTheReferencesDefaults(t *testing.T) {
	if zero := (users.Policy{}); zero.EnableMediaPlayback || zero.LoginAttemptsBeforeLockout != 0 {
		t.Fatalf("the Go zero value has changed shape, and this test no longer guards what it says it does")
	}

	policy, err := users.DecodePolicy([]byte(`{"IsAdministrator":true}`))
	if err != nil {
		t.Fatalf("decoding a policy document holding one property: %v", err)
	}

	if !policy.IsAdministrator {
		t.Error("the one property the document carried did not survive the decode")
	}
	if !policy.EnableMediaPlayback {
		t.Error("EnableMediaPlayback decoded false; the reference's constructor sets it true, " +
			"and false here refuses playback to every account whose document predates a property")
	}
	if policy.LoginAttemptsBeforeLockout != -1 {
		t.Errorf("LoginAttemptsBeforeLockout decoded %d, want -1; 0 is not 'no limit' but the "+
			"reference's sentinel for lock-after-three", policy.LoginAttemptsBeforeLockout)
	}
	if !policy.EnableAllFolders || !policy.EnableAllChannels || !policy.EnableRemoteAccess {
		t.Error("the three other constructor-set permissions decoded false")
	}
	if policy.AccessSchedules == nil || policy.BlockedTags == nil {
		t.Error("an array decoded nil, which serialises as null; the reference sends []")
	}
}

// TestAStoredPolicyDocumentStillOverridesADefaultItDisagreesWith is the other
// half of the rule above, and without it a decode that ignored the document
// entirely would pass every assertion in this file that matters.
//
// It is why `omitempty` is banned on a non-pointer (ADR-0002, architecture 2):
// a policy that really does forbid playback stores EnableMediaPlayback false,
// and an encoder that dropped false would make that policy indistinguishable
// from one that never mentioned the property — which decodes back to true.
func TestAStoredPolicyDocumentStillOverridesADefaultItDisagreesWith(t *testing.T) {
	policy, err := users.DecodePolicy([]byte(`{"EnableMediaPlayback":false,"LoginAttemptsBeforeLockout":3}`))
	if err != nil {
		t.Fatalf("decoding a policy document: %v", err)
	}
	if policy.EnableMediaPlayback {
		t.Error("a document saying EnableMediaPlayback false decoded true: the default won over the document")
	}
	if policy.LoginAttemptsBeforeLockout != 3 {
		t.Errorf("LoginAttemptsBeforeLockout decoded %d, want the document's 3", policy.LoginAttemptsBeforeLockout)
	}

	// And the false survives a write, which is the half a decode test cannot
	// see: an encoder that omitted it would store a policy that reads back as
	// its opposite.
	document, err := policy.Document()
	if err != nil {
		t.Fatalf("writing a policy document: %v", err)
	}
	if !strings.Contains(string(document), `"EnableMediaPlayback":false`) {
		t.Errorf("the written document does not carry EnableMediaPlayback false:\n%s", document)
	}
}

// TestTheModelDeclaresFortyFourMembersAndTheBodyCarriesFortyTwo is the
// arithmetic tasks.md T2 asks for: 44 − 2 = 42.
//
// The measured number is 42 [probe: tools/probe_auth_mechanisms.py,
// Jellyfin 10.11.11, 2026-08-28]. Asserted as the literal 42 it would be a
// number typed in to match, and a model that lost a property and a test that
// said "42" would agree with each other and disagree with the reference. So the
// 42 is *derived*: the model's own members, minus the two behaviours 1.7 drops,
// must equal what the body carries — and the two that are dropped are named,
// not counted.
func TestTheModelDeclaresFortyFourMembersAndTheBodyCarriesFortyTwo(t *testing.T) {
	declared := reflect.TypeOf(users.Policy{}).NumField()
	if declared != len(policyOrder) {
		t.Fatalf("users.Policy declares %d fields and the reference declares %d", declared, len(policyOrder))
	}

	// A policy an operator has restricted: both nullable members set, so every
	// declared member is encodable and the encoding shows all of them.
	restricted := users.DefaultPolicy()
	rating, subRating := 10, 4
	restricted.MaxParentalRating = &rating
	restricted.MaxParentalSubRating = &subRating

	whole, err := restricted.Document()
	if err != nil {
		t.Fatalf("writing a policy document: %v", err)
	}
	wholeMembers := memberNamesInOrder(t, whole)
	if len(wholeMembers) != len(policyOrder) {
		t.Fatalf("an encoded policy with nothing null carries %d members, want %d:\n%s",
			len(wholeMembers), len(policyOrder), strings.Join(wholeMembers, ", "))
	}

	sent := memberNamesInOrder(t, policyBodyOverTheWire(t, users.DefaultPolicy()))

	var absent []string
	for _, name := range wholeMembers {
		if !slices.Contains(sent, name) {
			absent = append(absent, name)
		}
	}
	if !slices.Equal(absent, theTwoThatDoNotTravel) {
		t.Fatalf("the members a default account does not send are %v, want %v",
			absent, theTwoThatDoNotTravel)
	}

	if got, want := len(sent), len(wholeMembers)-len(absent); got != want {
		t.Fatalf("the body carries %d members and %d − %d is %d",
			got, len(wholeMembers), len(absent), want)
	}
	if len(sent) != 42 {
		t.Errorf("the body carries %d members and the reference sends 42 "+
			"[probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]", len(sent))
	}
}

// TestThePolicyIsWrittenInTheReferencesDeclarationOrder asserts the **byte
// order** of the encoded document, and not a set.
//
// POST /Users/AuthenticateByName is this feature's one L3 row (surface.yaml),
// and L3 compares bytes. A set comparison passes on a reordered model, so it
// would let a property move between two builds of this server and be found by
// the differential run instead of by this package.
//
// Both encodings are asserted, because they are two encodings of one
// declaration: the stored document, which is encoding/json's, and the body
// internal/wire writes.
func TestThePolicyIsWrittenInTheReferencesDeclarationOrder(t *testing.T) {
	restricted := users.DefaultPolicy()
	rating, subRating := 10, 4
	restricted.MaxParentalRating = &rating
	restricted.MaxParentalSubRating = &subRating

	document, err := restricted.Document()
	if err != nil {
		t.Fatalf("writing a policy document: %v", err)
	}
	assertOrder(t, "the stored document", memberNamesInOrder(t, document), policyOrder)

	var sentOrder []string
	for _, name := range policyOrder {
		if !slices.Contains(theTwoThatDoNotTravel, name) {
			sentOrder = append(sentOrder, name)
		}
	}
	assertOrder(t, "the body internal/wire writes",
		memberNamesInOrder(t, policyBodyOverTheWire(t, users.DefaultPolicy())), sentOrder)
}

// modelWithTwoMembersTransposed is the whole evidence that the order assertion
// above can fail, and that a set comparison in its place could not.
//
// It exists only in this file, so it is in no binary this project ships. The
// two members are the reference's first two, swapped — the smallest change a
// reordering can be, and one that a comparison of sorted names or of a map
// would not notice at all.
type modelWithTwoMembersTransposed struct {
	IsHidden        bool
	IsAdministrator bool
}

// TestTheOrderAssertionCatchesATransposition proves the check can fail, and
// proves the weaker check it replaces would not have.
//
// A sweep that has never failed has proved nothing, and this is the failure
// this file's most important assertion is guarding against.
func TestTheOrderAssertionCatchesATransposition(t *testing.T) {
	document, err := json.Marshal(modelWithTwoMembersTransposed{})
	if err != nil {
		t.Fatalf("encoding the transposed model: %v", err)
	}
	got := memberNamesInOrder(t, document)
	want := []string{"IsAdministrator", "IsHidden"}

	if slices.Equal(got, want) {
		t.Fatalf("the transposed model encoded in the declared order, so this file's order "+
			"assertion is not asserting what it says: %v", got)
	}

	// The comparison the order assertion replaces. It passes, which is the
	// point: this is what would have been shipped.
	asSet := slices.Clone(got)
	slices.Sort(asSet)
	sortedWant := slices.Clone(want)
	slices.Sort(sortedWant)
	if !slices.Equal(asSet, sortedWant) {
		t.Errorf("a set comparison failed on the transposed model, so it is not the weaker "+
			"check this test claims it is: %v", asSet)
	}
}

// TestEveryMemberNameIsOneThePinnedDocumentDeclares checks the 60 names of the
// two models against the 1,026 property names extracted from the pinned OpenAPI
// document (docs/compatibility/property-names.json).
//
// It is the cheapest check that a member is the reference's and not a
// plausible misspelling of it. The casing sweep in internal/httpapi answers a
// different question — is this name PascalCase — and would pass
// "EnableMediaPlaybak" without comment. These two models cannot be added to
// that sweep's registry until their routes are registered, because the registry
// is checked against the router in both directions (T17).
func TestEveryMemberNameIsOneThePinnedDocumentDeclares(t *testing.T) {
	declared := propertyNamesOfThePinnedDocument(t)

	restricted := users.DefaultPolicy()
	rating, subRating := 10, 4
	restricted.MaxParentalRating = &rating
	restricted.MaxParentalSubRating = &subRating
	policyDocument, err := restricted.Document()
	if err != nil {
		t.Fatalf("writing a policy document: %v", err)
	}
	configurationDocument, err := users.DefaultConfiguration().Document()
	if err != nil {
		t.Fatalf("writing a configuration document: %v", err)
	}

	for _, model := range []struct {
		name    string
		members []string
	}{
		{"Policy", memberNamesInOrder(t, policyDocument)},
		{"Configuration", memberNamesInOrder(t, configurationDocument)},
	} {
		for _, member := range model.members {
			if !declared[member] {
				t.Errorf("%s writes the property name %q, which the pinned OpenAPI document "+
					"does not declare", model.name, member)
			}
		}
	}
}

// policyBodyOverTheWire writes a policy the way a response carries it: through
// internal/wire, under the profile a request with no Accept header gets.
//
// internal/users does not import internal/wire — the domain may not import the
// serialiser (architecture 2) — and this test does, deliberately. The member
// count and the key order are facts about bytes, and the only honest place to
// assert them is over the bytes the project's own serialiser produces.
func policyBodyOverTheWire(t *testing.T, policy users.Policy) []byte {
	t.Helper()

	recorder := httptest.NewRecorder()
	if err := wire.Write(recorder, http.StatusOK, policy, wire.ProfilePlain); err != nil {
		t.Fatalf("writing a policy through internal/wire: %v", err)
	}
	return recorder.Body.Bytes()
}

// memberNamesInOrder reads the member names of a JSON object in the order the
// bytes carry them.
//
// json.Decoder's token stream is what makes this possible: unmarshalling into a
// map would lose the order, which is the property the L3 row depends on. Each
// value is consumed whole with Decode, so a nested object's own members are
// never mistaken for this one's.
func memberNamesInOrder(t *testing.T, document []byte) []string {
	t.Helper()

	decoder := json.NewDecoder(bytes.NewReader(document))
	open, err := decoder.Token()
	if err != nil {
		t.Fatalf("reading the document: %v", err)
	}
	if delimiter, ok := open.(json.Delim); !ok || delimiter != '{' {
		t.Fatalf("the document is not a JSON object: %s", document)
	}

	var names []string
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading the document: %v", err)
		}
		name, ok := token.(string)
		if !ok {
			t.Fatalf("expected a member name and read %v", token)
		}
		names = append(names, name)

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatalf("reading the value of %s: %v", name, err)
		}
	}
	return names
}

// assertOrder compares two ordered lists of member names and reports the first
// position they differ at, because a diff of two 44-element lists is unreadable
// and the position is what a reordering is.
func assertOrder(t *testing.T, what string, got, want []string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s carries %d members, want %d:\n got: %s\nwant: %s",
			what, len(got), len(want), strings.Join(got, ", "), strings.Join(want, ", "))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s writes %q at position %d, want %q — declaration order is wire order",
				what, got[i], i, want[i])
		}
	}
}

// propertyNamesOfThePinnedDocument reads the generated list of every property
// name the pinned OpenAPI document declares.
func propertyNamesOfThePinnedDocument(t *testing.T) map[string]bool {
	t.Helper()

	const path = "../../docs/compatibility/property-names.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	var document struct {
		Count int      `json:"count"`
		Names []string `json:"names"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if len(document.Names) != document.Count {
		t.Fatalf("%s holds %d names and claims %d", path, len(document.Names), document.Count)
	}

	declared := make(map[string]bool, len(document.Names))
	for _, name := range document.Names {
		declared[name] = true
	}
	return declared
}
