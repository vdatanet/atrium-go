package conformance_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"regexp"
	"slices"
	"testing"
)

// 002 AC-6, AC-7, AC-8 and AC-12 against the running binary.
//
// # Why these are here as well as in internal/httpapi
//
// All four are already asserted at the HTTP boundary inside internal/httpapi,
// over handlers built in a test with a stubbed store. They are asserted again
// here, and the reason is 001's closing audit stated as a policy by
// 002 plan 8: **a criterion written about a request is not met by a test about
// the mechanism that serves it, however good that test is.** The similarity
// between the two files is the thing being tested. What is new here is that
// the accounts were made by the command an operator runs, the tokens were
// minted by the process, the routing is the binary's own, and the bytes came
// off a socket.
//
// # One installation, four criteria, four subtests
//
// T18's rule, inherited rather than re-derived: provisioning 002 plan 8's
// fixture costs six Argon2id derivations, each holding a 64 MiB arena, and
// enough of them in parallel is measurable in **another package** —
// internal/users' timing equalisation, the one check ADR-0006's argument
// stands on, failed CI at 71.3 ms against 107.5 ms with a 17.8 ms margin when
// this package held nine installations open
// `[measurement: GitHub Actions, go test ./..., 2026-09-03]`. So a criterion
// does not get an installation; a group of criteria that do not disturb one
// another gets one, and each criterion stays a named subtest so a failure
// still names one criterion.
//
// **These four do disturb one another in exactly one direction, and the order
// below is that direction written down.** Every reading here is a byte
// comparison between two responses, so what matters is that no *write* happens
// between two reads that are compared. AC-8 is the only subtest that writes —
// it replaces a configuration — so it runs last, and it writes the
// `restricted` account's rather than the administrator's, whose object AC-6,
// AC-7 and AC-12 all read. The seats are logged in once, up front, for the
// same reason: a login stamps `LastLoginDate`, and a login in the middle of a
// subtest would move a body another line of the same subtest had already read.

// The seats this file authenticates, and the device each one holds its
// credential on.
//
// Four of plan 8's six accounts can hold a seat. `disabled` cannot — the
// route answers it 403 whatever password it carries (AC-2) — and it is a
// *subject* below rather than a seat, which is the whole point of the matrix:
// an account nobody can log in as is still an account every seat can read.
var userRouteSeats = []string{
	administratorAccount,
	restrictedAccount,
	hiddenAccount,
	passwordlessAccount,
}

// passwordOf is what a seat sends on the login route.
//
// The passwordless account authenticates with the empty string and nothing
// else, which is ADR-0006's rule at the wire: an account with no credential
// record is not an account that accepts any credential.
func passwordOf(account string) string {
	if account == passwordlessAccount {
		return ""
	}
	return fixturePassword
}

func TestTheUserRoutesAtTheWire(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)

	// Every seat, logged in before any subtest reads anything — see the
	// ordering paragraph above.
	seats := make(map[string]credential, len(userRouteSeats))
	for _, seat := range userRouteSeats {
		seats[seat] = logIn(t, server, "user-routes-"+seat, seat, passwordOf(seat))
	}

	t.Run("AC-6: the public users are the same bytes an authenticated caller reads", func(t *testing.T) {
		assertThePublicUsersMatchTheAuthenticatedReading(t, server, seats)
	})
	t.Run("AC-7: the caller matrix", func(t *testing.T) {
		assertTheCallerMatrix(t, server, seats)
	})
	t.Run("AC-12: the caller's own object, whole", func(t *testing.T) {
		assertTheCurrentUserIsWhole(t, server)
	})
	t.Run("AC-8: every configuration property round-trips and an unknown one is dropped", func(t *testing.T) {
		assertTheConfigurationRoundTrips(t, server, seats[restrictedAccount])
	})
}

// 002 AC-6 on the fixture that has a hidden user.
//
// # The byte comparison is the criterion, and the exclusion is the floor
//
// spec 3.4 measured `/Users/Public` to be byte-identical to the same users
// read through an authenticated route, so what is asserted is an equality
// rather than two shape checks that happen to agree
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]. Each
// element of the public array is compared with that account read through
// `GET /Users/{userId}` — the raw element bytes against the raw response
// bytes, so a difference in key order, in casing, in a numeric type or in a
// null-versus-absent is a failure.
//
// # Why the five objects are asserted to differ from one another
//
// T13's lesson, and T14 paid it a second time: **a test over data with only
// one possible answer proves nothing.** A handler that answered one account's
// object to every request would satisfy every equality in this function. The
// five accounts here are provisioned to be distinguishable — an administrator,
// a restricted seat, a disabled account, one with no password and one with a
// lockout threshold of two — and the assertion that they read as five
// different objects is what makes the equalities above mean what they say.
func assertThePublicUsersMatchTheAuthenticatedReading(t *testing.T, server *server, seats map[string]credential) {
	t.Helper()

	held := seats[administratorAccount]

	public := server.get(t, publicUsersPath, goldenHost, nil)
	if public.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", publicUsersPath, public.status, http.StatusOK, public.body)
	}
	if contentType := public.header.Get("Content-Type"); contentType != jsonContentType {
		t.Errorf("Content-Type: got %q, want %q", contentType, jsonContentType)
	}

	listed := arrayElements(t, public.body)

	// The hidden account is excluded and the other five are listed, in the
	// order the store answers — `username_folded, id`.
	//
	// The reference orders by the **unfolded** username
	// [source: Jellyfin.Api/Controllers/UserController.cs:653-655 @ v10.11.11],
	// which is a reading and not a measurement; every name in this fixture is
	// already lower case, so the two orders agree here and this assertion is
	// about determinism (Principle VII) rather than about the divergence. The
	// register at T23 is owed the row and plan 6.2 carries it.
	names := make([]string, 0, len(listed))
	for _, element := range listed {
		names = append(names, unquote(t, rawField(t, element, "Name")))
	}
	want := []string{
		administratorAccount, disabledAccount, lockedOutAccount, passwordlessAccount, restrictedAccount,
	}
	if !equalStrings(names, want) {
		t.Fatalf("the public users are\n got %v\nwant %v", names, want)
	}

	// The exclusion is an exclusion and not an absence: the hidden account
	// exists, holds a seat, and is readable through the authenticated route.
	if slices.Contains(names, hiddenAccount) {
		t.Errorf("the hidden account is on the login screen: %v", names)
	}
	hiddenPath := userPath(seats[hiddenAccount].userID)
	if got := server.get(t, hiddenPath, goldenHost, held.bearing()); got.status != http.StatusOK {
		t.Fatalf("the hidden account is not readable either, so its absence above proves nothing: status %d\nbody: %s",
			got.status, got.body)
	}

	objects := make([][]byte, 0, len(listed))
	for i, element := range listed {
		objects = append(objects, element)
		t.Run(names[i], func(t *testing.T) {
			// Whole objects: Configuration and Policy travel to an
			// unauthenticated caller, which spec 3.4 measured and
			// behaviours 3.5 decided to replicate. This project's own table
			// asserted the opposite until it was measured.
			assertTheWholeUserObject(t, element, hasLoggedIn(names[i]))

			id := unquote(t, rawField(t, element, "Id"))
			read := server.get(t, userPath(id), goldenHost, held.bearing())
			if read.status != http.StatusOK {
				t.Fatalf("%s: status %d, want %d\nbody: %s", userPath(id), read.status, http.StatusOK, read.body)
			}
			if !bytes.Equal(element, read.body) {
				t.Errorf("the public reading and the authenticated reading of %s differ.\n got %s\nwant %s\n%s",
					names[i], element, read.body, firstDifference(element, read.body))
			}
		})
	}

	assertPairwiseDifferent(t, "the five public users", names, objects)
}

// 002 AC-7 at the wire: every seat against every subject, and the three
// refusals that are not about who asked.
//
// # What the matrix proves, and what a status assertion would not
//
// spec 3.7 measured that this route refuses no authenticated caller and that
// the bytes do not depend on who asked
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01]. The
// criterion is therefore an equality, and a "no refusal" check is not it: this
// route answered a 403 in this project's own specification from the day it was
// written until 2026-09-01, and the successor mistake is a handler that
// answers 200 with a **redacted** body, which every status assertion passes.
//
// So each subject is read by every seat and the readings are compared byte for
// byte, and the six subjects are asserted to read as six different objects —
// without which a handler answering one account to everybody satisfies the
// whole matrix.
func assertTheCallerMatrix(t *testing.T, server *server, seats map[string]credential) {
	t.Helper()

	// Every account in plan 8's fixture is a subject, including the disabled
	// one, which no seat can be.
	subjects := subjectIdentifiers(t, server, seats)
	subjectNames := []string{
		administratorAccount, restrictedAccount, hiddenAccount,
		disabledAccount, passwordlessAccount, lockedOutAccount,
	}

	readings := make([][]byte, 0, len(subjectNames))
	for _, subject := range subjectNames {
		var first []byte
		for _, seat := range userRouteSeats {
			t.Run(seat+" reads "+subject, func(t *testing.T) {
				got := server.get(t, userPath(subjects[subject]), goldenHost, seats[seat].bearing())
				if got.status != http.StatusOK {
					t.Fatalf("status %d, want %d — this route refuses no authenticated caller\nbody: %s",
						got.status, http.StatusOK, got.body)
				}
				assertTheWholeUserObject(t, got.body, hasLoggedIn(subject))
				if first == nil {
					first = got.body
					return
				}
				if !bytes.Equal(got.body, first) {
					t.Errorf("%s reads a different %s from %s did.\n got %s\nwant %s\n%s",
						seat, subject, userRouteSeats[0], got.body, first, firstDifference(got.body, first))
				}
			})
		}
		readings = append(readings, first)
	}

	// The clause spec 3.7 names by itself, asserted by name so that a failure
	// says which pair moved: the administrator's object as read by a
	// restricted stranger, against that administrator's own reading of it.
	t.Run("a restricted stranger reads an administrator's whole object", func(t *testing.T) {
		path := userPath(subjects[administratorAccount])
		stranger := server.get(t, path, goldenHost, seats[restrictedAccount].bearing())
		own := server.get(t, path, goldenHost, seats[administratorAccount].bearing())
		if stranger.status != http.StatusOK || own.status != http.StatusOK {
			t.Fatalf("statuses %d and %d, want %d for both", stranger.status, own.status, http.StatusOK)
		}
		if !bytes.Equal(stranger.body, own.body) {
			t.Errorf("the stranger's reading is redacted.\n got %s\nwant %s\n%s",
				stranger.body, own.body, firstDifference(stranger.body, own.body))
		}
		if !bytes.Contains(own.body, []byte(`"IsAdministrator":true`)) {
			t.Errorf("the administrator's object carries no flag a redacting handler would be tempted "+
				"to withhold, so the equality above proves less than it says:\n%s", own.body)
		}
	})

	assertPairwiseDifferent(t, "the six subjects", subjectNames, readings)

	t.Run("an identifier nobody has is the same 404 to everybody", func(t *testing.T) {
		assertTheUnknownIdentifierIsFourOhFour(t, server, seats)
	})
	t.Run("an identifier that is not one is the validation 400", func(t *testing.T) {
		assertAMalformedIdentifierIsTheValidationRefusal(t, server, seats[administratorAccount])
	})
	t.Run("the all-zero identifier is the twenty-five byte 400", func(t *testing.T) {
		assertTheEmptyIdentifierIsTheControllerRefusal(t, server, seats[administratorAccount])
	})
	t.Run("no credential is the empty 401", func(t *testing.T) {
		assertNoCredentialIsTheEmptyRefusal(t, server, subjects[administratorAccount])
	})
}

// unknownIdentifier is well formed and belongs to no account on any
// installation this package builds: every identifier here is derived from a
// folded account name (Principle VII), and no name folds to this.
const unknownIdentifier = "deadbeefdeadbeefdeadbeefdeadbeef"

// spec 3.7's second row: 404 carrying the JSON-encoded bare string
// "User not found", **the same body to an administrator and to a
// non-administrator** [probe: tools/probe_user_read.py, Jellyfin 10.11.11,
// 2026-09-01].
//
// The sameness is the assertion. A server that concealed which identifiers
// exist from a non-administrator — by answering it 403, or by answering a
// different message — is what this row rules out, and it is the same decision
// spec 3.4 already takes on /Users/Public: the reference conceals nothing from
// anybody who can reach the port.
//
// behaviours 1.11's **fourth** error shape, not the problem details every
// other handler-raised 404 in this project answers, which is why the body is a
// golden rather than a field comparison.
func assertTheUnknownIdentifierIsFourOhFour(t *testing.T, server *server, seats map[string]credential) {
	t.Helper()

	var first []byte
	for _, seat := range []string{administratorAccount, restrictedAccount} {
		got := server.get(t, userPath(unknownIdentifier), goldenHost, seats[seat].bearing())
		if got.status != http.StatusNotFound {
			t.Fatalf("%s: status %d, want %d\nbody: %s", seat, got.status, http.StatusNotFound, got.body)
		}
		if contentType := got.header.Get("Content-Type"); contentType != jsonContentType {
			t.Errorf("%s: Content-Type %q, want %q", seat, contentType, jsonContentType)
		}
		assertGolden(t, "user_not_found.json", got.body)
		if first == nil {
			first = got.body
			continue
		}
		if !bytes.Equal(got.body, first) {
			t.Errorf("the two seats are told different things about an identifier nobody has.\n got %s\nwant %s",
				got.body, first)
		}
	}
}

// malformedIdentifier is the segment spec 3.7's third row quotes back, spelled
// as the probe sent it.
const malformedIdentifier = "not-an-identifier"

// spec 3.7's third row: the model binder's validation 400, keyed on the
// parameter's **own** spelling and quoting the value back
// [probe: tools/probe_user_read.py, Jellyfin 10.11.11, 2026-09-01].
//
// # The golden states the trace identifier, and nothing else
//
// behaviours 1.11 records `traceId` as per-request by definition: it is
// compared by shape and never by value. Every other byte of this body — the
// problem-details type and title, the status, the key `userId`, and the
// apostrophes around the quoted value escaped to \u0027 by behaviours 1.16's
// escape pass — is compared as bytes. The shape is asserted before the value
// is stated, which is assertGoldenWithStatedMembers' own rule.
func assertAMalformedIdentifierIsTheValidationRefusal(t *testing.T, server *server, held credential) {
	t.Helper()

	got := server.get(t, userPath(malformedIdentifier), goldenHost, held.bearing())
	if got.status != http.StatusBadRequest {
		t.Fatalf("status %d, want %d\nbody: %s", got.status, http.StatusBadRequest, got.body)
	}
	if contentType := got.header.Get("Content-Type"); contentType != jsonContentType {
		t.Errorf("Content-Type: got %q, want %q", contentType, jsonContentType)
	}

	trace := unquote(t, rawField(t, got.body, "traceId"))
	if !traceIdentifier.MatchString(trace) {
		t.Fatalf("traceId = %q, want a W3C trace-context identifier", trace)
	}

	// The escape pass is asserted on the bytes as well as through the golden,
	// because a golden recorded from a build that had already lost it would
	// record the loss.
	if !bytes.Contains(got.body, []byte(`\u0027`+malformedIdentifier+`\u0027`)) {
		t.Errorf("the quoted value does not carry escaped apostrophes (behaviours 1.16):\n%s", got.body)
	}

	assertGoldenWithStatedMembers(t, "user_by_id_malformed.json", got.body, []statedMember{
		{name: "TraceId", value: trace},
	})
}

// traceIdentifier is the shape behaviours 1.11 compares a `traceId` by:
// W3C trace-context, version `00`, a 16-byte trace id, an 8-byte span id and
// `00` flags, all lower-case hex.
var traceIdentifier = regexp.MustCompile(`^00-[0-9a-f]{32}-[0-9a-f]{16}-00$`)

// allZeroIdentifier is .NET's Guid.Empty written the way every identifier on
// this API is written.
const allZeroIdentifier = "00000000000000000000000000000000"

// The fourth answer on a route spec 3.7's table gives three, asserted here as
// what this server does.
//
// # This row is owed to the table, and T22 is where it is owed
//
// An all-zero `userId` is well formed and belongs to nobody, so spec 3.7's
// table as written makes it the 16-byte 404. It is not: the reference's
// account lookup refuses an empty identifier before it queries anything
// [source: Jellyfin.Server.Implementations/Users/UserManager.cs:123-133 @ v10.11.11],
// the exception is mapped to 400 under text/plain
// [source: Jellyfin.Api/Middleware/ExceptionMiddleware.cs:92-99,123-136 @ v10.11.11],
// and the same request **measured** on another route that resolves an
// identifier answered exactly these 25 bytes (009 spec 3.8's identifier table,
// 2026-09-01). T14 implemented it rather than recording it, because the
// alternative was answering 404 to a request the reference refuses.
//
// **spec 3.7's table is still owed the row, and T22 owns that document.** This
// test asserts what the implementation does and says where the row is owed, so
// that the day the table gains it, the two already agree.
//
// The body is compared against the same golden AC-2's three refusals are
// compared against: six responses on one file, which is what makes "they carry
// the same 25 bytes" one assertion rather than six written alike.
func assertTheEmptyIdentifierIsTheControllerRefusal(t *testing.T, server *server, held credential) {
	t.Helper()

	got := server.get(t, userPath(allZeroIdentifier), goldenHost, held.bearing())
	if got.status != http.StatusBadRequest {
		t.Fatalf("status %d, want %d — the all-zero identifier is not the 404\nbody: %s",
			got.status, http.StatusBadRequest, got.body)
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

// spec 3.7's fourth row: no credential is the empty 401, on a path that names
// an account that exists.
//
// The subject exists on purpose. A 401 for an identifier nobody has would be
// the right status for the wrong reason, and this route reads the credential
// **before** it binds the segment (plan 7) — which is the order this assertion
// stands on and which the malformed case above shows from the other side.
func assertNoCredentialIsTheEmptyRefusal(t *testing.T, server *server, subject string) {
	t.Helper()

	assertEmptyRefusal(t, server.get(t, userPath(subject), goldenHost, nil),
		http.StatusUnauthorized, "no credential on a path naming an account that exists")
}

// 002 AC-12: GET /Users/Me returns the caller's spec 3.5 object in full,
// configuration and policy included.
//
// # The criterion is proven as two equalities rather than as a shape check
//
// The route is compared with the `User` member of the authentication result
// that issued the token, and with the same account read through
// `GET /Users/{userId}`. Three routes, one account, one set of bytes — which
// is plan 6.6's single filler asserted at the wire rather than at the function
// that implements it. A shape check over one of the three would pass on a
// build where the other two had drifted.
//
// The login is made from a device of its own so that the seats logged in by
// the parent test keep their tokens (plan 6.5 revokes a device's previous
// token), and it is made inside this subtest so that the `User` member it
// compares against carries this login's own `LastLoginDate`.
func assertTheCurrentUserIsWhole(t *testing.T, server *server) {
	t.Helper()

	const device = "current-user"
	login := authenticate(t, server, device, administratorAccount, fixturePassword)
	if login.status != http.StatusOK {
		t.Fatalf("authenticating: status %d, want %d\nbody: %s", login.status, http.StatusOK, login.body)
	}
	token := unquote(t, rawField(t, login.body, "AccessToken"))
	held := credential{token: token, device: device}
	inTheResult := []byte(rawField(t, login.body, "User"))

	me := server.get(t, currentUserPath, goldenHost, held.bearing())
	if me.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", currentUserPath, me.status, http.StatusOK, me.body)
	}
	if contentType := me.header.Get("Content-Type"); contentType != jsonContentType {
		t.Errorf("Content-Type: got %q, want %q", contentType, jsonContentType)
	}

	assertTheWholeUserObject(t, me.body, true)

	if !bytes.Equal(me.body, inTheResult) {
		t.Errorf("%s and the authentication result's User member differ.\n got %s\nwant %s\n%s",
			currentUserPath, me.body, inTheResult, firstDifference(me.body, inTheResult))
	}

	id := unquote(t, rawField(t, me.body, "Id"))
	byIdentifier := server.get(t, userPath(id), goldenHost, held.bearing())
	if byIdentifier.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", userPath(id), byIdentifier.status, http.StatusOK, byIdentifier.body)
	}
	if !bytes.Equal(me.body, byIdentifier.body) {
		t.Errorf("%s and %s answer different bytes for one account.\n got %s\nwant %s\n%s",
			currentUserPath, userPath(id), me.body, byIdentifier.body,
			firstDifference(me.body, byIdentifier.body))
	}
}

// The members of spec 3.5's object that travel, in the order the reference
// declares them [source: MediaBrowser.Model/Dto/UserDto.cs:26-105 @ v10.11.11].
//
// Four of the fourteen declared members are null on every account this binary
// can hold, and behaviours 1.7 omits a null globally: `ServerName`,
// `PrimaryImageTag`, `PrimaryImageAspectRatio` and — for as long as nothing
// calls TouchActivity — `LastActivityDate`.
//
// **`LastLoginDate` is the fifth, and it is absent per account rather than per
// build**, which is why it is a parameter here and not a row. spec 3.5 says
// the member is absent until the first login; the store keeps NULL for it, and
// a non-pointer field would answer `0001-01-01T00:00:00.0000000Z` for an
// account that has never logged in — a value where the reference sends no
// member at all, and the exact opposite of `SessionInfo.LastPlaybackCheckIn`,
// where the zero tick *is* the value. This fixture holds both states at once:
// four accounts hold seats and two — `disabled`, which cannot log in, and
// `locked-out`, which is never asked to — have never logged in, so the
// distinction is asserted at the wire rather than only where a store can be
// stubbed.
func userObjectMembers(hasLoggedIn bool) []string {
	members := []string{
		"Name", "ServerId", "Id",
		"HasPassword", "HasConfiguredPassword", "HasConfiguredEasyPassword", "EnableAutoLogin",
	}
	if hasLoggedIn {
		members = append(members, "LastLoginDate")
	}
	return append(members, "Configuration", "Policy")
}

// hasLoggedIn reports whether an account of this fixture has authenticated by
// the time the user routes are read.
//
// The seats have, by construction: the parent test logs every one of them in
// before any subtest runs. Nothing logs `disabled` in — the route refuses it
// 403 (AC-2) — and nothing logs `locked-out` in either, which is what makes it
// the second account in the absent state.
func hasLoggedIn(account string) bool {
	return slices.Contains(userRouteSeats, account)
}

// The sixteen configuration properties of spec 3.6, in the reference's
// declaration order
// [source: MediaBrowser.Model/Configuration/UserConfiguration.cs:35-76 @ v10.11.11].
var configurationMembers = []string{
	"AudioLanguagePreference", "PlayDefaultAudioTrack", "SubtitleLanguagePreference",
	"DisplayMissingEpisodes", "GroupedFolders", "SubtitleMode", "DisplayCollectionsView",
	"EnableLocalPassword", "OrderedViews", "LatestItemsExcludes", "MyMediaExcludes",
	"HidePlayedInLatest", "RememberAudioSelections", "RememberSubtitleSelections",
	"EnableNextEpisodeAutoPlay", "CastReceiverId",
}

// policyMembersThatTravel is spec 3.5's measured count: the reference sends
// **42** policy properties
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28] of the
// 44 it declares
// [source: MediaBrowser.Model/Users/UserPolicy.cs:16-68,112-114 @ v10.11.11],
// the absent two being the parental ratings, which are null and which
// behaviours 1.7 omits.
const policyMembersThatTravel = 42

// The parental ratings, named so that "42" is the arithmetic 44 − 2 rather
// than a number typed in to match.
var policyMembersThatAreNull = []string{"MaxParentalRating", "MaxParentalSubRating"}

// assertTheWholeUserObject is what "the whole object, Configuration and Policy
// included" means on the wire, in one place.
//
// It is one function because AC-6, AC-7 and AC-12 all say it about the same
// bytes and three copies would be three chances to weaken one. The member
// names are read **in the order the bytes carry them**, which a decoded object
// cannot see and which L3 compares.
func assertTheWholeUserObject(t *testing.T, object []byte, loggedIn bool) {
	t.Helper()

	want := userObjectMembers(loggedIn)
	if names := propertyNames(t, object); !equalStrings(names, want) {
		t.Errorf("the user object's members are\n got %v\nwant %v", names, want)
	}
	if loggedIn {
		// The member is there; that it is a date rather than a well-formed
		// something else is the cross-cutting L1 sweep's rule applied to a
		// value, and it is the assertion the member's mere presence is not.
		if value := unquote(t, rawField(t, object, "LastLoginDate")); !wireDate.MatchString(value) {
			t.Errorf("LastLoginDate = %q, want a date with seven fractional digits and a Z", value)
		}
	}

	configuration := json.RawMessage(rawField(t, object, "Configuration"))
	if names := propertyNames(t, configuration); !equalStrings(names, configurationMembers) {
		t.Errorf("Configuration's members are\n got %v\nwant %v", names, configurationMembers)
	}

	policy := json.RawMessage(rawField(t, object, "Policy"))
	names := propertyNames(t, policy)
	if len(names) != policyMembersThatTravel {
		t.Errorf("Policy carries %d members, want %d", len(names), policyMembersThatTravel)
	}
	for _, absent := range policyMembersThatAreNull {
		if slices.Contains(names, absent) {
			t.Errorf("Policy carries %s, which is null at the reference and omitted under behaviours 1.7", absent)
		}
	}
}

// 002 AC-8 at the wire: post all sixteen properties, read them back through
// /Users/Me, then post an unknown one and watch it be dropped.
//
// # The round trip is a byte equality because the posted document is the
// answer
//
// The document below is written in the model's own declaration order, so what
// the route stores and what /Users/Me answers can be compared with the very
// bytes that were sent. That makes one assertion out of three: every property
// survived, none was renamed or retyped, and the order on the way out is the
// order spec 3.6 records.
//
// **Every one of the sixteen values differs from the reference's default.** A
// document that agreed with the defaults anywhere would leave a handler that
// ignored the body passing on that property — T13's lesson, and the reason
// this literal looks perverse.
//
// # The unknown property is dropped, and that is the opposite of a session's
// capabilities
//
// spec 3.6: unknown properties are ignored, not rejected. behaviours 5.9
// records the capabilities document doing the **opposite** — an unknown
// property there survives into /Sessions, as a declared divergence. The two
// look like one question and are not, which is why both are asserted at the
// wire rather than one being read off the other.
func assertTheConfigurationRoundTrips(t *testing.T, server *server, held credential) {
	t.Helper()

	stored := postConfiguration(t, server, held, everyConfigurationProperty)
	if !bytes.Equal(stored, []byte(everyConfigurationProperty)) {
		t.Fatalf("the configuration did not round-trip.\n got %s\nwant %s\n%s",
			stored, everyConfigurationProperty,
			firstDifference(stored, []byte(everyConfigurationProperty)))
	}

	afterUnknown := postConfiguration(t, server, held, aChangedPropertyAndAnUnknownOne)
	if !bytes.Equal(afterUnknown, []byte(theChangedProperty)) {
		t.Errorf("the unknown property was not dropped, or a declared one did not survive.\n got %s\nwant %s\n%s",
			afterUnknown, theChangedProperty,
			firstDifference(afterUnknown, []byte(theChangedProperty)))
	}
	if bytes.Contains(afterUnknown, []byte(unknownConfigurationProperty)) {
		t.Errorf("%s survived into the user object:\n%s", unknownConfigurationProperty, afterUnknown)
	}
}

// postConfiguration replaces the caller's configuration and hands back what
// /Users/Me then carries under `Configuration`.
//
// The 204 is asserted here rather than in a row of its own because it is the
// route's whole success answer: a status, no body, and no content type. AC-13
// asserts the same three about the capabilities route at T20.
func postConfiguration(t *testing.T, server *server, held credential, document string) []byte {
	t.Helper()

	written := server.send(t, http.MethodPost, userConfigurationPath, goldenHost,
		held.bearing(), []byte(document))
	if written.status != http.StatusNoContent {
		t.Fatalf("%s: status %d, want %d\nbody: %s",
			userConfigurationPath, written.status, http.StatusNoContent, written.body)
	}
	if len(written.body) != 0 {
		t.Errorf("the 204 carries a body: %q", written.body)
	}
	if contentType := written.header.Get("Content-Type"); contentType != "" {
		t.Errorf("Content-Type: got %q, want the field to be absent on a 204", contentType)
	}

	me := server.get(t, currentUserPath, goldenHost, held.bearing())
	if me.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", currentUserPath, me.status, http.StatusOK, me.body)
	}
	return []byte(rawField(t, me.body, "Configuration"))
}

// The three documents AC-8 posts.
//
// Written as literals rather than marshalled, for the reason loginBody is:
// Principle VIII, what a test states is what goes on the wire, and a
// marshaller is a second opinion about that. The identifiers in the four
// arrays are 32 lowercase hex because that is the shape a Guid array carries
// (behaviours 1.4) — nothing here reads them, and a value that could not
// travel would be testing the encoder rather than the route.
const (
	everyConfigurationProperty = `{"AudioLanguagePreference":"eng",` +
		`"PlayDefaultAudioTrack":false,` +
		`"SubtitleLanguagePreference":"fra",` +
		`"DisplayMissingEpisodes":true,` +
		`"GroupedFolders":["1111111111111111111111111111111a","2222222222222222222222222222222b"],` +
		`"SubtitleMode":"Always",` +
		`"DisplayCollectionsView":true,` +
		`"EnableLocalPassword":true,` +
		`"OrderedViews":["3333333333333333333333333333333c"],` +
		`"LatestItemsExcludes":["4444444444444444444444444444444d"],` +
		`"MyMediaExcludes":["5555555555555555555555555555555e"],` +
		`"HidePlayedInLatest":false,` +
		`"RememberAudioSelections":false,` +
		`"RememberSubtitleSelections":false,` +
		`"EnableNextEpisodeAutoPlay":false,` +
		`"CastReceiverId":"atrium-has-no-cast-receiver"}`

	// The same sixteen with one value moved, which is what makes "the declared
	// ones survive" an assertion about this request rather than about the one
	// before it.
	theChangedProperty = `{"AudioLanguagePreference":"deu",` +
		`"PlayDefaultAudioTrack":false,` +
		`"SubtitleLanguagePreference":"fra",` +
		`"DisplayMissingEpisodes":true,` +
		`"GroupedFolders":["1111111111111111111111111111111a","2222222222222222222222222222222b"],` +
		`"SubtitleMode":"Always",` +
		`"DisplayCollectionsView":true,` +
		`"EnableLocalPassword":true,` +
		`"OrderedViews":["3333333333333333333333333333333c"],` +
		`"LatestItemsExcludes":["4444444444444444444444444444444d"],` +
		`"MyMediaExcludes":["5555555555555555555555555555555e"],` +
		`"HidePlayedInLatest":false,` +
		`"RememberAudioSelections":false,` +
		`"RememberSubtitleSelections":false,` +
		`"EnableNextEpisodeAutoPlay":false,` +
		`"CastReceiverId":"atrium-has-no-cast-receiver"}`

	// unknownConfigurationProperty is a name no version of this model has and
	// no reference response carries.
	unknownConfigurationProperty = "NotAConfigurationProperty"

	aChangedPropertyAndAnUnknownOne = `{"` + unknownConfigurationProperty + `":"kept?",` +
		`"AudioLanguagePreference":"deu",` +
		`"PlayDefaultAudioTrack":false,` +
		`"SubtitleLanguagePreference":"fra",` +
		`"DisplayMissingEpisodes":true,` +
		`"GroupedFolders":["1111111111111111111111111111111a","2222222222222222222222222222222b"],` +
		`"SubtitleMode":"Always",` +
		`"DisplayCollectionsView":true,` +
		`"EnableLocalPassword":true,` +
		`"OrderedViews":["3333333333333333333333333333333c"],` +
		`"LatestItemsExcludes":["4444444444444444444444444444444d"],` +
		`"MyMediaExcludes":["5555555555555555555555555555555e"],` +
		`"HidePlayedInLatest":false,` +
		`"RememberAudioSelections":false,` +
		`"RememberSubtitleSelections":false,` +
		`"EnableNextEpisodeAutoPlay":false,` +
		`"CastReceiverId":"atrium-has-no-cast-receiver"}`
)

// 002 AC-6's other half: the installation where every account is hidden.
//
// It is asserted as a subtest of the all-hidden fixture in mechanisms_test.go
// rather than on an installation of its own, which is T18's rule — see the
// ordering paragraph at the top of this file.
//
// # `[]` is only a criterion when the account is demonstrably there
//
// An empty array is what an installation with no accounts answers too, and
// that is the same bytes for a different reason. So the account is logged in
// and read through the authenticated route in the same subtest: an account
// that exists, holds a token and answers a whole object, and is still absent
// from the login screen. Without the second half this is a test that a server
// serving nothing would pass.
func assertTheAllHiddenPublicListIsAnExclusion(t *testing.T, server *server, held credential) {
	t.Helper()

	got := server.get(t, publicUsersPath, goldenHost, nil)
	if got.status != http.StatusOK {
		t.Fatalf("%s: status %d, want %d\nbody: %s", publicUsersPath, got.status, http.StatusOK, got.body)
	}
	if contentType := got.header.Get("Content-Type"); contentType != jsonContentType {
		t.Errorf("Content-Type: got %q, want %q", contentType, jsonContentType)
	}
	// The bytes, not a decoded length: internal/wire writes a nil slice as
	// `null`, which is a different document from an empty array and one no
	// client expects here (spec 3.4).
	if string(got.body) != "[]" {
		t.Fatalf("%s answered %s, want []", publicUsersPath, got.body)
	}

	read := server.get(t, userPath(held.userID), goldenHost, held.bearing())
	if read.status != http.StatusOK {
		t.Fatalf("the hidden account is not readable, so the empty array above proves nothing: status %d\nbody: %s",
			read.status, read.body)
	}
	assertTheWholeUserObject(t, read.body, true)
	if !unquoteBool(t, rawField(t, json.RawMessage(rawField(t, read.body, "Policy")), "IsHidden")) {
		t.Errorf("the account this fixture is built on is not hidden, so `[]` is not an exclusion:\n%s", read.body)
	}
}

// jsonContentType is what every 200 and every JSON refusal in this feature
// carries [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11,
// 2026-08-28].
const jsonContentType = "application/json; charset=utf-8"

// userPath is GET /Users/{userId} with the segment filled in.
func userPath(id string) string {
	return "/Users/" + id
}

// arrayElements splits a JSON array into its elements **without re-encoding
// them**, so each one is the bytes the server sent.
//
// That is what makes the comparison in AC-6 a byte comparison: a decode and a
// re-encode would agree about a body whose key order or numeric types had
// moved, which is the whole class of difference L3 exists to catch.
func arrayElements(t *testing.T, body []byte) []json.RawMessage {
	t.Helper()

	var elements []json.RawMessage
	if err := json.Unmarshal(body, &elements); err != nil {
		t.Fatalf("the body is not a JSON array: %v\n%s", err, body)
	}
	return elements
}

// subjectIdentifiers is every account's identifier, read off the wire rather
// than derived.
//
// This package cannot derive one — it may import nothing of ours — and
// transcribing what the server computed would be a test agreeing with itself.
// The five visible accounts come from /Users/Public and the hidden one from
// its own login, which is the only route that names it.
func subjectIdentifiers(t *testing.T, server *server, seats map[string]credential) map[string]string {
	t.Helper()

	subjects := map[string]string{hiddenAccount: seats[hiddenAccount].userID}
	for _, element := range arrayElements(t, server.get(t, publicUsersPath, goldenHost, nil).body) {
		subjects[unquote(t, rawField(t, element, "Name"))] = unquote(t, rawField(t, element, "Id"))
	}
	for _, name := range []string{
		administratorAccount, restrictedAccount, hiddenAccount,
		disabledAccount, passwordlessAccount, lockedOutAccount,
	} {
		if subjects[name] == "" {
			t.Fatalf("no identifier was found for %s, so the matrix below would be short a subject", name)
		}
	}
	return subjects
}

// assertPairwiseDifferent is T13's lesson written once and called twice: a
// test over data with only one possible answer proves nothing.
//
// Every equality in this file is satisfied by a server that answers one user
// object to every request. This is the assertion that rules that server out,
// and it belongs beside each equality rather than in a file of its own.
func assertPairwiseDifferent(t *testing.T, what string, names []string, objects [][]byte) {
	t.Helper()

	for i := range objects {
		for j := i + 1; j < len(objects); j++ {
			if bytes.Equal(objects[i], objects[j]) {
				t.Errorf("%s: %s and %s are the same bytes, so every equality asserted about them "+
					"is satisfied by a server answering one object to everybody:\n%s",
					what, names[i], names[j], objects[i])
			}
		}
	}
}

// unquoteBool reads a raw JSON boolean, the way unquote reads a raw string.
//
// A separate reader rather than a decode into an interface, because Principle
// VIII: the string "true" and the boolean true are one value to an interface
// and two documents to a client.
func unquoteBool(t *testing.T, raw string) bool {
	t.Helper()

	var value bool
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("%s is not a JSON boolean: %v", raw, err)
	}
	return value
}
