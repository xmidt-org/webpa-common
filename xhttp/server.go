package xhttp

import (
	stdlog "log"
	"net"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	serverKey interface{} = "server"
)

// ServerKey returns the contextual logging key for the server name
func ServerKey() interface{} {
	return serverKey
}

// NewServerLogger adapts a go-kit Logger onto a golang Logger in a way that is appropriate
// for http.Server.ErrorLog.
func NewServerLogger(logger log.Logger, serverName string) *stdlog.Logger {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return stdlog.New(
		log.NewStdlibAdapter(log.With(logger, ServerKey(), serverName)),
		"", // having a prefix gives the adapter trouble
		stdlog.LstdFlags|stdlog.LUTC,
	)
}

// NewServerConnStateLogger adapts a go-kit Logger onto a connection state handler appropriate
// for http.Server.ConnState.
func NewServerConnStateLogger(logger log.Logger, serverName string) func(net.Conn, http.ConnState) {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return func(c net.Conn, cs http.ConnState) {
		logger.Log(
			ServerKey(), serverName,
			"remoteAddress", c.RemoteAddr(),
			"state", cs,
		)
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

// StartOptions describes a set of startup options to apply to http.Server instances
type StartOptions struct {
	Logger   log.Logger
	Listener net.Listener

	DisableKeepAlives bool
	CertificateFile   string
	KeyFile           string
}

// NewStarter returns a starter closure for the given HTTP server.  The start options are first
// applied to the server instance, and the server instance must not have already been started prior
// to invoking this method.
//
// The returned closure will invoke the correct method on the server to start it, e.g. Serve, ServeTLS, etc.
// The selection of which server method is based on the options.  For example, if CertificateFile and KeyFile
// are set, either of the xxxTLS methods will be invoked based on whether there is a Listener configured.
func NewStarter(o StartOptions, s httpServer) func() error {
	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	s.SetKeepAlivesEnabled(!o.DisableKeepAlives)

	var starter func() error
	if len(o.CertificateFile) > 0 && len(o.KeyFile) > 0 {
		if o.Listener != nil {
			starter = func() error {
				return s.ServeTLS(o.Listener, o.CertificateFile, o.KeyFile)
			}
		} else {
			starter = func() error {
				return s.ListenAndServeTLS(o.CertificateFile, o.KeyFile)
			}
		}
	} else {
		if o.Listener != nil {
			starter = func() error {
				return s.Serve(o.Listener)
			}
		} else {
			starter = func() error {
				return s.ListenAndServe()
			}
		}
	}

	return func() error {
		o.Logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "starting server")
		err := starter()
		if err == http.ErrServerClosed {
			o.Logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "server closed")
		} else {
			o.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "server exited", logging.ErrorKey(), err)
		}

		return err
	}
}
