package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/health"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	"time"
)

const (
	// DefaultName is the application name used when none is supplied
	DefaultName = "webpa"

	// DefaultAddress is the bind address of the primary server (e.g. talaria, petasos, etc)
	DefaultAddress = ":8080"

	// DefaultHealthAddress is the bind address of the health check server
	DefaultHealthAddress = ":8081"

	// DefaultHealthLogInterval is the interval at which health statistics are emitted
	// when a non-positive log interval is specified
	DefaultHealthLogInterval time.Duration = time.Duration(60 * time.Second)

	// DefaultLogConnectionState is the default setting for logging connection state messages.  This
	// value is primarily used when a *WebPA value is nil.
	DefaultLogConnectionState = false

	// HealthSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the health server.
	HealthSuffix = "health"

	// PprofSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the pprof server.
	PprofSuffix = "pprof"
)

// serverExecutor is an internal type used to start an HTTP server.  *http.Server implements
// this interface.  It can be mocked for testing.
type serverExecutor interface {
	ListenAndServe() error
	ListenAndServeTLS(certificateFile, keyFile string) error
}

// WebPA represents a server component within the WebPA cluster.  It is used for both
// primary servers (e.g. petasos) and supporting, embedded servers such as pprof.
//
// The methods associated with a WebPA allow the method target to be nil.  This permits
// easy default servers to be stoodup for testing or development.
type WebPA struct {
	// Name is a human-readable label for the primary server.  Used primary in logging output.
	Name string

	// LogConnectionState determines whether a ConnState logging function is associated with servers.
	// By default this is false, but is useful to set to true in test and development.
	LogConnectionState bool

	// Address is the bind address of the primary server.  If empty, DefaultAddress is used.
	Address string

	// CertificateFile is the file system path of the SSL certificate.  If both this and KeyFile are supplied,
	// TLS is used for connections.  Otherwise, standard HTTP is used.
	CertificateFile string

	// KeyFile is the file system path of the SSL key.  If both this and CertificateFile are supplied,
	// then TLS is used for connections.  Otherwise, standard HTTP is used.
	KeyFile string

	// HealthAddress is the bind address of the health server.  If empty, DefaultHealthAddress is used.
	HealthAddress string

	// HealthLogInterval is the interval at which health information is logged.  If nonpositive,
	// DefaultHealthLogInterval is used.
	HealthLogInterval time.Duration

	// PprofAddress is the bind address of the pprof server.  If empty, no pprof server is used.
	PprofAddress string
}

func (w *WebPA) name() string {
	if w != nil && len(w.Name) > 0 {
		return w.Name
	}

	return DefaultName
}

func (w *WebPA) address() string {
	if w != nil && len(w.Address) > 0 {
		return w.Address
	}

	return DefaultAddress
}

func (w *WebPA) healthAddress() string {
	if w != nil && len(w.HealthAddress) > 0 {
		return w.HealthAddress
	}

	return DefaultHealthAddress
}

func (w *WebPA) pprofAddress() string {
	if w != nil {
		return w.PprofAddress
	}

	return ""
}

func (w *WebPA) healthLogInterval() time.Duration {
	if w != nil && w.HealthLogInterval > 0 {
		return w.HealthLogInterval
	}

	return DefaultHealthLogInterval
}

func (w *WebPA) logConnectionState() bool {
	if w != nil {
		return w.LogConnectionState
	}

	return DefaultLogConnectionState
}

func (w *WebPA) certificateFile() string {
	if w != nil {
		return w.CertificateFile
	}

	return ""
}

func (w *WebPA) keyFile() string {
	if w != nil {
		return w.KeyFile
	}

	return ""
}

// NewPrimaryServer creates an http.Server configured with this WebPA.  This server will typically
// be executed with secureIfPossible set to true when calling RunServer.
func (w *WebPA) NewPrimaryServer(logger logging.Logger, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:     w.address(),
		Handler:  handler,
		ErrorLog: NewErrorLog(w.name(), logger),
	}

	if w.logConnectionState() {
		server.ConnState = NewConnectionStateLogger(w.name(), logger)
	}

	return server
}

// NewHealthServer constructs both a Health subsystem and a server for serving health check content.
func (w *WebPA) NewHealthServer(logger logging.Logger, options ...health.Option) (*health.Health, *http.Server) {
	var (
		healthName = fmt.Sprintf("%s.%s", w.name(), HealthSuffix)

		healthHandler = health.New(
			w.healthLogInterval(),
			logger,
			options...,
		)

		healthServer = &http.Server{
			Addr:     w.healthAddress(),
			Handler:  healthHandler,
			ErrorLog: NewErrorLog(healthName, logger),
		}
	)

	if w.logConnectionState() {
		healthServer.ConnState = NewConnectionStateLogger(healthName, logger)
	}

	return healthHandler, healthServer
}

// NewPprofServer constructs an *http.Server that serves up the net/http/pprof content.  This
// method will return nil if no PprofAddress is configured.
func (w *WebPA) NewPprofServer(logger logging.Logger, pprofHandler http.Handler) *http.Server {
	pprofAddress := w.pprofAddress()
	if len(pprofAddress) == 0 {
		// the pprof server is optional
		return nil
	}

	if pprofHandler == nil {
		// this allows the simplicity of just doing "import _ net/http/pprof"
		pprofHandler = http.DefaultServeMux
	}

	var (
		pprofName = fmt.Sprintf("%s.%s", w.name(), PprofSuffix)

		pprofServer = &http.Server{
			Addr:     pprofAddress,
			Handler:  pprofHandler,
			ErrorLog: NewErrorLog(pprofName, logger),
		}
	)

	if w.logConnectionState() {
		pprofServer.ConnState = NewConnectionStateLogger(pprofName, logger)
	}

	return pprofServer
}

// RunServer executes a configured server, optionally using the secure configuration if specified.
// This method spawns another goroutine that the server executes from.
//
// A TLS server is used if and only if both (1) secureIfPossible is true, and (2) both CertificateFile
// and KeyFile are set.  Otherwise, a plain HTTP server is started.
//
// This method allows the executor to be nil, in which case this method simply returns.
func (w *WebPA) RunServer(logger logging.Logger, secureIfPossible bool, executor serverExecutor) {
	if executor == nil {
		return
	}

	if secureIfPossible {
		go func(certificateFile, keyFile string) {
			if len(certificateFile) > 0 && len(keyFile) > 0 {
				logger.Error(
					executor.ListenAndServeTLS(certificateFile, keyFile),
				)
			} else {
				logger.Error(
					executor.ListenAndServe(),
				)
			}
		}(w.certificateFile(), w.keyFile())
	} else {
		go func() {
			logger.Error(
				executor.ListenAndServe(),
			)
		}()
	}
}
