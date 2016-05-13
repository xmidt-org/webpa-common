package server

import (
	"github.com/Comcast/webpa-common/health"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	_ "net/http/pprof"
	"sync"
)

// Executor is a local interface describing the set of methods the underlying
// server object must implement.
type Executor interface {
	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// WebPA represents a server within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type WebPA struct {
	name            string
	executor        Executor
	certificateFile string
	keyFile         string
	logger          logging.Logger
	once            sync.Once
}

// Name returns the human-readable identifier for this WebPA instance
func (w *WebPA) Name() string {
	return w.name
}

// Logger returns the logging.Logger associated with this WebPA instance
func (w *WebPA) Logger() logging.Logger {
	return w.logger
}

// Https tests if this WebPA instance represents a secure server that uses HTTPS
func (w *WebPA) Https() bool {
	return len(w.certificateFile) > 0 && len(w.keyFile) > 0
}

// Run executes this WebPA server.  If Https() returns true, this method will start
// an HTTPS server using the configured certificate and key.  Otherwise, it will
// start an HTTP server.
//
// This method spawns a goroutine that actually executes the appropriate http.Server.ListenXXX method.
// The supplied sync.WaitGroup is incremented, and sync.WaitGroup.Done() is called when the
// spawned goroutine exits.
//
// Run is idemptotent.  It can only be execute once, and subsequent invocations have
// no effect.  Once this method is invoked, this WebPA instance is considered immutable.
func (w *WebPA) Run(waitGroup *sync.WaitGroup) error {
	w.once.Do(func() {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			var err error
			w.logger.Info("Starting [%s]", w.name)
			if w.Https() {
				err = w.executor.ListenAndServeTLS(w.certificateFile, w.keyFile)
			} else {
				err = w.executor.ListenAndServe()
			}

			w.logger.Error("%v", err)
		}()
	})

	return nil
}

// New creates a new, nonsecure WebPA instance.  It delegates to NewSecure(), with empty strings
// for certificateFile and keyFile.
func New(logger logging.Logger, name string, executor Executor) *WebPA {
	return NewSecure(logger, name, executor, "", "")
}

// NewSecure creates a new, optionally secure WebPA instance.  The certificateFile and keyFile parameters
// may be empty strings, in which case the returned instance will start an HTTP server.
func NewSecure(logger logging.Logger, name string, executor Executor, certificateFile, keyFile string) *WebPA {
	return &WebPA{
		name:            name,
		executor:        executor,
		certificateFile: certificateFile,
		keyFile:         keyFile,
		logger:          logger,
	}
}

// Primary is a factory function for the primary server defined in the configuration
func Primary(logger logging.Logger, name string, configuration *Configuration, handler http.Handler) *WebPA {
	executor := &http.Server{
		Addr:      configuration.PrimaryAddress(),
		Handler:   handler,
		ConnState: logging.NewConnectionStateLogger(logger, name),
		ErrorLog:  logging.NewErrorLog(logger, name),
	}

	return NewSecure(
		logger,
		name,
		executor,
		configuration.CertificateFile,
		configuration.KeyFile,
	)
}

// Health is a factory function for both the WebPA server that exposes health statistics
// and the underlying Health object, both of which are Runnable.
func Health(logger logging.Logger, name string, configuration *Configuration, options ...health.Option) (*WebPA, *health.Health) {
	healthHandler := health.New(configuration.HealthCheckInterval(), logger, options...)

	executor := &http.Server{
		Addr:      configuration.HealthAddress(),
		Handler:   healthHandler,
		ConnState: logging.NewConnectionStateLogger(logger, name),
		ErrorLog:  logging.NewErrorLog(logger, name),
	}

	return New(logger, name, executor), healthHandler
}

// Pprof is a factory function for the pprof server defined in the configuration
func Pprof(logger logging.Logger, name string, configuration *Configuration) *WebPA {
	// http.DefaultServeMux is where the pprof handlers are automatically registered
	executor := &http.Server{
		Addr:      configuration.PprofAddress(),
		Handler:   http.DefaultServeMux,
		ConnState: logging.NewConnectionStateLogger(logger, name),
		ErrorLog:  logging.NewErrorLog(logger, name),
	}

	return New(logger, name, executor)
}
