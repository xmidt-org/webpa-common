package aws

import (
	"github.com/Comcast/webpa-common/xmetrics"
)

func MakeTestRegistry() xmetrics.Registry {
	o := &xmetrics.Options{}
	registry, _ := xmetrics.NewRegistry(o)
	
	return registry
}