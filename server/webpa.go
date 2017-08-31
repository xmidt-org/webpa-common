package server

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/health"
	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

var (
	// ErrorNoPrimaryAddress is the error returned when no primary address is specified in a WebPA instance
	ErrorNoPrimaryAddress = errors.New("No primary address configured")
)

// executor is an internal type used to start an HTTP server.  *http.Server implements
// this interface.  It can be mocked for testing.
type executor interface {
	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// Secure exposes the optional certificate information to be used when starting an HTTP server.
type Secure interface {
	// Certificate returns the certificate information associated with this Secure instance.
	// BOTH the returned file paths must be non-empty if a TLS server is desired.
	Certificate() (certificateFile, keyFile string)
}

// ListenAndServe invokes the appropriate server method based on the secure information.
// If Secure.Certificate() returns both a certificateFile and a keyFile, e.ListenAndServeTLS()
// is called to start the server.  Otherwise, e.ListenAndServe() is used.
func ListenAndServe(logger log.Logger, s Secure, e executor) {
	certificateFile, keyFile := s.Certificate()
	if len(certificateFile) > 0 && len(keyFile) > 0 {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.ListenAndServeTLS(certificateFile, keyFile),
			)
		}()
	} else {
		go func() {
			logging.Error(logger).Log(
				logging.ErrorKey(), e.ListenAndServe(),
			)
		}()
	}
}

// Basic describes a simple HTTP server.  Typically, this struct has its values
// injected via Viper.  See the New function in this package.
type Basic struct {
	Name               string
	Address            string
	CertificateFile    string
	KeyFile            string
	LogConnectionState bool
}

func (b *Basic) Certificate() (certificateFile, keyFile string) {
	return b.CertificateFile, b.KeyFile
}

// New creates an http.Server using this instance's configuration.  The given logger is required,
// but the handler may be nil.  If the handler is nil, http.DefaultServeMux is used, which matches
// the behavior of http.Server.
//
// This method returns nil if the configured address is empty, effectively disabling
// this server from startup.
func (b *Basic) New(logger log.Logger, handler http.Handler) *http.Server {
	if len(b.Address) == 0 {
		return nil
	}

	server := &http.Server{
		Addr:     b.Address,
		Handler:  handler,
		ErrorLog: NewErrorLog(b.Name, logger),
	}

	if b.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(b.Name, logger)
	}

	return server
}

// Health represents a configurable factory for a Health server.  As with the Basic type,
// if the Address is not specified, health is considered to be disabled.
//
// Due to a limitation of Viper, this struct does not use an embedded Basic
// instance.  Rather, it duplicates the fields so that Viper can inject them.
type Health struct {
	Name               string
	Address            string
	CertificateFile    string
	KeyFile            string
	LogConnectionState bool
	LogInterval        time.Duration
	Options            []string
}

func (h *Health) Certificate() (certificateFile, keyFile string) {
	return h.CertificateFile, h.KeyFile
}

// NewHealth creates a Health instance from this instance's configuration.  If the Address
// field is not supplied, this method returns nil.
func (h *Health) NewHealth(logger log.Logger, options ...health.Option) *health.Health {
	if len(h.Address) == 0 {
		return nil
	}

	for _, value := range h.Options {
		options = append(options, health.Stat(value))
	}

	return health.New(
		h.LogInterval,
		logger,
		options...,
	)
}

// New creates an HTTP server instance for serving health statistics.  If the health parameter
// is nil, then h.NewHealth is used to create a Health instance.  Otherwise, the health parameter
// is returned as is.
//
// If the Address option is not supplied, the health module is considered to be disabled.  In that
// case, this method simply returns the health parameter as the monitor and a nil server instance.
func (h *Health) New(logger log.Logger, health *health.Health) (*health.Health, *http.Server) {
	if len(h.Address) == 0 {
		// health is disabled
		return nil, nil
	}

	if health == nil {
		if health = h.NewHealth(logger); health == nil {
			// should never hit this case, since NewHealth performs the same
			// Address field check as this method.  but, just to be safe ...
			return nil, nil
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/health", health)

	server := &http.Server{
		Addr:     h.Address,
		Handler:  mux,
		ErrorLog: NewErrorLog(h.Name, logger),
	}

	if h.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(h.Name, logger)
	}

	return health, server
}

// WebPA represents a server component within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type WebPA struct {
	// Primary is the main server for this application, e.g. petasos.
	Primary Basic

	// Alternate is an alternate server which serves the primary application logic.
	// Used to have the same API served on more than one port and possibly more than
	// one protocol, e.g. HTTP and HTTPS.
	Alternate Basic

	// Health describes the health server for this application.  Note that if the Address
	// is empty, no health server is started.
	Health Health

	// Pprof describes the pprof server for this application.  Note that if the Address
	// is empty, no pprof server is started.
	Pprof Basic

	// Log is the logging configuration for this application.
	Log *logging.Options
}

// Prepare gets a WebPA server ready for execution.  This method does not return errors, but the returned
// Runnable may return an error.  The supplied logger will usually come from the New function, but the
// WebPA.Log object can be used to create a different logger if desired.
//
// The caller may pass an arbitrary Health instance.  If this parameter is nil, this method will attempt to
// create one using Health.NewHealth.  In either case, if Health.Address is not supplied, no health server
// will be instantiated.
//
// The supplied http.Handler is used for the primary server.  If the alternate server has an address,
// it will also be used for that server.  The health server uses an internally create handler, while the pprof
// server uses http.DefaultServeMux.  The health Monitor created from configuration is returned so that other
// infrastructure can make use of it.
func (w *WebPA) Prepare(logger log.Logger, health *health.Health, primaryHandler http.Handler) (health.Monitor, concurrent.Runnable) {
	// allow the health instance to be non-nil, in which case it will be used in favor of
	// the WebPA-configured instance.
	healthHandler, healthServer := w.Health.New(logger, health)
	infoLog := logging.Info(logger)

	return healthHandler, concurrent.RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		if healthHandler != nil && healthServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Health.Name, "address", w.Health.Address)
			ListenAndServe(logger, &w.Health, healthServer)
			healthHandler.Run(waitGroup, shutdown)

			// wrap the primary handler in the RequestTracker decorator
			primaryHandler = healthHandler.RequestTracker(primaryHandler)
		}

		if pprofServer := w.Pprof.New(logger, nil); pprofServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Pprof.Name, "address", w.Pprof.Address)
			ListenAndServe(logger, &w.Pprof, pprofServer)
		}

		if primaryServer := w.Primary.New(logger, primaryHandler); primaryServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Primary.Name, "address", w.Primary.Address)
			ListenAndServe(logger, &w.Primary, primaryServer)
		} else {
			return ErrorNoPrimaryAddress
		}

		if alternateServer := w.Alternate.New(logger, primaryHandler); alternateServer != nil {
			infoLog.Log(logging.MessageKey(), "starting server", "name", w.Alternate.Name, "address", w.Alternate.Address)
			ListenAndServe(logger, &w.Alternate, alternateServer)
		}

		return nil
	})
}
