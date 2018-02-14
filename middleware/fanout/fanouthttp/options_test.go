package fanouthttp

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testOptionsDefaults(t *testing.T, o *Options) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
	)

	assert.NotNil(o.logger())
	assert.Empty(o.endpoints())
	assert.Empty(o.authorization())

	transport := o.transport()
	require.NotNil(transport)
	assert.Equal(DefaultMaxIdleConnsPerHost, transport.MaxIdleConnsPerHost)

	client := o.NewClient()
	require.NotNil(client)
	assert.Equal(*transport, *client.Transport.(*http.Transport))

	assert.Equal(DefaultFanoutTimeout, o.fanoutTimeout())
	assert.Equal(DefaultClientTimeout, o.clientTimeout())
	assert.Equal(DefaultMaxClients, o.maxClients())
	assert.Equal(DefaultConcurrency, o.concurrency())

	var (
		expectedRequest  = "expected request"
		expectedResponse = "expected response"
		expectedError    = errors.New("expected error")
	)

	endpointCalled := false
	e := o.loggerMiddleware(func(ctx context.Context, actualRequest interface{}) (interface{}, error) {
		endpointCalled = true
		assert.Equal(expectedRequest, actualRequest)
		assert.Equal(logging.DefaultLogger(), logging.FromContext(ctx))
		return expectedResponse, expectedError
	})

	require.NotNil(e)
	e(context.Background(), expectedRequest)
	assert.True(endpointCalled)

	assert.NotNil(o.FanoutMiddleware())
}

func testOptionsConfigured(t *testing.T) {
	var (
		require        = require.New(t)
		assert         = assert.New(t)
		expectedLogger = logging.NewTestLogger(nil, t)

		o = Options{
			Logger:        expectedLogger,
			Endpoints:     []string{"http://host1.com:8080/api", "http://host2.com:9090/api"},
			Authorization: "QWxhZGRpbjpPcGVuU2VzYW1l",
			Transport: http.Transport{
				IdleConnTimeout:     30 * time.Minute,
				MaxIdleConnsPerHost: 256,
			},
			FanoutTimeout: 500 * time.Second,
			ClientTimeout: 37 * time.Second,
			MaxClients:    38734,
			Concurrency:   3249,
		}
	)

	assert.Equal(expectedLogger, o.logger())
	assert.Equal([]string{"http://host1.com:8080/api", "http://host2.com:9090/api"}, o.endpoints())
	assert.Equal("QWxhZGRpbjpPcGVuU2VzYW1l", o.authorization())

	transport := o.transport()
	require.NotNil(transport)
	assert.Equal(30*time.Minute, transport.IdleConnTimeout)
	assert.Equal(256, transport.MaxIdleConnsPerHost)

	client := o.NewClient()
	require.NotNil(client)
	assert.Equal(*transport, *client.Transport.(*http.Transport))

	assert.Equal(500*time.Second, o.fanoutTimeout())
	assert.Equal(37*time.Second, o.clientTimeout())
	assert.Equal(int64(38734), o.maxClients())
	assert.Equal(3249, o.concurrency())

	var (
		expectedRequest  = "expected request"
		expectedResponse = "expected response"
		expectedError    = errors.New("expected error")
	)

	endpointCalled := false
	e := o.loggerMiddleware(func(ctx context.Context, actualRequest interface{}) (interface{}, error) {
		endpointCalled = true
		assert.Equal(expectedRequest, actualRequest)
		assert.Equal(expectedLogger, logging.FromContext(ctx))
		return expectedResponse, expectedError
	})

	require.NotNil(e)
	e(context.Background(), expectedRequest)
	assert.True(endpointCalled)

	assert.NotNil(o.FanoutMiddleware())
}

func TestOptions(t *testing.T) {
	t.Run("Defaults", func(t *testing.T) {
		testOptionsDefaults(t, nil)
		testOptionsDefaults(t, new(Options))
	})

	t.Run("Configured", testOptionsConfigured)
}
