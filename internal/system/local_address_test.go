package system_test

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/vdatanet/atrium-go/internal/system"
)

// prefix and addr keep the table below readable. A malformed literal in a table
// is a bug in the table, so they panic rather than returning an error nobody
// would check.
func prefix(s string) netip.Prefix { return netip.MustParsePrefix(s) }
func addr(s string) netip.Addr     { return netip.MustParseAddr(s) }

// homeNetwork is the fixture behaviours 2.3 was measured on: an operator's
// server at 192.168.1.39, serving 192.168.1.0/24 over plain HTTP on 8096.
var homeNetwork = system.BoundAddress{
	Address: addr("192.168.1.39"),
	Subnet:  prefix("192.168.1.0/24"),
	Scheme:  "http",
	Port:    8096,
}

// vpnNetwork is the second network of AC-8. behaviours 2.3's requirement is
// that a request arriving over a VPN is answered with the VPN-side address, so
// the two have to be told apart by the requester alone.
var vpnNetwork = system.BoundAddress{
	Address: addr("10.8.0.1"),
	Subnet:  prefix("10.8.0.0/24"),
	Scheme:  "http",
	Port:    8096,
}

// TestLocalAddressChoosesByTierAndByRequester is spec 6's "table-driven test
// over the three tiers with synthesised requester addresses". Every row names
// the criterion or the measured behaviour it is there for; a row with neither
// is a row nobody can argue with.
func TestLocalAddressChoosesByTierAndByRequester(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  system.RequestFacts
		cfg  system.AddressConfig
		want string
	}{
		// ---- Tier 1, plain form (AC-7) ----
		{
			name: "AC-7: a published url is returned verbatim",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "192.168.1.39", Scheme: "http", Port: 8096},
			cfg: system.AddressConfig{
				PublishedURL:   "https://media.example.com/jellyfin",
				BoundAddresses: []system.BoundAddress{homeNetwork},
			},
			want: "https://media.example.com/jellyfin",
		},
		{
			name: "AC-7: one trailing slash is removed",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg:  system.AddressConfig{PublishedURL: "https://media.example.com/"},
			want: "https://media.example.com",
		},
		{
			// The reference spells this Trim('/'), which takes every one of
			// them from both ends
			// [source: Emby.Server.Implementations/ApplicationHost.cs:877 @ v10.11.11].
			// plan 6.6 said "one trailing /", which is narrower than the line
			// it cites; this row is what the difference looks like.
			name: "AC-7: every trailing slash is removed, not one",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg:  system.AddressConfig{PublishedURL: "https://media.example.com///"},
			want: "https://media.example.com",
		},
		{
			name: "AC-7: a published url with no slash to remove is untouched",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg:  system.AddressConfig{PublishedURL: "http://box.lan:8096"},
			want: "http://box.lan:8096",
		},
		{
			// "must not be second-guessed" (spec 3.4). A published URL naming
			// the scheme's default port keeps it, because tier 2's omission is
			// a rule about a URL this server assembles and this one it did not.
			name: "AC-7: a published url keeps a default port it spelled",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg:  system.AddressConfig{PublishedURL: "https://media.example.com:443"},
			want: "https://media.example.com:443",
		},
		{
			// The whole reason tier 1 is first: the published URL is what an
			// operator behind a reverse proxy sets, and derivation from the
			// request would answer the proxy's own idea of the host instead.
			name: "tier 1 wins over tier 2",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "10.0.0.9", Scheme: "http", Port: 8096},
			cfg: system.AddressConfig{
				PublishedURL:      "https://media.example.com",
				DeriveFromRequest: true,
			},
			want: "https://media.example.com",
		},

		// ---- Tier 1, subnet-scoped form (behaviours 2.3, branch 3) ----
		{
			// behaviours 2.3's corrected table: an operator's server carrying
			// 192.168.1.0/24=http://192.168.1.39:7096 answers plain HTTP to a
			// caller inside that subnet, against a request that arrived over
			// TLS. The scoped override replaces the answer; it is not merged
			// with the server's scheme.
			name: "a scoped override answers a caller inside its subnet, scheme and all",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https", Port: 443},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://192.168.1.39:7096"},
				},
				BoundAddresses: []system.BoundAddress{homeNetwork},
			},
			want: "http://192.168.1.39:7096",
		},
		{
			// A function of who is asking: the same server, a caller outside
			// the override's subnet, and tier 3 answers instead.
			name: "a scoped override does not answer a caller outside its subnet",
			req:  system.RequestFacts{RemoteAddress: addr("10.8.0.44"), Host: "media.example.com", Scheme: "https", Port: 443},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://192.168.1.39:7096"},
				},
				BoundAddresses: []system.BoundAddress{homeNetwork, vpnNetwork},
			},
			want: "http://10.8.0.1:8096",
		},
		{
			// OrderByDescending(PrefixLength)
			// [source: src/Jellyfin.Networking/Manager/NetworkManager.cs:1000-1016 @ v10.11.11].
			// The wide entry is written first, so a first-match rule would
			// answer it and this row is what tells the two apart.
			name: "the most specific matching prefix wins over a wider one written first",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.0.0/16"), URL: "http://wide.lan:8096"},
					{Subnet: prefix("192.168.1.0/24"), URL: "http://narrow.lan:8096"},
				},
			},
			want: "http://narrow.lan:8096",
		},
		{
			name: "the most specific matching prefix wins over a wider one written last",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://narrow.lan:8096"},
					{Subnet: prefix("192.168.0.0/16"), URL: "http://wide.lan:8096"},
				},
			},
			want: "http://narrow.lan:8096",
		},
		{
			name: "two prefixes of the same width are answered in declaration order",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://first.lan:8096"},
					{Subnet: prefix("192.168.1.0/24"), URL: "http://second.lan:8096"},
				},
			},
			want: "http://first.lan:8096",
		},
		{
			name: "the plain published url wins over a matching scoped one",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{
				PublishedURL: "https://media.example.com",
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://192.168.1.39:7096"},
				},
			},
			want: "https://media.example.com",
		},
		{
			name: "a scoped override has its trailing slashes removed too",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{
				PublishedURLBySubnet: []system.SubnetURL{
					{Subnet: prefix("192.168.1.0/24"), URL: "http://192.168.1.39:7096/"},
				},
			},
			want: "http://192.168.1.39:7096",
		},

		// ---- Tier 2, derive from the request (AC-13) ----
		{
			name: "AC-13: the request's own host and scheme come back",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https", Port: 8920},
			cfg: system.AddressConfig{
				DeriveFromRequest: true,
				BoundAddresses:    []system.BoundAddress{homeNetwork},
			},
			want: "https://media.example.com:8920",
		},
		{
			name: "AC-13: port 80 is omitted on http",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "http", Port: 80},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "http://media.example.com",
		},
		{
			name: "AC-13: port 443 is omitted on https",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https", Port: 443},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "https://media.example.com",
		},
		{
			// The omission is per scheme, not a list of ports to drop. 443 on
			// http and 80 on https are both written out, and the reference's
			// condition is the pair
			// [source: Emby.Server.Implementations/ApplicationHost.cs:890-896 @ v10.11.11].
			name: "AC-13: port 443 is kept on http, because the default is per scheme",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "http", Port: 443},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "http://media.example.com:443",
		},
		{
			name: "AC-13: port 80 is kept on https, for the same reason",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https", Port: 80},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "https://media.example.com:80",
		},
		{
			// A request that named no port at all. The reference folds this
			// into the same -1 it uses for a default port.
			name: "AC-13: a request that named no port produces none",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https"},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "https://media.example.com",
		},
		{
			// A bare IPv6 literal in an authority is unparseable to a client.
			// A table over IPv4 alone never sees this.
			name: "AC-13: an IPv6 host is bracketed",
			req:  system.RequestFacts{RemoteAddress: addr("fd00::55"), Host: "fd00::1", Scheme: "http", Port: 8096},
			cfg:  system.AddressConfig{DeriveFromRequest: true},
			want: "http://[fd00::1]:8096",
		},
		{
			name: "tier 2 wins over tier 3",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "media.example.com", Scheme: "https", Port: 8920},
			cfg: system.AddressConfig{
				DeriveFromRequest: true,
				BoundAddresses:    []system.BoundAddress{homeNetwork},
			},
			want: "https://media.example.com:8920",
		},

		// ---- Tier 3, the requester's network (AC-8, behaviours 2.3) ----
		{
			name: "AC-8: a requester on the home network is answered with the home address",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "192.168.1.39", Scheme: "http", Port: 8096},
			cfg:  system.AddressConfig{BoundAddresses: []system.BoundAddress{homeNetwork, vpnNetwork}},
			want: "http://192.168.1.39:8096",
		},
		{
			name: "AC-8: a requester over the VPN is answered with the VPN-side address",
			req:  system.RequestFacts{RemoteAddress: addr("10.8.0.44"), Host: "192.168.1.39", Scheme: "http", Port: 8096},
			cfg:  system.AddressConfig{BoundAddresses: []system.BoundAddress{homeNetwork, vpnNetwork}},
			want: "http://10.8.0.1:8096",
		},
		{
			// The match is on the requester, not on the order of the list: the
			// same two networks with the VPN written first still answer the
			// home address to a home requester.
			name: "AC-8: the match is on the requester, not on which network is listed first",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg:  system.AddressConfig{BoundAddresses: []system.BoundAddress{vpnNetwork, homeNetwork}},
			want: "http://192.168.1.39:8096",
		},
		{
			name: "a requester on no bound network falls back to the first bound address",
			req:  system.RequestFacts{RemoteAddress: addr("203.0.113.7")},
			cfg:  system.AddressConfig{BoundAddresses: []system.BoundAddress{homeNetwork, vpnNetwork}},
			want: "http://192.168.1.39:8096",
		},
		{
			name: "tier 3 omits a default port too",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{BoundAddresses: []system.BoundAddress{{
				Address: addr("192.168.1.39"), Subnet: prefix("192.168.1.0/24"), Scheme: "http", Port: 80,
			}}},
			want: "http://192.168.1.39",
		},
		{
			name: "tier 3 brackets an IPv6 bound address",
			req:  system.RequestFacts{RemoteAddress: addr("fd00::55")},
			cfg: system.AddressConfig{BoundAddresses: []system.BoundAddress{{
				Address: addr("fd00::1"), Subnet: prefix("fd00::/64"), Scheme: "http", Port: 8096,
			}}},
			want: "http://[fd00::1]:8096",
		},
		{
			// netip keeps the families apart: an IPv4 requester is in no IPv6
			// prefix `[measurement: net/netip, Go 1.27.0, 2026-09-03]`, so a
			// dual-stack server answers each family its own address.
			name: "an IPv4 requester does not match an IPv6 bound network",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55")},
			cfg: system.AddressConfig{BoundAddresses: []system.BoundAddress{
				{Address: addr("fd00::1"), Subnet: prefix("fd00::/64"), Scheme: "http", Port: 8096},
				{Address: addr("192.168.1.39"), Subnet: prefix("192.168.1.0/24"), Scheme: "http", Port: 8096},
			}},
			want: "http://192.168.1.39:8096",
		},
		{
			// spec 3.4 tier 3 is "the scheme and port the server is actually
			// reachable on". A server fronted by nothing and listening on TLS
			// itself says https, because that is what it is reachable on — the
			// divergence is about ignoring the certificate, not about never
			// saying https.
			name: "tier 3 says https when that is what the address is reachable on",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Scheme: "http"},
			cfg: system.AddressConfig{BoundAddresses: []system.BoundAddress{{
				Address: addr("192.168.1.39"), Subnet: prefix("192.168.1.0/24"), Scheme: "https", Port: 8920,
			}}},
			want: "https://192.168.1.39:8920",
		},

		// ---- Nothing configured at all ----
		{
			// Not a tier of spec 3.4, and it has to answer something: an empty
			// LocalAddress is a field every client reads and none can use.
			name: "with nothing configured the request itself is the last resort",
			req:  system.RequestFacts{RemoteAddress: addr("192.168.1.55"), Host: "box.lan", Scheme: "http", Port: 8096},
			cfg:  system.AddressConfig{},
			want: "http://box.lan:8096",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if got := system.LocalAddress(test.req, test.cfg); got != test.want {
				t.Errorf("LocalAddress() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestTierThreeIgnoresAConfiguredCertificate is the deliberate divergence of
// behaviours 4.2, asserted rather than assumed.
//
// The measurement it is written against: on an instance this project stood up,
// gave a self-signed certificate and restarted, the same route over the same
// plain-HTTP request answered http://<address>:8096 before the certificate and
// https://<address>:8920 after it — the scheme and the port
// [probe: tools/probe_local_address.py, Jellyfin 10.11.11, 2026-09-02].
//
// The certificate is therefore configured here, with the HTTPS port beside it,
// and the answer must be the *before* value. A check that left both fields at
// their zero values would pass whatever LocalAddress did with them, which is
// assuming the divergence rather than asserting it.
func TestTierThreeIgnoresAConfiguredCertificate(t *testing.T) {
	t.Parallel()

	req := system.RequestFacts{
		RemoteAddress: addr("192.168.1.55"),
		Host:          "192.168.1.39",
		Scheme:        "http",
		Port:          8096,
	}
	cfg := system.AddressConfig{
		BoundAddresses:        []system.BoundAddress{homeNetwork},
		CertificateConfigured: true,
		HTTPSPort:             8920,
	}

	const (
		reachable  = "http://192.168.1.39:8096"  // what Atrium answers
		overridden = "https://192.168.1.39:8920" // what the reference answers
	)

	got := system.LocalAddress(req, cfg)
	if got == overridden {
		t.Fatalf("LocalAddress() = %q: the reference's HTTPS override was replicated, and behaviours 4.2 says it is not", got)
	}
	if got != reachable {
		t.Fatalf("LocalAddress() = %q, want %q", got, reachable)
	}
}

// TestACertificateChangesNothingOnAnyTier widens the row above.
//
// The divergence is that the certificate is not an input at all, and one row
// proves only that it is not an input to one branch. Every tier is run twice
// over the same facts, once with a certificate and once without, and the two
// answers have to be the same string.
func TestACertificateChangesNothingOnAnyTier(t *testing.T) {
	t.Parallel()

	req := system.RequestFacts{
		RemoteAddress: addr("192.168.1.55"),
		Host:          "media.example.com",
		Scheme:        "http",
		Port:          8096,
	}

	tiers := []struct {
		name string
		cfg  system.AddressConfig
	}{
		{"tier 1, plain", system.AddressConfig{PublishedURL: "http://published.example.com:8096"}},
		{"tier 1, scoped", system.AddressConfig{PublishedURLBySubnet: []system.SubnetURL{
			{Subnet: prefix("192.168.1.0/24"), URL: "http://scoped.example.com:8096"},
		}}},
		{"tier 2", system.AddressConfig{DeriveFromRequest: true}},
		{"tier 3", system.AddressConfig{BoundAddresses: []system.BoundAddress{homeNetwork}}},
		{"nothing configured", system.AddressConfig{}},
	}

	for _, tier := range tiers {
		t.Run(tier.name, func(t *testing.T) {
			t.Parallel()

			without := system.LocalAddress(req, tier.cfg)

			with := tier.cfg
			with.CertificateConfigured = true
			with.HTTPSPort = 8920

			if got := system.LocalAddress(req, with); got != without {
				t.Errorf("with a certificate LocalAddress() = %q, without one %q: behaviours 4.2 says the certificate is not an input", got, without)
			}
			if strings.HasPrefix(without, "https://") {
				t.Errorf("the fixture answers %q, which is https before any certificate is configured: this row cannot see the override", without)
			}
		})
	}
}

// TestTwoNetworksReceiveTwoDifferentAddresses is AC-8 stated the way the
// criterion states it, rather than as two rows that happen to differ.
//
// "Two requests from two different networks receive two different LocalAddress
// values, each on the requester's network" is an inequality plus a containment,
// and a table of expected strings asserts neither: it would pass unchanged if
// both rows were edited to the same value.
func TestTwoNetworksReceiveTwoDifferentAddresses(t *testing.T) {
	t.Parallel()

	cfg := system.AddressConfig{BoundAddresses: []system.BoundAddress{homeNetwork, vpnNetwork}}

	home := system.LocalAddress(system.RequestFacts{RemoteAddress: addr("192.168.1.55")}, cfg)
	vpn := system.LocalAddress(system.RequestFacts{RemoteAddress: addr("10.8.0.44")}, cfg)

	if home == vpn {
		t.Fatalf("both networks were answered %q; AC-8 asks for two different values", home)
	}
	if !strings.Contains(home, homeNetwork.Address.String()) {
		t.Errorf("the home requester was answered %q, which is not on its own network", home)
	}
	if !strings.Contains(vpn, vpnNetwork.Address.String()) {
		t.Errorf("the VPN requester was answered %q, which is not on its own network", vpn)
	}
}

// TestAnUnknownRequesterSkipsEveryPerCallerBranch covers the zero RemoteAddress.
//
// The edge cannot always name the requester — a unix socket, a proxy that
// rewrote it, a fact the caller declined to fill in — and both per-caller
// branches match on it. Neither may match everything when it is unknown, and
// neither may panic.
func TestAnUnknownRequesterSkipsEveryPerCallerBranch(t *testing.T) {
	t.Parallel()

	req := system.RequestFacts{Host: "box.lan", Scheme: "http", Port: 8096}

	scoped := system.AddressConfig{PublishedURLBySubnet: []system.SubnetURL{
		{Subnet: prefix("0.0.0.0/0"), URL: "http://everything.example.com"},
	}}
	if got := system.LocalAddress(req, scoped); got != "http://box.lan:8096" {
		t.Errorf("an unknown requester matched a scoped override: LocalAddress() = %q", got)
	}

	bound := system.AddressConfig{BoundAddresses: []system.BoundAddress{vpnNetwork, homeNetwork}}
	if got := system.LocalAddress(req, bound); got != "http://10.8.0.1:8096" {
		t.Errorf("an unknown requester did not fall back to the first bound address: LocalAddress() = %q", got)
	}
}

// TestParseSubnetPublishedURL covers the operator spelling of the scoped form.
func TestParseSubnetPublishedURL(t *testing.T) {
	t.Parallel()

	t.Run("the documented form", func(t *testing.T) {
		t.Parallel()
		got, err := system.ParseSubnetPublishedURL("192.168.1.0/24=http://host:port")
		if err != nil {
			t.Fatalf("ParseSubnetPublishedURL() error = %v", err)
		}
		if got.Subnet != prefix("192.168.1.0/24") {
			t.Errorf("subnet = %v, want 192.168.1.0/24", got.Subnet)
		}
		if got.URL != "http://host:port" {
			t.Errorf("url = %q, want %q", got.URL, "http://host:port")
		}
	})

	t.Run("a url containing an equals sign keeps it", func(t *testing.T) {
		t.Parallel()
		got, err := system.ParseSubnetPublishedURL("10.0.0.0/8=http://host:8096/base?a=b")
		if err != nil {
			t.Fatalf("ParseSubnetPublishedURL() error = %v", err)
		}
		if got.URL != "http://host:8096/base?a=b" {
			t.Errorf("url = %q: only the first %q separates the two halves", got.URL, "=")
		}
	})

	t.Run("a host address is masked to the range it matches", func(t *testing.T) {
		t.Parallel()
		got, err := system.ParseSubnetPublishedURL("192.168.1.5/24=http://host:8096")
		if err != nil {
			t.Fatalf("ParseSubnetPublishedURL() error = %v", err)
		}
		if got.Subnet.String() != "192.168.1.0/24" {
			t.Errorf("subnet = %v, want 192.168.1.0/24", got.Subnet)
		}
	})

	t.Run("an IPv6 prefix is accepted", func(t *testing.T) {
		t.Parallel()
		got, err := system.ParseSubnetPublishedURL("fd00::/64=http://[fd00::1]:8096")
		if err != nil {
			t.Fatalf("ParseSubnetPublishedURL() error = %v", err)
		}
		if !got.Subnet.Contains(addr("fd00::55")) {
			t.Errorf("subnet %v does not contain fd00::55", got.Subnet)
		}
	})

	t.Run("surrounding space is not part of either half", func(t *testing.T) {
		t.Parallel()
		got, err := system.ParseSubnetPublishedURL(" 192.168.1.0/24 = http://host:8096 ")
		if err != nil {
			t.Fatalf("ParseSubnetPublishedURL() error = %v", err)
		}
		if got.URL != "http://host:8096" {
			t.Errorf("url = %q, want %q", got.URL, "http://host:8096")
		}
	})

	refusals := []struct {
		name  string
		entry string
	}{
		{"no separator", "192.168.1.0/24"},
		{"no url", "192.168.1.0/24="},
		{"no subnet", "=http://host:8096"},
		{"a bare address rather than a prefix", "192.168.1.0=http://host:8096"},
		{"a prefix length wider than the family", "192.168.1.0/33=http://host:8096"},
		{"nonsense", "nonsense"},
	}
	for _, refusal := range refusals {
		t.Run("refused: "+refusal.name, func(t *testing.T) {
			t.Parallel()
			_, err := system.ParseSubnetPublishedURL(refusal.entry)
			if err == nil {
				t.Fatalf("ParseSubnetPublishedURL(%q) was accepted", refusal.entry)
			}
			// Every refusal is answered by an operator looking at their own
			// configuration line, so every refusal has to quote it.
			if !strings.Contains(err.Error(), refusal.entry) {
				t.Errorf("error %q does not name the entry it refused", err)
			}
		})
	}
}
