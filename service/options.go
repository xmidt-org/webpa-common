package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/strava/go.serversets"
	"time"
)

const (
	DefaultScheme           = "http"
	DefaultHost             = "localhost"
	DefaultZookeeperServer  = "localhost:2181"
	DefaultZookeeperTimeout = 5 * time.Second
	DefaultBaseDirectory    = "/webpa"
	DefaultMemberPrefix     = "webpa_"
	DefaultEnvironment      = serversets.Local
	DefaultServiceName      = "test"
	DefaultVnodeCount       = 10000
)

var (
	defaultPorts = map[string]uint16{
		"http":  80,
		"https": 443,
	}
)

// Registration describes a single endpoint to register.
type Registration struct {
	Scheme string `json:"scheme"`
	Host   string `json:"host"`
	Port   uint16 `json:"port"`
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
	Logger           logging.Logger `json:"-"`
	ZookeeperServers []string       `json:"zookeeperServers"`
	ZookeeperTimeout time.Duration  `json:"zookeeperTimeout"`
	BaseDirectory    string         `json:"baseDirectory"`
	MemberPrefix     string         `json:"memberPrefix"`
	Environment      string         `json:"environment"`
	ServiceName      string         `json:"serviceName"`
	Registrations    []Registration `json:"registrations,omitempty"`
	VnodeCount       uint           `json:"vnodeCount"`
	PingFunc         func() error   `json:"-"`
}

func (o *Options) logger() logging.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}

func (o *Options) zookeeperServers() []string {
	if o != nil && len(o.ZookeeperServers) > 0 {
		return o.ZookeeperServers
	}

	return []string{DefaultZookeeperServer}
}

func (o *Options) zookeeperTimeout() time.Duration {
	if o != nil && o.ZookeeperTimeout > 0 {
		return time.Duration(o.ZookeeperTimeout)
	}

	return DefaultZookeeperTimeout
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
