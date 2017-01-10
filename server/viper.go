package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	FileFlagName      = "file"
	FileFlagShorthand = "f"
)

// ConfigureFlagSet adds the standard set of WebPA flags to the supplied FlagSet.  Use of this function
// is optional, and necessary only if the standard flags should be supported.  However, this is highly desirable,
// as ConfigureViper can make use of the standard flags to tailor how configuration is loaded.
func ConfigureFlagSet(applicationName string, f *pflag.FlagSet) {
	f.StringP(FileFlagName, FileFlagShorthand, applicationName, "base name of the configuration file")
}

// ConfigureViper configures a Viper instances using the opinionated WebPA settings.  All WebPA servers should
// use this function.
//
// The flagSet is optional.  If supplied, it will be bound to the given Viper instance.  Additionally, if the
// flagSet has a FileFlagName flag, it will be used as the configuration name to hunt for instead of the
// application name.
func ConfigureViper(applicationName string, f *pflag.FlagSet, v *viper.Viper) error {
	v.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	v.AddConfigPath(fmt.Sprintf("$HOME/.%s", applicationName))
	v.AddConfigPath(".")

	v.SetEnvPrefix(applicationName)
	v.AutomaticEnv()

	v.SetDefault("name", applicationName)
	v.SetDefault("address", DefaultAddress)
	v.SetDefault("healthAddress", DefaultHealthAddress)
	v.SetDefault("healthLogInterval", DefaultHealthLogInterval)

	if f != nil {
		if fileFlag := f.Lookup(FileFlagName); fileFlag != nil {
			// use the command-line to specify the base name of the file to be searched for
			v.SetConfigName(fileFlag.Value.String())
		} else {
			v.SetConfigName(applicationName)
		}

		if err := v.BindPFlags(f); err != nil {
			return err
		}
	} else {
		v.SetConfigName(applicationName)
	}

	return nil
}

/*
Configure is a one-stop shopping function for reading in WebPA configuration.  Typical usage is:

    var (
      f = pflag.NewFlagSet()
      v = viper.New()
    )

    if err := server.Configure("petasos", os.Args, f, v); err != nil {
      // deal with the error, possibly just exiting
    }
*/
func Configure(applicationName string, arguments []string, f *pflag.FlagSet, v *viper.Viper) error {
	if f != nil {
		ConfigureFlagSet(applicationName, f)
		if err := f.Parse(arguments); err != nil {
			return err
		}
	}

	if err := ConfigureViper(applicationName, f, v); err != nil {
		return err
	}

	return nil
}
