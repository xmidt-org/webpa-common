// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"net"
	"net/http"
	"time"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/sallust/sallusthttp"
	"go.uber.org/zap"
)

var (
	serverKey = "server"
)

// ServerKey returns the contextual logging key for the server name
func ServerKey() string {
	return serverKey
}

// StartOptions represents the subset of server options that have to do with how
// an HTTP server is started.
type StartOptions struct {
	// Logger is the go-kit Logger to use for server startup and error logging.  If not
	// supplied, sallust.Default() is used instead.
	Logger *zap.Logger `json:"-"`

	// Listener is the optional net.Listener to use.  If not supplied, the http.Server default
	// listener is used.
	Listener net.Listener `json:"-"`

	// DisableKeepAlives indicates whether the server should honor keep alives
	DisableKeepAlives bool `json:"disableKeepAlives,omitempty"`

	// CertificateFile is the HTTPS certificate file(s).  If both this field and KeyFile are set,
	// an HTTPS starter function is created.
	CertificateFile []string `json:"certificateFile,omitempty"`

	// KeyFile is the HTTPS key file(s).  If both this field and CertificateFile are set,
	// an HTTPS starter function is created.
	KeyFile []string `json:"keyFile,omitempty"`
}

// NewStarter returns a starter closure for the given HTTP server.  The start options are first
// applied to the server instance, and the server instance must not have already been started prior
// to invoking this method.
//
// The returned closure will invoke the correct method on the server to start it, e.g. Serve, ServeTLS, etc.
// The selection of which server method is based on the options.  For example, if CertificateFile and KeyFile
// are set, either of the xxxTLS methods will be invoked based on whether there is a Listener configured.
//
// Note: tlsConfig is expected to already be set
func NewStarter(o StartOptions, s httpServer) func() error {
	if o.Logger == nil {
		o.Logger = sallust.Default()
	}

	s.SetKeepAlivesEnabled(!o.DisableKeepAlives)

	var starter func() error

	// nolint: typecheck
	if o.Listener != nil {
		starter = func() error {
			return s.Serve(o.Listener)
		}
	} else {
		starter = func() error {
			return s.ListenAndServe()
		}
	}

	return func() error {
		o.Logger.Info("starting server")
		err := starter()
		if err == http.ErrServerClosed {
			o.Logger.Error("server closed", zap.Error(http.ErrServerClosed))
		} else {
			o.Logger.Error("server exited", zap.Error(err))
		}

		return err
	}
}

// httpServer exposes the set of methods expected of an http.Server by this package.
type httpServer interface {
	ListenAndServe() error
	ListenAndServeTLS(string, string) error

	Serve(net.Listener) error
	ServeTLS(net.Listener, string, string) error

	SetKeepAlivesEnabled(bool)
}

// ServerOptions describes the superset of options for both construction an http.Server and
// starting it.
type ServerOptions struct {
	// Logger is the go-kit Logger to use for server startup and error logging.  If not
	// supplied, sallust.Default() is used instead.
	Logger *zap.Logger `json:"-"`

	// Address is the bind address of the server.  If not supplied, defaults to the internal net/http default.
	Address string `json:"address,omitempty"`

	// ReadTimeout is the maximum duration for reading the entire request.  If not supplied, defaults to the
	// internal net/http default.
	ReadTimeout time.Duration `json:"readTimeout,omitempty"`

	// ReadHeaderTimeout is the amount of time allowed to read request headers.  If not supplied, defaults to
	// the internal net/http default.
	ReadHeaderTimeout time.Duration `json:"readHeaderTimeout,omitempty"`

	// WriteTimeout is the maximum duration before timing out writes of the response.  If not supplied, defaults
	// to the internal net/http default.
	WriteTimeout time.Duration `json:"writeTimeout,omitempty"`

	// IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled.
	// If not supplied, defaults to the internal net/http default.
	IdleTimeout time.Duration `json:"idleTimeout,omitempty"`

	// MaxHeaderBytes controls the maximum number of bytes the server will read parsing the request header's
	// keys and values.  If not supplied, defaults to the internal net/http default.
	MaxHeaderBytes int `json:"maxHeaderBytes,omitempty"`

	// Listener is the optional net.Listener to use.  If not supplied, the http.Server default
	// listener is used.
	Listener net.Listener `json:"-"`

	// DisableKeepAlives indicates whether the server should honor keep alives
	DisableKeepAlives bool `json:"disableKeepAlives,omitempty"`

	// CertificateFile is the HTTPS certificate file.  If both this field and KeyFile are set,
	// an HTTPS starter function is created.
	CertificateFile []string `json:"certificateFile,omitempty"`

	// KeyFile is the HTTPS key file.  If both this field and CertificateFile are set,
	// an HTTPS starter function is created.
	KeyFile []string `json:"keyFile,omitempty"`
}

// StartOptions produces a StartOptions with the corresponding values from this ServerOptions
func (so *ServerOptions) StartOptions() StartOptions {
	logger := so.Logger
	if logger == nil {
		logger = sallust.Default()
	}

	return StartOptions{
		Logger: logger.With(
			zap.String("address", so.Address),
		),
		Listener:          so.Listener,
		DisableKeepAlives: so.DisableKeepAlives,
		CertificateFile:   so.CertificateFile,
		KeyFile:           so.KeyFile,
	}
}

// NewServer creates a Server from a supplied set of options.
func NewServer(o ServerOptions) *http.Server {
	return &http.Server{
		Addr:              o.Address,
		ReadTimeout:       o.ReadTimeout,
		ReadHeaderTimeout: o.ReadHeaderTimeout,
		WriteTimeout:      o.WriteTimeout,
		IdleTimeout:       o.IdleTimeout,
		MaxHeaderBytes:    o.MaxHeaderBytes,
		ErrorLog:          sallust.NewServerLogger("xhttp.Server", o.Logger),
		ConnState:         sallusthttp.NewConnStateLogger(o.Logger, zap.DebugLevel),
	}
}
