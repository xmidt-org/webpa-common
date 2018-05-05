package fanout

import (
	"net/http"
	"net/url"
)

// Endpoints is a strategy interface for determining the set of HTTP URL endpoints that a fanout
// should use.  Each returned endpoint will be associated with a single http.Request object and transaction.
type Endpoints interface {
	NewEndpoints(*http.Request) ([]*url.URL, error)
}

type EndpointsFunc func(*http.Request) ([]*url.URL, error)

func (ef EndpointsFunc) NewEndpoints(original *http.Request) ([]*url.URL, error) {
	return ef(original)
}
