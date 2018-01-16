package aws

import (
	"github.com/Comcast/webpa-common/xmetrics"
)

func makeTestRegistry() xmetrics.Registry {
	o := &xmetrics.Options{}
	registry, _ := xmetrics.NewRegistry(o)
	
	return registry
}