package wrphttp

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/middleware"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	gokithttp "github.com/go-kit/kit/transport/http"
)

const (
	DefaultMethod                            = "POST"
	DefaultEndpoint                          = "http://localhost:7000/api/v2/device"
	DefaultMaxIdleConnsPerHost               = 20
	DefaultFanoutTimeout       time.Duration = 45 * time.Second
	DefaultClientTimeout       time.Duration = 30 * time.Second
	DefaultMaxClients          int64         = 10000
	DefaultConcurrency                       = 1000
	DefaultEncoderPoolSize                   = 100
	DefaultDecoderPoolSize                   = 100
)

// FanoutOptions describe the options available for a go-kit HTTP server that does fanout via middleware.Fanout.
type FanoutOptions struct {
	// Logger is the go-kit logger to use when creating the service fanout.  If not set, logging.DefaultLogger is used.
	Logger log.Logger `json:"-"`

	// Method is the HTTP method to use for all endpoints.  If not set, DefaultMethod is used.
	Method string `json:"method,omitempty"`

	// Endpoints are the URLs for each endpoint to fan out to.  If not set, DefaultEndpoint is used.
	Endpoints []string `json:"endpoints,omitempty"`

	// Transport is the http.Client transport
	Transport http.Transport `json:"transport"`

	// FanoutTimeout is the timeout for the entire fanout operation.  If not supplied, DefaultFanoutTimeout is used.
	FanoutTimeout time.Duration `json:"timeout"`

	// ClientTimeout is the http.Client Timeout.  If not set, DefaultClientTimeout is used.
	ClientTimeout time.Duration `json:"clientTimeout"`

	// MaxClients is the maximum number of concurrent clients that can be using the fanout.  This should be set to
	// something larger than the Concurrency field.
	MaxClients int64 `json:"maxClients"`

	// Concurrency is the maximum number of concurrent fanouts allowed.  This is enforced via a Concurrent middleware.
	// If this is not set, DefaultConcurrency is used.
	Concurrency int `json:"concurrency"`

	// EncoderPoolSize is the size of the WRP encoder pool.  If not set, DefaultEncoderPoolSize is used.
	EncoderPoolSize int

	// DecoderPoolSize is the size of the WRP encoder pool.  If not set, DefaultDecoderPoolSize is used.
	DecoderPoolSize int

	// Middleware is the extra Middleware to append, which can (and often is) empty
	Middleware []endpoint.Middleware `json:"-"`
}

func (f *FanoutOptions) logger() log.Logger {
	if f != nil && f.Logger != nil {
		return f.Logger
	}

	return logging.DefaultLogger()
}

func (f *FanoutOptions) method() string {
	if f != nil && len(f.Method) > 0 {
		return f.Method
	}

	return DefaultMethod
}

func (f *FanoutOptions) endpoints() []string {
	if f != nil && len(f.Endpoints) > 0 {
		return f.Endpoints
	}

	return []string{DefaultEndpoint}
}

func (f *FanoutOptions) urls() ([]*url.URL, error) {
	var urls []*url.URL
	for _, endpoint := range f.endpoints() {
		url, err := url.Parse(endpoint)
		if err != nil {
			return nil, err
		}

		urls = append(urls, url)
	}

	return urls, nil
}

func (f *FanoutOptions) transport() *http.Transport {
	transport := new(http.Transport)

	if f != nil {
		*transport = f.Transport
	}

	if transport.MaxIdleConnsPerHost < 1 {
		transport.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}

	return transport
}

func (f *FanoutOptions) fanoutTimeout() time.Duration {
	if f != nil && f.FanoutTimeout > 0 {
		return f.FanoutTimeout
	}

	return DefaultFanoutTimeout
}

func (f *FanoutOptions) clientTimeout() time.Duration {
	if f != nil && f.ClientTimeout > 0 {
		return f.ClientTimeout
	}

	return DefaultClientTimeout
}

func (f *FanoutOptions) maxClients() int64 {
	if f != nil && f.MaxClients > 0 {
		return f.MaxClients
	}

	return DefaultMaxClients
}

func (f *FanoutOptions) concurrency() int {
	if f != nil && f.Concurrency > 0 {
		return f.Concurrency
	}

	return DefaultConcurrency
}

func (f *FanoutOptions) encoderPoolSize() int {
	if f != nil && f.EncoderPoolSize > 0 {
		return f.EncoderPoolSize
	}

	return DefaultEncoderPoolSize
}

func (f *FanoutOptions) decoderPoolSize() int {
	if f != nil && f.DecoderPoolSize > 0 {
		return f.DecoderPoolSize
	}

	return DefaultDecoderPoolSize
}

func (f *FanoutOptions) middleware() []endpoint.Middleware {
	if f != nil {
		return f.Middleware
	}

	return nil
}

// NewFanoutEndpoint uses the supplied options to produce a go-kit HTTP server endpoint which
// fans out to the HTTP endpoints specified in the options.  The endpoint returned from this
// can be used to build one or more go-kit transport/http.Server objects.
//
// The FanoutOptions can be nil, in which case a set of defaults is used.
func NewFanoutEndpoint(o *FanoutOptions) (endpoint.Endpoint, error) {
	var (
		encoderPool = wrp.NewEncoderPool(o.encoderPoolSize(), wrp.Msgpack)
		decoderPool = wrp.NewDecoderPool(o.decoderPoolSize(), wrp.Msgpack)

		httpClient = &http.Client{
			Transport: o.transport(),
			Timeout:   o.clientTimeout(),
		}

		endpoints       = o.endpoints()
		fanoutEndpoints = make(map[string]endpoint.Endpoint, len(endpoints))
	)

	for _, e := range endpoints {
		url, err := url.Parse(e)
		if err != nil {
			return nil, err
		}

		fanoutEndpoints[e] =
			gokithttp.NewClient(
				o.method(),
				url,
				ClientEncodeRequestBody(encoderPool, http.Header{"Accept": []string{"application/msgpack"}}),
				ClientDecodeResponseBody(decoderPool),
				gokithttp.SetClient(httpClient),
			).Endpoint()
	}

	var (
		middlewareChain = append(
			[]endpoint.Middleware{
				middleware.Busy(o.maxClients(), &httperror.E{Code: http.StatusServiceUnavailable, Text: "Server Busy"}),
				middleware.Timeout(o.fanoutTimeout()),
				middleware.Concurrent(o.concurrency(), &httperror.E{Code: http.StatusTooManyRequests, Text: "Too Many Requests"}),
			},
			o.middleware()...,
		)
	)

	return endpoint.Chain(
			middlewareChain[0],
			middlewareChain[1:]...,
		)(middleware.Fanout(tracing.NewSpanner(), fanoutEndpoints)),
		nil
}
