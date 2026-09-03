package sessions_test

import (
	"slices"
	"testing"
	"time"

	"github.com/vdatanet/atrium-go/internal/ports"
	"github.com/vdatanet/atrium-go/internal/sessions"
	"github.com/vdatanet/atrium-go/internal/units"
)

// # One thing here is deliberately NOT asserted, and this is the note saying so
//
// There is no case named for the fact that DeviceID runs *before* the
// visibility rule, and there must not be one.
//
// The two are predicates over one list, so they commute: a caller who is not an
// administrator naming somebody else's device is answered [] whichever ran
// first, and no request this route serves tells the two sequences apart. A case
// called "deviceId runs before the visibility rule" would therefore be green on
// both implementations — it would be asserting the conjunction and reporting
// the order — and a reader meeting it later would take a passing suite as proof
// of something nothing checked.
//
// This is not a hypothetical tidiness. Spec 3.8 records that writing AC-15 is
// what discovered it: the criterion reached first for the *order* being
// observable, which is how behaviours 2.25's own wording reads, and the claim
// did not survive being written down. The claim already cost this feature once,
// and the reason it cost only once is that it was written down instead of
// deleted.
//
// What a request *can* see is asserted, under its own name:
//
//   - TestNamingAnotherUsersDeviceAndNamingAnotherUserAnswerDifferently — the
//     two parameters that name somebody else's property give two answers, which
//     is the observable half spec 3.8 keeps.
//   - TestTheCombinationOfDeviceIdAndControllableByUserIsItsOwnCase — the
//     combination, which 002 plan 6.10 makes the only part of the sequence a
//     request reaches.
//
// The order in Visible is written the reference's way regardless
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1947-2034 @ v10.11.11],
// because being right is not conditional on being observable. It is just not
// asserted, and this paragraph is the record of why.

const (
	ada = "ada-user-identifier"
	bob = "bob-user-identifier"
)

// now is the instant every case is evaluated at. It is fixed rather than read
// off a clock, because activeWithinSeconds is arithmetic against it.
var now = units.At(time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC))

// mediaControlDeclaration is a capabilities document in which the client claims
// SupportsMediaControl. Spec 3.8 measures that the reference echoes that true
// back inside Capabilities while reporting false at the top level: the
// declaration is the client's, the flag is the server's judgement about it
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
var mediaControlDeclaration = []byte(`{"PlayableMediaTypes":["Audio"],"SupportsMediaControl":true}`)

// The fixture. Four sessions over two accounts, three devices and two clients.
//
// No session carries an empty DeviceID, which is load-bearing: it is what makes
// the "deviceId= is absent, not a device nothing is named after" case
// discriminating rather than decorative. An implementation that filtered on
// equality unconditionally would answer [] for that request instead of the
// whole list.
//
// The two living-room rows are spelled in different cases on purpose, because
// spec 3.8 matches the parameter without regard to case and a fixture spelling
// them alike would let a case-sensitive comparison pass.
func fixture() []ports.Session {
	return []ports.Session{
		session("ada-living-room", ada, "Embeat", "living-room", 0, nil),
		session("ada-phone", ada, "Embeat", "phone", time.Hour, nil),
		session("bob-living-room", bob, "Atrium TV", "LIVING-ROOM", 10*time.Second, nil),
		session("bob-kitchen", bob, "Embeat", "kitchen", 90*time.Second, mediaControlDeclaration),
	}
}

func session(name, user, client, device string, idle time.Duration, capabilities []byte) ports.Session {
	return ports.Session{
		ID:                    name,
		UserID:                user,
		Client:                client,
		DeviceID:              device,
		DeviceName:            name,
		CapabilitiesDocument:  capabilities,
		CreatedAt:             units.At(now.Instant().Add(-24 * time.Hour)),
		LastActivityAt:        units.At(now.Instant().Add(-idle)),
		LastPlaybackCheckInAt: units.TimeFromTicks(0),
	}
}

var (
	administrator = sessions.Caller{UserID: bob, IsAdministrator: true}
	adaCaller     = sessions.Caller{UserID: ada}
	bobCaller     = sessions.Caller{UserID: bob}
)

// The table is spec 3.8, row for row. Each case names what it is for, because
// several of them answer the same list for different reasons and a reader has
// to be able to tell which rule a failure broke.
func TestVisible(t *testing.T) {
	cases := []struct {
		name   string
		caller sessions.Caller
		sel    sessions.Selection
		want   []string
	}{
		{
			name:   "no parameter, an administrator sees every session",
			caller: administrator,
			want:   []string{"ada-living-room", "ada-phone", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "no parameter, everybody else sees only their own",
			caller: adaCaller,
			want:   []string{"ada-living-room", "ada-phone"},
		},
		{
			name:   "deviceId, spelled as the session holds it",
			caller: administrator,
			sel:    sessions.Selection{DeviceID: "LIVING-ROOM"},
			want:   []string{"ada-living-room", "bob-living-room"},
		},
		{
			name:   "deviceId in another case matches the same sessions",
			caller: administrator,
			sel:    sessions.Selection{DeviceID: "living-room"},
			want:   []string{"ada-living-room", "bob-living-room"},
		},
		{
			name:   "deviceId in a case neither session uses",
			caller: administrator,
			sel:    sessions.Selection{DeviceID: "LiViNg-RoOm"},
			want:   []string{"ada-living-room", "bob-living-room"},
		},
		{
			name:   "deviceId empty is absent, not a device nothing is named after",
			caller: administrator,
			sel:    sessions.Selection{DeviceID: ""},
			want:   []string{"ada-living-room", "ada-phone", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "deviceId matching nothing is an empty list and not an unfiltered one",
			caller: administrator,
			sel:    sessions.Selection{DeviceID: "no-such-device"},
			want:   nil,
		},
		{
			name:   "a non-administrator naming another user's device",
			caller: adaCaller,
			sel:    sessions.Selection{DeviceID: "kitchen"},
			want:   nil,
		},
		{
			name:   "a non-administrator naming a device they share with somebody else",
			caller: adaCaller,
			sel:    sessions.Selection{DeviceID: "living-room"},
			want:   []string{"ada-living-room"},
		},
		{
			name:   "activeWithinSeconds at 0 is the unfiltered list",
			caller: administrator,
			sel:    sessions.Selection{ActiveWithinSeconds: 0},
			want:   []string{"ada-living-room", "ada-phone", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "activeWithinSeconds at -5 is the unfiltered list and not a refusal",
			caller: administrator,
			sel:    sessions.Selection{ActiveWithinSeconds: -5},
			want:   []string{"ada-living-room", "ada-phone", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "activeWithinSeconds excludes a row whose LastActivityDate is older",
			caller: administrator,
			sel:    sessions.Selection{ActiveWithinSeconds: 60},
			want:   []string{"ada-living-room", "bob-living-room"},
		},
		{
			name:   "activeWithinSeconds keeps a row exactly that old, because the bound is inclusive",
			caller: administrator,
			sel:    sessions.Selection{ActiveWithinSeconds: 90},
			want:   []string{"ada-living-room", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "activeWithinSeconds narrows the caller's own list rather than replacing it",
			caller: adaCaller,
			sel:    sessions.Selection{ActiveWithinSeconds: 60},
			want:   []string{"ada-living-room"},
		},
		{
			name:   "activeWithinSeconds larger than any tick count does not wrap",
			caller: administrator,
			sel:    sessions.Selection{ActiveWithinSeconds: 1 << 62},
			want:   []string{"ada-living-room", "ada-phone", "bob-living-room", "bob-kitchen"},
		},
		{
			name:   "controllableByUserId naming the caller",
			caller: adaCaller,
			sel:    sessions.Selection{ControllableByUser: ada},
			want:   nil,
		},
		{
			name:   "controllableByUserId named by an administrator",
			caller: administrator,
			sel:    sessions.Selection{ControllableByUser: ada},
			want:   nil,
		},
		{
			name:   "controllableByUserId and deviceId together",
			caller: adaCaller,
			sel:    sessions.Selection{DeviceID: "living-room", ControllableByUser: ada},
			want:   nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := names(sessions.Visible(fixture(), testCase.caller, testCase.sel, now))
			if !slices.Equal(got, testCase.want) {
				t.Errorf("Visible = %v, want %v", got, testCase.want)
			}
		})
	}
}

// The observable half of spec 3.8's order, under its own name.
//
// One route, two parameters that each name somebody else's property, two
// answers. Naming another user's *device* is an empty 200 — the ordinary result
// of a filter that matched a row the caller may not see, and not a redaction of
// a refusal. Naming another *user* in controllableByUserId is a 403, which is
// the handler's and is asserted at the wire; what this level can say is that the
// domain answers the empty list for it rather than refusing, so the refusal is
// the edge's decision and not a status leaking out of a pure function.
func TestNamingAnotherUsersDeviceAndNamingAnotherUserAnswerDifferently(t *testing.T) {
	othersDevice := sessions.Visible(fixture(), adaCaller, sessions.Selection{DeviceID: "kitchen"}, now)
	if len(othersDevice) != 0 {
		t.Errorf("naming another user's device answered %v, want the empty list", names(othersDevice))
	}

	otherUser := sessions.Visible(fixture(), adaCaller, sessions.Selection{ControllableByUser: bob}, now)
	if len(otherUser) != 0 {
		t.Errorf("naming another user answered %v, want the empty list from the domain", names(otherUser))
	}
}

// The combination is its own case because 002 plan 6.10 makes it the only part
// of the sequence a request reaches, and it is worth being exact about what it
// catches and what it cannot.
//
// It catches the two implementations that would otherwise look right:
//
//   - one that *ignores* controllableByUserId, which plan 6.10 names as the
//     failure mode — it would answer ada's living-room session where the
//     reference's own first clause answers nothing;
//   - one that lets controllableByUserId *widen* the list back to everybody on
//     that device, which is what a rule appended rather than substituted would
//     do.
//
// It cannot tell which of DeviceID and ControllableByUser ran first, because
// v1's answer to a non-empty ControllableByUser is empty whatever DeviceID
// narrowed to. That is spec 3.8's own argument — v1 attaches no control channel
// so no session supports remote control
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1977 @ v10.11.11]
// — and it is stated here rather than papered over, because the day a feature
// attaches a control channel this case becomes discriminating and whoever
// writes it should know it was not before.
func TestTheCombinationOfDeviceIdAndControllableByUserIsItsOwnCase(t *testing.T) {
	both := sessions.Selection{DeviceID: "living-room", ControllableByUser: ada}
	if got := sessions.Visible(fixture(), adaCaller, both, now); len(got) != 0 {
		t.Errorf("deviceId with controllableByUserId answered %v, want the empty list", names(got))
	}

	deviceOnly := sessions.Visible(fixture(), adaCaller, sessions.Selection{DeviceID: "living-room"}, now)
	if len(deviceOnly) == 0 {
		t.Fatal("the device alone answers nothing, so the case above proves nothing about controllableByUserId")
	}
}

// This asserts the *reason* spec 3.8 gives, not the emptiness — the emptiness
// is already in the table twice.
//
// bob-kitchen posts SupportsMediaControl: true. The reference echoes that true
// back inside Capabilities and still reports SupportsMediaControl false at the
// top level, because the declaration is the client's and the flag is the
// server's judgement about it
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. The
// clause controllableByUserId applies first keeps only the sessions the
// *server* says support remote control
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1977 @ v10.11.11],
// and v1 says that of none of them (spec 3.8, behaviours 2.14).
//
// So this is the one branch in this feature whose correctness is an argument
// rather than a comparison, and the assertion that matches the argument is that
// a session which *declares* media control is absent anyway. An implementation
// that read the capabilities document — the plausible mistake, and the only way
// this branch can be got wrong while looking thorough — answers this session and
// fails here. The guard below is what stops the case from passing vacuously on a
// fixture that lost the declaration.
func TestASessionDeclaringMediaControlIsStillAbsentFromAControllableByUserAnswer(t *testing.T) {
	all := fixture()
	declaring := 0
	for _, s := range all {
		if s.ID == "bob-kitchen" && slices.Equal(s.CapabilitiesDocument, mediaControlDeclaration) {
			declaring++
		}
	}
	if declaring != 1 {
		t.Fatal("the fixture no longer declares SupportsMediaControl, so this case asserts nothing")
	}

	got := sessions.Visible(all, administrator, sessions.Selection{ControllableByUser: bob}, now)
	if len(got) != 0 {
		t.Errorf("Visible = %v, want the empty list: a client's declaration is not the server's flag", names(got))
	}
}

// An empty answer is [] on the wire and not null, which is a byte-level
// difference invisible to anything that parses the body (Principle VIII).
// wire.Write serialises a nil slice as null, so the shape has to be decided
// here rather than at the handler.
func TestAnEmptyAnswerIsAnEmptySliceAndNotNil(t *testing.T) {
	empty := []sessions.Selection{
		{DeviceID: "no-such-device"},
		{ControllableByUser: ada},
		{DeviceID: "living-room", ControllableByUser: ada},
	}
	for _, sel := range empty {
		if got := sessions.Visible(fixture(), adaCaller, sel, now); got == nil {
			t.Errorf("Visible(%+v) returned nil, which is null on the wire and not []", sel)
		}
	}
	if got := sessions.Visible(nil, adaCaller, sessions.Selection{}, now); got == nil {
		t.Error("Visible over no sessions at all returned nil, which is null on the wire and not []")
	}
}

// Visible filters in place, over a copy it makes itself. The copy is the whole
// of what keeps that safe, and nothing else in this package would notice if it
// went away: every case above builds a fresh fixture. A caller that read the
// session list once and asked twice would get an answer narrowed by the first
// question.
func TestVisibleDoesNotNarrowTheListItWasGiven(t *testing.T) {
	all := fixture()
	before := names(all)

	sessions.Visible(all, adaCaller, sessions.Selection{DeviceID: "living-room"}, now)

	if after := names(all); !slices.Equal(before, after) {
		t.Errorf("the caller's list is now %v, was %v", after, before)
	}
}

// A caller nobody filled in sees nothing, which is the direction a zero value
// has to fail in. It matters because sessions.Caller is built by the handler
// out of httpapi.Caller, one field at a time: a conversion that forgot
// IsAdministrator answers fewer sessions, and one that forgot UserID answers
// none, but neither answers everybody's.
func TestTheZeroCallerSeesNothing(t *testing.T) {
	if got := sessions.Visible(fixture(), sessions.Caller{}, sessions.Selection{}, now); len(got) != 0 {
		t.Errorf("the zero caller saw %v, want nothing", names(got))
	}
}

func names(all []ports.Session) []string {
	if len(all) == 0 {
		return nil
	}
	out := make([]string, 0, len(all))
	for _, session := range all {
		out = append(out, session.ID)
	}
	return out
}
