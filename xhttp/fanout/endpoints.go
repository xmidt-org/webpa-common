package fanout

import (
	"errors"
	"net/http"
	"net/url"
)

var (
	errNoConfiguredEndpoints = errors.New("No configured endpoints")
)

// Endpoints is a strategy interface for determining the set of HTTP URL endpoints that a fanout
// should use.
type Endpoints interface {
	// FanoutURLs determines the URLs that an original request should be dispatched
	// to as part of a fanout.  Each returned URL will be associated with a single http.Request
	// object and transaction.
	FanoutURLs(*http.Request) ([]*url.URL, error)
}

type EndpointsFunc func(*http.Request) ([]*url.URL, error)

func (ef EndpointsFunc) FanoutURLs(original *http.Request) ([]*url.URL, error) {
	return ef(original)
}

// MustFanoutURLs invokes FanoutURLs on the given Endpoints instance, and panics if there's an error.
func MustFanoutURLs(e Endpoints, original *http.Request) []*url.URL {
	endpointURLs, err := e.FanoutURLs(original)
	if err != nil {
		panic(err)
	}

	return endpointURLs
}

// FixedEndpoints represents a set of URLs that act as base URLs for a fanout.
type FixedEndpoints []*url.URL

// ParseURLs parses each URL to produce a FixedEndpoints.  Each supplied URL should have a scheme
// instead of being abbreviated, e.g. "http://hostname" or "http://hostname:1234" instead of "hostname" or "hostname:1234"
func ParseURLs(urls ...string) (FixedEndpoints, error) {
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

// MustParseURLs is like ParseURLs, except that it panics instead of returning an error.
func MustParseURLs(urls ...string) FixedEndpoints {
	fe, err := ParseURLs(urls...)
	if err != nil {
		panic(err)
	}

	return fe
}

func (fe FixedEndpoints) FanoutURLs(original *http.Request) ([]*url.URL, error) {
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

// NewEndpoints accepts a set of Options, typically injected via configuration, and an alternate function
// that can create an Endpoints.  If Options has a fixed set of endpoints, this function returns a
// FixedEndpoints built from those URLs.  Otherwise, the alternate function is invoked to produce
// and Endpoints instance to return.
//
// This function allows an application-layer Endpoints, returned by alternate, to be used when injected
// endpoints are not present.
func NewEndpoints(o Options, alternate func() (Endpoints, error)) (Endpoints, error) {
	if endpoints := o.endpoints(); len(endpoints) > 0 {
		return ParseURLs(endpoints...)
	}

	if alternate != nil {
		return alternate()
	}

	return nil, errNoConfiguredEndpoints
}

// MustNewEndpoints is like NewEndpoints, save that it panics upon any error.
func MustNewEndpoints(o Options, alternate func() (Endpoints, error)) Endpoints {
	e, err := NewEndpoints(o, alternate)
	if err != nil {
		panic(err)
	}

	return e
}
