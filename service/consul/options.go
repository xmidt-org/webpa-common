package consul

import (
	"github.com/Comcast/webpa-common/service"
	"github.com/hashicorp/consul/api"
)

type Watch struct {
	Service     string   `json:"service,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	PassingOnly bool     `json:"passingOnly"`
}

type Options struct {
	Client             *api.Config                    `json:"client"`
	DisableGenerateID  bool                           `json:"disableGenerateID"`
	RegistrationScheme string                         `json:"registrationScheme"`
	Registrations      []api.AgentServiceRegistration `json:"registrations,omitempty"`
	Watches            []Watch                        `json:"watches,omitempty"`
}

func (o *Options) config() *api.Config {
	if o != nil && o.Client != nil {
		return o.Client
	}

	return api.DefaultConfig()
}

func (o *Options) disableGenerateID() bool {
	if o != nil {
		return o.DisableGenerateID
	}

	return false
}

func (o *Options) registrationScheme() string {
	if o != nil && len(o.RegistrationScheme) > 0 {
		return o.RegistrationScheme
	}

	return service.DefaultScheme
}

func (o *Options) registrations() []api.AgentServiceRegistration {
	if o != nil && len(o.Registrations) > 0 {
		return o.Registrations
	}

	return nil
}

func (o *Options) watches() []Watch {
	if o != nil && len(o.Watches) > 0 {
		return o.Watches
	}

	return nil
}
