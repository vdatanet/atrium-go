package users_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/users"
)

// configurationOrder is the reference's declaration order, transcribed from the
// reference and not from this project's struct
// [source: MediaBrowser.Model/Configuration/UserConfiguration.cs:35-76 @
// v10.11.11]. See policyOrder for why it is written out.
var configurationOrder = []string{
	"AudioLanguagePreference",
	"PlayDefaultAudioTrack",
	"SubtitleLanguagePreference",
	"DisplayMissingEpisodes",
	"GroupedFolders",
	"SubtitleMode",
	"DisplayCollectionsView",
	"EnableLocalPassword",
	"OrderedViews",
	"LatestItemsExcludes",
	"MyMediaExcludes",
	"HidePlayedInLatest",
	"RememberAudioSelections",
	"RememberSubtitleSelections",
	"EnableNextEpisodeAutoPlay",
	"CastReceiverId",
}

// TestAStoredConfigurationDocumentDecodesOntoTheReferencesDefaults is the
// policy rule applied to the second document, and it fails in the same place
// for the same reason.
//
// The direction the zero value is wrong in here is quieter than the policy's
// and still a delta: PlayDefaultAudioTrack, HidePlayedInLatest,
// RememberAudioSelections, RememberSubtitleSelections and
// EnableNextEpisodeAutoPlay are all true on a fresh account
// [source: src/Jellyfin.Database/Jellyfin.Database.Implementations/Entities/User.cs:42-56
// @ v10.11.11], and SubtitleMode is the enum name "Default" rather than the
// empty string.
func TestAStoredConfigurationDecodesOntoTheReferencesDefaults(t *testing.T) {
	if zero := (users.Configuration{}); zero.PlayDefaultAudioTrack || zero.SubtitleMode != "" {
		t.Fatalf("the Go zero value has changed shape, and this test no longer guards what it says it does")
	}

	configuration, err := users.DecodeConfiguration([]byte(`{"DisplayMissingEpisodes":true}`))
	if err != nil {
		t.Fatalf("decoding a configuration document holding one property: %v", err)
	}

	if !configuration.DisplayMissingEpisodes {
		t.Error("the one property the document carried did not survive the decode")
	}
	if !configuration.PlayDefaultAudioTrack || !configuration.HidePlayedInLatest ||
		!configuration.RememberAudioSelections || !configuration.RememberSubtitleSelections ||
		!configuration.EnableNextEpisodeAutoPlay {
		t.Error("a preference the reference sets true on a fresh account decoded false")
	}
	if configuration.SubtitleMode != "Default" {
		t.Errorf("SubtitleMode decoded %q, want %q — an empty string is not a member of the "+
			"reference's enumeration", configuration.SubtitleMode, "Default")
	}
	if configuration.OrderedViews == nil || configuration.GroupedFolders == nil {
		t.Error("an array decoded nil, which serialises as null; the reference sends []")
	}
}

// TestAnUnknownPropertyInAStoredConfigurationIsDroppedAndTheDeclaredOnesSurvive
// is spec 3.6's "unknown properties are ignored, not rejected", asserted on
// both halves: the unknown one does not survive the round trip, and the
// declared ones do.
//
// Ignoring is not the same as tolerating. A decode that refused would make an
// installation unreadable by the build that comes after the one that wrote it;
// a decode that *kept* the unknown property would echo a property the reference
// does not declare, which is a delta by Principle I on the count.
//
// **This is the opposite of what the session's capabilities do.** behaviours
// 5.9 records an unknown capabilities property surviving into /Sessions as a
// deliberate divergence, because the reference keeps it there; there is no such
// divergence here (plan 6.6, 6.10). The two are stated together because
// generalising from either one gives the wrong answer about the other, and T16
// is where the other half is written.
func TestAnUnknownPropertyInAStoredConfigurationIsDroppedAndTheDeclaredOnesSurvive(t *testing.T) {
	stored := `{"DisplayMissingEpisodes":true,"SubtitleMode":"Always",` +
		`"EnableTimeTravel":true,"AudioLanguagePreference":"spa"}`

	configuration, err := users.DecodeConfiguration([]byte(stored))
	if err != nil {
		t.Fatalf("an unknown property was rejected, and spec 3.6 says it is ignored: %v", err)
	}

	if !configuration.DisplayMissingEpisodes {
		t.Error("DisplayMissingEpisodes did not survive beside the unknown property")
	}
	if configuration.SubtitleMode != "Always" {
		t.Errorf("SubtitleMode is %q, want %q", configuration.SubtitleMode, "Always")
	}
	if configuration.AudioLanguagePreference != "spa" {
		t.Errorf("AudioLanguagePreference is %q, want %q", configuration.AudioLanguagePreference, "spa")
	}

	document, err := configuration.Document()
	if err != nil {
		t.Fatalf("writing a configuration document: %v", err)
	}
	if strings.Contains(string(document), "EnableTimeTravel") {
		t.Errorf("the unknown property survived the round trip, so this server would echo a "+
			"property the reference does not declare:\n%s", document)
	}
	if members := memberNamesInOrder(t, document); len(members) != len(configurationOrder) {
		t.Errorf("the written document carries %d members, want the declared %d:\n%s",
			len(members), len(configurationOrder), strings.Join(members, ", "))
	}
}

// TestTheConfigurationIsWrittenInTheReferencesDeclarationOrder asserts the byte
// order, for the reason the policy's does: this object travels inside the same
// L3 body, and a set comparison passes on a reordered model.
//
// All sixteen travel — none of them is nullable in this model, and spec 3.6
// records the reference sending sixteen
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28] — so
// the stored document and the body carry the same list and there is no
// subtraction to do here.
func TestTheConfigurationIsWrittenInTheReferencesDeclarationOrder(t *testing.T) {
	document, err := users.DefaultConfiguration().Document()
	if err != nil {
		t.Fatalf("writing a configuration document: %v", err)
	}
	assertOrder(t, "the stored document", memberNamesInOrder(t, document), configurationOrder)

	if len(configurationOrder) != 16 {
		t.Fatalf("this test transcribes %d members and spec 3.6 measured 16", len(configurationOrder))
	}
}

// TestTheDeclaredSetsDoNotOverlap is a small guard on the two transcriptions
// above rather than on the models.
//
// Policy and Configuration travel side by side inside one user object, and a
// name appearing in both lists would mean one of the two was transcribed from
// the wrong reference model — the kind of mistake that reads as correct in
// review and produces a body with a member in the wrong object.
func TestTheDeclaredSetsDoNotOverlap(t *testing.T) {
	for _, name := range configurationOrder {
		if slices.Contains(policyOrder, name) {
			t.Errorf("%q is declared by both models, so one of the two transcriptions is wrong", name)
		}
	}
}
