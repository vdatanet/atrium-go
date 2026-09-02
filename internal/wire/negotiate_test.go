package wire_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vdatanet/atrium-go/internal/wire"
)

// TestNegotiateFollowsTheFourRules is spec 3.0.2's negotiation, one group of
// rows per rule so that a failure names the rule rather than a header.
//
// The rules are behaviours 1.13's, measured
// `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`:
// the profile is compared case-insensitively and unquoted; a charset beside it
// stops the match; an unknown profile falls back; and ranking is by q with the
// client's order kept on a tie.
func TestNegotiateFollowsTheFourRules(t *testing.T) {
	cases := []struct {
		name   string
		accept string
		want   wire.Profile
	}{
		// Rule 1 — the profile is compared case-insensitively and unquoted.
		{
			name:   "the canonical spelling",
			accept: `application/json; profile="CamelCase"`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "unquoted",
			accept: `application/json; profile=CamelCase`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "lower case",
			accept: `application/json; profile="camelcase"`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "upper case and unquoted",
			accept: `application/json; profile=CAMELCASE`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "the parameter's own name in another case",
			accept: `application/json; PROFILE="CamelCase"`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "the media type in another case",
			accept: `APPLICATION/JSON; profile="PascalCase"`,
			want:   wire.ProfilePascal,
		},
		{
			name:   "the other profile",
			accept: `application/json; profile="pascalcase"`,
			want:   wire.ProfilePascal,
		},

		// Rule 2 — a charset beside the profile stops the match. This is the
		// rule least likely to be guessed: the request still names a profile,
		// and it still gets the plain type.
		{
			name:   "a charset after the profile",
			accept: `application/json; profile="CamelCase"; charset=utf-8`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "a charset before the profile",
			accept: `application/json; charset=utf-8; profile="CamelCase"`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "a charset beside the profile that is already PascalCase",
			accept: `application/json; profile="PascalCase"; charset=utf-8`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "a charset on its own is not a profile to stop",
			accept: `application/json; charset=utf-8`,
			want:   wire.ProfilePlain,
		},

		// Rule 3 — an unknown profile falls back to plain.
		{
			name:   "a profile this server has never heard of",
			accept: `application/json; profile="Klingon"`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "an empty profile",
			accept: `application/json; profile=""`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "a profile that is nearly one of the two",
			accept: `application/json; profile="CamelCase2"`,
			want:   wire.ProfilePlain,
		},
		{
			// The quoted value is one parameter, comma and all, and it is not
			// a profile. The row exercises the quoted-string split; it cannot
			// distinguish it, because every fragment a naive split produces is
			// a media type this server has no representation for and is
			// dropped for that reason instead.
			name:   "a profile with a comma in it",
			accept: `application/json; profile="Camel,Case"`,
			want:   wire.ProfilePlain,
		},

		// Rule 4 — rank by q descending, and on a tie the client's order wins.
		// The first two rows are the pair behaviours 1.13 states outright.
		{
			name:   "a tie, plain written first",
			accept: `application/json, application/json; profile="CamelCase"`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "the same tie, camelCase written first",
			accept: `application/json; profile="CamelCase", application/json`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "a higher q later in the header",
			accept: `application/json;q=0.5, application/json;profile="CamelCase";q=0.9`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "a higher q earlier in the header",
			accept: `application/json;profile="CamelCase";q=0.2, application/json;q=0.8`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "q=0 is not acceptable, not merely last",
			accept: `application/json;profile="CamelCase";q=0, application/json;profile="PascalCase";q=0.1`,
			want:   wire.ProfilePascal,
		},
		{
			// The row above passes whether `q=0` means "refused" or merely
			// "ranked last", because something outranks it either way. This
			// one is the difference: the only range there is says camelCase is
			// **not** acceptable, so answering it in camelCase would be
			// answering with the one thing the client ruled out.
			name:   "the only range refuses what it names",
			accept: `application/json;profile="CamelCase";q=0`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "three ranges, the middle one wins",
			accept: `application/json;q=0.3, application/json;profile="PascalCase";q=0.7, application/json;profile="CamelCase";q=0.5`,
			want:   wire.ProfilePascal,
		},
		{
			name:   "a media type this server cannot produce is no obstacle",
			accept: `text/plain, application/json; profile="CamelCase"`,
			want:   wire.ProfileCamel,
		},
		{
			// A wildcard ranks as an ordinary range that names no profile, so
			// it wins the tie it is written first in. The reference does not
			// discard it: RespectBrowserAcceptHeader is set
			// `[source: Jellyfin.Server/Extensions/ApiServiceCollectionExtensions.cs:125-126 @ v10.11.11]`.
			name:   "a wildcard first",
			accept: `*/*, application/json; profile="CamelCase"`,
			want:   wire.ProfilePlain,
		},
		{
			name:   "a wildcard behind a profile",
			accept: `application/json; profile="CamelCase", */*`,
			want:   wire.ProfileCamel,
		},
		{
			name:   "a wildcard subtype names no profile of its own",
			accept: `application/*; profile="CamelCase"`,
			want:   wire.ProfilePlain,
		},
		{
			// RFC 9110 §12.5.1: parameters after q belong to the negotiation,
			// not to the media type, so this asks for plain JSON at q=0.9 and
			// not for camelCase. ⚠️ UNVERIFIED against the reference.
			name:   "a profile written after the q",
			accept: `application/json;q=0.9;profile="CamelCase"`,
			want:   wire.ProfilePlain,
		},

		// The fallbacks. None of these is a 406: behaviours 1.11 measured four
		// refusal shapes and none of them is one.
		{name: "no Accept header at all", accept: "", want: wire.ProfilePlain},
		{name: "only a media type this server has not got", accept: "text/plain", want: wire.ProfilePlain},
		{name: "everything refused", accept: "application/json;q=0", want: wire.ProfilePlain},
		{name: "a trailing comma", accept: `application/json; profile="CamelCase",`, want: wire.ProfileCamel},
		{name: "whitespace around everything", accept: "  application/json ;  profile = \"CamelCase\"  ", want: wire.ProfileCamel},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := wire.Negotiate(c.accept); got != c.want {
				t.Errorf("Negotiate(%q) = %s, want %s", c.accept, got, c.want)
			}
		})
	}
}

// TestNegotiateAnswersEveryProfileWithItsOwnContentType is the second half of
// the winner's job: it decides the naming policy **and** the content type, and
// the content type puts the profile before the charset (plan 6.3).
func TestNegotiateAnswersEveryProfileWithItsOwnContentType(t *testing.T) {
	cases := []struct {
		accept string
		want   string
	}{
		{accept: "application/json", want: "application/json; charset=utf-8"},
		{
			accept: `application/json; profile="PascalCase"`,
			want:   `application/json; profile="PascalCase"; charset=utf-8`,
		},
		{
			accept: `application/json; profile="CamelCase"`,
			want:   `application/json; profile="CamelCase"; charset=utf-8`,
		},
		{
			// The echo carries the canonical spelling, not the request's.
			accept: `application/json; profile=camelcase`,
			want:   `application/json; profile="CamelCase"; charset=utf-8`,
		},
	}

	for _, c := range cases {
		t.Run(c.accept, func(t *testing.T) {
			if got := answer(t, c.accept).Header().Get("Content-Type"); got != c.want {
				t.Errorf("Content-Type = %q, want %q", got, c.want)
			}
		})
	}
}

// profiled is a body with everything the three requests have to be compared
// over: a name with a leading run of capitals, a nested object so that depth is
// covered, and a dictionary whose keys must survive untouched.
type profiled struct {
	Id          string
	UICulture   string
	Nested      nested
	ProviderIds map[string]string
}

type nested struct {
	ServerName string
}

// sample is the one value all three requests are answered with, so that "the
// same values" is a property of the bytes rather than of two constructions.
var sample = profiled{
	Id:          "3f9cb0f2a1d4e6b8c0a2e4f6b8d0a2c4",
	UICulture:   "en-US",
	Nested:      nested{ServerName: "atrium"},
	ProviderIds: map[string]string{"Imdb": "tt0000001"},
}

// answer issues one request through the negotiation and the writer, and hands
// back what reached the client.
//
// It goes through an http.Handler rather than calling Write with a profile
// chosen in the test, because AC-9 is about what three *requests* receive. The
// negotiation reading the header it was sent is half of what is being checked.
func answer(t *testing.T, accept string) *httptest.ResponseRecorder {
	t.Helper()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := wire.Write(w, http.StatusOK, sample, wire.Negotiate(r.Header.Get("Accept"))); err != nil {
			t.Errorf("Write(...) = %v, want no error", err)
		}
	})

	request := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
	if accept != "" {
		request.Header.Set("Accept", accept)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

// TestTheThreeDeclaredContentTypesAnswerAsTheReferenceDoes is AC-9, whole.
//
// Three requests, three content types, two bodies: the plain type and the
// `PascalCase` profile must be **byte-identical**, `CamelCase` must carry the
// same values under converted names, and each response must say in its own
// content type which of the two it used
// `[probe: tools/probe_content_type_profiles.py, Jellyfin 10.11.11, 2026-08-26]`.
//
// Every assertion is on bytes (Principle VIII). A decoder turns `Id` and `id`
// into two keys of a map and `é` and `é` into one string, so a test that
// unmarshalled could report agreement between a server that converted the names
// and one that did not, depending on which side it looked at.
func TestTheThreeDeclaredContentTypesAnswerAsTheReferenceDoes(t *testing.T) {
	const (
		pascalBody = `{"Id":"3f9cb0f2a1d4e6b8c0a2e4f6b8d0a2c4","UICulture":"en-US",` +
			`"Nested":{"ServerName":"atrium"},"ProviderIds":{"Imdb":"tt0000001"}}`
		camelBody = `{"id":"3f9cb0f2a1d4e6b8c0a2e4f6b8d0a2c4","uiCulture":"en-US",` +
			`"nested":{"serverName":"atrium"},"providerIds":{"Imdb":"tt0000001"}}`
	)

	plain := answer(t, "application/json")
	pascal := answer(t, `application/json; profile="PascalCase"`)
	camel := answer(t, `application/json; profile="CamelCase"`)

	// "byte-identical" is the criterion's own word, so the two are compared
	// with each other as well as with the expectation. Comparing only against
	// the constant would let both drift to the same wrong answer and still
	// pass the clause that matters.
	if plain.Body.String() != pascal.Body.String() {
		t.Errorf("the plain type and the PascalCase profile differ:\n\t%s\n\t%s",
			plain.Body.String(), pascal.Body.String())
	}
	if got := plain.Body.String(); got != pascalBody {
		t.Errorf("plain body =\n\t%s\nwant\n\t%s", got, pascalBody)
	}

	// The same values, under converted names, at every depth — and the
	// dictionary's key `Imdb` untouched, which is the half of spec 3.0.2 a
	// conversion applied to the finished bytes would get wrong.
	if got := camel.Body.String(); got != camelBody {
		t.Errorf("camelCase body =\n\t%s\nwant\n\t%s", got, camelBody)
	}

	// Each response names the profile it used.
	contentTypes := []struct {
		name     string
		recorder *httptest.ResponseRecorder
		want     string
	}{
		{"plain", plain, "application/json; charset=utf-8"},
		{"PascalCase", pascal, `application/json; profile="PascalCase"; charset=utf-8`},
		{"CamelCase", camel, `application/json; profile="CamelCase"; charset=utf-8`},
	}
	for _, c := range contentTypes {
		if got := c.recorder.Header().Get("Content-Type"); got != c.want {
			t.Errorf("%s Content-Type = %q, want %q", c.name, got, c.want)
		}
	}

	// The two PascalCase answers are the same bytes and different headers,
	// which is the whole of "three names for two behaviours". If this ever
	// stops holding, one of the two clauses above is wrong rather than both.
	if plain.Header().Get("Content-Type") == pascal.Header().Get("Content-Type") {
		t.Errorf("the plain type and the PascalCase profile echoed the same content type %q",
			plain.Header().Get("Content-Type"))
	}
}
