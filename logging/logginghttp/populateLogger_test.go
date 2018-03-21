package logginghttp

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestProtoKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(requestProtoKey, RequestProtoKey())
}

func TestRequestMethodKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(requestMethodKey, RequestMethodKey())
}

func TestRequestURIKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(requestURIKey, RequestURIKey())
}

func TestRemoteAddrKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(remoteAddrKey, RemoteAddrKey())
}

func testPopulateLogger(t *testing.T, base log.Logger) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		nextCalled = false
		next       = http.HandlerFunc(func(rw http.ResponseWriter, request *http.Request) {
			nextCalled = true
			assert.Equal(response, rw)
			assert.NotNil(logging.GetLogger(request.Context()))
		})

		constructor = PopulateLogger(base)
	)

	require.NotNil(constructor)

	decorated := constructor(next)
	require.NotNil(decorated)

	decorated.ServeHTTP(response, request)
	assert.True(nextCalled)
}

func TestPopulateLogger(t *testing.T) {
	t.Run("DefaultLogger", func(t *testing.T) {
		testPopulateLogger(t, nil)
	})

	t.Run("CustomLogger", func(t *testing.T) {
		testPopulateLogger(t, logging.NewTestLogger(nil, t))
	})
}
