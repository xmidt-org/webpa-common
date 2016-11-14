package service

import (
	"fmt"
	"github.com/strava/go.serversets"
)

// Registrar is the interface which is used to register endpoints.
// *serversets.ServerSet implements this interface.
type Registrar interface {
	RegisterEndpoint(string, int, func() error) (*serversets.Endpoint, error)
}

// Watcher is the interface used to observe changes to the set of endpoints.
// *serversets.ServerSet implements this interface.
type Watcher interface {
	Watch() (*serversets.Watch, error)
}

// RegistrarWatcher is simply the union of the serversets interfaces in this package.
type RegistrarWatcher interface {
	Registrar
	Watcher
}

// NewRegistrarWatcher produces a serversets.ServerSet using a supplied set of options.
// Because of limitations with the underlying go.serversets library, this function should
// be called exactly once for any given process.
func NewRegistrarWatcher(o *Options) RegistrarWatcher {
	// yuck, really? in 2016 people use global variables for configuration?
	serversets.BaseDirectory = o.baseDirectory()
	serversets.MemberPrefix = o.memberPrefix()

	serverSet := serversets.New(
		o.environment(),
		o.serviceName(),
		o.servers(),
	)

	serverSet.ZKTimeout = o.timeout()
	return serverSet
}

// RegisterOne creates an endpoint for the given registration with a specific Registrar.
func RegisterOne(registrar Registrar, registration Registration, pingFunc func() error) (*serversets.Endpoint, error) {
	host := fmt.Sprintf("%s://%s", registration.scheme(), registration.host())
	port := registration.port()
	if port == 0 {
		return nil, fmt.Errorf("No port configured for %s", host)
	}

	return registrar.RegisterEndpoint(
		host,
		int(port),
		pingFunc,
	)
}

// RegisterAll registers all host:port strings found in o.Registrations.
func RegisterAll(registrar Registrar, o *Options) ([]*serversets.Endpoint, error) {
	registrations := o.registrations()
	if len(registrations) > 0 {
		var (
			logger    = o.logger()
			pingFunc  = o.pingFunc()
			endpoints = make([]*serversets.Endpoint, 0, len(registrations))
		)

		for _, registration := range registrations {
			logger.Info(
				"Registering endpoint: scheme=%s, host=%s, port=%d",
				registration.Scheme,
				registration.Host,
				registration.Port,
			)

			endpoint, err := RegisterOne(registrar, registration, pingFunc)
			if err != nil {
				return endpoints, err
			}

			endpoints = append(endpoints, endpoint)
		}

		return endpoints, nil
	}

	return nil, nil
}
