package fanout

import (
	"context"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/xcontext"
	"github.com/justinas/alice"
)

const (
	DefaultFanoutTimeout time.Duration = 45 * time.Second
	DefaultClientTimeout time.Duration = 30 * time.Second
	DefaultConcurrency                 = 1000
)

// Options defines the configuration structure for externally configuring a fanout.
type Options struct {
	// Endpoints are the URLs for each endpoint to fan out to.  If unset, the default is supplied
	// by application code, which is normally a set of endpoints driven by service discovery.
	Endpoints []string `json:"endpoints,omitempty"`

	// Authorization is the Basic Auth token.  There is no default for this field.
	Authorization string `json:"authorization"`

	// Transport is the http.Client transport
	Transport http.Transport `json:"transport"`

	// FanoutTimeout is the timeout for the entire fanout operation.  If not supplied, DefaultFanoutTimeout is used.
	FanoutTimeout time.Duration `json:"timeout"`

	// ClientTimeout is the http.Client Timeout.  If not set, DefaultClientTimeout is used.
	ClientTimeout time.Duration `json:"clientTimeout"`

	// Concurrency is the maximum number of concurrent fanouts allowed.  If this is not set, DefaultConcurrency is used.
	Concurrency int `json:"concurrency"`

	// MaxRedirects defines the maximum number of redirects each fanout will allow
	MaxRedirects int `json:"maxRedirects"`

	// RedirectExcludeHeaders are the headers that will *not* be copied on a redirect
	RedirectExcludeHeaders []string `json:"redirectExcludeHeaders,omitempty"`
}

func (o *Options) endpoints() []string {
	if o != nil {
		return o.Endpoints
	}

	return nil
}

func (o *Options) authorization() string {
	if o != nil && len(o.Authorization) > 0 {
		return o.Authorization
	}

	return ""
}

func (o *Options) fanoutTimeout() time.Duration {
	if o != nil && o.FanoutTimeout > 0 {
		return o.FanoutTimeout
	}

	return DefaultFanoutTimeout
}

func (o *Options) clientTimeout() time.Duration {
	if o != nil && o.ClientTimeout > 0 {
		return o.ClientTimeout
	}

	return DefaultClientTimeout
}

func (o *Options) transport() *http.Transport {
	transport := new(http.Transport)

	if o != nil {
		*transport = o.Transport
	}

	return transport
}

func (o *Options) concurrency() int {
	if o != nil && o.Concurrency > 0 {
		return o.Concurrency
	}

	return DefaultConcurrency
}

func (o *Options) maxRedirects() int {
	if o != nil {
		return o.MaxRedirects
	}

	return 0
}

func (o *Options) redirectExcludeHeaders() []string {
	if o != nil {
		return o.RedirectExcludeHeaders
	}

	return nil
}

func (o *Options) checkRedirect() func(*http.Request, []*http.Request) error {
	return xhttp.CheckRedirect(xhttp.RedirectPolicy{
		MaxRedirects:   o.maxRedirects(),
		ExcludeHeaders: o.redirectExcludeHeaders(),
	})
}

// NewTransactor constructs an HTTP client transaction function from a set of fanout options.
func NewTransactor(o Options) func(*http.Request) (*http.Response, error) {
	return (&http.Client{
		Transport:     o.transport(),
		CheckRedirect: o.checkRedirect(),
		Timeout:       o.clientTimeout(),
	}).Do
}

// NewChain constructs an Alice constructor Chain from a set of fanout options and zero or
// more application-layer request functions.
func NewChain(o Options, rf ...func(context.Context, *http.Request) context.Context) alice.Chain {
	return alice.New(
		xcontext.Populate(o.fanoutTimeout(), rf...),
		xhttp.Busy(o.concurrency()),
	)
}
