package server

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/logging"
	"sync"
)

// serverExecutor is a local interface describing the set of methods the underlying
// server object must implement.
type serverExecutor interface {
	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// webPA represents a server within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
type webPA struct {
	name            string
	address         string
	serverExecutor  serverExecutor
	certificateFile string
	keyFile         string
	logger          logging.Logger
	once            sync.Once
}

func (w *webPA) String() string {
	data, err := w.MarshalJSON()
	if err != nil {
		return err.Error()
	}

	return string(data)
}

func (w *webPA) MarshalJSON() ([]byte, error) {
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

func (w *webPA) Close() error {
	return nil
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
func (w *webPA) Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
	w.once.Do(func() {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			var err error
			w.logger.Info("Starting [%s]", w.name)
			if len(w.certificateFile) > 0 && len(w.keyFile) > 0 {
				err = w.serverExecutor.ListenAndServeTLS(w.certificateFile, w.keyFile)
			} else {
				err = w.serverExecutor.ListenAndServe()
			}

			w.logger.Error("%s exiting: %v", w.name, err)
		}()
	})

	return nil
}
