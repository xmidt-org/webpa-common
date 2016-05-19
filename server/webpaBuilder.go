package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	_ "net/http/pprof"
	"time"
)

// WebPABuilder implements the instantiation logic for each WebPA server component.
// This builder type is the standard way to construct and start a WebPA server.
type WebPABuilder struct {
	// LoggerFactory is used to create logging.Logger objects for use in
	// each server
	LoggerFactory logging.LoggerFactory

	// Configuration is the parsed configuration data, normaly from a JSON configuration file
	Configuration *Configuration

	// PrimaryHandler is the http.Handler used for the primary server
	PrimaryHandler http.Handler

	// PprofHandler is the optional handler for pprof traffic.  If omitted, http.DefaultServeMux
	// will be used instead
	PprofHandler http.Handler

	// HealthHandler is the http.handler for health traffic
	HealthHandler http.Handler
}

func (builder *WebPABuilder) ServerName() string {
	if builder.Configuration != nil && len(builder.Configuration.ServerName) > 0 {
		return builder.Configuration.ServerName
	}

	return DefaultServerName
}

// PrimaryAddress returns the listen address for the primary server, i.e.
// the server that listens on c.Port.
func (builder *WebPABuilder) PrimaryAddress() string {
	port := DefaultPort
	if builder.Configuration != nil && builder.Configuration.Port > 0 {
		port = builder.Configuration.Port
	}

	return fmt.Sprintf(":%d", port)
}

// HealthAddress returns the listen address for the health server
func (builder *WebPABuilder) HealthAddress() string {
	port := DefaultHealthCheckPort
	if builder.Configuration != nil && builder.Configuration.HealthCheckPort > 0 {
		port = builder.Configuration.HealthCheckPort
	}

	return fmt.Sprintf(":%d", port)
}

// HealthCheckInterval returns the health check interval as
// a time.Duration, using the default if c.HCInterval is nonpositive.
func (builder *WebPABuilder) HealthCheckInterval() time.Duration {
	if builder.Configuration != nil && builder.Configuration.HealthCheckInterval > 0 {
		return time.Duration(builder.Configuration.HealthCheckInterval)
	}

	return DefaultHealthCheckInterval
}

// PprofAddress returns the listen address for the pprof server
func (builder *WebPABuilder) PprofAddress() string {
	port := DefaultPprofPort
	if builder.Configuration != nil && builder.Configuration.PprofPort > 0 {
		port = builder.Configuration.PprofPort
	}

	return fmt.Sprintf(":%d", port)
}

// BuildPrimary returns a Runnable that will execute the primary server
func (builder *WebPABuilder) BuildPrimary() (Runnable, error) {
	name := builder.ServerName()
	address := builder.PrimaryAddress()
	logger, err := builder.LoggerFactory.NewLogger(name)
	if err != nil {
		return nil, err
	}

	certificateFile, keyFile := "", ""
	if builder.Configuration != nil {
		certificateFile = builder.Configuration.CertificateFile
		keyFile = builder.Configuration.KeyFile
	}

	return &webPA{
		name:            name,
		address:         address,
		logger:          logger,
		certificateFile: certificateFile,
		keyFile:         keyFile,
		serverExecutor: &http.Server{
			Addr:      address,
			Handler:   builder.PrimaryHandler,
			ConnState: NewConnectionStateLogger(logger, name),
			ErrorLog:  NewErrorLog(logger, name),
		},
	}, nil
}

// BuildHealth is a factory function for both the WebPA server that exposes health statistics
// and the underlying Health object, both of which are Runnable.
func (builder *WebPABuilder) BuildHealth() (Runnable, error) {
	name := builder.ServerName() + healthSuffix
	address := builder.HealthAddress()
	logger, err := builder.LoggerFactory.NewLogger(name)
	if err != nil {
		return nil, err
	}

	return &webPA{
		name:    name,
		address: address,
		logger:  logger,
		serverExecutor: &http.Server{
			Addr:      address,
			Handler:   builder.HealthHandler,
			ConnState: NewConnectionStateLogger(logger, name),
			ErrorLog:  NewErrorLog(logger, name),
		},
	}, nil
}

// BuildPprof is a factory function for the pprof server defined in the configuration
func (builder *WebPABuilder) BuildPprof() (Runnable, error) {
	name := builder.ServerName() + pprofSuffix
	address := builder.PprofAddress()
	logger, err := builder.LoggerFactory.NewLogger(name)
	if err != nil {
		return nil, err
	}

	pprofHandler := builder.PprofHandler
	if pprofHandler == nil {
		// by default, net/http/pprof registers the handles
		// using the default server mux
		pprofHandler = http.DefaultServeMux
	}

	return &webPA{
		name:    name,
		address: address,
		logger:  logger,
		serverExecutor: &http.Server{
			Addr:      address,
			Handler:   pprofHandler,
			ConnState: NewConnectionStateLogger(logger, name),
			ErrorLog:  NewErrorLog(logger, name),
		},
	}, nil
}

// BuildAll returns a RunnableSet that executes all server components produced
// by this builder: pprof, health, and the primary server
func (builder *WebPABuilder) BuildAll() (runnableSet RunnableSet, err error) {
	var runnables [3]Runnable
	runnables[0], err = builder.BuildPprof()
	if err != nil {
		return
	}

	runnables[1], err = builder.BuildHealth()
	if err != nil {
		return
	}

	runnables[2], err = builder.BuildPrimary()
	if err != nil {
		return
	}

	runnableSet = RunnableSet(runnables[0:3])
	return
}
