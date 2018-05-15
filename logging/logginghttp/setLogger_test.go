package logginghttp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

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

func testSetLoggerNilBase(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		SetLogger(nil)
	})
}

func testSetLoggerBaseOnly(t *testing.T) {
	var (
		assert = assert.New(t)

		base    = logging.NewTestLogger(nil, t)
		request = httptest.NewRequest("GET", "/", nil)
		ctx     = SetLogger(base)(context.Background(), request)
	)

	assert.Equal(base, logging.GetLogger(ctx))
}

func testSetLoggerCustom(t *testing.T) {
}

func TestSetLogger(t *testing.T) {
	t.Run("NilBase", testSetLoggerNilBase)
	t.Run("BaseOnly", testSetLoggerBaseOnly)
	t.Run("Custom", testSetLoggerCustom)
}
