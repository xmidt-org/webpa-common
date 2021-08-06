package xhttp

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/logging"
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

func testloadconfig(certificatFiles, keyFiles []string) *tls.Config {
	certPem := []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)
	keyPem := []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)
	cert, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		panic(err)
	}
	cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
	cfg.BuildNameToCertificate()
	return cfg
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
			httpServer.On("ListenAndServe").Return(expectedError).Once()
			o.CertificateFile = []string{expectedCertificateFile}
			o.KeyFile = []string{expectedKeyFile}

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
			httpServer.On("Serve", listener).Return(expectedError).Once()
			o.Listener = listener
			o.CertificateFile = []string{expectedCertificateFile}
			o.KeyFile = []string{expectedKeyFile}

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

func TestServerOptions(t *testing.T) {
	var (
		assert   = assert.New(t)
		logger   = logging.NewTestLogger(nil, t)
		listener = new(mockListener)

		o = ServerOptions{
			Logger:            logger,
			Listener:          listener,
			DisableKeepAlives: true,
			CertificateFile:   []string{"cert.pem"},
			KeyFile:           []string{"key.pem"},
		}
	)

	so := o.StartOptions()
	assert.NotNil(so.Logger)
	assert.Equal(listener, so.Listener)
	assert.True(so.DisableKeepAlives)
	assert.Equal([]string{"cert.pem"}, so.CertificateFile)
	assert.Equal([]string{"key.pem"}, so.KeyFile)
	listener.AssertExpectations(t)
}

func TestNewServer(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		logger   = logging.NewTestLogger(nil, t)
		listener = new(mockListener)

		o = ServerOptions{
			Logger:            logger,
			Address:           "localhost:1234",
			ReadTimeout:       31 * time.Hour,
			ReadHeaderTimeout: 12356 * time.Second,
			WriteTimeout:      391 * time.Minute,
			IdleTimeout:       102 * time.Millisecond,
			MaxHeaderBytes:    48287231,
			Listener:          listener,
			DisableKeepAlives: true,
			CertificateFile:   []string{"cert.pem"},
			KeyFile:           []string{"key.pem"},
		}
	)

	s := NewServer(o)
	require.NotNil(s)

	assert.Equal("localhost:1234", s.Addr)
	assert.Equal(31*time.Hour, s.ReadTimeout)
	assert.Equal(12356*time.Second, s.ReadHeaderTimeout)
	assert.Equal(391*time.Minute, s.WriteTimeout)
	assert.Equal(102*time.Millisecond, s.IdleTimeout)
	assert.Equal(48287231, s.MaxHeaderBytes)
	assert.NotNil(s.ErrorLog)
	assert.NotNil(s.ConnState)
}
