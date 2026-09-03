package conformance_test

import (
	"bytes"
	"net/http"
	"testing"
)

// 002 AC-3 against the running binary: the five token mechanisms of spec 3.1,
// the precedence between them, and the grammar of spec 3.2 that two of them are
// read with.
//
// internal/httpapi asserts all of this over two pure functions, exhaustively and
// far faster. What a request adds — and the reason plan 8 puts the criterion
// here as well — is that the *routes* read a credential through those functions
// rather than through something of their own, on a request that went through
// query canonicalisation, path folding and the whole pipeline first.

// The route classes AC-3 names, and the one it does not.
const (
	// A route that requires a token (002 plan 6.2). /Users/Me rather than
	// /Sessions, because its answer differs per caller and so a mechanism that
	// authenticated the wrong account would show up as different bytes rather
	// than as the same empty list.
	requiredRoute = currentUserPath

	// A route that reads no credential at all. It is not "accepts a token" in
	// the reference's sense — that is the image and delivery pair below — but
	// it is the one route in this feature where presenting one must change
	// nothing, which is the half of AC-3 that this feature can prove.
	unauthenticatedRoute = publicUsersPath
)

// tokenMechanism is one of 002 spec 3.1's five, expressed as the request it
// makes: a mechanism is a place to put a token, and at the wire that is a
// header field or a query string.
type tokenMechanism struct {
	name string

	// place puts a token into a request under construction. Both a header map
	// and a query string are given because two mechanisms are fields and two
	// are query names, and a pair sets one of each.
	place func(token string, header http.Header, query *string)
}

// The five, in the order 002 spec 3.1 measures them resolving.
var (
	viaAuthorization = tokenMechanism{"Authorization", func(token string, header http.Header, _ *string) {
		header.Set("Authorization", clientIdentification("mechanisms", token))
	}}
	viaEmbyAuthorization = tokenMechanism{"X-Emby-Authorization", func(token string, header http.Header, _ *string) {
		header.Set("X-Emby-Authorization", clientIdentification("mechanisms", token))
	}}
	viaEmbyToken = tokenMechanism{"X-Emby-Token", func(token string, header http.Header, _ *string) {
		header.Set("X-Emby-Token", token)
	}}
	viaApiKey = tokenMechanism{"?ApiKey=", func(token string, _ http.Header, query *string) {
		*query = appendQuery(*query, "ApiKey="+token)
	}}
	viaApiKeyUnderscore = tokenMechanism{"?api_key=", func(token string, _ http.Header, query *string) {
		*query = appendQuery(*query, "api_key="+token)
	}}
)

func allMechanisms() []tokenMechanism {
	return []tokenMechanism{viaAuthorization, viaEmbyAuthorization, viaEmbyToken, viaApiKey, viaApiKeyUnderscore}
}

func appendQuery(raw, fragment string) string {
	if raw == "" {
		return fragment
	}
	return raw + "&" + fragment
}

// request issues one GET carrying a token by the mechanisms given.
func requestWith(t *testing.T, s *server, path string, place func(header http.Header, query *string)) *response {
	t.Helper()

	header := http.Header{}
	query := ""
	place(header, &query)
	if query != "" {
		path += "?" + query
	}
	return s.get(t, path, goldenHost, header)
}

// AC-3's first half: all five mechanisms authenticate the same request
// identically on a route that requires a token.
//
// **Identically means the same bytes**, not merely 200 each time. A server that
// resolved one mechanism to a different account would answer 200 five times and
// two different bodies, and only a comparison sees it — which is why the
// fixture has six accounts rather than one, and why the token belongs to the
// restricted seat rather than to the administrator: /Users/Me answers the
// caller's own object, so a mechanism that fell back to "somebody" would answer
// a different object rather than the same one.
func TestTheFiveMechanismsAuthenticateOneRequestIdentically(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)
	held := logIn(t, server, "mechanisms", restrictedAccount, fixturePassword)

	var first []byte
	for _, mechanism := range allMechanisms() {
		t.Run(mechanism.name, func(t *testing.T) {
			got := requestWith(t, server, requiredRoute, func(header http.Header, query *string) {
				mechanism.place(held.token, header, query)
			})
			if got.status != http.StatusOK {
				t.Fatalf("%s on %s: status %d, want %d\nbody: %s",
					mechanism.name, requiredRoute, got.status, http.StatusOK, got.body)
			}
			if first == nil {
				first = got.body
				return
			}
			if !bytes.Equal(got.body, first) {
				t.Errorf("%s answered different bytes from the first mechanism.\n got %s\nwant %s",
					mechanism.name, got.body, first)
			}
		})
	}

	// The floor the comparison stands on: a body with nothing in it is the same
	// bytes five times over and proves nothing. T17's lesson, which cost a
	// whole sweep that walked an empty list while passing.
	if len(first) == 0 {
		t.Fatal("the compared responses were empty")
	}
}

// The same five on a route that reads no credential, where the criterion is
// that presenting one is never itself a reason to refuse — and never a reason
// to answer differently either (002 spec 3.4's measured byte-equality).
//
// Run over two installations. The one with six accounts answers a list with
// something in it; the all-hidden one answers `[]`, which is the case where a
// handler that consulted the credential would be most tempted to. AC-6's own
// assertions over that fixture — the exclusion, and the comparison with an
// authenticated reading of the same users — belong to T19; what is asserted
// here is only that the credential changes nothing.
func TestPresentingATokenChangesNothingOnARouteThatRequiresNone(t *testing.T) {
	t.Parallel()

	for _, installation := range []struct {
		name  string
		start func(t *testing.T) *server
	}{
		{"six accounts", func(t *testing.T) *server { return newInstallation(t) }},
		{"every account hidden", func(t *testing.T) *server { return newAllHiddenInstallation(t) }},
	} {
		t.Run(installation.name, func(t *testing.T) {
			t.Parallel()

			server := installation.start(t)
			held := logIn(t, server, "mechanisms", hiddenAccount, fixturePassword)

			anonymous := server.get(t, unauthenticatedRoute, goldenHost, nil)
			if anonymous.status != http.StatusOK {
				t.Fatalf("%s without a credential: status %d, want %d\nbody: %s",
					unauthenticatedRoute, anonymous.status, http.StatusOK, anonymous.body)
			}

			for _, mechanism := range allMechanisms() {
				t.Run(mechanism.name, func(t *testing.T) {
					got := requestWith(t, server, unauthenticatedRoute, func(header http.Header, query *string) {
						mechanism.place(held.token, header, query)
					})
					if got.status != http.StatusOK {
						t.Fatalf("%s carrying %s: status %d, want %d\nbody: %s",
							unauthenticatedRoute, mechanism.name, got.status, http.StatusOK, got.body)
					}
					if !bytes.Equal(got.body, anonymous.body) {
						t.Errorf("%s answered different bytes from the anonymous reading.\n got %s\nwant %s",
							mechanism.name, got.body, anonymous.body)
					}
				})
			}
		})
	}
}

// unknownToken is a well-formed credential this server never issued: 32
// lowercase hexadecimal characters that resolve to no session.
//
// It is the loser in every precedence pair below. A pair proves an order only
// when the two sides carry credentials that answer differently, and "one live
// token and one dead one" is the pair a request can see: the status says which
// of the two the server read.
const unknownToken = "ffffffffffffffffffffffffffffffff"

// The precedence of 002 spec 3.1, at the wire, in both directions.
//
//	Authorization  >  X-Emby-Authorization  >  X-Emby-Token  >  ?ApiKey= / ?api_key=
//
// Each pair runs twice. The winner carries the live token and the loser the
// unknown one, which must be 200; then they swap, which must be 401. **The
// second direction is the one that does the work** — a server that simply tried
// every mechanism until one authenticated would pass the first direction of
// every row here and fail every second one.
//
// The three adjacent pairs are measured pair by pair and in both directions
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]; the
// three non-adjacent ones are inferred from the chain
// (behaviours 2.4) and are labelled so nobody reads them as measurements. No
// pair here rests on plan 6.1's unmeasured generalisation — the row that does
// is the next test, and it is separated for exactly that reason.
func TestThePrecedenceBetweenTwoLiveMechanismsHoldsAtTheWire(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)
	held := logIn(t, server, "mechanisms", restrictedAccount, fixturePassword)

	for _, pair := range []struct {
		winner, loser tokenMechanism
		provenance    string
	}{
		{viaAuthorization, viaEmbyAuthorization, "measured"},
		{viaEmbyAuthorization, viaEmbyToken, "measured"},
		{viaEmbyToken, viaApiKey, "measured"},
		{viaAuthorization, viaEmbyToken, "inferred from the chain"},
		{viaAuthorization, viaApiKey, "inferred from the chain"},
		{viaEmbyAuthorization, viaApiKey, "inferred from the chain"},
	} {
		t.Run(pair.winner.name+" beats "+pair.loser.name+" ("+pair.provenance+")", func(t *testing.T) {
			for _, direction := range []struct {
				name                    string
				winnerToken, loserToken string
				status                  int
			}{
				{"the winner carries the live token", held.token, unknownToken, http.StatusOK},
				{"the winner carries the unknown token", unknownToken, held.token, http.StatusUnauthorized},
			} {
				got := requestWith(t, server, requiredRoute, func(header http.Header, query *string) {
					pair.winner.place(direction.winnerToken, header, query)
					pair.loser.place(direction.loserToken, header, query)
				})
				if got.status != direction.status {
					t.Errorf("%s: status %d, want %d — %s was read where %s should have been\nbody: %s",
						direction.name, got.status, direction.status, pair.loser.name, pair.winner.name, got.body)
				}
			}
		})
	}
}

// The rows that rest on plan 6.1's *"a header that is present but yields
// nothing does not stop the search"*, separated from the pairs above because
// one of them is a candidate undeclared difference and the others are not.
//
// # ⚠️ UNVERIFIED, and only for the first row
//
// T8 read the reference's own resolver while writing the reader and found that
// it falls back from Authorization to X-Emby-Authorization only when the first
// field is **absent**, not when it is present and unreadable
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:231-238 @ v10.11.11].
// The probe's measured pairs each carried a readable scheme word, so nothing
// measured contradicts either reading. plan 6.1's generalisation is what ships;
// **one request settles it**, and until it is sent, the first row below is this
// server answering 200 where the reference's source says 401.
//
// The other rows are not in that position. X-Emby-Token and the two query names
// are read from their own fields rather than through the Authorization
// fallback [source: .../AuthorizationContext.cs:98-111 @ v10.11.11], so an
// unreadable Authorization beside any of them authenticates on both servers.
// They are here because they are the rows an `if hasHeader { return }`
// implementation fails, which is the mistake the rule exists to prevent.
func TestAHeaderThatYieldsNothingDoesNotStopTheSearchAtTheWire(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)
	held := logIn(t, server, "mechanisms", restrictedAccount, fixturePassword)

	for _, row := range []struct {
		name       string
		mechanism  tokenMechanism
		provenance string
	}{
		{
			name:      "a readable X-Emby-Authorization",
			mechanism: viaEmbyAuthorization,
			provenance: "⚠️ UNVERIFIED: the reference falls back only when Authorization is absent, " +
				"so its source says 401 here [source: .../AuthorizationContext.cs:231-238 @ v10.11.11]",
		},
		{name: "X-Emby-Token", mechanism: viaEmbyToken, provenance: "read from its own field on both servers"},
		{name: "?ApiKey=", mechanism: viaApiKey, provenance: "read from the query on both servers"},
		{name: "?api_key=", mechanism: viaApiKeyUnderscore, provenance: "read from the query on both servers"},
	} {
		t.Run("Authorization: Bearer beside "+row.name, func(t *testing.T) {
			got := requestWith(t, server, requiredRoute, func(header http.Header, query *string) {
				// Present, and contributing no Token component at all, because
				// its scheme word is neither MediaBrowser nor Emby. It is not a
				// mechanism that disagreed; it is one that was not used.
				header.Set("Authorization", "Bearer "+held.token)
				row.mechanism.place(held.token, header, query)
			})
			if got.status != http.StatusOK {
				t.Errorf("status %d, want %d (%s)\nbody: %s",
					got.status, http.StatusOK, row.provenance, got.body)
			}
		})
	}
}

// 002 spec 3.2's grammar table, sent as a client would send it, over **both**
// field names the reference reads with that grammar.
//
// The two strict rows are asserted as refusals rather than as different values,
// which is the same shape internal/httpapi asserts them in and for the same
// reason: a parser that was kind about whitespace around the `=` or about a
// lowercase component name would let a client be built against Atrium that
// fails against the reference (behaviours 6's non-improvements). At the wire
// that becomes 401 — the token was not read, so the request carries no
// credential at all.
func TestTheClientIdentificationGrammarHoldsOverBothFieldNamesAtTheWire(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)
	held := logIn(t, server, "mechanisms", restrictedAccount, fixturePassword)
	token := held.token

	for _, field := range []string{"Authorization", "X-Emby-Authorization"} {
		t.Run(field, func(t *testing.T) {
			for _, row := range []struct {
				variation string
				value     string
				status    int
			}{
				// The six lenient rows of spec 3.2's table.
				{"components in the order spec 3.2 writes them",
					`MediaBrowser Client="Atrium Conformance", Device="A Seat", DeviceId="grammar", Version="1.0.0", Token="` + token + `"`,
					http.StatusOK},
				{"components in another order",
					`MediaBrowser Token="` + token + `", Version="1.0.0", DeviceId="grammar", Device="A Seat", Client="Atrium Conformance"`,
					http.StatusOK},
				{"values bare rather than quoted",
					`MediaBrowser Client=Atrium Conformance, DeviceId=grammar, Token=` + token,
					http.StatusOK},
				{"no space after a comma",
					`MediaBrowser Client="Atrium Conformance",DeviceId="grammar",Token="` + token + `"`,
					http.StatusOK},
				{"a space before a comma",
					`MediaBrowser Client="Atrium Conformance" , DeviceId="grammar" , Token="` + token + `"`,
					http.StatusOK},
				{"extra spaces after the scheme word",
					`MediaBrowser    Client="Atrium Conformance", Token="` + token + `"`,
					http.StatusOK},
				{"a trailing comma",
					`MediaBrowser Client="Atrium Conformance", Token="` + token + `",`,
					http.StatusOK},
				{"an unknown component, which is ignored",
					`MediaBrowser Client="Atrium Conformance", Sausage="mystery", Token="` + token + `"`,
					http.StatusOK},

				// The scheme word, which decides whether anything is read.
				{"the historical Emby scheme word", `Emby Token="` + token + `"`, http.StatusOK},
				{"the scheme word in another case", `mediabrowser Token="` + token + `"`, http.StatusOK},
				{"a scheme word that is neither, where every component would have parsed",
					`Jellyfin Client="Atrium Conformance", DeviceId="grammar", Token="` + token + `"`,
					http.StatusUnauthorized},
				{"no scheme word at all", `Token="` + token + `"`, http.StatusUnauthorized},

				// The two strict rows.
				{"whitespace around the equals", `MediaBrowser Client="Atrium Conformance", Token = "` + token + `"`,
					http.StatusUnauthorized},
				{"a space before the equals only", `MediaBrowser Client="Atrium Conformance", Token ="` + token + `"`,
					http.StatusUnauthorized},
				{"a lowercase component name", `MediaBrowser Client="Atrium Conformance", token="` + token + `"`,
					http.StatusUnauthorized},
			} {
				t.Run(row.variation, func(t *testing.T) {
					got := server.get(t, requiredRoute, goldenHost, http.Header{field: {row.value}})
					if got.status != row.status {
						t.Errorf("%s: %q\nstatus %d, want %d\nbody: %s",
							field, row.value, got.status, row.status, got.body)
					}
				})
			}
		})
	}
}

// The half of AC-3 this feature does not prove, named rather than skipped.
//
// AC-3's second sentence is about the **image and delivery route classes**,
// where the reference accepts all five mechanisms and requires none — measured,
// and the opposite of what the criterion assumed before it was probed
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-26]
// (behaviours 2.10). Those routes belong to features 006 and 008. What Atrium
// does about them is theirs to decide and theirs to assert, and spec 3.1 says
// so in as many words.
//
// This is an assertion rather than a comment because a comment saying "not
// routed yet" stops being true silently. Principle VI: an endpoint with no
// named consumer in v1 is not routed at all, so both classes answer 404 here —
// and the day either is served, this test fails and whoever served it inherits
// AC-3's second sentence along with the route.
func TestTheImageAndDeliveryClassesOfAcThreeBelongToFeaturesSixAndEight(t *testing.T) {
	t.Parallel()

	server := newInstallation(t)
	held := logIn(t, server, "mechanisms", restrictedAccount, fixturePassword)

	for _, path := range []string{
		// One path of each class, spelled as api-surface-v1.md spells the rows
		// those features own.
		"/Items/" + held.userID + "/Images/Primary",
		"/Audio/" + held.userID + "/universal",
	} {
		t.Run(path, func(t *testing.T) {
			got := server.get(t, path, goldenHost, held.bearing())
			if got.status != http.StatusNotFound {
				t.Errorf("%s answered %d, want %d — if this class is served now, AC-3's second "+
					"sentence is owed an assertion by whichever feature serves it\nbody: %s",
					path, got.status, http.StatusNotFound, got.body)
			}
		})
	}
}
