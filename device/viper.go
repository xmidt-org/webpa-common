package device

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/spf13/viper"
)

func NewOptions(logger logging.Logger, v *viper.Viper) (o *Options, err error) {
	o = new(Options)
	if v != nil {
		err = v.Unmarshal(o)
	}

	o.Logger = logger
	return
}
