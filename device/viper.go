package device

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	// DeviceManagerKey is the Viper subkey under which device.Options are typically stored
	// In a JSON configuration file, this will be expressed as:
	//
	//   {
	//     /* other stuff can be here */
	//
	//     "device": {
	//       "manager": {
	//       }
	//     }
	//   }
	DeviceManagerKey = "device.manager"
)

// NewOptions unmarshals a device.Options from a Viper environment.  Listeners
// must be configured separately.
func NewOptions(logger *zap.Logger, v *viper.Viper) (o *Options, err error) {
	o = new(Options)
	if v != nil {
		err = v.Unmarshal(o)
	}

	o.Logger = logger
	return
}
