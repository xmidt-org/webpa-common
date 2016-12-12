package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

// NewViper produces a Viper instance configured with WebPA conventions.
// The applicationName is used as the configuration file name, the environment prefix,
// and  to generate the path under /etc and $HOME to look for configuration files.
// Automatic environment mode is turned on.
func NewViper(applicationName string) *viper.Viper {
	viper := viper.New()
	viper.SetConfigName(applicationName)
	viper.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.%s", applicationName))
	viper.AddConfigPath(".")

	viper.SetEnvPrefix(applicationName)
	viper.AutomaticEnv()

	return viper
}

// ParseAndBind parses the given flag set using the supplied arguments and then binds
// the flag set to the specified Viper instance.  If arguments is nil, os.Args is used instead.
func ParseAndBind(viper *viper.Viper, flagSet *pflag.FlagSet, arguments []string) error {
	if arguments == nil {
		arguments = os.Args
	}

	if err := flagSet.Parse(arguments); err != nil {
		return err
	}

	viper.BindPFlags(flagSet)
	return nil
}
