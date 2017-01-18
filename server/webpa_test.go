package server

import (
	"errors"
	"github.com/Comcast/webpa-common/health"
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

type mockServerExecutor struct {
	mock.Mock
}

func (m *mockServerExecutor) ListenAndServe() error {
	return m.Called().Error(0)
}

func (m *mockServerExecutor) ListenAndServeTLS(certificateFile, keyFile string) error {
	return m.Called(certificateFile, keyFile).Error(0)
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
			Name:               "TestNewPrimaryServerLogConnectionState",
			Address:            ":331",
			LogConnectionState: true,
		}

		primaryServer = webPA.NewPrimaryServer(logger, handler)
	)

	require.NotNil(primaryServer)
	assert.Equal(":331", primaryServer.Addr)
	assert.Equal(handler, primaryServer.Handler)
	assertErrorLog(assert, verify, "TestNewPrimaryServerLogConnectionState", primaryServer.ErrorLog)
	assertConnState(assert, verify, primaryServer.ConnState)

	handler.AssertExpectations(t)
}

func TestNewHealthServer(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		options = []health.Option{health.Stat("option1"), health.Stat("option2")}

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:              "TestNewHealthServer",
			HealthAddress:     ":7181",
			HealthLogInterval: 15 * time.Second,
		}

		healthHandler, healthServer = webPA.NewHealthServer(logger, options...)
	)

	require.NotNil(healthHandler)
	require.NotNil(healthServer)
	assert.Equal(":7181", healthServer.Addr)
	assert.Equal(healthHandler, healthServer.Handler)
	assertErrorLog(assert, verify, "TestNewHealthServer", healthServer.ErrorLog)
	assert.Nil(healthServer.ConnState)
}

func TestNewHealthServerLogConnectionState(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		options = []health.Option{health.Stat("option1"), health.Stat("option2")}

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:               "TestNewHealthServerLogConnectionState",
			HealthAddress:      ":165",
			HealthLogInterval:  45 * time.Minute,
			LogConnectionState: true,
		}

		healthHandler, healthServer = webPA.NewHealthServer(logger, options...)
	)

	require.NotNil(healthHandler)
	require.NotNil(healthServer)
	assert.Equal(":165", healthServer.Addr)
	assert.Equal(healthHandler, healthServer.Handler)
	assertErrorLog(assert, verify, "TestNewHealthServerLogConnectionState", healthServer.ErrorLog)
	assertConnState(assert, verify, healthServer.ConnState)
}

func TestNewPprofServer(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:         "TestNewPprofServer",
			PprofAddress: ":996",
		}

		pprofServer = webPA.NewPprofServer(logger, handler)
	)

	require.NotNil(pprofServer)
	assert.Equal(":996", pprofServer.Addr)
	assert.Equal(handler, pprofServer.Handler)
	assert.Nil(pprofServer.ConnState)
	assertErrorLog(assert, verify, "TestNewPprofServer", pprofServer.ErrorLog)

	handler.AssertExpectations(t)
}

func TestNewPprofServerDefaultHandler(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:         "TestNewPprofServerDefaultHandler",
			PprofAddress: ":1299",
		}

		pprofServer = webPA.NewPprofServer(logger, nil)
	)

	require.NotNil(pprofServer)
	assert.Equal(":1299", pprofServer.Addr)
	assert.Equal(http.DefaultServeMux, pprofServer.Handler)
	assert.Nil(pprofServer.ConnState)
	assertErrorLog(assert, verify, "TestNewPprofServerDefaultHandler", pprofServer.ErrorLog)
}

func TestNewPprofServerNoPprofAddress(t *testing.T) {
	var (
		assert = assert.New(t)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name: "TestNewPprofServerNoPprofAddress",
		}

		pprofServer = webPA.NewPprofServer(logger, nil)
	)

	assert.Nil(pprofServer)
	assert.Empty(verify.String())
}

func TestNewPprofServerLogConnectionState(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)

		verify, logger = newTestLogger()
		webPA          = WebPA{
			Name:               "TestNewPprofServerLogConnectionState",
			PprofAddress:       ":16077",
			LogConnectionState: true,
		}

		pprofServer = webPA.NewPprofServer(logger, handler)
	)

	require.NotNil(pprofServer)
	assert.Equal(":16077", pprofServer.Addr)
	assert.Equal(handler, pprofServer.Handler)
	assertErrorLog(assert, verify, "TestNewPprofServerLogConnectionState", pprofServer.ErrorLog)
	assertConnState(assert, verify, pprofServer.ConnState)

	handler.AssertExpectations(t)
}

func TestRunServerNonSecure(t *testing.T) {
	var (
		simpleError = errors.New("TestRunServerNonSecure")
		testData    = []struct {
			webPA            WebPA
			secureIfPossible bool
			expectedError    error
		}{
			{WebPA{}, false, nil},
			{WebPA{}, false, simpleError},
			{WebPA{CertificateFile: "file.cert", KeyFile: "file.key"}, false, nil},
			{WebPA{CertificateFile: "file.cert", KeyFile: "file.key"}, false, simpleError},
			{WebPA{}, true, nil},
			{WebPA{}, true, simpleError},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			executorCalled     = make(chan struct{})
			_, logger          = newTestLogger()
			mockServerExecutor = &mockServerExecutor{}
		)

		// When no certificate and key are configured, a nonsecure server should be run
		mockServerExecutor.
			On("ListenAndServe").
			Return(record.expectedError).
			Once().
			Run(func(mock.Arguments) {
				close(executorCalled)
			})

		record.webPA.RunServer(logger, record.secureIfPossible, mockServerExecutor)
		<-executorCalled

		mockServerExecutor.AssertExpectations(t)
	}
}

func TestRunServerSecure(t *testing.T) {
	const (
		certificateFile = "file.cert"
		keyFile         = "file.key"
	)

	var (
		simpleError = errors.New("TestRunServerSecure")
		testData    = []struct {
			webPA         WebPA
			expectedError error
		}{
			{WebPA{CertificateFile: certificateFile, KeyFile: keyFile}, nil},
			{WebPA{CertificateFile: certificateFile, KeyFile: keyFile}, simpleError},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			executorCalled     = make(chan struct{})
			_, logger          = newTestLogger()
			mockServerExecutor = &mockServerExecutor{}
		)

		mockServerExecutor.
			On("ListenAndServeTLS", certificateFile, keyFile).
			Return(record.expectedError).
			Once().
			Run(func(mock.Arguments) {
				close(executorCalled)
			})

		record.webPA.RunServer(logger, true, mockServerExecutor)
		<-executorCalled

		mockServerExecutor.AssertExpectations(t)
	}
}

func TestRunServerNoExecutor(t *testing.T) {
	testData := []struct {
		webPA            WebPA
		secureIfPossible bool
	}{
		{WebPA{}, false},
		{WebPA{CertificateFile: "file.cert", KeyFile: "file.key"}, false},
		{WebPA{}, true},
		{WebPA{CertificateFile: "file.cert", KeyFile: "file.key"}, true},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		_, logger := newTestLogger()
		record.webPA.RunServer(logger, record.secureIfPossible, nil)
	}
}
