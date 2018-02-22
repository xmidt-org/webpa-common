package xhttp

import (
	stdlog "log"
	"net"
	"net/http"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xlistener"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/spf13/viper"
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
func NewServerLogger(logger log.Logger) *stdlog.Logger {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return stdlog.New(
		log.NewStdlibAdapter(logger),
		"", // having a prefix gives the adapter trouble
		stdlog.LstdFlags|stdlog.LUTC,
	)
}

// NewServerConnStateLogger adapts a go-kit Logger onto a connection state handler appropriate
// for http.Server.ConnState.
func NewServerConnStateLogger(logger log.Logger) func(net.Conn, http.ConnState) {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	return func(c net.Conn, cs http.ConnState) {
		logger.Log(
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
	// Logger is the go-kit Logger to use for server startup and error logging.  If not
	// supplied, logging.DefaultLogger() is used instead.
	Logger log.Logger

	// Listener is the optional net.Listener to use.  If not supplied, the http.Server default
	// listener is used.
	Listener net.Listener

	// DisableKeepAlives indicates whether the server should honor keep alives
	DisableKeepAlives bool

	// CertificateFile is the HTTPS certificate file.  If both this field and KeyFile are set,
	// an HTTPS starter function is created.
	CertificateFile string

	// KeyFile is the HTTPS key file.  If both this field and CertificateFile are set,
	// an HTTPS starter function is created.
	KeyFile string
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

// ServerOption represents an optional configuration applied to a server after unmarshalling from Viper.
type ServerOption func(log.Logger, *viper.Viper, *http.Server) error

// ServerLogging is a ServerOption that sets the ErrorLog and ConnState to objects that will log appropriate messages.
func ServerLogging(logger log.Logger, _ *viper.Viper, s *http.Server) error {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	s.ErrorLog = NewServerLogger(logger)
	s.ConnState = NewServerConnStateLogger(logger)
	return nil
}

// UnmarshalServer unmarshals an http.Server instance from a Viper environment.  An optional
// set of ServerOptions can be supplied to provide post-processing on the http.Server instance.
func UnmarshalServer(logger log.Logger, v *viper.Viper, o ...ServerOption) (*http.Server, error) {
	v.RegisterAlias("address", "addr")
	s := new(http.Server)
	if err := v.Unmarshal(s); err != nil {
		return nil, err
	}

	for _, f := range o {
		if err := f(logger, v, s); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// NewServer handles extracting a complete http.Server and starter closure from a Viper environment.
// The server itself, a xlistener.Options, and a StartOptions are all extracted from the
// supplied Viper instance to fully create the http.Server.
func NewServer(logger log.Logger, v *viper.Viper, o ...ServerOption) (*http.Server, func() error, error) {
	server, err := UnmarshalServer(logger, v, o...)
	if err != nil {
		return nil, nil, err
	}

	lo := xlistener.Options{
		Logger: logger,
	}

	if err := v.Unmarshal(&lo); err != nil {
		return nil, nil, err
	}

	listener, err := xlistener.New(lo)
	if err != nil {
		return nil, nil, err
	}

	so := StartOptions{
		Logger:   logger,
		Listener: listener,
	}

	if err := v.Unmarshal(&so); err != nil {
		// The listener has already been started
		listener.Close()
		return nil, nil, err
	}

	return server, NewStarter(so, server), nil
}
