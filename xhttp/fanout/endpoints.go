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

// MustNewEndpoints invokes NewEndpoints on the given Endpoints instance, and panics if there's an error.
func MustNewEndpoints(e Endpoints, original *http.Request) []*url.URL {
	endpointURLs, err := e.NewEndpoints(original)
	if err != nil {
		panic(err)
	}

	return endpointURLs
}

// FixedEndpoints represents a set of URLs that act as base URLs for a fanout.
type FixedEndpoints []*url.URL

// NewFixedEndpoints parses each URL to produce a FixedEndpoints.  Each supplied URL should have a scheme
// instead of being abbreviated, e.g. "http://hostname" or "http://hostname:1234" instead of "hostname" or "hostname:1234"
func NewFixedEndpoints(urls ...string) (FixedEndpoints, error) {
	fe := make(FixedEndpoints, 0, len(urls))

	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			return nil, err
		}

		fe = append(fe, parsed)
	}

	return fe, nil
}

// MustNewFixedEndpoints is like NewFixedEndpoints, except that it panics instead of returning an error.
func MustNewFixedEndpoints(urls ...string) FixedEndpoints {
	fe, err := NewFixedEndpoints(urls...)
	if err != nil {
		panic(err)
	}

	return fe
}

func (fe FixedEndpoints) NewEndpoints(original *http.Request) ([]*url.URL, error) {
	endpoints := make([]*url.URL, len(fe))
	for i := 0; i < len(fe); i++ {
		endpoints[i] = new(url.URL)
		*endpoints[i] = *fe[i]

		endpoints[i].Path = original.URL.Path
		endpoints[i].RawPath = original.URL.RawPath
		endpoints[i].RawQuery = original.URL.RawQuery
		endpoints[i].Fragment = original.URL.Fragment
	}

	return endpoints, nil
}
