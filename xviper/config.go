package xviper

import (
	"fmt"

	"github.com/spf13/pflag"
)

// Configer is the subset of Viper behavior dealing with configuration paths and locations
type Configer interface {
	AddConfigPath(string)
	SetConfigName(string)
	SetConfigFile(string)
}

// AddStandardConfigPaths adds the standard *nix-style configuration paths
func AddStandardConfigPaths(c Configer, applicationName string) {
	c.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	c.AddConfigPath(fmt.Sprintf("$HOME/%s", applicationName))
	c.AddConfigPath(".")
}

// FlagLookup is the behavior expected of a pflag.FlagSet to lookup individual flags by longhand name.
type FlagLookup interface {
	Lookup(string) *pflag.Flag
}

// BindConfigName extracts the name of the Viper configuration file from a flagset.  If the given flag
// is set, its value is passed to c.SetConfigName and this function returns true.  If the flag was missing,
// this method returns false and the supplied Configer is not changed.
//
// This function is useful to allow the name of the file that Viper searches for to be specified or
// overridden from the command line.
func BindConfigName(c Configer, fl FlagLookup, flag string) bool {
	if f := fl.Lookup(flag); f != nil {
		configName := f.Value.String()
		if len(configName) > 0 {
			c.SetConfigName(configName)
			return true
		}
	}

	return false
}

// BindConfigName extracts the path of the Viper configuration file from a flagset.  If the given flag
// is set, its value is passed to c.SetConfigFile and this function returns true.  If the flag was missing,
// this method returns false and the supplied Configer is not changed.
//
// This function is useful to allow the fully-qualified path of the file that Viper uses to be specified
// or overridden from the command line.
func BindConfigFile(c Configer, fl FlagLookup, flag string) bool {
	if f := fl.Lookup(flag); f != nil {
		configFile := f.Value.String()
		if len(configFile) > 0 {
			c.SetConfigFile(configFile)
			return true
		}
	}

	return false
}

// BindConfig attempts first to bind the configuration file via BindConfigFile.  Failing that, it
// attempts to bind the configuration name via BindConfigName.  If either succeeds, this function
// returns true.  Otherwise, if no binding took place, this function returns false.
func BindConfig(c Configer, fl FlagLookup, fileFlag, nameFlag string) bool {
	result := BindConfigFile(c, fl, fileFlag)
	if !result {
		result = BindConfigName(c, fl, nameFlag)
	}

	return result
}
