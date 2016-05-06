package server

import (
	"github.com/Comcast/webpa-common/context"
	"log"
	"net/http"
	"sync"
)

// WebPA represents a server within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type WebPA struct {
	name            string
	server          *http.Server
	certificateFile string
	keyFile         string
	logger          context.Logger
}

// Name returns the human-readable identifier for this WebPA instance
func (w *WebPA) Name() string {
	return w.name
}

// Logger returns the context.Logger associated with this WebPA instance
func (w *WebPA) Logger() context.Logger {
	return w.logger
}

// Https tests if this WebPA instance represents a secure server that uses HTTPS
func (w *WebPA) Https() bool {
	return len(w.certificateFile) > 0 && len(w.keyFile) > 0
}

// Run executes this WebPA server.  If Https() returns true, this method will start
// an HTTPS server using the configured certificate and key.  Otherwise, it will
// start an HTTP server.
func (w *WebPA) Run(waitGroup *sync.WaitGroup) {
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		var err error
		if w.Https() {
			w.logger.Info("Starting [%s] as HTTPS on %s", w.name, w.server.Addr)
			err = w.server.ListenAndServeTLS(w.certificateFile, w.keyFile)
		} else {
			w.logger.Info("Starting [%s] as HTTP on %s", w.name, w.server.Addr)
			err = w.server.ListenAndServe()
		}

		w.logger.Error("%v", err)
	}()
}

// New creates a new, nonsecure WebPA instance.  It delegates to NewSecure(), with empty strings
// for certificateFile and keyFile.
func New(name string, server *http.Server, logger context.Logger) *WebPA {
	return NewSecure(name, server, "", "", logger)
}

// NewSecure creates a new, secure WebPA instance.  If no ErrorLog is associated with the given http.Server,
// this method attaches an ErrorLog that delegates to the configured context.Logger.
func NewSecure(name string, server *http.Server, certificateFile, keyFile string, logger context.Logger) *WebPA {
	if server.ErrorLog == nil {
		server.ErrorLog = log.New(
			&context.ErrorWriter{logger},
			name,
			log.LUTC|log.LstdFlags,
		)
	}

	return &WebPA{
		name:            name,
		server:          server,
		certificateFile: certificateFile,
		keyFile:         keyFile,
		logger:          logger,
	}
}
