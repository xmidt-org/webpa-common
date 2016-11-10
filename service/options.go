package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/strava/go.serversets"
)

const (
	DefaultZookeeper   = "localhost:2181"
	DefaultEnvironment = serversets.Local
	DefaultServiceName = "test"
	DefaultVnodeCount  = 220
)

// Options represents the set of configurable attributes for service discovery and registration
type Options struct {
	Logger        logging.Logger `json:"-"`
	Zookeepers    []string       `json:"zookeepers"`
	Environment   string         `json:"environment"`
	ServiceName   string         `json:"serviceName"`
	Registrations []string       `json:"registrations,omitempty"`
	VnodeCount    int            `json:"vnodeCount"`
	PingFunc      func() error   `json:"-"`
}

func (o *Options) logger() logging.Logger {
	if o != nil && o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}

func (o *Options) zookeepers() []string {
	if o != nil && len(o.Zookeepers) > 0 {
		return o.Zookeepers
	}

	return []string{DefaultZookeeper}
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

func (o *Options) vnodeCount() int {
	if o != nil && o.VnodeCount > 0 {
		return o.VnodeCount
	}

	return DefaultVnodeCount
}
