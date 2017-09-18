package wrphttp

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFanoutOptionsDefaults(t *testing.T, o *FanoutOptions) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
	)

	assert.NotNil(o.logger())
	assert.Equal(DefaultMethod, o.method())
	assert.Equal([]string{DefaultEndpoint}, o.endpoints())

	urls, err := o.urls()
	require.Len(urls, 1)
	assert.NoError(err)
	assert.Equal(DefaultEndpoint, urls[0].String())

	transport := o.transport()
	require.NotNil(transport)
	assert.Equal(DefaultMaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)

	assert.Equal(DefaultFanoutTimeout, o.fanoutTimeout())
	assert.Equal(DefaultClientTimeout, o.clientTimeout())
	assert.Equal(DefaultMaxClients, o.maxClients())
	assert.Equal(DefaultConcurrency, o.concurrency())
	assert.Equal(DefaultEncoderPoolSize, o.encoderPoolSize())
	assert.Equal(DefaultDecoderPoolSize, o.decoderPoolSize())
	assert.Empty(o.middleware())
}

func testFanoutOptionsConfigured(t *testing.T) {
	var (
		require          = require.New(t)
		assert           = assert.New(t)
		expectedLogger   = logging.NewTestLogger(nil, t)
		middlewareCalled = false

		o = FanoutOptions{
			Logger:    expectedLogger,
			Method:    "GET",
			Endpoints: []string{"http://host1.com:8080/api", "http://host2.com:9090/api"},
			Transport: http.Transport{
				IdleConnTimeout:     30 * time.Minute,
				MaxIdleConnsPerHost: 256,
			},
			FanoutTimeout:   500 * time.Second,
			ClientTimeout:   37 * time.Second,
			MaxClients:      38734,
			Concurrency:     3249,
			EncoderPoolSize: 56,
			DecoderPoolSize: 98234,
			Middleware: []endpoint.Middleware{
				func(e endpoint.Endpoint) endpoint.Endpoint {
					middlewareCalled = true
					return nil
				},
			},
		}
	)

	assert.Equal(expectedLogger, o.logger())
	assert.Equal("GET", o.method())
	assert.Equal([]string{"http://host1.com:8080/api", "http://host2.com:9090/api"}, o.endpoints())

	urls, err := o.urls()
	require.Len(urls, 2)
	assert.NoError(err)
	assert.Equal("http://host1.com:8080/api", urls[0].String())
	assert.Equal("http://host2.com:9090/api", urls[1].String())

	transport := o.transport()
	require.NotNil(transport)
	assert.Equal(30*time.Minute, transport.IdleConnTimeout)
	assert.Equal(256, transport.MaxIdleConnsPerHost)

	assert.Equal(500*time.Second, o.fanoutTimeout())
	assert.Equal(37*time.Second, o.clientTimeout())
	assert.Equal(int64(38734), o.maxClients())
	assert.Equal(3249, o.concurrency())
	assert.Equal(56, o.encoderPoolSize())
	assert.Equal(98234, o.decoderPoolSize())

	middleware := o.middleware()
	require.Len(middleware, 1)
	middleware[0](nil)
	assert.True(middlewareCalled)
}

func testFanoutOptionsBadURL(t *testing.T) {
	var (
		assert = assert.New(t)
		o      = FanoutOptions{
			Endpoints: []string{" : 7 // this is a bad url"},
		}
	)

	urls, err := o.urls()
	assert.Empty(urls)
	assert.Error(err)
}

func TestFanoutOptions(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		testFanoutOptionsDefaults(t, nil)
		testFanoutOptionsDefaults(t, new(FanoutOptions))
	})

	t.Run("Configured", testFanoutOptionsConfigured)
	t.Run("BadURL", testFanoutOptionsBadURL)
}

func testNewFanoutEndpointSendReceive(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
		logger  = logging.NewTestLogger(nil, t)

		expectedMessage = &wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:123412341234",
			ContentType: "text/plain",
			Payload:     []byte("yay!"),
		}

		server = httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			assert.Equal(wrp.Msgpack.ContentType(), request.Header.Get("Content-Type"))
			assert.Equal(wrp.Msgpack.ContentType(), request.Header.Get("Accept"))

			contents, _ := ioutil.ReadAll(request.Body)
			actualRequest, err := wrpendpoint.DecodeRequestBytes(logger, contents, wrp.NewDecoderPool(1, wrp.Msgpack))
			if !assert.NoError(err) {
				response.WriteHeader(http.StatusInternalServerError)
				return
			} else if !assert.Equal(*expectedMessage, *actualRequest.Message()) {
				response.WriteHeader(http.StatusInternalServerError)
				return
			}

			response.Header().Set("Content-Type", "application/msgpack")
			response.Write(contents)
		}))

		o = &FanoutOptions{
			Endpoints: []string{server.URL},
		}
	)

	defer server.Close()
	fanoutEndpoint, err := NewFanoutEndpoint(o)
	require.NotNil(fanoutEndpoint)
	assert.NoError(err)

	result, err := fanoutEndpoint(
		context.Background(),
		wrpendpoint.WrapAsRequest(logger, expectedMessage),
	)

	require.NotNil(result)
	require.NoError(err)
	assert.Equal(*expectedMessage, *result.(wrpendpoint.Response).Message())
}

func testNewFanoutEndpointBadURL(t *testing.T) {
	var (
		assert = assert.New(t)
		o      = &FanoutOptions{
			Endpoints: []string{" : 7 // this is a bad url"},
		}
	)

	fanoutEndpoint, err := NewFanoutEndpoint(o)
	assert.Nil(fanoutEndpoint)
	assert.Error(err)
}

func TestNewFanoutEndpoint(t *testing.T) {
	t.Run("SendReceive", testNewFanoutEndpointSendReceive)
	t.Run("BadURL", testNewFanoutEndpointBadURL)
}
