package server

import (
	"errors"
	//	"github.com/Comcast/webpa-common/health"
	"crypto/tls"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestListenAndServeNonSecure(t *testing.T) {
	var (
		simpleError = errors.New("expected")
		testData    = []struct {
			certificateFile, keyFile string
			expectedError            error
			shouldCallFinal          bool
		}{
			{"", "", http.ErrServerClosed, true},
			{"", "", simpleError, false},
			{"file.cert", "", http.ErrServerClosed, true},
			{"file.cert", "", simpleError, false},
			{"", "file.key", http.ErrServerClosed, true},
			{"", "file.key", simpleError, false},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			assert = assert.New(t)

			_, logger      = newTestLogger()
			executorCalled = make(chan struct{}, 1)
			mockExecutor   = new(mockExecutor)

			finalizerCalled = make(chan struct{})
			finalizer       = func() {
				close(finalizerCalled)
			}
		)

		mockExecutor.On("ListenAndServe").
			Return(record.expectedError).
			Run(func(mock.Arguments) { executorCalled <- struct{}{} })

		ListenAndServe(logger, mockExecutor, finalizer)
		select {
		case <-executorCalled:
			// passing
		case <-time.After(time.Second):
			assert.Fail("the executor was not called")
		}

		select {
		case <-finalizerCalled:
			// passing
		case <-time.After(time.Second):
			if record.shouldCallFinal {
				assert.Fail("the finalizer was not called")
			}
		}

		mockExecutor.AssertExpectations(t)
	}
}

func TestListenAndServeSecure(t *testing.T) {
	var (
		testData = []struct {
			expectedError   error
			shouldCallFinal bool
		}{
			{http.ErrServerClosed, true},
			{errors.New("expected"), false},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		var (
			assert = assert.New(t)

			_, logger      = newTestLogger()
			executorCalled = make(chan struct{}, 1)
			mockExecutor   = new(mockExecutor)

			finalizerCalled = make(chan struct{})
			finalizer       = func() {
				close(finalizerCalled)
			}
		)

		mockExecutor.On("ListenAndServe").
			Return(record.expectedError).
			Run(func(mock.Arguments) { executorCalled <- struct{}{} })

		ListenAndServe(logger, mockExecutor, finalizer)
		select {
		case <-executorCalled:
			// passing
		case <-time.After(time.Second):
			assert.Fail("the executor was not called")
		}

		select {
		case <-finalizerCalled:
			// passing
		case <-time.After(time.Second):
			if record.shouldCallFinal {
				assert.Fail("the finalizer was not called")
			}
		}

		mockExecutor.AssertExpectations(t)
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

func TestHealthNew(t *testing.T) {
	const (
		expectedName                      = "TestHealthNew"
		expectedLogInterval time.Duration = 45 * time.Second
	)

	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedHandlerType *http.ServeMux = nil

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

			handler, server = health.New(logger, alice.New(), nil)
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
	)

	r, err := xmetrics.NewRegistry(nil, Metrics)
	require.NoError(err)
	require.NotNil(r)

	var (
		handler = new(mockHandler)
		webPA   = WebPA{}

		_, logger               = newTestLogger()
		monitor, runnable, done = webPA.Prepare(logger, nil, xmetrics.MustNewRegistry(nil), handler)
	)

	assert.Nil(monitor)
	require.NotNil(runnable)
	assert.NotNil(done)

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
	)

	r, err := xmetrics.NewRegistry(nil, Metrics)
	require.NoError(err)
	require.NotNil(r)

	var (
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

			Metric: Metric{
				Name:    "test.metrics",
				Address: ":0",
			},
		}

		_, logger               = newTestLogger()
		monitor, runnable, done = webPA.Prepare(logger, nil, xmetrics.MustNewRegistry(nil), handler)
	)

	assert.NotNil(monitor)
	require.NotNil(runnable)
	assert.NotNil(done)

	var (
		waitGroup = new(sync.WaitGroup)
		shutdown  = make(chan struct{})
	)

	assert.Nil(runnable.Run(waitGroup, shutdown))
	close(shutdown)
	waitGroup.Wait() // the http.Server instances will still be running after this returns
	handler.AssertExpectations(t)
}

func TestBasicNewWithClientCACert(t *testing.T) {
	const expectedName = "TestBasicNewClientCA"

	var (
		assert   = assert.New(t)
		require  = require.New(t)
		testData = []struct {
			address            string
			certificateFiles   []string
			keyFiles           []string
			clientCACertFile   string
			handler            *mockHandler
			logConnectionState bool
		}{
			{"localhost:80", []string{"cert.pem"}, []string{"cert.key"}, "client_ca.pem", new(mockHandler), true},
			{"localhost:8090", []string{"file.cert"}, []string{"file.key"}, "invalid.cert", new(mockHandler), true},
			{"localhost:8090", []string{"file.cert"}, []string{"file.key"}, "", new(mockHandler), true},
			{":8081", []string{}, []string{}, "", new(mockHandler), false},
		}
	)

	for index, record := range testData {
		t.Logf("%#v", record)

		var (
			verify, logger = newTestLogger()
			basic          = Basic{
				Name:               expectedName,
				Address:            record.address,
				CertificateFile:    record.certificateFiles,
				KeyFile:            record.keyFiles,
				ClientCACertFile:   record.clientCACertFile,
				LogConnectionState: record.logConnectionState,
			}

			server = basic.New(logger, record.handler)
		)

		t.Logf("%#v", server)

		require.NotNil(server)
		assert.Equal(record.address, server.Addr)
		assert.Equal(record.handler, server.Handler)
		assertErrorLog(assert, verify, expectedName, server.ErrorLog)

		switch index {
		case 0:
			tlsConfig := server.TLSConfig
			require.NotNil(tlsConfig)
			assert.Equal(tls.RequireAndVerifyClientCert, tlsConfig.ClientAuth)
			require.NotNil(tlsConfig.ClientCAs)
		case 1:
			require.Nil(server.TLSConfig)
		case 2:
			require.Nil(server.TLSConfig)
		case 3:
			require.Nil(server.TLSConfig)
		}

		if record.handler != nil {
			record.handler.AssertExpectations(t)
		}
	}
}
