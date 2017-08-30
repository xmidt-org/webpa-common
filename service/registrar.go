package service

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log/level"
	"github.com/strava/go.serversets"
)

// Watch is the subset of methods required by this package that *serversets.Watch implements
type Watch interface {
	Close()
	IsClosed() bool
	Event() <-chan struct{}
	Endpoints() []string
}

// Registrar is the interface which is used to register and watch endpoints
type Registrar interface {
	RegisterEndpoint(string, int, func() error) (*serversets.Endpoint, error)
	Watch() (Watch, error)
}

// registrar is an internal type used to make ServerSet conform to the Registrar interface
type registrar serversets.ServerSet

func (r *registrar) RegisterEndpoint(host string, port int, ping func() error) (*serversets.Endpoint, error) {
	return (*serversets.ServerSet)(r).RegisterEndpoint(host, port, ping)
}

func (r *registrar) Watch() (Watch, error) {
	return (*serversets.ServerSet)(r).Watch()
}

// NewRegistrar produces a serversets.ServerSet using a supplied set of options.
// Because of limitations with the underlying go.serversets library, this function should
// be called exactly once for any given process.
func NewRegistrar(o *Options) Registrar {
	// yuck, really? in 2016 people use global variables for configuration?
	serversets.BaseDirectory = o.baseDirectory()
	serversets.MemberPrefix = o.memberPrefix()

	serverSet := serversets.New(
		o.environment(),
		o.serviceName(),
		o.servers(),
	)

	serverSet.ZKTimeout = o.timeout()
	return (*registrar)(serverSet)
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
func RegisterAll(registrar Registrar, o *Options) (RegisteredEndpoints, error) {
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
			infoLog   = logging.DefaultCaller(o.logger(), level.Key(), level.InfoValue())
			pingFunc  = o.pingFunc()
			endpoints = make(RegisteredEndpoints, len(registrations))
		)

		for index, host := range hosts {
			port := ports[index]
			infoLog.Log(logging.MessageKey, "Registering endpoint", "host", host, "port", port)

			registeredEndpoint, err := registrar.RegisterEndpoint(host, port, pingFunc)
			if err != nil {
				return endpoints, err
			}

			endpoints.AddHostPort(host, port, registeredEndpoint)
		}

		return endpoints, nil
	}

	return nil, nil
}
