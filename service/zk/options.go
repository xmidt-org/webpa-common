package zk

import (
	"strings"
	"time"
)

const (
	DefaultServer      = "localhost:2181"
	DefaultServiceName = "test"
	DefaultPath        = "/xmidt/test"
	DefaultAddress     = "localhost"
	DefaultPort        = 8080
	DefaultScheme      = "http"

	DefaultConnectTimeout time.Duration = 5 * time.Second
	DefaultSessionTimeout time.Duration = 10 * time.Second
)

type Registration struct {
	// Name is the service name under which to register.  If not supplied, DefaultServiceName is used.
	Name string `json:"name,omitempty"`

	// Path is the znode path under which to register.  If not supplied, DefaultPath is used.
	Path string `json:"path,omitempty"`

	// Address is the FQDN or hostname of the server which hosts the service.  If not supplied, DefaultAddress is used.
	Address string `json:"address,omitempty"`

	// Port is the TCP port on which the service listens.  If not supplied, DefaultPort is used.
	Port int `json:"port,omitempty"`

	// Scheme specific the protocl used for the service.  If not supplied, DefaultScheme is used.
	Scheme string `json:"scheme,omitempty"`
}

func (r Registration) name() string {
	if len(r.Name) > 0 {
		return r.Name
	}

	return DefaultServiceName
}

func (r Registration) path() string {
	if len(r.Path) > 0 {
		return r.Path
	}

	return DefaultPath
}

func (r Registration) address() string {
	if len(r.Address) > 0 {
		return r.Address
	}

	return DefaultAddress
}

func (r Registration) port() int {
	if r.Port > 0 {
		return r.Port
	}

	return DefaultPort
}

func (r Registration) scheme() string {
	if len(r.Scheme) > 0 {
		return r.Scheme
	}

	return DefaultScheme
}

// Options represents the set of configurable attributes for Zookeeper
type Options struct {
	// Connection is the comma-delimited Zookeeper connection string.  Both this and
	// Servers may be set, and they will be merged together when connecting to Zookeeper.
	Connection string `json:"connection,omitempty"`

	// Servers is the array of Zookeeper servers.  Both this and Connection may be set,
	// and they will be merged together when connecting to Zookeeper.
	Servers []string `json:"servers,omitempty"`

	// ConnectTimeout is the Zookeeper connection timeout.
	ConnectTimeout time.Duration `json:"connectTimeout"`

	// SessionTimeout is the Zookeeper session timeout.
	SessionTimeout time.Duration `json:"sessionTimeout"`

	// Registrations are the ways in which the host process should be registered with zookeeper.
	// There is no default for this field.
	Registrations []Registration `json:"registrations,omitempty"`

	// Watches are the zookeeper paths to watch for updates.  There is no default for this field.
	Watches []string `json:"watches,omitempty"`
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

func (o *Options) connectTimeout() time.Duration {
	if o != nil && o.ConnectTimeout > 0 {
		return o.ConnectTimeout
	}

	return DefaultConnectTimeout
}

func (o *Options) sessionTimeout() time.Duration {
	if o != nil && o.SessionTimeout > 0 {
		return o.SessionTimeout
	}

	return DefaultSessionTimeout
}

func (o *Options) registrations() []Registration {
	if o != nil && len(o.Registrations) > 0 {
		return o.Registrations
	}

	return nil
}

func (o *Options) watches() []string {
	if o != nil && len(o.Watches) > 0 {
		return o.Watches
	}

	return nil
}
