package server

import (
	"github.com/Comcast/webpa-common/logging"
	"net/http"
)

// Builder provides a simple, configurable factory for WebPA instances.
type Builder struct {
	Name            string
	Address         string
	CertificateFile string
	KeyFile         string
	Logger          logging.Logger
	Handler         http.Handler
}

// Build creates a a distinct WebPA instance from this builder's configuration
func (b *Builder) Build() *WebPA {
	return &WebPA{
		name:            b.Name,
		address:         b.Address,
		certificateFile: b.CertificateFile,
		keyFile:         b.KeyFile,
		serverExecutor: &http.Server{
			Addr:      b.Address,
			Handler:   b.Handler,
			ErrorLog:  NewErrorLog(b.Name, b.Logger),
			ConnState: NewConnectionStateLogger(b.Name, b.Logger),
		},
		logger: b.Logger,
	}
}
