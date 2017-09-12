package wrphttp

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Comcast/webpa-common/wrp"
	"github.com/Comcast/webpa-common/wrp/wrpendpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerDecodeRequestBody(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		pool        = wrp.NewDecoderPool(1, wrp.JSON)
		httpRequest = httptest.NewRequest("GET", "/", strings.NewReader(`
			{"msg_type": 3, "source": "test", "dest": "mac:123412341234"}
		`))
	)

	value, err := ServerDecodeRequestBody(pool)(context.Background(), httpRequest)
	require.NotNil(value)
	require.NoError(err)

	wrpRequest, ok := value.(wrpendpoint.Request)
	require.True(ok)

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleRequestResponseMessageType,
			Source:      "test",
			Destination: "mac:123412341234",
		},
		*wrpRequest.Message(),
	)
}

func testServerDecodeRequestHeadersSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	httpRequest.Header.Set(MessageTypeHeader, "SimpleEvent")
	httpRequest.Header.Set(SourceHeader, "test")
	httpRequest.Header.Set(DestinationHeader, "mac:432143214321")

	value, err := ServerDecodeRequestHeaders(context.Background(), httpRequest)
	require.NotNil(value)
	require.NoError(err)

	wrpRequest, ok := value.(wrpendpoint.Request)
	require.True(ok)

	assert.Equal(
		wrp.Message{
			Type:        wrp.SimpleEventMessageType,
			Source:      "test",
			Destination: "mac:432143214321",
		},
		*wrpRequest.Message(),
	)
}

func testServerDecodeRequestHeadersBadHeaders(t *testing.T) {
	var (
		assert      = assert.New(t)
		httpRequest = httptest.NewRequest("GET", "/", nil)
	)

	value, err := ServerDecodeRequestHeaders(context.Background(), httpRequest)
	assert.Nil(value)
	assert.Error(err)
}

func TestServerDecodeRequestHeaders(t *testing.T) {
	t.Run("Success", testServerDecodeRequestHeadersSuccess)
	t.Run("BadHeaders", testServerDecodeRequestHeadersBadHeaders)
}
