package httpapi

import (
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"github.com/vdatanet/atrium-go/internal/system"
)

// requestFacts is the edge's half of plan 5's seam: it turns an *http.Request
// into the four facts system.LocalAddress needs, so that the domain never sees
// a request and spec 3.4's three-tier table stays a table test.
//
// # This is where the seam goes wrong silently, so each half says what it does
//
// Every one of the four fields has a spelling that is *almost* right and fails
// on a shape a test over IPv4 and a port never produces:
//
//   - r.Host is "host:port", and for an IPv6 literal "[::1]:8096". Splitting it
//     is net.SplitHostPort — and that function **returns an error** when there
//     is no port at all, which a request to a server on port 80 really does
//     send. Treating the error as "no host" would drop the value the whole
//     branch answers with, so a Host that does not split is kept whole.
//   - r.RemoteAddr is also "host:port", and the address arithmetic wants the
//     host half. An address that does not parse leaves the zero netip.Addr,
//     which is the documented "unknown requester": both per-caller branches
//     match nothing and the answer falls through, rather than matching some
//     default network.
//   - The scheme is r.TLS != nil, and nothing else. 001 declares no
//     forwarded-header handling, so a deployment behind a proxy that
//     terminates TLS is told "http" — which is honest about what this process
//     served. Reading X-Forwarded-Proto here would trust a header any client
//     can send, on a server that has not decided which proxies it believes;
//     that decision is a feature's, not a reflex.
//
// RequestFacts.Host carries no brackets by its own documentation, so a bare
// bracketed literal loses them here rather than reaching system.formatURL,
// which would otherwise leave them alone and be right by accident.
func requestFacts(r *http.Request) system.RequestFacts {
	host, port := splitHostPort(r.Host)

	facts := system.RequestFacts{
		Host:   host,
		Port:   port,
		Scheme: "http",
	}
	if r.TLS != nil {
		facts.Scheme = "https"
	}

	// r.RemoteAddr is set by net/http and is "host:port" for TCP. It is empty
	// for a request that never came off a connection, which is what an
	// httptest.NewRequest with no RemoteAddr produces, and both forms end at
	// the same zero Addr.
	if remote, _ := splitHostPort(r.RemoteAddr); remote != "" {
		if addr, err := netip.ParseAddr(remote); err == nil {
			// Unmap so that a request arriving on a dual-stack listener as
			// ::ffff:192.168.1.10 is matched against a 192.168.1.0/24 entry
			// the operator wrote in the obvious spelling. netip compares the
			// two as different addresses otherwise.
			facts.RemoteAddress = addr.Unmap()
		}
	}

	return facts
}

// splitHostPort splits an authority into its host and port, and answers the
// whole of it as the host when there is no port to split off.
//
// The zero port means "the request named none", which is what
// system.RequestFacts documents and what makes tier 2 omit it.
func splitHostPort(authority string) (host string, port int) {
	if authority == "" {
		return "", 0
	}

	host, portText, err := net.SplitHostPort(authority)
	if err != nil {
		// No port, or something this is not the place to reject: keep the
		// value. An authority that is a bare IPv6 literal is the one case
		// that still carries brackets, and RequestFacts.Host holds none.
		return strings.TrimSuffix(strings.TrimPrefix(authority, "["), "]"), 0
	}

	// A port that is not a number is a port this server cannot advertise, and
	// 0 is how RequestFacts spells "none named". The host is still good.
	number, err := strconv.Atoi(portText)
	if err != nil || number <= 0 {
		return host, 0
	}
	return host, number
}
