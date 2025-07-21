// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
	"go.uber.org/zap"
)

const (
	expectedCertificateFile = "certificateFile"
	expectedKeyFile         = "keyFile"
)

// startOptions generates the various permutations of StartOptions that we test with.
// Each options struct can be further modified by tests.
func startOptions(t *testing.T) []StartOptions {
	var o []StartOptions

	for _, logger := range []*zap.Logger{nil, sallust.Default()} {
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

			// nolint: typecheck
			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			// nolint: typecheck
			httpServer.On("ListenAndServe").Return(expectedError).Once()

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			// nolint: typecheck
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

			// nolint: typecheck
			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			// nolint: typecheck
			httpServer.On("Serve", listener).Return(expectedError).Once()
			o.Listener = listener

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			// nolint: typecheck
			listener.AssertExpectations(t)
			// nolint: typecheck
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

			// nolint: typecheck
			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			// nolint: typecheck
			httpServer.On("ListenAndServe").Return(expectedError).Once()
			o.CertificateFile = []string{expectedCertificateFile}
			o.KeyFile = []string{expectedKeyFile}

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			// nolint: typecheck
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

			// nolint: typecheck
			httpServer.On("SetKeepAlivesEnabled", !o.DisableKeepAlives).Once()
			// nolint: typecheck
			httpServer.On("Serve", listener).Return(expectedError).Once()
			o.Listener = listener
			o.CertificateFile = []string{expectedCertificateFile}
			o.KeyFile = []string{expectedKeyFile}

			starter := NewStarter(o, httpServer)
			require.NotNil(starter)

			assert.NotPanics(func() {
				assert.Equal(expectedError, starter())
			})

			// nolint: typecheck
			listener.AssertExpectations(t)
			// nolint: typecheck
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
		logger   = sallust.Default()
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
	// nolint: typecheck
	listener.AssertExpectations(t)
}

func TestNewServer(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		logger   = sallust.Default()
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
