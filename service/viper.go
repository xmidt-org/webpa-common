package service

import (
	"github.com/spf13/viper"
)

const (
	// ServiceKey is the expected Viper subkey containing service discovery configuration
	ServiceKey = "service"
)

// Sub returns the standard Viper subconfiguration for service discovery.
// If this function is passed nil, it returns nil.
func Sub(v *viper.Viper) *viper.Viper {
	if v != nil {
		return v.Sub(ServiceKey)
	}

	return nil
}

// FromViper returns an Options from a Viper environment.  This function accepts nil,
// in which case a non-nil default Options instance is returned.
func FromViper(v *viper.Viper) (*Options, error) {
	o := new(Options)
	if v != nil {
		if err := v.Unmarshal(o); err != nil {
			return nil, err
		}
	}

	return o, nil
}
