package servicecfg

import (
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/service/consul"
	"github.com/xmidt-org/webpa-common/service/zk"
)

// Options contains the superset of all necessary options for initializing service discovery.
type Options struct {
	VnodeCount    int    `json:"vnodeCount,omitempty"`
	DisableFilter bool   `json:"disableFilter"`
	DefaultScheme string `json:"defaultScheme"`

	Fixed     []string        `json:"fixed,omitempty"`
	Zookeeper *zk.Options     `json:"zookeeper,omitempty"`
	Consul    *consul.Options `json:"consul,omitempty"`
}

func (o *Options) vnodeCount() int {
	if o != nil && o.VnodeCount > 0 {
		return o.VnodeCount
	}

	return service.DefaultVnodeCount
}

func (o *Options) disableFilter() bool {
	if o != nil {
		return o.DisableFilter
	}

	return false
}

func (o *Options) defaultScheme() string {
	if o != nil && len(o.DefaultScheme) > 0 {
		return o.DefaultScheme
	}

	return service.DefaultScheme
}
