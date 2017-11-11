package middlewarehttp

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/middleware"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	gokithttp "github.com/go-kit/kit/transport/http"
)

const (
	DefaultMethod                            = "POST"
	DefaultMaxIdleConnsPerHost               = 20
	DefaultTimeout             time.Duration = 30 * time.Second
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

	// Endpoints are the URLs for each endpoint to fan out to.  Each URL may be a URI template, in which case it will
	// be made available to request encoders.
	Endpoints []string `json:"endpoints,omitempty"`

	// Authorization is the Basic Auth token.  There is no default for this field.
	Authorization string `json:"authorization"`

	// Timeout is the http.Client Timeout.  If not set, DefaultTimeout is used.
	ClientTimeout time.Duration `json:"timeout"`

	// EncodeRequest is the go-kit transport/http request encoder.  This field is required, and there is no default.
	EncodeRequest gokithttp.EncodeRequestFunc `json:"-"`

	// DecodeResponse is the go-kit transport/http response decoder.  This field is required, and there is no default.
	DecodeResponse gokithttp.DecodeResponseFunc `json:"-"`

	ClientOptions []gokithttp.ClientOption
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

	return nil
}

func (f *FanoutOptions) authorization() string {
	if f != nil && len(f.Authorization) > 0 {
		return f.Authorization
	}

	return ""
}

func (f *FanoutOptions) timeout() time.Duration {
	if f != nil && f.Timeout > 0 {
		return f.Timeout
	}

	return DefaultTimeout
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
	urls, err := o.urls()
	if err != nil {
		return nil, err
	}

	var (
		httpClient = &http.Client{
			Transport: o.transport(),
			Timeout:   o.clientTimeout(),
		}

		fanoutEndpoints = make(map[string]endpoint.Endpoint, len(urls))
		customHeader    = http.Header{
			"Accept": []string{"application/msgpack"},
		}
	)

	if authorization := o.authorization(); len(authorization) > 0 {
		customHeader.Set("Authorization", "Basic "+authorization)
	}

	for _, url := range urls {
		fanoutEndpoints[url.String()] =
			gokithttp.NewClient(
				o.method(),
				url,
				ClientEncodeRequestBody(encoderPool, customHeader),
				ClientDecodeResponseBody(decoderPool),
				gokithttp.SetClient(httpClient), gokithttp.ClientBefore(SetGetBodyFunc),
			).Endpoint()
	}

	var (
		middlewareChain = append(
			[]endpoint.Middleware{
				middleware.Logging,
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

//SetGetBodyFunc allows reading a request's body multiple times. 307 POST redirects are part of the specific cases
//where this becomes useful
func SetGetBodyFunc(context context.Context, request *http.Request) context.Context {
	if request == nil || request.Body == nil {
		return context
	}

	if freshBodyCopy, err := ioutil.ReadAll(request.Body); err == nil { //read it once and keep a copy
		var keepBuffer bytes.Buffer
		keepBuffer.Write(freshBodyCopy)
		request.Body = ioutil.NopCloser(&keepBuffer) //Extra: Also make request.Body re-readable

		//set up function used by clients such as net/http/client
		request.GetBody = func() (send io.ReadCloser, err error) {
			var sendBuffer bytes.Buffer
			sendBuffer.Write(freshBodyCopy)
			send = ioutil.NopCloser(&sendBuffer)
			return
		}
	}
	return context
}
