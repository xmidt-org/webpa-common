package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/logging/golog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	FileFlagName      = "file"
	FileFlagShorthand = "f"

	// LogKey is the standard Viper subkey which is expected to contain log configuration
	LogKey = "log"
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
func ConfigureViper(applicationName string, f *pflag.FlagSet, v *viper.Viper) (configName string, err error) {
	v.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	v.AddConfigPath(fmt.Sprintf("$HOME/.%s", applicationName))
	v.AddConfigPath(".")

	v.SetEnvPrefix(applicationName)
	v.AutomaticEnv()

	v.SetDefault("name", applicationName)
	v.SetDefault("address", DefaultAddress)
	v.SetDefault("healthAddress", DefaultHealthAddress)
	v.SetDefault("healthLogInterval", DefaultHealthLogInterval)

	configName = applicationName
	if f != nil {
		if fileFlag := f.Lookup(FileFlagName); fileFlag != nil {
			// use the command-line to specify the base name of the file to be searched for
			configName = fileFlag.Value.String()
		}

		err = v.BindPFlags(f)
	}

	v.SetConfigName(configName)
	return
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

Usage of this function is only necessary if custom configuration is needed.  Normally,
using New will suffice.
*/
func Configure(applicationName string, arguments []string, f *pflag.FlagSet, v *viper.Viper) (configName string, err error) {
	if f != nil {
		ConfigureFlagSet(applicationName, f)
		err = f.Parse(arguments)
		if err != nil {
			return
		}
	}

	configName, err = ConfigureViper(applicationName, f, v)
	return
}

// NewWebPA creates a WebPA instance from a Viper configuration.  The supplied Viper instance
// may be nil, in which case a default WebPA instance is returned.
func NewWebPA(v *viper.Viper) (webPA *WebPA, err error) {
	webPA = new(WebPA)
	if v != nil {
		err = v.Unmarshal(webPA)
	}

	return
}

// NewLoggerFactory creates a LoggerFactory from a Viper configuration.  The supplied Viper instance
// may be nil, in which case a default LoggerFactory is returned.
func NewLoggerFactory(v *viper.Viper) (loggerFactory logging.LoggerFactory, err error) {
	loggerFactory = new(golog.LoggerFactory)
	if v != nil {
		err = v.Unmarshal(loggerFactory)
	}

	return
}

/*
New is the primary constructor for this package.  It configures Viper and unmarshals the
appropriate objects.

    var (
      f = pflag.NewFlagSet()
      v = viper.New()
      webPA, loggerFactory, err = server.New("petasos", os.Args, f, v)
    )

    if err != nil {
      // deal with the error, possibly just exiting
    }

This function is typically all that's needed to fully use this package for a WebPA server.

As with the other NewXXX functions, this function permits a nil Viper instance.
*/
func New(applicationName string, arguments []string, f *pflag.FlagSet, v *viper.Viper) (webPA *WebPA, loggerFactory logging.LoggerFactory, err error) {
	var logViper *viper.Viper
	if v != nil {
		_, err = Configure(applicationName, arguments, f, v)
		if err != nil {
			return
		}

		err = v.ReadInConfig()
		if err != nil {
			return
		}

		logViper = v.Sub(LogKey)
	}

	webPA, err = NewWebPA(v)
	if err != nil {
		return
	}

	loggerFactory, err = NewLoggerFactory(logViper)
	return
}
