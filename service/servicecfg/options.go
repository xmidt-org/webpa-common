// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package servicecfg

import (
	"github.com/xmidt-org/webpa-common/v2/service"
	"github.com/xmidt-org/webpa-common/v2/service/accessor"
	"github.com/xmidt-org/webpa-common/v2/service/consul"
	"github.com/xmidt-org/webpa-common/v2/service/zk"
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

	return accessor.DefaultVnodeCount
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
