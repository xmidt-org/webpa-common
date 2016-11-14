package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/strava/go.serversets"
	"strings"
	"time"
)

const (
	DefaultScheme        = "http"
	DefaultHost          = "localhost"
	DefaultServer        = "localhost:2181"
	DefaultTimeout       = 5 * time.Second
	DefaultBaseDirectory = "/webpa"
	DefaultMemberPrefix  = "webpa_"
	DefaultEnvironment   = serversets.Local
	DefaultServiceName   = "test"
	DefaultVnodeCount    = 10000
)

var (
	defaultPorts = map[string]uint16{
		"http":  80,
		"https": 443,
	}
)

// Registration describes a single endpoint to register.
type Registration struct {
	// Scheme is the scheme portion of the registered URL, e.g. "http" or "https".
	// Non-HTTP schemes are allowed, but no default ports are set.
	Scheme string `json:"scheme"`

	// Host is the FQDN or DNS host name of the registered service.
	Host string `json:"host"`

	// Port is the IP port of the registered service.  If not set, and if Scheme
	// is either not set or set to an HTTP scheme, then the Port is set to the default
	// port when registering.
	Port uint16 `json:"port"`
}

func (r *Registration) scheme() string {
	if r != nil && len(r.Scheme) > 0 {
		return r.Scheme
	}

	return DefaultScheme
}

func (r *Registration) host() string {
	if r != nil && len(r.Host) > 0 {
		return r.Host
	}

	return DefaultHost
}

func (r *Registration) port() uint16 {
	if r != nil && r.Port > 0 {
		return r.Port
	}

	return defaultPorts[r.scheme()]
}

// Options represents the set of configurable attributes for service discovery and registration
type Options struct {
	// Logger is used by any component configured via this Options.  If unset, a default
	// logger is used.
	Logger logging.Logger `json:"-"`

	// Connection is the comma-delimited Zookeeper connection string.  Both this and
	// Servers may be set, and they will be merged together when connecting to Zookeeper.
	Connection string `json:"connection,omitempty"`

	// Servers is the array of Zookeeper servers.  Both this and Connection may be set,
	// and they will be merged together when connecting to Zookeeper.
	Servers []string `json:"servers,omitempty"`

	// Timeout is the Zookeeper connection timeout.
	Timeout time.Duration `json:"timeout"`

	// BaseDirectory is the base path for all znodes created via this Options.
	BaseDirectory string `json:"baseDirectory,omitempty"`

	// MemberPrefix is the prefix for ephemeral nodes regstered via this Options.
	MemberPrefix string `json:"memberPrefix,omitempty"`

	// Environment is the environment component of the ephemeral znode path.
	Environment string `json:"environment,omitempty"`

	// ServiceName is the name of the service being registered.
	ServiceName string `json:"serviceName,omitempty"`

	// Registrations holds the slice of information used to register endpoints.  Typically,
	// this slice will either (1) be empty for an application that only watches for changes,  or (2) have the single
	// Registration indicating how this service is known.  Multiple registrations, essentially
	// being aliases for the same application, are supported.
	Registrations []Registration `json:"registrations,omitempty"`

	// VnodeCount is used to tune the underlying consistent hash algorithm for servers.
	VnodeCount uint `json:"vnodeCount"`

	// PingFunc is the callback function used to determine if this application is still able
	// to respond to requests.  This can be nil, and there is no default.
	PingFunc func() error `json:"-"`
}

func (o *Options) logger() logging.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}

func (o *Options) servers() []string {
	servers := make([]string, 0, 10)

	if o != nil {
		if len(o.Connection) > 0 {
			for _, server := range strings.Split(o.Connection, ",") {
				servers = append(servers, strings.TrimSpace(server))
			}
		}

		if len(o.Servers) > 0 {
			servers = append(servers, o.Servers...)
		}
	}

	if len(servers) == 0 {
		servers = append(servers, DefaultServer)
	}

	return servers
}

func (o *Options) timeout() time.Duration {
	if o != nil && o.Timeout > 0 {
		return time.Duration(o.Timeout)
	}

	return DefaultTimeout
}

func (o *Options) baseDirectory() string {
	if o != nil && len(o.BaseDirectory) > 0 {
		return o.BaseDirectory
	}

	return DefaultBaseDirectory
}

func (o *Options) memberPrefix() string {
	if o != nil && len(o.MemberPrefix) > 0 {
		return o.MemberPrefix
	}

	return DefaultMemberPrefix
}

func (o *Options) environment() serversets.Environment {
	if o != nil && len(o.Environment) > 0 {
		return serversets.Environment(o.Environment)
	}

	return DefaultEnvironment
}

func (o *Options) serviceName() string {
	if o != nil && len(o.ServiceName) > 0 {
		return o.ServiceName
	}

	return DefaultServiceName
}

func (o *Options) registrations() []Registration {
	if o != nil {
		return o.Registrations
	}

	return nil
}

func (o *Options) vnodeCount() int {
	if o != nil && o.VnodeCount > 0 {
		return int(o.VnodeCount)
	}

	return DefaultVnodeCount
}

func (o *Options) pingFunc() func() error {
	if o != nil {
		return o.PingFunc
	}

	return nil
}
