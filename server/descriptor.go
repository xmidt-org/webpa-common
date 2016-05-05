package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/context"
	"net/http"
	"sync"
)

// Descriptor provides the basic description of a WebPA server.  This struct can be embedded
// and further tailored to a specific server's configuration.
type Descriptor struct {
	// Port is the IP port that serves the primary purpose of this server
	Port int `json:"port"`

	// CertificateFile is the path to the certificate file.  If this or KeyFile is not
	// supplied, the server that listens on Port will be an HTTP server.
	CertificateFile string `json:"cert"`

	// KeyFile is the path to the key file.  If this or KeyFile is not
	// supplied, the server that listens on Port will be an HTTP server.
	KeyFile string `json:"key"`
}

// NewServer creates an HTTP server from this descriptor's configuration
func (d *Descriptor) NewServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    fmt.Sprintf(":%d", d.Port),
		Handler: handler,
	}
}

// RunServer creates and runs a server in a separate goroutine.  Whether
// the server is HTTP or HTTPS is dependent on the CertificateFile and KeyFile
// attributes.
func (d *Descriptor) RunServer(logger context.Logger, waitGroup *sync.WaitGroup, server *http.Server) {
	waitGroup.Add(1)
	go func() {
		var err error
		if len(d.CertificateFile) > 0 && len(d.KeyFile) > 0 {
			logger.Info("Starting HTTPS server at %s", server.Addr)
			err = server.ListenAndServeTLS(d.CertificateFile, d.KeyFile)
		} else {
			logger.Info("Starting HTTP server at %s", server.Addr)
			err = server.ListenAndServe()
		}

		// The return from the ListenXXX method is always non-nil
		logger.Error("%v", err)
		waitGroup.Done()
	}()
}
