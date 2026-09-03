package httpapi

import (
	"fmt"
	"testing"
)

// The grammar of 002 spec 3.2, row by row.
//
// Both cores in this file are pure functions over strings, so the table is a
// table and not a request per row. What a request would add here is a
// transport and a router between the input and the assertion, and neither is
// part of the grammar.

// TestTheGrammarIsLenientInTheSixWaysItIsMeasuredToBe covers every "yes" row
// of 002 spec 3.2's table
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
func TestTheGrammarIsLenientInTheSixWaysItIsMeasuredToBe(t *testing.T) {
	full := ClientIdentification{
		Client:   "Jellyfin Android",
		Device:   "Pixel",
		DeviceID: "abc123",
		Version:  "2.4.1",
		Token:    "tok",
	}

	for _, row := range []struct {
		variation string
		field     string
		want      ClientIdentification
	}{
		{
			variation: "components in the order 002 spec 3.2 writes them",
			field:     `MediaBrowser Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok"`,
			want:      full,
		},
		{
			variation: "components in any other order",
			field:     `MediaBrowser Token="tok", Version="2.4.1", DeviceId="abc123", Device="Pixel", Client="Jellyfin Android"`,
			want:      full,
		},
		{
			variation: "values bare rather than quoted",
			field:     `MediaBrowser Client=Jellyfin Android, Device=Pixel, DeviceId=abc123, Version=2.4.1, Token=tok`,
			want:      full,
		},
		{
			variation: "values quoted and bare in one header",
			field:     `MediaBrowser Client="Jellyfin Android", Device=Pixel, DeviceId="abc123", Version=2.4.1, Token=tok`,
			want:      full,
		},
		{
			variation: "no space after a comma",
			field:     `MediaBrowser Client="Jellyfin Android",Device="Pixel",DeviceId="abc123",Version="2.4.1",Token="tok"`,
			want:      full,
		},
		{
			variation: "a space before a comma",
			field:     `MediaBrowser Client="Jellyfin Android" , Device="Pixel" , DeviceId="abc123" , Version="2.4.1" , Token="tok"`,
			want:      full,
		},
		{
			variation: "extra spaces after the scheme word",
			field:     `MediaBrowser    Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok"`,
			want:      full,
		},
		{
			variation: "a trailing comma",
			field:     `MediaBrowser Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok",`,
			want:      full,
		},
		{
			variation: "a trailing comma and a trailing space",
			field:     `MediaBrowser Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok", `,
			want:      full,
		},
		{
			variation: "an unknown component alongside, which is ignored",
			field:     `MediaBrowser Client="Jellyfin Android", Sausage="mystery", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok"`,
			want:      full,
		},
		{
			variation: "a comma inside a quoted value, which does not end the component",
			field:     `MediaBrowser Client="Jellyfin Android", Device="Living Room, Upstairs", DeviceId="abc123"`,
			want:      ClientIdentification{Client: "Jellyfin Android", Device: "Living Room, Upstairs", DeviceID: "abc123"},
		},
		{
			variation: "the four identification components and no token, which is served everywhere but one route",
			field:     `MediaBrowser Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1"`,
			want:      ClientIdentification{Client: "Jellyfin Android", Device: "Pixel", DeviceID: "abc123", Version: "2.4.1"},
		},
		{
			variation: "a token and nothing else, which is the shape most requests carry",
			field:     `MediaBrowser Token="tok"`,
			want:      ClientIdentification{Token: "tok"},
		},
	} {
		t.Run(row.variation, func(t *testing.T) {
			if got := ParseClientIdentification(row.field); got != row.want {
				t.Errorf("ParseClientIdentification(%q)\n got %+v\nwant %+v", row.field, got, row.want)
			}
		})
	}
}

// TestWhitespaceAroundTheEqualsIsNoComponentAtAll is the first of 002 spec
// 3.2's two strict rows, and it is asserted as an absence.
//
// The point is not that the component's value differs. It is that there is no
// component: a parser that was kind about this form would let somebody build a
// client against Atrium that fails against the reference, which measured 401
// for it (behaviours 2.12, and 6's non-improvements)
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
//
// Every row carries two well-formed components beside the malformed one, so a
// failure says the *component* vanished rather than the header.
func TestWhitespaceAroundTheEqualsIsNoComponentAtAll(t *testing.T) {
	for _, row := range []struct {
		variation string
		field     string
	}{
		{"a space on both sides", `MediaBrowser Client="Jellyfin Android", Token = "tok", Device="Pixel"`},
		{"a space before the equals", `MediaBrowser Client="Jellyfin Android", Token ="tok", Device="Pixel"`},
		{"a space after the equals", `MediaBrowser Client="Jellyfin Android", Token= "tok", Device="Pixel"`},
		{"a tab after the equals", "MediaBrowser Client=\"Jellyfin Android\", Token=\t\"tok\", Device=\"Pixel\""},
		{"a space on both sides of a bare value", `MediaBrowser Client="Jellyfin Android", Token = tok, Device="Pixel"`},
	} {
		t.Run(row.variation, func(t *testing.T) {
			got := ParseClientIdentification(row.field)
			if got.Token != "" {
				t.Errorf("ParseClientIdentification(%q).Token = %q, want no component at all", row.field, got.Token)
			}
			if got.Client != "Jellyfin Android" || got.Device != "Pixel" {
				t.Errorf("ParseClientIdentification(%q) lost a well-formed component too: %+v", row.field, got)
			}
		})
	}
}

// TestALowercaseComponentNameIsNoComponentAtAll is 002 spec 3.2's second
// strict row, asserted the same way and for the same reason. The reference
// keys its parts ordinally and asks for five exact spellings, so `token=` is a
// key nothing ever reads
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:86-90 @ v10.11.11]
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28].
func TestALowercaseComponentNameIsNoComponentAtAll(t *testing.T) {
	for _, row := range []struct {
		variation string
		field     string
		absent    func(ClientIdentification) string
		name      string
	}{
		{"token=", `MediaBrowser Client="Jellyfin Android", token="tok", Device="Pixel"`, func(c ClientIdentification) string { return c.Token }, "Token"},
		{"deviceid=", `MediaBrowser Client="Jellyfin Android", deviceid="abc", Device="Pixel"`, func(c ClientIdentification) string { return c.DeviceID }, "DeviceID"},
		{"deviceId= (only the first letter wrong)", `MediaBrowser Client="Jellyfin Android", deviceId="abc", Device="Pixel"`, func(c ClientIdentification) string { return c.DeviceID }, "DeviceID"},
		{"DEVICEID=", `MediaBrowser Client="Jellyfin Android", DEVICEID="abc", Device="Pixel"`, func(c ClientIdentification) string { return c.DeviceID }, "DeviceID"},
		{"version=", `MediaBrowser Client="Jellyfin Android", version="2.4.1", Device="Pixel"`, func(c ClientIdentification) string { return c.Version }, "Version"},
	} {
		t.Run(row.variation, func(t *testing.T) {
			got := ParseClientIdentification(row.field)
			if value := row.absent(got); value != "" {
				t.Errorf("ParseClientIdentification(%q).%s = %q, want no component at all", row.field, row.name, value)
			}
			if got.Client != "Jellyfin Android" || got.Device != "Pixel" {
				t.Errorf("ParseClientIdentification(%q) lost a well-formed component too: %+v", row.field, got)
			}
		})
	}
}

// TestTheSchemeWordDecidesWhetherAnythingIsReadAtAll is the case a lenient
// parser passes and the reference refuses: every component below would have
// parsed, and none of them is read, because the word in front is not one of
// the two
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:246-268 @ v10.11.11].
func TestTheSchemeWordDecidesWhetherAnythingIsReadAtAll(t *testing.T) {
	components := `Client="Jellyfin Android", Device="Pixel", DeviceId="abc123", Version="2.4.1", Token="tok"`
	full := ClientIdentification{
		Client:   "Jellyfin Android",
		Device:   "Pixel",
		DeviceID: "abc123",
		Version:  "2.4.1",
		Token:    "tok",
	}

	for _, row := range []struct {
		word string
		want ClientIdentification
	}{
		{"MediaBrowser", full},
		{"mediabrowser", full},
		{"MEDIABROWSER", full},
		{"MediaBrowseR", full},
		{"Emby", full},
		{"emby", full},
		{"EMBY", full},
		{"Bearer", ClientIdentification{}},
		{"Basic", ClientIdentification{}},
		{"MediaBrowse", ClientIdentification{}},
		{"MediaBrowserr", ClientIdentification{}},
		{"Jellyfin", ClientIdentification{}},
		{"", ClientIdentification{}},
	} {
		t.Run(fmt.Sprintf("scheme word %q", row.word), func(t *testing.T) {
			field := row.word + " " + components
			if got := ParseClientIdentification(field); got != row.want {
				t.Errorf("ParseClientIdentification(%q)\n got %+v\nwant %+v", field, got, row.want)
			}
		})
	}
}

// TestAFieldValueWithNoSchemeWordYieldsNothing covers the shapes that have no
// scheme word to compare at all. The reference splits on the first space and
// gives up when there is none
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:248-254 @ v10.11.11],
// so a tab is not a separator and a bare token is not a header.
func TestAFieldValueWithNoSchemeWordYieldsNothing(t *testing.T) {
	for _, field := range []string{
		"",
		"MediaBrowser",
		`MediaBrowser,Token="tok"`,
		"MediaBrowser\tToken=\"tok\"",
		"tok",
		`Token="tok"`,
	} {
		if got := (ParseClientIdentification(field)); got != (ClientIdentification{}) {
			t.Errorf("ParseClientIdentification(%q) = %+v, want nothing at all", field, got)
		}
	}
}

// mechanism is one of 002 spec 3.1's five, expressed as where it puts a token
// among the four strings the reader reads.
type mechanism struct {
	name  string
	place func(token string, in *inputs)
}

// inputs are the four strings presentedToken reads: three header field values
// and a raw query string.
type inputs struct {
	authorization     string
	embyAuthorization string
	embyToken         string
	rawQuery          string
}

var (
	viaAuthorization = mechanism{"Authorization", func(token string, in *inputs) {
		in.authorization = `MediaBrowser Client="c", DeviceId="d", Token="` + token + `"`
	}}
	viaEmbyAuthorization = mechanism{"X-Emby-Authorization", func(token string, in *inputs) {
		in.embyAuthorization = `MediaBrowser Client="c", DeviceId="d", Token="` + token + `"`
	}}
	viaEmbyToken = mechanism{"X-Emby-Token", func(token string, in *inputs) {
		in.embyToken = token
	}}
	viaApiKey = mechanism{"?ApiKey=", func(token string, in *inputs) {
		in.rawQuery = appendQuery(in.rawQuery, "ApiKey="+token)
	}}
	viaApiKeyUnderscore = mechanism{"?api_key=", func(token string, in *inputs) {
		in.rawQuery = appendQuery(in.rawQuery, "api_key="+token)
	}}
)

func appendQuery(raw, fragment string) string {
	if raw == "" {
		return fragment
	}
	return raw + "&" + fragment
}

func read(in inputs) string {
	return presentedToken(in.authorization, in.embyAuthorization, in.embyToken, in.rawQuery)
}

// TestEachMechanismOnItsOwnYieldsItsToken is the floor the precedence rows
// stand on: a pair proves nothing about an order if one of its two sides never
// worked alone.
func TestEachMechanismOnItsOwnYieldsItsToken(t *testing.T) {
	for _, m := range []mechanism{viaAuthorization, viaEmbyAuthorization, viaEmbyToken, viaApiKey, viaApiKeyUnderscore} {
		t.Run(m.name, func(t *testing.T) {
			var in inputs
			m.place("alpha", &in)
			if got := read(in); got != "alpha" {
				t.Errorf("%s alone yielded %q, want %q", m.name, got, "alpha")
			}
		})
	}
}

// TestThePrecedenceHoldsInBothDirections covers every pair of 002 spec 3.1's
// chain:
//
//	Authorization > X-Emby-Authorization > X-Emby-Token > ?ApiKey= / ?api_key=
//
// Each pair runs twice, with the two tokens swapped between the mechanisms, so
// a row cannot pass because one token happened to be the answer. The three
// adjacent pairs are measured pair by pair and in both directions
// [probe: tools/probe_auth_mechanisms.py, Jellyfin 10.11.11, 2026-08-28]; the
// three non-adjacent ones are inferred from the chain (behaviours 2.4) and are
// labelled so nobody reads them as measurements.
func TestThePrecedenceHoldsInBothDirections(t *testing.T) {
	for _, pair := range []struct {
		winner, loser mechanism
		provenance    string
	}{
		{viaAuthorization, viaEmbyAuthorization, "measured"},
		{viaEmbyAuthorization, viaEmbyToken, "measured"},
		{viaEmbyToken, viaApiKey, "measured"},
		{viaAuthorization, viaEmbyToken, "inferred from the chain"},
		{viaAuthorization, viaApiKey, "inferred from the chain"},
		{viaEmbyAuthorization, viaApiKey, "inferred from the chain"},
	} {
		name := fmt.Sprintf("%s beats %s (%s)", pair.winner.name, pair.loser.name, pair.provenance)
		t.Run(name, func(t *testing.T) {
			for _, direction := range []struct {
				winnerToken, loserToken string
			}{
				{"alpha", "beta"},
				{"beta", "alpha"},
			} {
				var in inputs
				pair.winner.place(direction.winnerToken, &in)
				pair.loser.place(direction.loserToken, &in)

				if got := read(in); got != direction.winnerToken {
					t.Errorf("%s carried %q and %s carried %q: read %q, want %q",
						pair.winner.name, direction.winnerToken, pair.loser.name, direction.loserToken, got, direction.winnerToken)
				}
			}
		})
	}
}

// TestAHeaderThatYieldsNothingDoesNotStopTheSearch is plan 6.1's rule and the
// row an `if hasHeader { return }` implementation fails.
//
// Authorization: Bearer x is a header that is *present*. It contributes no
// Token component, because its scheme word is neither MediaBrowser nor Emby,
// so it is not a mechanism that disagreed — it is a mechanism that was not
// used, and the search goes on to the query.
func TestAHeaderThatYieldsNothingDoesNotStopTheSearch(t *testing.T) {
	for _, row := range []struct {
		name string
		in   inputs
	}{
		{
			name: "Bearer beside ?api_key=",
			in:   inputs{authorization: "Bearer x", rawQuery: "api_key=good"},
		},
		{
			name: "Bearer beside ?ApiKey=",
			in:   inputs{authorization: "Bearer x", rawQuery: "ApiKey=good"},
		},
		{
			name: "Bearer beside X-Emby-Token",
			in:   inputs{authorization: "Bearer x", embyToken: "good"},
		},
		{
			name: "Bearer beside a readable X-Emby-Authorization",
			in:   inputs{authorization: "Bearer x", embyAuthorization: `MediaBrowser Token="good"`},
		},
		{
			name: "a header carrying only identification, beside ?api_key=",
			in:   inputs{authorization: `MediaBrowser Client="c", DeviceId="d"`, rawQuery: "api_key=good"},
		},
		{
			name: "a header whose Token component is malformed, beside ?api_key=",
			in:   inputs{authorization: `MediaBrowser Client="c", Token = "stale"`, rawQuery: "api_key=good"},
		},
		{
			name: "an empty X-Emby-Token beside ?api_key=",
			in:   inputs{embyToken: "   ", rawQuery: "api_key=good"},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			if got := read(row.in); got != "good" {
				t.Errorf("read(%+v) = %q, want %q", row.in, got, "good")
			}
		})
	}
}

// TestTheQueryNamesAreReadInEverySpelling covers behaviours 1.15's
// case-insensitivity applied to 002 spec 3.1's two query names.
func TestTheQueryNamesAreReadInEverySpelling(t *testing.T) {
	for _, name := range []string{"ApiKey", "apikey", "APIKEY", "apiKey", "ApIkEy", "api_key", "API_KEY", "Api_Key"} {
		t.Run(name, func(t *testing.T) {
			if got := queryToken(name + "=tok"); got != "tok" {
				t.Errorf("queryToken(%q) = %q, want %q", name+"=tok", got, "tok")
			}
		})
	}
}

// TestTheFirstOccurrenceInTheRawQueryWins is plan 6.1's own decision rather
// than a measurement: ApiKey and api_key are two names that do not fold
// together, and behaviours 2.4 records that the two spellings were never set
// against each other. The rule is stated so that it is a decision rather than
// a property of a map.
//
// ⚠️ The reference's source resolves it by name rather than by position — it
// reads ApiKey and consults api_key only when that is empty
// [source: Jellyfin.Server.Implementations/Security/AuthorizationContext.cs:103-111 @ v10.11.11] —
// so the second row below is the one a probe would move.
func TestTheFirstOccurrenceInTheRawQueryWins(t *testing.T) {
	for _, row := range []struct {
		rawQuery string
		want     string
	}{
		{"ApiKey=first&api_key=second", "first"},
		{"api_key=first&ApiKey=second", "first"},
		{"APIKEY=first&ApiKey=second", "first"},
		{"ApiKey=first&ApiKey=second", "first"},
		{"unrelated=1&api_key=first&ApiKey=second", "first"},
	} {
		t.Run(row.rawQuery, func(t *testing.T) {
			if got := queryToken(row.rawQuery); got != row.want {
				t.Errorf("queryToken(%q) = %q, want %q", row.rawQuery, got, row.want)
			}
		})
	}
}

// TestAQueryFragmentThatNamesNoCredentialIsSkipped keeps the reader from
// answering with somebody else's parameter, and keeps an empty credential from
// ending the search — an empty value is no token, exactly as an unreadable
// header is no token.
func TestAQueryFragmentThatNamesNoCredentialIsSkipped(t *testing.T) {
	for _, row := range []struct {
		rawQuery string
		want     string
	}{
		{"", ""},
		{"limit=10", ""},
		{"ApiKeys=tok", ""},
		{"XApiKey=tok", ""},
		{"apikey", ""},
		{"ApiKey=", ""},
		{"ApiKey=&api_key=good", "good"},
		{"&&ApiKey=good", "good"},
		{"=good", ""},
		{"ApiKey=Ab%20Cd", "Ab Cd"},
		{"ApiKey=%zz", "%zz"},
	} {
		t.Run(fmt.Sprintf("%q", row.rawQuery), func(t *testing.T) {
			if got := queryToken(row.rawQuery); got != row.want {
				t.Errorf("queryToken(%q) = %q, want %q", row.rawQuery, got, row.want)
			}
		})
	}
}

// TestXEmbyTokenIsTrimmedAndNothingElse is plan 6.1's second line. The whole
// field value is the token; only surrounding whitespace comes off.
func TestXEmbyTokenIsTrimmedAndNothingElse(t *testing.T) {
	for _, row := range []struct {
		field string
		want  string
	}{
		{"tok", "tok"},
		{"  tok  ", "tok"},
		{"\ttok\t", "tok"},
		{`"tok"`, `"tok"`},
		{"MediaBrowser Token=tok", "MediaBrowser Token=tok"},
		{"   ", ""},
		{"", ""},
	} {
		t.Run(fmt.Sprintf("%q", row.field), func(t *testing.T) {
			if got := presentedToken("", "", row.field, ""); got != row.want {
				t.Errorf("presentedToken with X-Emby-Token %q = %q, want %q", row.field, got, row.want)
			}
		})
	}
}

// TestARequestWithNoCredentialYieldsNoToken is the answer that decides
// admission, so it is asserted rather than assumed: nothing in this reader can
// fail, and an empty string is how it says "no credential".
func TestARequestWithNoCredentialYieldsNoToken(t *testing.T) {
	for _, in := range []inputs{
		{},
		{authorization: "Bearer x"},
		{authorization: `MediaBrowser Client="c", Device="d", DeviceId="e", Version="1"`},
		{embyAuthorization: `Bearer Token="tok"`},
		{rawQuery: "limit=10&sortBy=SortName"},
		{authorization: "Bearer x", embyAuthorization: "Basic y", embyToken: " ", rawQuery: "nothing=here"},
	} {
		if got := read(in); got != "" {
			t.Errorf("read(%+v) = %q, want no token", in, got)
		}
	}
}
