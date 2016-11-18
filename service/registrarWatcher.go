package service

import (
	"fmt"
	"github.com/strava/go.serversets"
	"regexp"
	"strconv"
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

var (
	defaultPorts = map[string]uint16{
		"http":  80,
		"https": 443,
	}

	// registrationPattern is the regular expression used to parse registrations.
	// any string will match this pattern, because of the way the <host> subexpression
	// is defined.
	registrationPattern = regexp.MustCompile(`((?P<scheme>[a-z]+)://)*(?P<host>[^:]+)(:(?P<port>[0-9]+))*`)
)

// ParseRegistration accepts a string containing host and port and an optional
// scheme and returns the host and port for registering as an endpoint.
//
// Examples of registration strings include "http://something.comcast.net:8080" and "foobar.com".
// If no scheme is supplied, "http" is used.  If no port is supplied, the default port for
// the scheme is used, if one is present.  Unrecognized schemes are permitted.
func ParseRegistration(value string) (string, uint16, error) {
	matches := registrationPattern.FindStringSubmatch(value)
	scheme := matches[2]
	if len(scheme) == 0 {
		scheme = DefaultScheme
	}

	host := fmt.Sprintf("%s://%s", scheme, matches[3])

	port := matches[5]
	if len(port) == 0 {
		return host, defaultPorts[scheme], nil
	}

	// the <port> subexpression is guaranteed to be a valid unsigned integer
	portValue, err := strconv.ParseUint(port, 10, 16)
	if err != nil {
		return host, 0, err
	}

	if portValue > 0 {
		return host, uint16(portValue), nil
	}

	// if the registration explicitly set the port to 0, use the default port
	return host, defaultPorts[scheme], nil
}

// RegisterAll registers all host:port strings found in o.Registrations.
func RegisterAll(registrar Registrar, o *Options) ([]*serversets.Endpoint, error) {
	registrations := o.registrations()
	if len(registrations) > 0 {
		var (
			hosts = make([]string, 0, len(registrations))
			ports = make([]int, 0, len(registrations))
		)

		// first, parse all the registrations.  if any are invalid, we
		// don't want to register any endpoint as the configuration is invalid.
		for _, registration := range registrations {
			host, port, err := ParseRegistration(registration)
			if err != nil {
				return nil, fmt.Errorf("Invalid registration %s: %s", registration, err)
			}

			hosts = append(hosts, host)
			ports = append(ports, int(port))
		}

		var (
			logger    = o.logger()
			pingFunc  = o.pingFunc()
			endpoints = make([]*serversets.Endpoint, 0, len(registrations))
		)

		for index, host := range hosts {
			port := ports[index]
			logger.Info("Registering endpoint: %s:%d", host, port)
			endpoint, err := registrar.RegisterEndpoint(host, port, pingFunc)
			if err != nil {
				return endpoints, err
			}

			endpoints = append(endpoints, endpoint)
		}

		return endpoints, nil
	}

	return nil, nil
}
