package httpapi

import (
	"net/http"

	"github.com/vdatanet/atrium-go/internal/build"
)

// ServerHeaderName is the field name of the product token every response
// carries.
const ServerHeaderName = "Server"

// serverProduct is the product half of the Server header's product token. The
// version half is the build stamp.
//
// It is deliberately not "Jellyfin Server". behaviours 4.1 makes this the one
// place where Atrium says what it really is, and Principle I is not violated
// by it because the reference does not send this value either: it sends
// `Server: Kestrel`
// [probe: tools/probe_routing.py, Jellyfin 10.11.11, 2026-08-28], which names
// a .NET web server rather than Jellyfin. A client that branched on it would
// already be broken against a reverse proxy.
const serverProduct = "Atrium"

// ServerHeader is the stage that identifies this server on every response
// (behaviours 4.1).
//
// # Why it is a stage and not a constant in a handler
//
// architecture 4 puts `Server: Atrium/<version>` in middleware, "from a
// build-stamped version", and never in "a constant edited by hand". Both
// halves of that row are load-bearing. A handler that set the header would set
// it on the responses a handler produces, which is not the same set as the
// responses this server sends — the refusals of behaviours 1.11 never reach
// one. And a hand-edited constant misidentifies a measured run: architecture 5
// makes a differential run refuse to start unless the two servers differ on
// this header, because ProductName is "Jellyfin Server" on both, so this is
// the one field that tells a report which binary produced it, and "a binary
// that cannot state its version cannot be measured".
type ServerHeader struct {
	value string
}

// NewServerHeader builds the stage from the version this binary was stamped
// with.
//
// Like NewResponseTimeStamp and unlike the three stages that read the route
// table, it cannot fail: build.Version never returns an empty string and never
// returns one carrying a byte that may not appear in an HTTP token, which is
// exactly the guarantee a product version needs. TestTheServerHeaderIsAValidProductToken
// asserts that rather than trusting it.
func NewServerHeader() *ServerHeader {
	return &ServerHeader{value: serverProduct + "/" + build.Version()}
}

// Value is the finished header value: the product token this server sends.
func (s *ServerHeader) Value() string {
	return s.value
}

// Wrap is the middleware.
//
// The header is set on the way in rather than at the moment the response
// starts, which is the difference between this stage and the response-time
// stamp beside it: the value is a constant, so there is nothing to measure and
// no reason to wrap the writer. Setting it early also means every later stage
// sees it in the header map, which is what makes it a deliberate act for one
// of them to remove it — and none does. The empty refusals of behaviours 1.11
// delete Content-Type and WWW-Authenticate by name and nothing else, so a 404,
// a 405 and a 401 all keep it.
func (s *ServerHeader) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set, not Add: two field lines here would be one field value with a
		// comma in it, and a Server header names one product.
		w.Header().Set(ServerHeaderName, s.value)
		next.ServeHTTP(w, r)
	})
}
