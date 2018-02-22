package xhttp

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testNewServerLogger(t *testing.T, logger log.Logger) {
	var (
		assert       = assert.New(t)
		require      = require.New(t)
		serverLogger = NewServerLogger(logger)
	)

	require.NotNil(serverLogger)
	assert.NotPanics(func() {
		serverLogger.Println("this is a message")
	})
}

func TestNewServerLogger(t *testing.T) {
	t.Run("NilLogger", func(t *testing.T) {
		testNewServerLogger(t, nil)
	})

	t.Run("CustomLogger", func(t *testing.T) {
		testNewServerLogger(t, log.With(logging.NewTestLogger(nil, t), ServerKey(), "test"))
	})
}

func testNewServerConnStateLogger(t *testing.T, logger log.Logger) {
	var (
		assert    = assert.New(t)
		require   = require.New(t)
		connState = NewServerConnStateLogger(logger)
	)

	require.NotNil(connState)
	assert.NotPanics(func() {
		connState(new(net.IPConn), http.StateNew)
	})
}

func TestNewServerConnStateLogger(t *testing.T) {
	t.Run("NilLogger", func(t *testing.T) {
		testNewServerConnStateLogger(t, nil)
	})

	t.Run("CustomLogger", func(t *testing.T) {
		testNewServerConnStateLogger(t, log.With(logging.NewTestLogger(nil, t), ServerKey(), "test"))
	})
}

const (
	expectedCertificateFile = "certificateFile"
	expectedKeyFile         = "keyFile"
)

// startOptions generates the various permutations of StartOptions that we test with.
// Each options struct can be further modified by tests.
func startOptions(t *testing.T) []StartOptions {
	var o []StartOptions

	for _, logger := range []log.Logger{nil, logging.NewTestLogger(nil, t)} {
		for _, disableKeepAlives := range []bool{false, true} {
			o = append(o, StartOptions{
				Logger:            logger,
				DisableKeepAlives: disableKeepAlives,
			})
		}
	}

	return o
}

func testNewStarterListenAndServe(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	for _, o := range startOptions(t) {
		t.Logf("StartOptions: %v", o)

		for _, expectedError := range []error{errors.New("expected"), http.ErrServerClosed} {
			httpServer := new(mockHTTPServer)

			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			httpServer.On("ListenAndServe").Return(expectedError).Once()

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			httpServer.AssertExpectations(t)
		}
	}
}

func testNewStarterServe(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	for _, o := range startOptions(t) {
		t.Logf("StartOptions: %v", o)

		for _, expectedError := range []error{errors.New("expected"), http.ErrServerClosed} {
			var (
				listener   = new(mockListener)
				httpServer = new(mockHTTPServer)
			)

			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			httpServer.On("Serve", listener).Return(expectedError).Once()
			o.Listener = listener

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			listener.AssertExpectations(t)
			httpServer.AssertExpectations(t)
		}
	}
}

func testNewStarterListenAndServeTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	for _, o := range startOptions(t) {
		t.Logf("StartOptions: %v", o)

		for _, expectedError := range []error{errors.New("expected"), http.ErrServerClosed} {
			httpServer := new(mockHTTPServer)

			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			httpServer.On("ListenAndServeTLS", expectedCertificateFile, expectedKeyFile).Return(expectedError).Once()
			o.CertificateFile = expectedCertificateFile
			o.KeyFile = expectedKeyFile

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			httpServer.AssertExpectations(t)
		}
	}
}

func testNewStarterServeTLS(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	for _, o := range startOptions(t) {
		t.Logf("StartOptions: %v", o)

		for _, expectedError := range []error{errors.New("expected"), http.ErrServerClosed} {
			var (
				listener   = new(mockListener)
				httpServer = new(mockHTTPServer)
			)

			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			httpServer.On("ServeTLS", listener, expectedCertificateFile, expectedKeyFile).Return(expectedError).Once()
			o.Listener = listener
			o.CertificateFile = expectedCertificateFile
			o.KeyFile = expectedKeyFile

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			listener.AssertExpectations(t)
			httpServer.AssertExpectations(t)
		}
	}
}

func TestNewStarter(t *testing.T) {
	t.Run("ListenAndServe", testNewStarterListenAndServe)
	t.Run("Serve", testNewStarterServe)
	t.Run("ListenAndServeTLS", testNewStarterListenAndServeTLS)
	t.Run("ServeTLS", testNewStarterServeTLS)
}

func testServerLoggingCustom(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
		server http.Server
	)

	ServerLogging(logger, v, &server)
	require.NotNil(server.ErrorLog)
	require.NotNil(server.ConnState)

	assert.NotPanics(func() {
		server.ErrorLog.Println("test")
	})

	i, o := net.Pipe()
	defer i.Close()
	defer o.Close()

	assert.NotPanics(func() {
		server.ConnState(i, http.StateNew)
	})
}

func testServerLoggingDefault(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		v      = viper.New()
		server http.Server
	)

	ServerLogging(nil, v, &server)
	require.NotNil(server.ErrorLog)
	require.NotNil(server.ConnState)

	assert.NotPanics(func() {
		server.ErrorLog.Println("test")
	})

	i, o := net.Pipe()
	defer i.Close()
	defer o.Close()

	assert.NotPanics(func() {
		server.ConnState(i, http.StateNew)
	})
}

func TestServerLogging(t *testing.T) {
	t.Run("Custom", testServerLoggingCustom)
	t.Run("Default", testServerLoggingDefault)
}

func testUnmarshalServerBasic(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s"
		}
	`))

	require.NoError(err)

	server, err := UnmarshalServer(logger, v)
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(":8080", server.Addr)
	assert.Equal(30*time.Second, server.ReadTimeout)
}

func testUnmarshalServerUnmarshalError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "this is not a valid duration"
		}
	`))

	require.NoError(err)

	server, err := UnmarshalServer(logger, v)
	assert.Error(err)
	assert.Nil(server)
}

func testUnmarshalServerGoodOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()

		optionCounter = 0
		option        = func(actualLogger log.Logger, actualViper *viper.Viper, server *http.Server) error {
			optionCounter++
			assert.Equal(logger, actualLogger)
			assert.True(v == actualViper)
			assert.NotNil(server)
			return nil
		}
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s"
		}
	`))

	require.NoError(err)

	server, err := UnmarshalServer(logger, v, option, option)
	require.NoError(err)
	require.NotNil(server)

	assert.Equal(":8080", server.Addr)
	assert.Equal(30*time.Second, server.ReadTimeout)
	assert.Equal(2, optionCounter)
}

func testUnmarshalServerBadOption(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger        = logging.NewTestLogger(nil, t)
		v             = viper.New()
		expectedError = errors.New("expected")

		firstCalled = false
		first       = func(actualLogger log.Logger, actualViper *viper.Viper, server *http.Server) error {
			firstCalled = true
			assert.Equal(logger, actualLogger)
			assert.True(v == actualViper)
			assert.NotNil(server)
			return nil
		}

		secondCalled = false
		second       = func(actualLogger log.Logger, actualViper *viper.Viper, server *http.Server) error {
			secondCalled = true
			assert.Equal(logger, actualLogger)
			assert.True(v == actualViper)
			assert.NotNil(server)
			return expectedError
		}

		third = func(actualLogger log.Logger, actualViper *viper.Viper, server *http.Server) error {
			assert.Fail("The third option should not have been called")
			return nil
		}
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s"
		}
	`))

	require.NoError(err)

	server, err := UnmarshalServer(logger, v, first, second, third)
	assert.Nil(server)
	assert.Equal(expectedError, err)
	assert.True(firstCalled)
	assert.True(secondCalled)
}

func TestUnmarshalServer(t *testing.T) {
	t.Run("Basic", testUnmarshalServerBasic)
	t.Run("UnmarshalError", testUnmarshalServerUnmarshalError)
	t.Run("GoodOptions", testUnmarshalServerGoodOptions)
	t.Run("BadOption", testUnmarshalServerBadOption)
}

func testNewServerUnmarshalServerError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "this is not a valid duration"
		}
	`))

	require.NoError(err)

	server, starter, err := NewServer(logger, v)
	assert.Error(err)
	assert.Nil(server)
	assert.Nil(starter)
}

func testNewServerUnmarshalListenerError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s",
			"maxConnections": "this is not a valid integer"
		}
	`))

	require.NoError(err)

	server, starter, err := NewServer(logger, v)
	assert.Error(err)
	assert.Nil(server)
	assert.Nil(starter)
}

func testNewServerBadListenerConfiguration(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s",
			"maxConnections": 30,
			"network": "this can't be a valid network"
		}
	`))

	require.NoError(err)

	server, starter, err := NewServer(logger, v)
	assert.Error(err)
	assert.Nil(server)
	assert.Nil(starter)
}

func testNewServerUnmarshalStartOptionsError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s",
			"maxConnections": 100,
			"disableKeepAlives": "this is not a valid bool"
		}
	`))

	require.NoError(err)

	server, starter, err := NewServer(logger, v)
	assert.Error(err)
	assert.Nil(server)
	assert.Nil(starter)
}

func testNewServerSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger = logging.NewTestLogger(nil, t)
		v      = viper.New()
	)

	v.SetConfigType("json")
	err := v.ReadConfig(strings.NewReader(`
		{
			"address": ":8080",
			"readTimeout": "30s",
			"maxConnections": 100,
			"certificateFile": "cert",
			"keyFile": "key"
		}
	`))

	require.NoError(err)

	server, starter, err := NewServer(logger, v)
	require.NotNil(server)
	defer server.Close() // necessary since the listener was started by NewServer

	assert.NotNil(starter)
	assert.NoError(err)
}

func TestNewServer(t *testing.T) {
	t.Run("UnmarshalServerError", testNewServerUnmarshalServerError)
	t.Run("UnmarshalListenerError", testNewServerUnmarshalListenerError)
	t.Run("BadListenerConfiguration", testNewServerBadListenerConfiguration)
	t.Run("UnmarshalStartOptionsError", testNewServerUnmarshalStartOptionsError)
	t.Run("Success", testNewServerSuccess)
}
