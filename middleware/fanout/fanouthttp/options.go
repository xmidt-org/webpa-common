package fanouthttp

import (
	"context"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/middleware"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
)

const (
	DefaultMaxIdleConnsPerHost               = 20
	DefaultFanoutTimeout       time.Duration = 45 * time.Second
	DefaultClientTimeout       time.Duration = 30 * time.Second
	DefaultMaxClients          int64         = 10000
	DefaultConcurrency                       = 1000
)

// Options defines the common options useful for creating HTTP fanouts.
type Options struct {
	// Logger is the go-kit logger to use when creating the service fanout.  If not set, logging.DefaultLogger is used.
	Logger log.Logger `json:"-"`

	// Endpoints are the URLs for each endpoint to fan out to.  If not set, DefaultEndpoint is used.
	Endpoints []string `json:"endpoints,omitempty"`

	// Authorization is the Basic Auth token.  There is no default for this field.
	Authorization string `json:"authorization"`

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
}

func (o *Options) logger() log.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
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

	if transport.MaxIdleConnsPerHost < 1 {
		transport.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}

	return transport
}

func (o *Options) maxClients() int64 {
	if o != nil && o.MaxClients > 0 {
		return o.MaxClients
	}

	return DefaultMaxClients
}

func (o *Options) concurrency() int {
	if o != nil && o.Concurrency > 0 {
		return o.Concurrency
	}

	return DefaultConcurrency
}

// NewClient returns a distinct HTTP client synthesized from these options
func (o *Options) NewClient() *http.Client {
	return &http.Client{
		Transport: o.transport(),
		Timeout:   o.clientTimeout(),
	}
}

func (o *Options) loggerMiddleware(next endpoint.Endpoint) endpoint.Endpoint {
	logger := o.logger()
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx = logging.WithLogger(ctx, logger)
		return next(ctx, request)
	}
}

// FanoutMiddleware uses these options to produce a go-kit middleware decorator for the
// fanout endpoint.
func (o *Options) FanoutMiddleware() endpoint.Middleware {
	return endpoint.Chain(
		// logging is the outermost middleware, so everything downstream can log consistently
		o.loggerMiddleware,
		middleware.Busy(o.maxClients(), &httperror.E{Code: http.StatusTooManyRequests, Text: "Server Busy"}),
		middleware.Timeout(o.fanoutTimeout()),
		middleware.Concurrent(o.concurrency(), &httperror.E{Code: http.StatusServiceUnavailable, Text: "Server Busy"}),
	)
}
