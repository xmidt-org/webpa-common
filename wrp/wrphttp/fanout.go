package wrphttp

import (
	"net/http"
	"net/url"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

const (
	DefaultMethod              = "POST"
	DefaultEndpoint            = "http://localhost:7000/api/v2/device"
	DefaultTimeout             = 30 * time.Second
	DefaultMaxIdleConnsPerHost = 20
	DefaultConcurrency         = 1000
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

	// Concurrency is the maximum number of concurrent fanouts allowed.  This is enforced via a Concurrent middleware.
	// If this is not set, DefaultConcurrency is used.
	Concurrency int `json:"concurrency"`

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

func (f *Fanout) concurrency() int {
	if f != nil && f.Concurrency > 0 {
		return f.Concurrency
	}

	return DefaultConcurrency
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
