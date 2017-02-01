package server

import (
	"errors"
	"github.com/Comcast/webpa-common/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"sync"
	"testing"
	"time"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.Called(response, request)
}

type mockExecutor struct {
	mock.Mock
}

func (m *mockExecutor) ListenAndServe() error {
	return m.Called().Error(0)
}

func (m *mockExecutor) ListenAndServeTLS(certificateFile, keyFile string) error {
	return m.Called(certificateFile, keyFile).Error(0)
}

type mockSecure struct {
	mock.Mock
}

func (m *mockSecure) Certificate() (string, string) {
	arguments := m.Called()
	return arguments.String(0), arguments.String(1)
}

func TestListenAndServeNonSecure(t *testing.T) {
	var (
		simpleError = errors.New("expected")
		testData    = []struct {
			certificateFile, keyFile string
			expectedError            error
		}{
			{"", "", nil},
			{"", "", simpleError},
			{"file.cert", "", nil},
			{"file.cert", "", simpleError},
			{"", "file.key", nil},
			{"", "file.key", simpleError},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			_, logger      = newTestLogger()
			executorCalled = make(chan struct{})
			mockSecure     = new(mockSecure)
			mockExecutor   = new(mockExecutor)
		)

		mockSecure.On("Certificate").
			Return(record.certificateFile, record.keyFile).
			Once()

		mockExecutor.On("ListenAndServe").
			Return(record.expectedError).
			Run(func(mock.Arguments) { close(executorCalled) }).
			Once()

		ListenAndServe(logger, mockSecure, mockExecutor)
		<-executorCalled

		mockSecure.AssertExpectations(t)
		mockExecutor.AssertExpectations(t)
	}
}

func TestListenAndServeSecure(t *testing.T) {
	var (
		testData = []struct {
			expectedError error
		}{
			{nil},
			{errors.New("expected")},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			_, logger      = newTestLogger()
			executorCalled = make(chan struct{})
			mockSecure     = new(mockSecure)
			mockExecutor   = new(mockExecutor)
		)

		mockSecure.On("Certificate").
			Return("file.cert", "file.key").
			Once()

		mockExecutor.On("ListenAndServeTLS", "file.cert", "file.key").
			Return(record.expectedError).
			Run(func(mock.Arguments) { close(executorCalled) }).
			Once()

		ListenAndServe(logger, mockSecure, mockExecutor)
		<-executorCalled

		mockSecure.AssertExpectations(t)
		mockExecutor.AssertExpectations(t)
	}
}

func TestBasicCertificate(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			certificateFile, keyFile string
		}{
			{"", ""},
			{"", "file.key"},
			{"file.cert", ""},
			{"file.cert", "file.key"},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			basic = Basic{
				CertificateFile: record.certificateFile,
				KeyFile:         record.keyFile,
			}

			actualCertificateFile, actualKeyFile = basic.Certificate()
		)

		assert.Equal(record.certificateFile, actualCertificateFile)
		assert.Equal(record.keyFile, actualKeyFile)
	}
}

func TestBasicNew(t *testing.T) {
	const expectedName = "TestBasicNew"

	var (
		assert   = assert.New(t)
		require  = require.New(t)
		testData = []struct {
			address            string
			handler            *mockHandler
			logConnectionState bool
		}{
			{"", nil, false},
			{"", nil, true},
			{"", new(mockHandler), false},
			{"", new(mockHandler), true},
			{":901", nil, false},
			{":19756", nil, true},
			{"localhost:80", new(mockHandler), false},
			{":http", new(mockHandler), true},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			verify, logger = newTestLogger()
			basic          = Basic{
				Name:               expectedName,
				Address:            record.address,
				LogConnectionState: record.logConnectionState,
			}

			server = basic.New(logger, record.handler)
		)

		if len(record.address) > 0 {
			require.NotNil(server)
			assert.Equal(record.address, server.Addr)
			assert.Equal(record.handler, server.Handler)
			assertErrorLog(assert, verify, expectedName, server.ErrorLog)

			if record.logConnectionState {
				assertConnState(assert, verify, server.ConnState)
			} else {
				assert.Nil(server.ConnState)
			}
		} else {
			require.Nil(server)
		}

		if record.handler != nil {
			record.handler.AssertExpectations(t)
		}
	}
}

func TestHealthCertificate(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			certificateFile, keyFile string
		}{
			{"", ""},
			{"", "file.key"},
			{"file.cert", ""},
			{"file.cert", "file.key"},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			health = Health{
				CertificateFile: record.certificateFile,
				KeyFile:         record.keyFile,
			}

			actualCertificateFile, actualKeyFile = health.Certificate()
		)

		assert.Equal(record.certificateFile, actualCertificateFile)
		assert.Equal(record.keyFile, actualKeyFile)
	}
}

func TestHealthNew(t *testing.T) {
	const (
		expectedName                      = "TestHealthNew"
		expectedLogInterval time.Duration = 45 * time.Second
	)

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedHandlerType *health.Health = nil

		testData = []struct {
			address            string
			logConnectionState bool
			options            []string
		}{
			{"", false, nil},
			{"", false, []string{}},
			{"", false, []string{"Value1"}},
			{"", false, []string{"Value1", "Value2"}},

			{"", true, nil},
			{"", true, []string{}},
			{"", true, []string{"Value1"}},
			{"", true, []string{"Value1", "Value2"}},

			{":901", false, nil},
			{":1987", false, []string{}},
			{":http", false, []string{"Value1"}},
			{":https", false, []string{"Value1", "Value2"}},

			{"locahost:9001", true, nil},
			{":57899", true, []string{}},
			{":ftp", true, []string{"Value1"}},
			{":0", true, []string{"Value1", "Value2"}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			verify, logger = newTestLogger()
			health         = Health{
				Name:               expectedName,
				Address:            record.address,
				LogConnectionState: record.logConnectionState,
				LogInterval:        expectedLogInterval,
				Options:            record.options,
			}

			handler, server = health.New(logger)
		)

		if len(record.address) > 0 {
			require.NotNil(handler)
			require.NotNil(server)
			assert.Equal(record.address, server.Addr)
			assert.IsType(expectedHandlerType, server.Handler)
			assertErrorLog(assert, verify, expectedName, server.ErrorLog)

			if record.logConnectionState {
				assertConnState(assert, verify, server.ConnState)
			} else {
				assert.Nil(server.ConnState)
			}
		} else {
			require.Nil(handler)
			require.Nil(server)
		}
	}
}

func TestWebPANoPrimaryAddress(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)
		webPA   = WebPA{}

		_, logger         = newTestLogger()
		monitor, runnable = webPA.Prepare(logger, handler)
	)

	assert.Nil(monitor)
	require.NotNil(runnable)

	var (
		waitGroup = new(sync.WaitGroup)
		shutdown  = make(chan struct{})
	)

	defer close(shutdown)
	assert.Equal(ErrorNoPrimaryAddress, runnable.Run(waitGroup, shutdown))
	waitGroup.Wait() // nothing should have incremented the wait group
	handler.AssertExpectations(t)
}

func TestWebPA(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		handler = new(mockHandler)

		// synthesize a WebPA instance that will start everything,
		// close to how it would be unmarshalled from Viper.
		webPA = WebPA{
			Primary: Basic{
				Name:    "test",
				Address: ":0",
			},
			Alternate: Basic{
				Name:    "test.alternate",
				Address: ":0",
			},
			Health: Health{
				Name:        "test.health",
				Address:     ":0",
				LogInterval: 60 * time.Minute,
				Options:     []string{"Option1", "Option2"},
			},
			Pprof: Basic{
				Name:    "test.pprof",
				Address: ":0",
			},
		}

		_, logger         = newTestLogger()
		monitor, runnable = webPA.Prepare(logger, handler)
	)

	assert.NotNil(monitor)
	require.NotNil(runnable)

	var (
		waitGroup = new(sync.WaitGroup)
		shutdown  = make(chan struct{})
	)

	assert.Nil(runnable.Run(waitGroup, shutdown))
	close(shutdown)
	waitGroup.Wait() // the http.Server instances will still be running after this returns
	handler.AssertExpectations(t)
}
