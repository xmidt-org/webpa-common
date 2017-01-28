package device

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/spf13/viper"
)

const (
	// DeviceManagerKey is the Viper subkey under which device.Options are typically stored
	DeviceManagerKey = "deviceManager"
)

// NewOptions unmarshals a device.Options from a Viper environment.  Listeners
// must be configured separately.
func NewOptions(logger logging.Logger, v *viper.Viper) (o *Options, err error) {
	o = new(Options)
	if v != nil {
		err = v.Unmarshal(o)
	}

	o.Logger = logger
	return
}
