package service

import (
	"bytes"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
)

const (
	DefaultServer         = "localhost:2181"
	DefaultConnectTimeout = 5 * time.Second
	DefaultSessionTimeout = 1 * time.Hour
	DefaultPath           = "/xmidt"
	DefaultServiceName    = "test"
	DefaultVnodeCount     = 211
)

// Options represents the set of configurable attributes for service discovery and registration
type Options struct {
	// Logger is used by any component configured via this Options.  If unset, a default
	// logger is used.
	Logger log.Logger `json:"-"`

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

	// UpdateDelay specifies the period of time between a service discovery update and when a client
	// is notified.  Updates during the wait time simply replace the waiting set of instances.
	// There is no default for this field.  If unset, all updates are immediately processed.
	UpdateDelay time.Duration `json:"updateDelay"`

	// Path is the base path for all znodes created via this Options.
	Path string `json:"path,omitempty"`

	// ServiceName is the name of the service being registered.
	ServiceName string `json:"serviceName,omitempty"`

	// Registration is the data stored about this service, typically host:port or scheme://host:port.
	Registration string `json:"registration,omitempty"`

	// VnodeCount is used to tune the underlying consistent hash algorithm for servers.
	VnodeCount uint `json:"vnodeCount"`

	// InstancesFilter is the optional filter for discovered instances.  If not set,
	// DefaultInstancesFilter will be used.
	InstancesFilter InstancesFilter `json:"-"`

	// AccessorFactory is the optional factory for Accessor instances.  If not set,
	// ConsistentAccessorFactory will be used.
	AccessorFactory AccessorFactory `json:"-"`

	// After is the optional function to use to obtain a channel which receives a time.Time
	// after a delay.  If not set, time.After is used.
	After func(time.Duration) <-chan time.Time `json:"-"`
}

func (o *Options) String() string {
	var output bytes.Buffer
	if o == nil {
		output.WriteString("<nil>")
	} else {
		if len(o.Connection) > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("connection=")
			output.WriteString(o.Connection)
		}

		if len(o.Servers) > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("servers=")
			output.WriteString(strings.Join(o.Servers, ","))
		}

		if o.ConnectTimeout > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("connectTimeout=")
			output.WriteString(o.ConnectTimeout.String())
		}

		if o.SessionTimeout > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("sessionTimeout=")
			output.WriteString(o.SessionTimeout.String())
		}

		if o.UpdateDelay > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("updateDelay=")
			output.WriteString(o.UpdateDelay.String())
		}

		if o.VnodeCount > 0 {
			if output.Len() > 0 {
				output.WriteString(", ")
			}

			output.WriteString("vnodeCount=")
			output.WriteString(strconv.FormatUint(uint64(o.VnodeCount), 10))
		}

		if output.Len() > 0 {
			output.WriteString(", ")
		}

		output.WriteString("path=")
		output.WriteString(o.Path)
		output.WriteString(", serviceName=")
		output.WriteString(o.ServiceName)
		output.WriteString(", registration=")
		output.WriteString(o.Registration)
	}

	return output.String()
}

func (o *Options) logger() log.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return log.NewNopLogger()
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

func (o *Options) updateDelay() time.Duration {
	if o != nil && o.UpdateDelay > 0 {
		return o.UpdateDelay
	}

	return 0
}

func (o *Options) path() string {
	if o != nil && len(o.Path) > 0 {
		return o.Path
	}

	return DefaultPath
}

func (o *Options) serviceName() string {
	if o != nil && len(o.ServiceName) > 0 {
		return o.ServiceName
	}

	return DefaultServiceName
}

func (o *Options) registration() string {
	if o != nil {
		return o.Registration
	}

	return ""
}

func (o *Options) vnodeCount() int {
	if o != nil && o.VnodeCount > 0 {
		return int(o.VnodeCount)
	}

	return DefaultVnodeCount
}

func (o *Options) instancesFilter() InstancesFilter {
	if o != nil && o.InstancesFilter != nil {
		return o.InstancesFilter
	}

	return DefaultInstancesFilter
}

func (o *Options) accessorFactory() AccessorFactory {
	if o != nil && o.AccessorFactory != nil {
		return o.AccessorFactory
	}

	return ConsistentAccessorFactory(o.vnodeCount())
}

func (o *Options) after() func(time.Duration) <-chan time.Time {
	if o != nil && o.After != nil {
		return o.After
	}

	return time.After
}
