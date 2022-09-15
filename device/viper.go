package device

import (
	"github.com/go-kit/log"
	"github.com/spf13/viper"
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
func NewOptions(logger log.Logger, v *viper.Viper) (o *Options, err error) {
	o = new(Options)
	if v != nil {
		err = v.Unmarshal(o)
	}

	o.Logger = logger
	return
}
