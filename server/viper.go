package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"os"
)

// viper is the internal interface which *viper.Viper instances must support
type viper interface {
	SetConfigName(string)
	AddConfigPath(string)
	SetEnvPrefix(string)
	AutomaticEnv()
	BindPFlags(*pflag.FlagSet) error
	ReadInConfig() error
}

// ConfigureViper configures the given *viper.Viper parameter with the standard
// pattern used by WebPA.  If supplied, the FlagSet is used to parse the given command-line
// arguments and then is bound to the Viper instance.  If fs  != nil and arguments == nil,
// then os.Args is used instead.
func ConfigureViper(applicationName string, v viper, fs *pflag.FlagSet, arguments []string) error {
	v.SetConfigName(applicationName)
	v.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	v.AddConfigPath(fmt.Sprintf("$HOME/.%s", applicationName))
	v.AddConfigPath(".")

	v.SetEnvPrefix(applicationName)
	v.AutomaticEnv()

	if fs != nil {
		if arguments == nil {
			arguments = os.Args
		}

		if err := fs.Parse(arguments); err != nil {
			return err
		}

		if err := v.BindPFlags(fs); err != nil {
			return err
		}
	}

	return nil
}

// ReadInConfig is an analog to viper.ReadInConfig.  This function applies WebPA usage patterns
// to reading in configuration.  It invokes ConfigureViper, then uses viper.ReadInConfig.
func ReadInConfig(applicationName string, v viper, fs *pflag.FlagSet, arguments []string) error {
	if err := ConfigureViper(applicationName, v, fs, arguments); err != nil {
		return err
	}

	return v.ReadInConfig()
}
