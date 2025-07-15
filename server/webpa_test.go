// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"bytes"
	"crypto/tls"
	"errors"
	"log"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/justinas/alice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
	"go.uber.org/zap/zapcore"
)

func assertBufferContains(assert *assert.Assertions, verify *bytes.Buffer, values ...string) {
	text := verify.String()
	for _, value := range values {
		assert.Contains(text, value)
	}
}

func assertErrorLog(assert *assert.Assertions, verify *bytes.Buffer, serverName string, errorLog *log.Logger) {
	if assert.NotNil(errorLog) {
		errorLog.Print("howdy!")
		assertBufferContains(assert, verify, serverName, "howdy!")
	}
}

func assertConnState(assert *assert.Assertions, verify *bytes.Buffer, connState func(net.Conn, http.ConnState)) {
	if assert.NotNil(connState) {
		conn1, conn2 := net.Pipe()
		defer conn1.Close()
		defer conn2.Close()

		assert.NotPanics(func() {
			connState(conn1, http.StateNew)
		})
		if verify != nil {
			assertBufferContains(assert, verify, conn1.LocalAddr().String(), http.StateNew.String())
		}
	}
}

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

			_, logger      = sallust.NewTestLogger(zapcore.InfoLevel)
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

			_, logger      = sallust.NewTestLogger(zapcore.InfoLevel)
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
			description        string
			address            string
			handler            *mockHandler
			certFile           []string
			keyFile            []string
			clientCACertFile   string
			minTLSVersion      uint16
			maxTLSVersion      uint16
			logConnectionState bool
			expectTLS          bool
			expectmTLS         bool
			nilServer          bool
		}{
			{
				description:        "No address",
				address:            "",
				handler:            nil,
				logConnectionState: false,
				nilServer:          true,
			},
			{
				description:        "Nil handler",
				address:            ":443",
				handler:            nil,
				logConnectionState: true,
			},

			{
				description:        "Invalid cert file",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem", "missing-pair.pem"},
				keyFile:            []string{"key.pem"},
				nilServer:          true,
			},

			{
				description:        "Invalid key file",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem"},
				keyFile:            []string{"key.pem", "missing-pair.pem"},
				nilServer:          true,
			},

			{
				description:        "Invalid client CA cert file",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem"},
				keyFile:            []string{"key.pem"},
				clientCACertFile:   "missing-file.pem",
				nilServer:          true,
			},

			{
				description:        "Invalid client CA cert file",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem"},
				keyFile:            []string{"key.pem"},
				clientCACertFile:   "missing-file.pem",
				nilServer:          true,
			},

			{
				description:        "TLS enabled",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem"},
				keyFile:            []string{"key.pem"},
				minTLSVersion:      tls.VersionTLS11,
				maxTLSVersion:      tls.VersionTLS12,
				expectTLS:          true,
			},

			{
				description:        "mTLS enabled",
				address:            ":443",
				handler:            new(mockHandler),
				logConnectionState: true,
				certFile:           []string{"cert.pem"},
				keyFile:            []string{"key.pem"},
				clientCACertFile:   "client_ca.pem",
				minTLSVersion:      tls.VersionTLS12,
				maxTLSVersion:      tls.VersionTLS13,
				expectTLS:          true,
				expectmTLS:         true,
			},
		}
	)

	for _, record := range testData {
		t.Run(record.description, func(t *testing.T) {
			var (
				verify, logger = sallust.NewTestLogger(zapcore.DebugLevel)
				basic          = Basic{
					Name:               expectedName,
					Address:            record.address,
					LogConnectionState: record.logConnectionState,
					CertificateFile:    record.certFile,
					KeyFile:            record.keyFile,
					ClientCACertFile:   record.clientCACertFile,
					MaxVersion:         record.maxTLSVersion,
					MinVersion:         record.minTLSVersion,
					DisableKeepAlives:  true,
				}
			)

			server := basic.New(logger, record.handler)

			if !record.nilServer {
				require.NotNil(server)
				assert.Equal(record.address, server.Addr)
				assert.Equal(record.handler, server.Handler)
				assertErrorLog(assert, verify, expectedName, server.ErrorLog)

				if record.logConnectionState {
					assertConnState(assert, verify, server.ConnState)
				} else {
					assert.Nil(server.ConnState)
				}

				if record.expectTLS {
					assert.NotZero(server.TLSConfig.MaxVersion)
					assert.Equal(record.minTLSVersion, server.TLSConfig.MinVersion)
					assert.Equal(record.maxTLSVersion, server.TLSConfig.MaxVersion)
					assert.NotNil(server.TLSConfig.Certificates)
					if record.expectmTLS {
						assert.NotNil(server.TLSConfig.ClientCAs)
						assert.Equal(tls.RequireAndVerifyClientCert, server.TLSConfig.ClientAuth)
					}
				} else {
					assert.Nil(server.TLSConfig)
				}
			} else {
				require.Nil(server)
			}

			if record.handler != nil {
				record.handler.AssertExpectations(t)
			}
		})
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
			verify, logger = sallust.NewTestLogger(zapcore.DebugLevel)
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

		_, logger               = sallust.NewTestLogger(zapcore.InfoLevel)
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

		_, logger               = sallust.NewTestLogger(zapcore.InfoLevel)
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
