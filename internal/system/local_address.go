package system

import (
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
)

// RequestFacts is the domain's own view of one request: the four things
// LocalAddress needs and nothing else (plan 5).
//
// It is the seam that keeps architecture 2 true. Taking an *http.Request here
// would put HTTP in the domain and make spec 3.4's three-tier table testable
// only by issuing requests; over synthesised facts the whole table is a table
// test, which is what spec 6 asks for.
//
// The edge fills it in. netip is address arithmetic, not networking — it opens
// no socket and resolves no name — so a domain that holds a netip.Addr has
// still imported nothing that can talk to anything.
type RequestFacts struct {
	// RemoteAddress is the requester's own address, which is what tiers 1's
	// scoped form and 3 match against. The zero Addr means "unknown", and both
	// tiers then match nothing.
	RemoteAddress netip.Addr

	// Host is the host the request named, without a port and without the
	// brackets an IPv6 literal carries in a Host header. It is what tier 2
	// answers with (spec 3.4).
	Host string

	// Scheme is the scheme the request arrived on, "http" or "https".
	Scheme string

	// Port is the port the request named, or 0 when it named none. Tier 2
	// omits it when it is the default for Scheme (AC-13).
	Port int
}

// SubnetURL is one entry of the subnet-scoped published-URL form,
// 192.168.1.0/24=http://host:port.
//
// The reference's published-URL override is per-caller, which makes
// LocalAddress a function of who is asking rather than of the server alone —
// behaviours 2.3 gained that branch on 2026-09-02, on an operator's server that
// answers plain HTTP from a scoped override against a request that arrived over
// TLS. plan 6.6 puts the form here so that the operator spelling has one
// definition rather than one per caller.
type SubnetURL struct {
	// Subnet is the range of requester addresses this entry answers for.
	Subnet netip.Prefix

	// URL is what a matching requester is told, verbatim.
	URL string
}

// BoundAddress is one address the server is reachable at, together with the
// network it serves and — the part that matters for 4.2 — the scheme and port
// it is actually reachable on.
//
// The scheme travels with the address rather than being derived from the
// server's certificate configuration, and that is the deliberate divergence:
// see AddressConfig.CertificateConfigured.
type BoundAddress struct {
	// Address is the address to advertise, as a host.
	Address netip.Addr

	// Subnet is the network this address serves. Tier 3 answers with the first
	// bound address whose subnet contains the requester.
	Subnet netip.Prefix

	// Scheme is what a client reaches this address over, "http" or "https".
	Scheme string

	// Port is the port this address is bound to. It is omitted from the answer
	// when it is the default for Scheme, which is what the reference's URI
	// builder does with one
	// [source: Emby.Server.Implementations/ApplicationHost.cs:941-947 @ v10.11.11].
	Port int
}

// AddressConfig is everything about this installation that spec 3.4's three
// tiers consult. It is configuration, so it is the same for every request; the
// per-request half is RequestFacts.
type AddressConfig struct {
	// PublishedURL is tier 1: what an operator behind a reverse proxy sets, and
	// what must not be second-guessed. Non-empty wins over everything below it.
	PublishedURL string

	// PublishedURLBySubnet is tier 1's scoped form. The most specific matching
	// prefix wins (plan 6.6), which is the reference's own rule
	// [source: src/Jellyfin.Networking/Manager/NetworkManager.cs:1000-1016 @ v10.11.11].
	PublishedURLBySubnet []SubnetURL

	// DeriveFromRequest is tier 2: build the address from the request's own
	// host and scheme.
	DeriveFromRequest bool

	// BoundAddresses is tier 3, in preference order. The first entry whose
	// subnet contains the requester answers; failing that, the first entry.
	//
	// Order is the caller's to establish, and it is where the reference's
	// interface ordering lives — it prefers non-loopback interfaces, then
	// local ones, then interface index
	// [source: src/Jellyfin.Networking/Manager/NetworkManager.cs:870-873 @ v10.11.11].
	// Keeping it here rather than sorting inside is Principle VII: the answer
	// derives from a stable input the caller states, never from map order.
	BoundAddresses []BoundAddress

	// CertificateConfigured records whether this installation holds a TLS
	// certificate, and HTTPSPort the port it would be served on.
	//
	// **Nothing in this package reads either one, and that is the point.**
	// They are the two inputs the reference consults, and behaviours 4.2 is the
	// deliberate divergence of not consulting them: with a certificate loaded
	// the reference rewrites tier 3's answer to the HTTPS scheme and the HTTPS
	// port whatever scheme the request arrived on — measured as
	// http://<address>:8096 before a certificate and https://<address>:8920
	// after it, on the same plain-HTTP request
	// [probe: tools/probe_local_address.py, Jellyfin 10.11.11, 2026-09-02].
	// Atrium answers the scheme and port it is actually reachable on, because
	// the override hands a DLNA renderer an address it has no TLS stack for and
	// that has cost real debugging time (behaviours 2.3, 4.2).
	//
	// A divergence with no input cannot be asserted, only assumed. These fields
	// exist so that T15's check can set them, see the answer not move, and
	// fail the day somebody makes the answer move.
	CertificateConfigured bool
	HTTPSPort             int
}

// LocalAddress is what /System/Info and /System/Info/Public report as
// LocalAddress: one string, chosen per requester, in spec 3.4's three tiers.
//
//  1. A configured published URL, verbatim with its trailing slashes removed —
//     the plain form first, then the most specific subnet-scoped entry whose
//     prefix contains the requester.
//  2. Otherwise, when the installation is configured to derive the address from
//     the request, the request's own host and scheme, with the port omitted
//     when it is that scheme's default (AC-13).
//  3. Otherwise, the bound address on the requester's network, with the scheme
//     and port the server is actually reachable on (behaviours 4.2).
//
// It never returns the empty string. An installation with no published URL, no
// derivation and no bound address has only the request left to answer from, and
// an empty LocalAddress on the wire is worse than a derived one.
//
// The order of the first two tiers is spec 3.4's, and the reference's own source
// puts them the other way round. LocalAddress is served by the HttpRequest
// overload [source: Emby.Server.Implementations/SystemManager.cs:77, 120 @
// v10.11.11], and that overload tests EnablePublishedServerUriByRequest before
// anything else, reaching the published URL only when it is off
// [source: Emby.Server.Implementations/ApplicationHost.cs:885-901 @ v10.11.11].
// The two disagree on exactly one installation: a published URL set *and*
// derivation switched on, where the reference answers the request's host and
// this answers the published URL. The specification is implemented as written
// and deliberately not amended on source evidence alone — AGENTS.md 1.3 makes
// the running server the authority and there is none here. plan 6.6 records what
// discharges it.
func LocalAddress(req RequestFacts, cfg AddressConfig) string {
	if url, ok := publishedURL(req, cfg); ok {
		return url
	}
	if cfg.DeriveFromRequest {
		return fromRequest(req)
	}
	if url, ok := fromBoundAddresses(req, cfg); ok {
		return url
	}
	return fromRequest(req)
}

// publishedURL is tier 1, both forms.
//
// The plain form is checked first because in the reference it short-circuits
// before the networking layer is consulted at all
// [source: Emby.Server.Implementations/ApplicationHost.cs:874-878 @ v10.11.11],
// while the scoped form is one of the branches the bind-address lookup takes
// [source: src/Jellyfin.Networking/Manager/NetworkManager.cs:851-854 @ v10.11.11].
//
// Both are returned verbatim but for their surrounding slashes. The reference
// spells that Trim('/') — leading as well as trailing, and every one of them,
// not one [source: Emby.Server.Implementations/ApplicationHost.cs:877 @
// v10.11.11] — so a URL configured with two trailing slashes comes back with
// none. spec 3.4 says "any trailing / removed"; plan 6.6 wrote "one", which is
// narrower than the line it cites, and the source is what is implemented here.
func publishedURL(req RequestFacts, cfg AddressConfig) (string, bool) {
	if cfg.PublishedURL != "" {
		return trimSlashes(cfg.PublishedURL), true
	}

	// Most specific matching prefix wins; a tie is broken by declaration order,
	// which the stable sort preserves. Both halves are Principle VII: the
	// answer may not depend on which entry the operator happened to write first
	// unless the prefixes are the same width, and then it says so.
	//
	// Containment is the only test. A guard for an unknown requester and a
	// guard for a malformed entry were both written here and both removed as
	// unreachable: Contains reports false for the zero Addr against every
	// prefix and for the zero Prefix against every address
	// `[measurement: net/netip, Go 1.27.0, 2026-09-03]`. A guard no case can
	// reach is a check that has never failed, and the tests that were meant to
	// cover the two are unchanged — they assert the behaviour, which netip
	// guarantees.
	matching := make([]SubnetURL, 0, len(cfg.PublishedURLBySubnet))
	for _, entry := range cfg.PublishedURLBySubnet {
		if entry.Subnet.Contains(req.RemoteAddress) {
			matching = append(matching, entry)
		}
	}
	if len(matching) == 0 {
		return "", false
	}
	slices.SortStableFunc(matching, func(a, b SubnetURL) int {
		return b.Subnet.Bits() - a.Subnet.Bits()
	})
	return trimSlashes(matching[0].URL), true
}

// fromRequest is tier 2: the request's own host and scheme.
//
// The reference passes the request's scheme in explicitly, which is what stops
// the certificate override from firing on this branch — the scheme is defaulted
// only when none was passed
// [source: Emby.Server.Implementations/ApplicationHost.cs:898, 939 @ v10.11.11].
func fromRequest(req RequestFacts) string {
	return formatURL(req.Scheme, req.Host, req.Port)
}

// fromBoundAddresses is tier 3.
//
// First match wins rather than most specific, because BoundAddresses is
// declared in preference order and the reference likewise returns the first
// interface whose subnet contains the caller
// [source: src/Jellyfin.Networking/Manager/NetworkManager.cs:891-905 @ v10.11.11].
// A requester on no bound network still gets an address — the first one — which
// is the same shape as the reference falling back to its preferred interface.
func fromBoundAddresses(req RequestFacts, cfg AddressConfig) (string, bool) {
	if len(cfg.BoundAddresses) == 0 {
		return "", false
	}

	// The same measurement as in publishedURL carries the two guards that are
	// not written here: an unknown requester is in no bound network, so the
	// loop matches nothing and the first address answers.
	chosen := cfg.BoundAddresses[0]
	for _, bound := range cfg.BoundAddresses {
		if bound.Subnet.Contains(req.RemoteAddress) {
			chosen = bound
			break
		}
	}

	// cfg.CertificateConfigured and cfg.HTTPSPort are deliberately not read.
	// behaviours 4.2 — this line is the divergence.
	return formatURL(chosen.Scheme, chosen.Address.String(), chosen.Port), true
}

// formatURL assembles scheme://host[:port], omitting the port when it is the
// default for the scheme or when there is no port to write.
//
// The omission is not a nicety: it is AC-13, and it is what the reference's URI
// builder does with a default port
// [source: Emby.Server.Implementations/ApplicationHost.cs:941-947 @ v10.11.11],
// via the -1 the request branch substitutes for one
// [source: Emby.Server.Implementations/ApplicationHost.cs:890-896 @ v10.11.11].
func formatURL(scheme, host string, port int) string {
	if scheme == "" {
		scheme = "http"
	}
	host = bracketIPv6(host)
	if port <= 0 || port == defaultPort(scheme) {
		return scheme + "://" + host
	}
	return scheme + "://" + host + ":" + strconv.Itoa(port)
}

// defaultPort is the port a scheme does not need to spell: 80 for http, 443 for
// https (spec 3.4 tier 2). A scheme this server does not serve has no default,
// and 0 is a port no URL carries, so its port is always written.
func defaultPort(scheme string) int {
	switch strings.ToLower(scheme) {
	case "http":
		return 80
	case "https":
		return 443
	default:
		return 0
	}
}

// bracketIPv6 wraps a bare IPv6 literal in the brackets a URL authority needs.
//
// A Host header carries them and net.SplitHostPort takes them off, so the host
// reaching RequestFacts has none; a bound address never had any. Writing
// http://::1:8096 would be an address no client can parse, and it is the kind
// of thing a table test over IPv4 alone never sees.
func bracketIPv6(host string) string {
	if host == "" || strings.HasPrefix(host, "[") {
		return host
	}
	addr, err := netip.ParseAddr(host)
	if err != nil || !addr.Is6() || addr.Is4In6() {
		return host
	}
	return "[" + host + "]"
}

// trimSlashes removes every leading and trailing slash, which is what the
// reference does to a published URL
// [source: Emby.Server.Implementations/ApplicationHost.cs:877 @ v10.11.11].
func trimSlashes(url string) string {
	return strings.Trim(url, "/")
}

// ParseSubnetPublishedURL reads one entry of the subnet-scoped form,
// 192.168.1.0/24=http://host:port, as an operator writes it.
//
// The parse lives here rather than at the entry layer because the form is part
// of what tier 1 accepts (plan 6.6), and a second reader somewhere else would
// be a second definition of the same operator spelling.
//
// It is strict. Every refusal names the entry, because every one of them is
// answered by an operator looking at their configuration, and a scoped override
// that silently did not load would show up as an address a renderer cannot
// reach — which is the failure behaviours 2.3 records the override being set to
// fix in the first place.
func ParseSubnetPublishedURL(entry string) (SubnetURL, error) {
	separator := strings.Index(entry, "=")
	if separator < 0 {
		return SubnetURL{}, fmt.Errorf("published url override %q: no %q separating the subnet from the url", entry, "=")
	}

	subnet, err := netip.ParsePrefix(strings.TrimSpace(entry[:separator]))
	if err != nil {
		return SubnetURL{}, fmt.Errorf("published url override %q: %w", entry, err)
	}

	// Masked, so the entry prints as the range it matches. An operator writing
	// 192.168.1.5/24 — the address of a host they were thinking of — means
	// 192.168.1.0/24, and matching is unaffected either way: Contains compares
	// only the leading bits, so an unmasked prefix already matches the same
	// addresses `[measurement: net/netip, Go 1.27.0, 2026-09-03]`. What changes
	// is what an error message and a log line say the override covers.
	subnet = subnet.Masked()

	url := strings.TrimSpace(entry[separator+1:])
	if url == "" {
		return SubnetURL{}, fmt.Errorf("published url override %q: no url after the %q", entry, "=")
	}
	return SubnetURL{Subnet: subnet, URL: url}, nil
}
