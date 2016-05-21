package server

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/logging"
	"sync"
)

// serverExecutor is a local interface describing the set of methods the underlying
// server object must implement. *http.Server, for example, implements this interface.
type serverExecutor interface {
	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// WebPA represents a server component within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type WebPA struct {
	name            string
	address         string
	serverExecutor  serverExecutor
	certificateFile string
	keyFile         string
	logger          logging.Logger
	once            sync.Once
}

var _ concurrent.Runnable = (*WebPA)(nil)

func (w *WebPA) Name() string {
	return w.name
}

func (w *WebPA) Address() string {
	return w.address
}

func (w *WebPA) CertificateFile() string {
	return w.certificateFile
}

func (w *WebPA) KeyFile() string {
	return w.keyFile
}

func (w *WebPA) Secure() bool {
	return len(w.certificateFile) > 0 && len(w.keyFile) > 0
}

func (w *WebPA) Logger() logging.Logger {
	return w.logger
}

func (w *WebPA) String() string {
	data, err := w.MarshalJSON()
	if err != nil {
		return err.Error()
	}

	return string(data)
}

func (w *WebPA) MarshalJSON() ([]byte, error) {
	data := struct {
		Name            string `json:"name"`
		Address         string `json:"address"`
		CertificateFile string `json:"cert"`
		KeyFile         string `json:"key"`
	}{
		Name:            w.name,
		Address:         w.address,
		CertificateFile: w.certificateFile,
		KeyFile:         w.keyFile,
	}

	return json.Marshal(&data)
}

// Run executes this WebPA server.  If both certificateFile and keyFile are non-empty, this method will start
// an HTTPS server using the configured certificate and key.  Otherwise, it will
// start an HTTP server.
//
// This method spawns a goroutine that actually executes the appropriate serverExecutor.ListenXXX method.
// The supplied sync.WaitGroup is incremented, and sync.WaitGroup.Done() is called when the
// spawned goroutine exits.
//
// Run is idemptotent.  It can only be execute once, and subsequent invocations have
// no effect.
func (w *WebPA) Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
	w.once.Do(func() {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			var err error
			w.logger.Info("Starting [%s]", w.name)
			if w.Secure() {
				err = w.serverExecutor.ListenAndServeTLS(w.certificateFile, w.keyFile)
			} else {
				err = w.serverExecutor.ListenAndServe()
			}

			w.logger.Error("%s exiting: %v", w.name, err)
		}()
	})

	return nil
}
