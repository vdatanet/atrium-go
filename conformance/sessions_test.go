package conformance_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

// 002 AC-9, AC-10, AC-13 and AC-15 against the running binary.
//
// # One installation, five criteria, and the order they run in
//
// T18's rule, which is a rule and not a preference: provisioning this fixture
// costs six Argon2id derivations at 64 MiB each, and enough of this package
// provisioning at once stopped being scheduling noise in `internal/users`,
// where the timing equalisation ADR-0006's whole argument stands on compares
// two medians with a margin sized for noise
// `[measurement: GitHub Actions, go test ./..., 2026-09-03]`. So criteria that
// do not disturb one another share a server and stay separate subtests.
//
// These five do not disturb one another **in this order**, and the order is
// therefore stated rather than implied:
//
//   - AC-9 posts a full capabilities declaration to the administrator's session
//     and reads it back. AC-13's replacement clause posts a smaller one to the
//     same session, so it has to run after AC-9 rather than before it.
//   - AC-15 reads session *lists* and asserts which rows come back. It does not
//     read capabilities, so the two posts above are invisible to it.
//   - AC-10 authenticates four times and succeeds once, which opens a session
//     the administrator would then see. It runs after AC-15 for that reason
//     alone.
//   - AC-13's session ceiling opens and evicts sessions belonging to one
//     account, and reads them back **as that account** — a non-administrator
//     sees only their own sessions (spec 3.8), so it is the one subtest here
//     that is insensitive to everything the others did.
//
// # Two accounts this fixture has that 002 plan 8's six do not
//
// They are appended to the fixture on this installation rather than added to
// `fixtureAccounts`, because that list is what AC-1's golden, AC-6's five
// public users and AC-7's six subjects are written against: a seventh account
// there would move three recorded bodies in another file to buy a fixture only
// this one needs.
func TestTheSessionsLockoutAndParameterMatrixAtTheWire(t *testing.T) {
	t.Parallel()

	server := newInstallation(t,
		// The second account of AC-10: a lockout threshold of two, like
		// `locked-out`, but reached with a success in between. Two accounts,
		// because a counter that resets and a counter that does not can only
		// be told apart by running both sequences, and the first of them
		// leaves its account permanently disabled (002 plan 6.7).
		withProvisionedAccount(resettingAccount, fixturePassword+"\n",
			"--login-attempts-before-lockout", fixtureLockoutThreshold, "--hidden=false"),

		// AC-13's session ceiling. One, because a ceiling of one is the
		// smallest number of logins that can exceed it, and because it is the
		// request register row U-13 names as the one that settles the
		// contradiction below.
		withProvisionedAccount(ceilingAccount, fixturePassword+"\n",
			"--max-active-sessions", sessionCeiling, "--hidden=false"),
	)

	// The two seats the parameter matrix needs: an administrator, who sees
	// every session, and a restricted account, who sees one. They are logged in
	// here rather than inside a subtest because a login is what creates the
	// session each of them is, and AC-15's exclusion row is arithmetic against
	// when the restricted seat was last seen.
	administrator := logIn(t, server, administratorDevice, administratorAccount, fixturePassword)
	restricted := logIn(t, server, restrictedDevice, restrictedAccount, fixturePassword)

	t.Run("AC-9: a posted declaration is hoisted, echoed, and not believed", func(t *testing.T) {
		assertTheDeclarationIsHoistedAndNotBelieved(t, server, administrator)
	})
	t.Run("AC-13: the declaration is replaced rather than merged into", func(t *testing.T) {
		assertASecondDeclarationReplacesTheFirst(t, server, administrator)
	})
	t.Run("AC-13: an authenticated request advances LastActivityDate", func(t *testing.T) {
		assertActivityAdvancesAcrossTwoRequests(t, server, administrator)
	})
	t.Run("AC-15: the parameter matrix", func(t *testing.T) {
		assertTheParameterMatrix(t, server, administrator, restricted)
	})
	t.Run("AC-10: the lockout, and the success that resets it", func(t *testing.T) {
		assertTheLockout(t, server)
	})
	t.Run("AC-13: exceeding MaxActiveSessions evicts the least recently used session", func(t *testing.T) {
		assertTheSessionCeilingEvicts(t, server)
	})
}

// The two accounts this file adds to 002 plan 8's fixture, and the ceiling one
// of them carries.
const (
	resettingAccount = "resetting"
	ceilingAccount   = "ceiling"
	sessionCeiling   = "1"
)

// The devices the two seats hold their credentials on.
//
// A device of their own, because a second authentication from one device
// revokes the first token (002 plan 6.5) — T17's note, which cost the sweep a
// half-finished run once — and because `deviceId` is the parameter half of
// AC-15 narrows on, so the two seats have to be distinguishable by device as
// well as by account.
const (
	administratorDevice = "sessions-administrator"
	restrictedDevice    = "sessions-restricted"
)

// pastTheActivityThrottle is longer than the interval a session's
// `LastActivityDate` may be written at (002 plan 6.10: at most once per session
// per second), and longer than the one-second window AC-15's exclusion row
// asks for.
//
// **It is real elapsed time because this package cannot do anything else.**
// `conformance/` starts the binary, so there is no clock to replace: the server
// reads its own, and a test that wanted to see a date move has to let one move.
// The margin over one second is 200 ms, and it only ever grows — `time.Sleep`
// blocks for *at least* the duration it is given, and every way a loaded
// machine can be slow makes the elapsed interval longer rather than shorter.
const pastTheActivityThrottle = 1200 * time.Millisecond

// 002 AC-9 at the wire: a client posts what it can do, and the session reports
// it — twice over, in two different senses, which is the whole of this
// criterion.
//
// # The three things a session says about a declaration
//
//  1. `PlayableMediaTypes` and `SupportedCommands` are **hoisted verbatim** to
//     the top level of the session (spec 3.8). They are compared as the raw
//     bytes of their arrays, so an implementation that sorted them, deduplicated
//     them or rebuilt them from a set is a failure: the reference copies the
//     client's list and this server has to as well.
//  2. `SupportsMediaControl` is the **server's** judgement and is `false`, beside
//     the client's own `true` echoed back inside `Capabilities`. That pair is
//     measured, not assumed — the reference reports false at the top level for a
//     session that declared true
//     [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28] —
//     and it is the one assertion here that a build hoisting *everything* fails.
//  3. The stored document is echoed **whole and unparsed**, which is where the
//     unknown property survives into /Sessions
//     (behaviours 5.9, a declared divergence and not parity).
//
// # The declaration carries a property the reference would drop
//
// `AtriumUnknownProperty` is in the posted document on purpose. The reference
// accepts it — the 204 — and drops it from `Capabilities`; this server keeps it.
// A test that posted only declared properties would pass on a build that
// decoded the document into a struct and re-encoded it, which is precisely the
// build behaviours 5.9 says this server must not be.
func assertTheDeclarationIsHoistedAndNotBelieved(t *testing.T, server *server, held credential) {
	t.Helper()

	posted := server.send(t, http.MethodPost, capabilitiesPath, goldenHost, held.bearing(),
		[]byte(fullDeclaration))

	// AC-13's first clause, asserted where the request is: 204 and no body at
	// all. The content type is asserted absent rather than empty, because a 204
	// carrying `application/json` and nothing else is a response a client can
	// see.
	if posted.status != http.StatusNoContent {
		t.Fatalf("posting a declaration: status %d, want %d\nbody: %s",
			posted.status, http.StatusNoContent, posted.body)
	}
	if len(posted.body) != 0 {
		t.Errorf("the 204 carries a body: %q", posted.body)
	}
	if contentType := posted.header.Get("Content-Type"); contentType != "" {
		t.Errorf("the 204 declares Content-Type %q, want none", contentType)
	}

	row := theCallersOwnSession(t, server, held)

	if got := rawField(t, row, "PlayableMediaTypes"); got != declaredMediaTypes {
		t.Errorf("PlayableMediaTypes = %s, want the declared list %s verbatim", got, declaredMediaTypes)
	}
	if got := rawField(t, row, "SupportedCommands"); got != declaredCommands {
		t.Errorf("SupportedCommands = %s, want the declared list %s verbatim", got, declaredCommands)
	}

	// The judgement, beside the declaration it disagrees with.
	for _, flag := range []string{"SupportsMediaControl", "SupportsRemoteControl"} {
		if got := rawField(t, row, flag); got != "false" {
			t.Errorf("%s = %s, want false — the declaration is the client's and the flag is the server's", flag, got)
		}
	}
	if got := rawField(t, row, "Capabilities"); got != fullDeclaration {
		t.Errorf("Capabilities = %s\nwant the posted document, byte for byte: %s", got, fullDeclaration)
	}
	if !strings.Contains(rawField(t, row, "Capabilities"), `"SupportsMediaControl":true`) {
		t.Error("the echoed declaration lost the client's own SupportsMediaControl: true, " +
			"which is the value the flag above is asserted beside")
	}
}

// The declaration AC-9 posts, and the two lists it hoists.
//
// The property names are PascalCase because that is what a real client sends,
// and the document is spelled as one literal rather than marshalled for
// fixture_test.go's reason: what a test states is what goes on the wire, and a
// marshaller is a second opinion about it. The two lists are spelled again
// separately so that the hoist is compared against the bytes that were sent
// rather than against a substring of the whole.
const (
	declaredMediaTypes = `["Audio","Video","Book"]`
	declaredCommands   = `["Play","Pause","SetVolume"]`

	fullDeclaration = `{"PlayableMediaTypes":` + declaredMediaTypes +
		`,"SupportedCommands":` + declaredCommands +
		`,"SupportsMediaControl":true,"AtriumUnknownProperty":"kept"}`
)

// smallerDeclaration is AC-13's second post, and it is smaller on purpose.
//
// It declares one property of the four above. A build that decoded the posted
// document over the stored one instead of replacing it would keep
// `SupportedCommands`, `SupportsMediaControl` and the unknown property, and
// would pass every "post it, read it back" assertion ever written — because
// everything a merge keeps was posted at some point. Only a second, *smaller*
// post can see the difference. T15 predicted this on /Users/Configuration and
// T16 confirmed it one route over; this is the same assertion at the wire.
const smallerDeclaration = `{"PlayableMediaTypes":["Audio"]}`

// 002 AC-13's replacement clause: the route is named `Full` and behaves like it.
func assertASecondDeclarationReplacesTheFirst(t *testing.T, server *server, held credential) {
	t.Helper()

	posted := server.send(t, http.MethodPost, capabilitiesPath, goldenHost, held.bearing(),
		[]byte(smallerDeclaration))
	if posted.status != http.StatusNoContent {
		t.Fatalf("posting the smaller declaration: status %d, want %d\nbody: %s",
			posted.status, http.StatusNoContent, posted.body)
	}

	row := theCallersOwnSession(t, server, held)

	if got := rawField(t, row, "Capabilities"); got != smallerDeclaration {
		t.Errorf("Capabilities = %s\nwant the second document alone: %s\n"+
			"Anything of the first document surviving here is a merge, and a merge passes every round trip.",
			got, smallerDeclaration)
	}
	if got := rawField(t, row, "PlayableMediaTypes"); got != `["Audio"]` {
		t.Errorf("PlayableMediaTypes = %s, want [\"Audio\"]", got)
	}
	// The list the second declaration does not name goes back to the empty
	// list the session started with, rather than keeping the first
	// declaration's. This is the assertion a merge fails.
	if got := rawField(t, row, "SupportedCommands"); got != `[]` {
		t.Errorf("SupportedCommands = %s, want [] — the second declaration names none, "+
			"so keeping the first one's is a merge", got)
	}
}

// 002 AC-13: an authenticated request advances the session's
// `LastActivityDate`.
//
// # Why this needs a wait and cannot be written without one
//
// The date is written at most once per session per second (002 plan 6.10), so
// two requests sent back to back read the same value — correctly. This package
// starts the binary and cannot replace its clock, so the only way to see the
// date move is to let a second pass. Two reads inside one second would be a
// test that passes on a server that never writes the date at all.
//
// # It is the caller's *own* session that moves, and that is the trap
//
// T16 recorded it one layer in and it is worth repeating here, because the next
// test in this file depends on it: the request that reads the list is itself an
// authenticated request, so it stamps the session it is sent from before the
// handler reads anything. A test that expected a session to stay still while
// asking about it would be asking the wrong question.
func assertActivityAdvancesAcrossTwoRequests(t *testing.T, server *server, held credential) {
	t.Helper()

	before := rawField(t, theCallersOwnSession(t, server, held), "LastActivityDate")

	time.Sleep(pastTheActivityThrottle)

	after := rawField(t, theCallersOwnSession(t, server, held), "LastActivityDate")

	if after == before {
		t.Fatalf("LastActivityDate is still %s after a second request more than %s later; "+
			"a session that never moves is a session activeWithinSeconds can never exclude",
			before, pastTheActivityThrottle)
	}
	// Forwards, and spelled the way every other date on this wire is spelled.
	// A build that wrote the zero instant on every request would move the value
	// and would fail here.
	if after <= before {
		t.Errorf("LastActivityDate went backwards: %s then %s", before, after)
	}
}

// theCallersOwnSession reads GET /Sessions as this credential and hands back
// the row belonging to the device it was minted from.
//
// It narrows by device rather than taking the first row, because an
// administrator sees every session on the installation and this helper is
// called with both seats.
func theCallersOwnSession(t *testing.T, server *server, held credential) []byte {
	t.Helper()

	got := server.get(t, sessionsPath, goldenHost, held.bearing())
	if got.status != http.StatusOK {
		t.Fatalf("GET %s: status %d, want %d\nbody: %s", sessionsPath, got.status, http.StatusOK, got.body)
	}
	for _, row := range arrayElements(t, got.body) {
		if unquote(t, rawField(t, row, "DeviceId")) == held.device {
			return row
		}
	}
	t.Fatalf("no session on device %s is in the answer:\n%s", held.device, got.body)
	return nil
}

// 002 AC-15: spec 3.8's three parameters over the wire, as the six requests
// 002 plan 8 names plus the combination.
//
// # What is compared as bytes, and what is compared as rows
//
// A body that is a constant is compared as bytes — the empty list is the two
// characters `[]` and never `null`, and the 403 is compared against the same
// golden file AC-2's three refusals are compared against. A body that is a
// *list of sessions* is compared by the identifiers in it, because the caller's
// own `LastActivityDate` moves between two reads by design (AC-13, one test up),
// so two readings of "the same list" are not the same bytes and asserting that
// they were would be asserting against this feature's own lifecycle.
//
// # The exclusion row has a fixture of its own, and this is why
//
// T16's finding, which cost a real failure one layer in: an authenticated
// request advances **its own** session's date, so a window measured over a
// fixture the restricted seat has already sent a request from is a window over
// two recent rows — a green case asserting nothing. So every request this seat
// makes happens first, then a second passes, and only then is the window asked
// for. The companion row with a window wide enough to reach both sessions is
// what makes the narrow one a filter rather than a list that was empty anyway.
func assertTheParameterMatrix(t *testing.T, server *server, administrator, restricted credential) {
	t.Helper()

	// Every row that is not the exclusion, from both seats. They run before the
	// wait for the reason above.
	t.Run("deviceId matches without regard to case", func(t *testing.T) {
		rows := sessionRows(t, server, administrator,
			sessionsPath+"?deviceId="+strings.ToUpper(restrictedDevice))
		if len(rows) != 1 {
			t.Fatalf("naming %s answered %d sessions, want 1: %v",
				strings.ToUpper(restrictedDevice), len(rows), rows)
		}
		// The stored spelling comes back, not the one that was asked for.
		if rows[0] != restrictedDevice {
			t.Errorf("the matched session's DeviceId is %q, want the stored spelling %q",
				rows[0], restrictedDevice)
		}
	})

	t.Run("deviceId= is ignored rather than read as a device nothing is named after", func(t *testing.T) {
		// spec 3.8, measured
		// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29].
		// No session in this fixture carries an empty DeviceId, so an
		// implementation comparing unconditionally answers [] here.
		assertBothSeatsAreListed(t, server, administrator, sessionsPath+"?deviceId=")
	})

	t.Run("a deviceId nothing matches is 200 with the empty list", func(t *testing.T) {
		assertTheEmptyList(t, server, administrator, sessionsPath+"?deviceId=no-such-device")
	})

	t.Run("activeWithinSeconds at zero and at a negative value are the unfiltered list", func(t *testing.T) {
		// Not a refusal, and not a window of nothing: spec 3.8 measures both as
		// ignored [probe: tools/probe_session_filters.py, Jellyfin 10.11.11,
		// 2026-08-29], and the domain applies the parameter only when it is
		// greater than zero.
		assertBothSeatsAreListed(t, server, administrator, sessionsPath+"?activeWithinSeconds=0")
		assertBothSeatsAreListed(t, server, administrator, sessionsPath+"?activeWithinSeconds=-5")
	})

	t.Run("a restricted seat naming another user's device is 200 with the empty list", func(t *testing.T) {
		// spec 3.8's observable half, first answer of two: naming somebody
		// else's *device* is an empty 200 and not a refusal, and it is the
		// ordinary result of a filter that matched a row this caller may not
		// see.
		assertTheEmptyList(t, server, restricted, sessionsPath+"?deviceId="+administratorDevice)
	})

	t.Run("controllableByUserId naming the caller is accepted from either seat", func(t *testing.T) {
		// Accepted from any caller, administrator or not (spec 3.8), and
		// answered with the empty list because v1 attaches no control channel.
		assertTheEmptyList(t, server, restricted,
			sessionsPath+"?controllableByUserId="+restricted.userID)
		assertTheEmptyList(t, server, administrator,
			sessionsPath+"?controllableByUserId="+administrator.userID)
	})

	t.Run("an administrator naming somebody else is 200 with the empty list", func(t *testing.T) {
		assertTheEmptyList(t, server, administrator,
			sessionsPath+"?controllableByUserId="+restricted.userID)
	})

	t.Run("a restricted seat naming somebody else is the 403 of AC-2's golden", func(t *testing.T) {
		assertTheControllableRefusal(t, server, restricted, administrator.userID)
	})

	t.Run("the combination is its own case", func(t *testing.T) {
		assertTheCombination(t, server, restricted)
	})

	// And now the exclusion, which is the one row that needs a fixture rather
	// than a request. Nothing above touched the restricted seat's session after
	// this point, and this wait is what puts it outside a one-second window
	// while the administrator's own session is stamped by the very request that
	// asks.
	time.Sleep(pastTheActivityThrottle)

	t.Run("activeWithinSeconds excludes a session that has been idle longer", func(t *testing.T) {
		narrow := sessionRows(t, server, administrator, sessionsPath+"?activeWithinSeconds=1")
		if len(narrow) != 1 || narrow[0] != administratorDevice {
			t.Fatalf("a one-second window answered %v, want only %s: the restricted seat has been "+
				"idle for more than %s", narrow, administratorDevice, pastTheActivityThrottle)
		}

		// The companion. Without it the row above passes on a server whose
		// session list was empty of everything but the caller for some other
		// reason, and the filter would never have been asked anything.
		assertBothSeatsAreListed(t, server, administrator, sessionsPath+"?activeWithinSeconds=3600")
	})
}

// assertBothSeatsAreListed is "the unfiltered list", asserted by the two
// devices that have to be in it.
//
// It is a superset assertion and not an equality, because later subtests on
// this installation open sessions of their own and the criterion is about which
// rows a parameter *keeps* rather than about how many sessions exist.
func assertBothSeatsAreListed(t *testing.T, server *server, held credential, path string) {
	t.Helper()

	rows := sessionRows(t, server, held, path)
	for _, device := range []string{administratorDevice, restrictedDevice} {
		if !contains(rows, device) {
			t.Errorf("%s answered %v, want it to include %s", path, rows, device)
		}
	}
}

// assertTheEmptyList compares the whole body against the two bytes an empty
// list is.
//
// Bytes, because `null` and `[]` are the same value to anything that parses the
// body and a different answer to every client that reads its length
// (Principle VIII). internal/wire serialises a nil slice as `null`, so this is
// a difference this server could produce.
func assertTheEmptyList(t *testing.T, server *server, held credential, path string) {
	t.Helper()

	got := server.get(t, path, goldenHost, held.bearing())
	if got.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", path, got.status, http.StatusOK, got.body)
	}
	if string(got.body) != "[]" {
		t.Errorf("%s answered %s, want []", path, got.body)
	}
}

// 002 AC-4's second half and AC-15's refusal, which are the same request: the
// one place in this feature where a valid token is refused for **who its holder
// is**.
//
// The body is compared against the golden AC-2's three refusals are compared
// against, which is what makes "the same 25 bytes" an assertion rather than a
// number written twice. The content type is compared as a whole field value
// because the measured field has **no charset parameter** and a `Contains`
// check would pass on `text/plain; charset=utf-8`, which is a different field
// [probe: tools/probe_session_filters.py, Jellyfin 10.11.11, 2026-08-29].
func assertTheControllableRefusal(t *testing.T, server *server, held credential, other string) {
	t.Helper()

	path := sessionsPath + "?controllableByUserId=" + other
	got := server.get(t, path, goldenHost, held.bearing())

	if got.status != http.StatusForbidden {
		t.Fatalf("%s: status %d, want %d\nbody: %s", path, got.status, http.StatusForbidden, got.body)
	}
	if contentType := got.header.Get("Content-Type"); contentType != "text/plain" {
		t.Errorf("Content-Type: got %q, want %q — the charset parameter is absent and that is the measurement",
			contentType, "text/plain")
	}
	if length := got.header.Get("Content-Length"); length != "25" {
		t.Errorf("Content-Length: got %q, want %q", length, "25")
	}
	assertGolden(t, refusalGolden, got.body)
}

// The combination case, named for the combination and **not** for the order.
//
// # What it catches
//
// Two wrong builds that every other row in this matrix is satisfied by:
//
//   - one that **ignores** `controllableByUserId`, which 002 plan 6.10 names as
//     the failure mode — it would answer this seat's own session on that device,
//     where the reference's first clause answers nothing;
//   - one that lets `controllableByUserId` **widen** the list back, which is
//     what a rule appended rather than substituted would do.
//
// The companion request is what makes either of those visible: the device alone
// answers a row, so the empty answer above is the second parameter's doing and
// not an empty list arriving by accident.
//
// # What it does not catch, and why the task list said otherwise
//
// It cannot tell which of the two ran first. The task list's own *Verified by*
// line asked for this case *"named for what it proves — that `deviceId` still
// narrows a request that also carries `controllableByUserId`"*, and that claim
// had already been struck: T9 removed it from 002 plan 6.10 on 2026-09-03 after
// making the domain case fail on it, because the early return answers `[]`
// **whatever `deviceId` narrowed to**. It is the third appearance of one wrong
// sentence — spec 3.8 lost it, plan 6.10 lost it, and the task list inherited it
// — and the task entry now records that. The half of that line that stands is
// the warning it ends with: this case must not be named for the *order*, which
// no request distinguishes.
//
// The day a feature attaches a control channel, this case becomes discriminating
// about the order too, and internal/sessions' visible_test.go says the same
// thing where the domain half of it lives.
func assertTheCombination(t *testing.T, server *server, held credential) {
	t.Helper()

	both := sessionsPath + "?deviceId=" + restrictedDevice + "&controllableByUserId=" + held.userID
	assertTheEmptyList(t, server, held, both)

	deviceOnly := sessionRows(t, server, held, sessionsPath+"?deviceId="+restrictedDevice)
	if len(deviceOnly) == 0 {
		t.Fatal("the device alone answers nothing, so the request above proves nothing about controllableByUserId")
	}
}

// sessionRows issues one GET /Sessions and answers the DeviceId of each row.
//
// The device rather than the identifier, because the device is what this file
// states — the session identifier is derived from the client and the device
// (002 plan 6.5) and transcribing it here would be a test agreeing with a
// derivation it cannot perform.
func sessionRows(t *testing.T, server *server, held credential, path string) []string {
	t.Helper()

	got := server.get(t, path, goldenHost, held.bearing())
	if got.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", path, got.status, http.StatusOK, got.body)
	}
	elements := arrayElements(t, got.body)
	devices := make([]string, 0, len(elements))
	for _, row := range elements {
		devices = append(devices, unquote(t, rawField(t, row, "DeviceId")))
	}
	return devices
}

func contains(all []string, wanted string) bool {
	for _, one := range all {
		if one == wanted {
			return true
		}
	}
	return false
}

// 002 AC-10 at the wire: after `LoginAttemptsBeforeLockout` failures the
// account answers **403 even with the correct credentials**, and one success
// afterwards resets the counter.
//
// # Two accounts, because one cannot show both halves
//
// The lockout is stored as the disabled flag (002 plan 6.7, and the reference
// does the same
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:634-641 @ v10.11.11]),
// so it is permanent until an operator clears it and v1 serves no route that
// does. An account that has proved the first half can prove nothing else.
//
// # The correct password is what makes the first half a criterion
//
// A locked-out account refused a *wrong* password would be indistinguishable
// from an account that was simply not locked out at all. The third request here
// carries the password that worked before the failures, which is what makes the
// 403 a statement about the account rather than about the credential — the same
// shape AC-2's disabled row takes.
//
// # 403 rather than 401, and where that number comes from
//
// It is v1's own decision and it is open as OQ-5: the reference's answer here
// has never been measured, because measuring it costs somebody's account a
// lockout counter no probe can reset. The source says why 403 is the right
// guess — a lockout *sets the disabled flag*, and a disabled account is
// measured to be 403
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26] — so
// on the reference a locked-out account *is* a disabled one on every later
// attempt. The body is compared against AC-2's golden for the same reason.
func assertTheLockout(t *testing.T, server *server) {
	t.Helper()

	// The threshold is two, and the reference's comparison increments before it
	// compares [source: .../UserManager.cs:636-641 @ v10.11.11], so the second
	// failure is the one that locks.
	for attempt := 1; attempt <= 2; attempt++ {
		refused := authenticate(t, server, "lockout-attempt", lockedOutAccount, "not-the-password")
		if refused.status != http.StatusUnauthorized {
			t.Fatalf("failure %d: status %d, want %d\nbody: %s",
				attempt, refused.status, http.StatusUnauthorized, refused.body)
		}
		assertGolden(t, refusalGolden, refused.body)
	}

	locked := authenticate(t, server, "lockout-correct", lockedOutAccount, fixturePassword)
	if locked.status != http.StatusForbidden {
		t.Fatalf("the correct password after %s failures answered %d, want %d — "+
			"an account that still authenticates has not been locked out\nbody: %s",
			fixtureLockoutThreshold, locked.status, http.StatusForbidden, locked.body)
	}
	assertGolden(t, refusalGolden, locked.body)

	// The second half, on the account nothing has locked: a failure, a success,
	// a failure, and then the correct password still works. Without the reset
	// the two failures reach the same threshold as above and this last request
	// is a 403 — which is exactly the build this sequence rules out.
	failing := authenticate(t, server, "reset-first-failure", resettingAccount, "not-the-password")
	if failing.status != http.StatusUnauthorized {
		t.Fatalf("the first failure answered %d, want %d\nbody: %s",
			failing.status, http.StatusUnauthorized, failing.body)
	}

	logIn(t, server, "reset-success", resettingAccount, fixturePassword)

	failing = authenticate(t, server, "reset-second-failure", resettingAccount, "not-the-password")
	if failing.status != http.StatusUnauthorized {
		t.Fatalf("the failure after the success answered %d, want %d\nbody: %s",
			failing.status, http.StatusUnauthorized, failing.body)
	}

	admitted := authenticate(t, server, "reset-final", resettingAccount, fixturePassword)
	if admitted.status != http.StatusOK {
		t.Fatalf("after a failure, a success and a failure the correct password answered %d, want %d — "+
			"the success in the middle did not reset the counter\nbody: %s",
			admitted.status, http.StatusOK, admitted.body)
	}
}

// 002 AC-13's last clause: exceeding `MaxActiveSessions` evicts the least
// recently used session **and its token**.
//
// # This test asserts the specification, and the reference does the opposite
//
// 002 plan 6.7 states the contradiction and decides it, and this comment carries
// it so that the request which settles it arrives at a test already saying what
// it expects. Spec 3.8's lifecycle table and AC-13 evict; the reference counts
// the user's sessions and throws
// `SecurityException("User is at their maximum number of sessions.")`
// [source: Emby.Server.Implementations/Session/SessionManager.cs:1623-1629 @ v10.11.11],
// which its exception filter turns into the 403 and the same 25 bytes this file
// compares three other refusals against. **So the second login below is a 200
// here and would be a 403 there**, and the two answers differ in the direction a
// client notices: eviction logs another device out, refusal keeps it.
//
// The specification is implemented as written because
// AGENTS.md 1.3 makes the running server the tie-breaker, there is none in this
// run, and source evidence does not discharge a specification. It is register
// row U-13 in docs/compatibility/reference-target.md, and this is that row's
// "one request settles it": two logins on an account whose `MaxActiveSessions`
// is 1. Whoever runs it against a reference changes the two expectations below
// and nothing else.
//
// # The account reads its own list, which is what makes this subtest portable
//
// `ceiling` is a non-administrator, so GET /Sessions answers **only its own
// sessions** (spec 3.8). Everything the other subtests on this installation
// opened is invisible here, which is why this one does not care what ran before
// it.
func assertTheSessionCeilingEvicts(t *testing.T, server *server) {
	t.Helper()

	first := logIn(t, server, "ceiling-first", ceilingAccount, fixturePassword)

	second := authenticate(t, server, "ceiling-second", ceilingAccount, fixturePassword)
	if second.status != http.StatusOK {
		t.Fatalf("the login that exceeds a ceiling of %s answered %d, want %d.\n"+
			"The reference refuses this request with %d and 25 bytes; this server evicts and admits "+
			"(spec 3.8, AC-13, register U-13), and a change of answer here is that register row "+
			"being settled rather than a bug.\nbody: %s",
			sessionCeiling, second.status, http.StatusOK, http.StatusForbidden, second.body)
	}
	live := credentialFrom(t, second, "ceiling-second")

	// The evicted session's token is gone with it. This is the half a build
	// that deleted the row and left the tokens would fail, and it is asserted
	// on a route that requires one.
	stale := server.get(t, currentUserPath, goldenHost, first.bearing())
	if stale.status != http.StatusUnauthorized {
		t.Errorf("the evicted session's token answered %d on %s, want %d — "+
			"a session evicted without its token is a credential naming nothing\nbody: %s",
			stale.status, currentUserPath, http.StatusUnauthorized, stale.body)
	}

	rows := sessionRows(t, server, live, sessionsPath)
	if len(rows) != 1 || rows[0] != "ceiling-second" {
		t.Fatalf("the account holds %v after a second login, want only ceiling-second: "+
			"a ceiling of %s that admits two sessions has enforced nothing", rows, sessionCeiling)
	}
}
