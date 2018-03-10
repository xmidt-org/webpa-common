package servicecfg

import (
	"github.com/Comcast/webpa-common/service"
	"github.com/Comcast/webpa-common/service/zk"
)

// Options contains the superset of all necessary options for initializing service discovery.
type Options struct {
	VnodeCount    int  `json:"vnodeCount,omitempty"`
	DisableFilter bool `json:"disableFilter"`

	Fixed     []string    `json:"fixed,omitempty"`
	Zookeeper *zk.Options `json:"zookeeper,omitempty"`
	Consul    interface{} `json:"consul,omitempty"`
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