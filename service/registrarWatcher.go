package service

import (
	"github.com/billhathaway/consistentHash"
	"github.com/strava/go.serversets"
	"strconv"
	"strings"
)

// Accessor provides access to services based around []byte keys.
// *consistentHash.ConsistentHash implements this interface.
type Accessor interface {
	Get([]byte) (string, error)
}

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

// ParseRegistration separates a string value into a host and a port.  This function assumes
// that value will have a format like "host:port".  If there is no semicolon, or if what comes
// after the last semicolon is not an integer, this function returns the value as the host
// and zero (0) for the port.
func ParseRegistration(value string) (string, int, error) {
	position := strings.LastIndex(value, ":")
	if position >= 0 {
		port, err := strconv.Atoi(value[position+1:])
		if err == nil {
			return value[0:position], port, nil
		}
	}

	return value, 0, nil
}

// NewRegistrarWatcher produces a serversets.ServerSet using a supplied set of options
func NewRegistrarWatcher(o *Options) RegistrarWatcher {
	return serversets.New(
		o.environment(),
		o.serviceName(),
		o.zookeepers(),
	)
}

// RegisterEndpoints registers all host:port strings found in o.Registrations.  This
// function returns a nil slice if o == nil or if o has no registrations.  If any errors
// occur, this function returns a partial slice of endpoints that it could successfully create.
func RegisterEndpoints(o *Options, registrar Registrar) ([]*serversets.Endpoint, error) {
	if o != nil && len(o.Registrations) > 0 {
		var (
			err               error
			registrationCount = len(o.Registrations)
			hosts             = make([]string, registrationCount)
			ports             = make([]int, registrationCount)
			endpoint          *serversets.Endpoint
		)

		for index, registration := range o.Registrations {
			hosts[index], ports[index], err = ParseRegistration(registration)
			if err != nil {
				return nil, err
			}
		}

		endpoints := make([]*serversets.Endpoint, 0, registrationCount)
		for index := 0; index < registrationCount; index++ {
			endpoint, err = registrar.RegisterEndpoint(
				hosts[index],
				ports[index],
				o.PingFunc,
			)

			if err != nil {
				return nil, err
			}

			endpoints = append(endpoints, endpoint)
		}

		return endpoints, nil
	}

	return nil, nil
}

func NewAccessor(vnodeCount int, endpoints []string) Accessor {
	hash := consistentHash.New()
	hash.SetVnodeCount(vnodeCount)
	for _, hostAndPort := range endpoints {
		hash.Add(hostAndPort)
	}

	return hash
}

func Subscribe(vnodeCount int, watcher Watcher) (<-chan Accessor, error) {
	watch, err := watcher.Watch()
	if err != nil {
		return nil, err
	}

	accessors := make(chan Accessor, 1)
	go func() {
		defer close(accessors)
		accessors <- NewAccessor(vnodeCount, watch.Endpoints())

		for {
			select {
			case <-watch.Event():
				if watch.IsClosed() {
					return
				}

				accessors <- NewAccessor(vnodeCount, watch.Endpoints())
			}
		}
	}()

	return accessors, nil
}
