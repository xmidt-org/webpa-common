package server

import (
	"errors"
	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/health"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/logging/golog"
	"net/http"
	"sync"
	"time"
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
func ListenAndServe(logger logging.Logger, s Secure, e executor) {
	certificateFile, keyFile := s.Certificate()
	if len(certificateFile) > 0 && len(keyFile) > 0 {
		go func() {
			logger.Error(
				e.ListenAndServeTLS(certificateFile, keyFile),
			)
		}()
	} else {
		go func() {
			logger.Error(
				e.ListenAndServe(),
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
func (b *Basic) New(logger logging.Logger, handler http.Handler) *http.Server {
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

// Health represents a configurable factory for a Health server.
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

// New creates both a health.Health monitor (which is also an HTTP handler) and an HTTP server
// which services health requests.
//
// This method returns nils if the configured Address is empty, which effectively disables
// the health server.
func (h *Health) New(logger logging.Logger) (handler *health.Health, server *http.Server) {
	if len(h.Address) == 0 {
		return
	}

	options := make([]health.Option, 0, len(h.Options))
	for _, value := range h.Options {
		options = append(options, health.Stat(value))
	}

	handler = health.New(
		h.LogInterval,
		logger,
		options...,
	)

	server = &http.Server{
		Addr:     h.Address,
		Handler:  handler,
		ErrorLog: NewErrorLog(h.Name, logger),
	}

	if h.LogConnectionState {
		server.ConnState = NewConnectionStateLogger(h.Name, logger)
	}

	return
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
	Log golog.LoggerFactory
}

// Prepare gets a WebPA server ready for execution.  This method does not return errors, but the returned
// Runnable may return an error.  The supplied logger will usually come from the New function, but the
// WebPA.Log object can be used to create a different logger if desired.
//
// The supplied http.Handler is used for the primary server.  If the alternate server has an address,
// it will also be used for that server.  The health server uses an internally create handler, while the pprof
// server uses http.DefaultServeMux.
func (w *WebPA) Prepare(logger logging.Logger, primaryHandler http.Handler) concurrent.Runnable {
	return concurrent.RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		if primaryServer := w.Primary.New(logger, primaryHandler); primaryServer != nil {
			logger.Info("Starting [%s] on [%s]", w.Primary.Name, w.Primary.Address)
			ListenAndServe(logger, &w.Primary, primaryServer)
		} else {
			return ErrorNoPrimaryAddress
		}

		if alternateServer := w.Alternate.New(logger, primaryHandler); alternateServer != nil {
			logger.Info("Starting [%s] on [%s]", w.Alternate.Name, w.Alternate.Address)
			ListenAndServe(logger, &w.Alternate, alternateServer)
		}

		if healthHandler, healthServer := w.Health.New(logger); healthHandler != nil && healthServer != nil {
			logger.Info("Starting [%s] on [%s]", w.Health.Name, w.Health.Address)
			ListenAndServe(logger, &w.Health, healthServer)
			healthHandler.Run(waitGroup, shutdown)
		}

		if pprofServer := w.Pprof.New(logger, nil); pprofServer != nil {
			logger.Info("Starting [%s] on [%s]", w.Pprof.Name, w.Pprof.Address)
			ListenAndServe(logger, &w.Pprof, pprofServer)
		}

		return nil
	})
}
