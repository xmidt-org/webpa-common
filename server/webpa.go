package server

import (
	"github.com/Comcast/webpa-common/logging"
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
func (w *WebPA) Run(waitGroup *sync.WaitGroup) {
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
