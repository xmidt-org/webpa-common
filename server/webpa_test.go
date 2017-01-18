package server

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.Called(response, request)
}

func TestWebPADefaults(t *testing.T) {
	assert := assert.New(t)
	for _, webPA := range []*WebPA{nil, new(WebPA)} {
		assert.Equal(DefaultName, webPA.name())
		assert.Equal(DefaultAddress, webPA.address())
		assert.Equal(DefaultHealthAddress, webPA.healthAddress())
		assert.Equal("", webPA.pprofAddress())
		assert.Equal(DefaultHealthLogInterval, webPA.healthLogInterval())
		assert.Equal(DefaultLogConnectionState, webPA.logConnectionState())
		assert.Equal("", webPA.certificateFile())
		assert.Equal("", webPA.keyFile())
	}
}

func TestWebPAAccessors(t *testing.T) {
	const healthLogInterval time.Duration = 46 * time.Minute

	var (
		assert = assert.New(t)
		webPA  = WebPA{
			Name:               "Custom Name",
			CertificateFile:    "custom.cert",
			KeyFile:            "custom.key",
			LogConnectionState: !DefaultLogConnectionState,
			Address:            "localhost:15001",
			HealthAddress:      "localhost:55",
			HealthLogInterval:  healthLogInterval,
			PprofAddress:       "foobar:7273",
		}
	)

	assert.Equal("Custom Name", webPA.name())
	assert.Equal("custom.cert", webPA.certificateFile())
	assert.Equal("custom.key", webPA.keyFile())
	assert.Equal(!DefaultLogConnectionState, webPA.logConnectionState())
	assert.Equal("localhost:15001", webPA.address())
	assert.Equal("localhost:55", webPA.healthAddress())
	assert.Equal(healthLogInterval, webPA.healthLogInterval())
	assert.Equal("foobar:7273", webPA.pprofAddress())
}

func TestNewPrimaryServer(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:    "TestNewPrimaryServer",
			Address: ":6007",
		}

		primaryServer = webPA.NewPrimaryServer(logger, handler)
	)

	require.NotNil(primaryServer)
	assert.Equal(":6007", primaryServer.Addr)
	assert.Equal(handler, primaryServer.Handler)
	assert.Nil(primaryServer.ConnState)
	assertErrorLog(assert, verify, "TestNewPrimaryServer", primaryServer.ErrorLog)

	handler.AssertExpectations(t)
}

func TestNewPrimaryServerLogConnectionState(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:               "TestNewPrimaryServer",
			Address:            ":6007",
			LogConnectionState: true,
		}

		primaryServer = webPA.NewPrimaryServer(logger, handler)
	)

	require.NotNil(primaryServer)
	assert.Equal(":6007", primaryServer.Addr)
	assert.Equal(handler, primaryServer.Handler)
	assertErrorLog(assert, verify, "TestNewPrimaryServer", primaryServer.ErrorLog)
	assertConnState(assert, verify, primaryServer.ConnState)

	handler.AssertExpectations(t)
}
