package service

import (
	"github.com/strava/go.serversets"
)

const (
	DefaultZookeeper   = "localhost:2181"
	DefaultEnvironment = serversets.Local
	DefaultServiceName = "test"
)

// Options represents the set of configurable attributes for service discovery and registration
type Options struct {
	Zookeepers    []string     `json:"zookeepers"`
	Environment   string       `json:"environment"`
	ServiceName   string       `json:"serviceName"`
	Registrations []string     `json:"registrations,omitempty"`
	PingFunc      func() error `json:"-"`
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
