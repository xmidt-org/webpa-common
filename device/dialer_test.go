package device

import (
	"net/http"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testDialerDialDeviceDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedURL      = "http://nowhere.com:8080/foo"
		expectedConn     = new(websocket.Conn)
		expectedResponse = new(http.Response)

		websocketDialer = new(mockWebsocketDialer)
		dialer          = NewDialer(DialerOptions{WSDialer: websocketDialer})
	)

	require.NotNil(dialer)

	dialer.DialDevice("mac:112233445566/service", expectedURL, nil)

	websocketDialer.AssertExpectations(t)
}

func TestDialer(t *testing.T) {
}
