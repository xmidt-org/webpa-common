package wrphttp

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	"github.com/go-kit/kit/log"
	gokithttp "github.com/go-kit/kit/transport/http"
)

const (
	DefaultMethod              = "POST"
	DefaultEndpoint            = "http://localhost:7000/api/v2/device"
	DefaultTimeout             = 30 * time.Second
	DefaultMaxIdleConnsPerHost = 20
	DefaultEncoderPoolSize     = 100
	DefaultDecoderPoolSize     = 100
)

// Fanout describes a WRP service with fans out to other HTTP endpoints, using wrp.NewServiceFanout.
type Fanout struct {
	// Logger is the go-kit logger to use when creating the service fanout.  If not set, logging.DefaultLogger is used.
	Logger log.Logger `json:"-"`

	// Method is the HTTP method to use for all endpoints.  If not set, DefaultMethod is used.
	Method string `json:"method,omitempty"`

	// Endpoints are the URLs for each endpoint to fan out to.  If not set, DefaultEndpoint is used.
	Endpoints []string `json:"endpoints,omitempty"`

	// Transport is the http.Client transport
	Transport http.Transport `json:"transport"`

	// Timeout is the http.Client Timeout.  If not set, DefaultTimeout is used.
	Timeout time.Duration `json:"timeout"`

	// EncoderPoolSize is the size of the WRP encoder pool.  If not set, DefaultEncoderPoolSize is used.
	EncoderPoolSize int

	// DecoderPoolSize is the size of the WRP encoder pool.  If not set, DefaultDecoderPoolSize is used.
	DecoderPoolSize int
}

func (f *Fanout) logger() log.Logger {
	if f != nil && f.Logger != nil {
		return f.Logger
	}

	return logging.DefaultLogger()
}

func (f *Fanout) method() string {
	if f != nil && len(f.Method) > 0 {
		return f.Method
	}

	return DefaultMethod
}

func (f *Fanout) endpoints() []string {
	if f != nil && len(f.Endpoints) > 0 {
		return f.Endpoints
	}

	return []string{DefaultEndpoint}
}

func (f *Fanout) urls() ([]*url.URL, error) {
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

func (f *Fanout) transport() *http.Transport {
	if f != nil {
		copyOf := f.Transport
		if copyOf.MaxIdleConnsPerHost < 1 {
			copyOf.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
		}

		return &copyOf
	}

	return &http.Transport{
		MaxIdleConnsPerHost: DefaultMaxIdleConnsPerHost,
	}
}

func (f *Fanout) timeout() time.Duration {
	if f != nil && f.Timeout > 0 {
		return f.Timeout
	}

	return DefaultTimeout
}

func (f *Fanout) encoderPoolSize() int {
	if f != nil && f.EncoderPoolSize > 0 {
		return f.EncoderPoolSize
	}

	return DefaultEncoderPoolSize
}

func (f *Fanout) decoderPoolSize() int {
	if f != nil && f.DecoderPoolSize > 0 {
		return f.DecoderPoolSize
	}

	return DefaultDecoderPoolSize
}

// NewServiceFanout uses the given Fanout to produce a WRP service that fans out to the given HTTP endpoints
func NewServiceFanout(f *Fanout) (wrpendpoint.Service, error) {
	urls, err := f.urls()
	if err != nil {
		return nil, err
	}

	var (
		httpClient = &http.Client{
			Transport: f.transport(),
			Timeout:   f.timeout(),
		}

		encoders = wrp.NewEncoderPool(f.encoderPoolSize(), wrp.Msgpack)
		decoders = wrp.NewDecoderPool(f.decoderPoolSize(), wrp.Msgpack)

		customHeader = http.Header{
			"Accept": []string{decoders.Format().ContentType()},
		}

		endpoints = make(map[string]wrpendpoint.Service, len(urls))
	)

	for _, url := range urls {
		name := url.String()
		if _, ok := endpoints[name]; ok {
			return nil, fmt.Errorf("Duplicate endpoint url: %s", url)
		}

		endpoints[name] = wrpendpoint.Wrap(
			gokithttp.NewClient(
				f.method(),
				url,
				ClientEncodeRequestBody(encoders, customHeader),
				ClientDecodeResponseBody(decoders),
				gokithttp.SetClient(httpClient),
			).Endpoint(),
		)
	}

	return wrpendpoint.NewServiceFanout(endpoints), nil
}
