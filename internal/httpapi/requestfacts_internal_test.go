package httpapi

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/vdatanet/atrium-go/internal/system"
)

// The seam between an *http.Request and system.RequestFacts is where spec 3.4
// can go wrong without any test of the three tiers noticing: the tiers are a
// pure function over facts, and a fact filled in wrongly is a correct answer to
// the wrong question. Every row here is a shape r.Host or r.RemoteAddr really
// takes.
func TestRequestFactsReadsTheFourFactsOffARequest(t *testing.T) {
	t.Parallel()

	for _, row := range []struct {
		name       string
		host       string
		remoteAddr string
		tls        bool
		want       system.RequestFacts
	}{
		{
			name:       "host and port",
			host:       "192.168.1.20:8096",
			remoteAddr: "192.168.1.44:51000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("192.168.1.44"),
				Host:          "192.168.1.20",
				Scheme:        "http",
				Port:          8096,
			},
		},
		{
			// A request to a server on port 80 carries no port at all, and
			// net.SplitHostPort *fails* on it. Dropping the host on that error
			// would empty the field tier 2 answers with.
			name:       "no port at all",
			host:       "jellyfin.example",
			remoteAddr: "203.0.113.9:44000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("203.0.113.9"),
				Host:          "jellyfin.example",
				Scheme:        "http",
				Port:          0,
			},
		},
		{
			name:       "IPv6 literal with a port loses its brackets",
			host:       "[2001:db8::1]:8096",
			remoteAddr: "[2001:db8::44]:51000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("2001:db8::44"),
				Host:          "2001:db8::1",
				Scheme:        "http",
				Port:          8096,
			},
		},
		{
			// The bracketed form with no port does not split either, and the
			// brackets have to come off here: RequestFacts.Host holds none, and
			// the domain puts them back when it builds a URL authority.
			name:       "IPv6 literal without a port loses its brackets too",
			host:       "[2001:db8::1]",
			remoteAddr: "[2001:db8::44]:51000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("2001:db8::44"),
				Host:          "2001:db8::1",
				Scheme:        "http",
				Port:          0,
			},
		},
		{
			// A dual-stack listener reports an IPv4 client as an
			// IPv4-mapped IPv6 address, and netip treats the mapped form and
			// the plain one as different addresses — so an operator's
			// 192.168.1.0/24 would match nothing without the unmapping.
			name:       "an IPv4-mapped requester is unmapped",
			host:       "192.168.1.20:8096",
			remoteAddr: "[::ffff:192.168.1.44]:51000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("192.168.1.44"),
				Host:          "192.168.1.20",
				Scheme:        "http",
				Port:          8096,
			},
		},
		{
			name:       "TLS is the only thing that makes the scheme https",
			host:       "jellyfin.example:8920",
			remoteAddr: "192.168.1.44:51000",
			tls:        true,
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("192.168.1.44"),
				Host:          "jellyfin.example",
				Scheme:        "https",
				Port:          8920,
			},
		},
		{
			// The zero Addr is the documented "unknown requester": both
			// per-caller branches then match nothing, which is what an
			// unparseable address should cost.
			name:       "an unparseable requester is the zero address",
			host:       "jellyfin.example:8096",
			remoteAddr: "pipe",
			want: system.RequestFacts{
				Host:   "jellyfin.example",
				Scheme: "http",
				Port:   8096,
			},
		},
		{
			name:       "no requester address at all is the zero address",
			host:       "jellyfin.example:8096",
			remoteAddr: "",
			want: system.RequestFacts{
				Host:   "jellyfin.example",
				Scheme: "http",
				Port:   8096,
			},
		},
		{
			// A port that is not a number is a port this server cannot
			// advertise, and the host is still perfectly good.
			name:       "a port that is not a number is no port",
			host:       "jellyfin.example:not-a-port",
			remoteAddr: "192.168.1.44:51000",
			want: system.RequestFacts{
				RemoteAddress: netip.MustParseAddr("192.168.1.44"),
				Host:          "jellyfin.example",
				Scheme:        "http",
				Port:          0,
			},
		},
	} {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()

			r := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
			r.Host = row.host
			r.RemoteAddr = row.remoteAddr
			if row.tls {
				r.TLS = &tls.ConnectionState{}
			}

			got := requestFacts(r)
			if got != row.want {
				t.Errorf("requestFacts(Host=%q, RemoteAddr=%q, tls=%t):\n got %+v\nwant %+v",
					row.host, row.remoteAddr, row.tls, got, row.want)
			}
		})
	}
}

// 001 declares no forwarded-header handling, so a proxy header must not move
// the answer. This is the test that fails the day somebody adds
// X-Forwarded-Proto by reflex rather than by decision — which is a decision
// about which proxies this server believes, and it has not been taken.
func TestRequestFactsIgnoresForwardedHeaders(t *testing.T) {
	t.Parallel()

	r := httptest.NewRequest(http.MethodGet, "/System/Info/Public", nil)
	r.Host = "jellyfin.example:8096"
	r.RemoteAddr = "192.168.1.44:51000"
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "public.example")
	r.Header.Set("X-Forwarded-For", "203.0.113.7")

	got := requestFacts(r)
	want := system.RequestFacts{
		RemoteAddress: netip.MustParseAddr("192.168.1.44"),
		Host:          "jellyfin.example",
		Scheme:        "http",
		Port:          8096,
	}
	if got != want {
		t.Errorf("a forwarded header moved the facts:\n got %+v\nwant %+v", got, want)
	}
}
