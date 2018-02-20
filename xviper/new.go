package xviper

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	DefaultNameFlag = "name"
	DefaultFileFlag = "file"
)

type option func(*viper.Viper) error

func AddConfigPaths(paths ...string) option {
	return func(v *viper.Viper) error {
		for _, p := range paths {
			v.AddConfigPath(p)
		}

		return nil
	}
}

func SetEnvPrefix(prefix string) option {
	return func(v *viper.Viper) error {
		v.SetEnvPrefix(prefix)
		return nil
	}
}

func SetConfigName(name string) option {
	return func(v *viper.Viper) error {
		v.SetConfigName(name)
		return nil
	}
}

func SetConfigFile(file string) option {
	return func(v *viper.Viper) error {
		v.SetConfigFile(file)
		return nil
	}
}

func AutomaticEnv(v *viper.Viper) error {
	v.AutomaticEnv()
	return nil
}

func BindPFlags(fs *pflag.FlagSet) option {
	return func(v *viper.Viper) error {
		return v.BindPFlags(fs)
	}
}

func BindConfigName(fs *pflag.FlagSet, flag string) option {
	return func(v *viper.Viper) error {
		if f := fs.Lookup(flag); f != nil {
			configName := f.Value.String()
			if len(configName) > 0 {
				v.SetConfigName(configName)
			}
		}

		return nil
	}
}

func BindConfigFile(fs *pflag.FlagSet, flag string) option {
	return func(v *viper.Viper) error {
		if f := fs.Lookup(flag); f != nil {
			configFile := f.Value.String()
			if len(configFile) > 0 {
				v.SetConfigFile(configFile)
			}
		}

		return nil
	}
}

func StdOptions(applicationName string, fs *pflag.FlagSet) option {
	return func(v *viper.Viper) error {
		err := AddConfigPaths(
			fmt.Sprintf("/etc/%s", applicationName),
			fmt.Sprintf("$HOME/.%s", applicationName),
			".",
		)(v)

		if err == nil {
			err = SetEnvPrefix(applicationName)(v)
		}

		if err == nil {
			err = AutomaticEnv(v)
		}

		if err == nil {
			err = SetConfigName(applicationName)(v)
		}

		if err == nil {
			err = BindPFlags(fs)(v)
		}

		return err
	}
}

func New(o ...option) (*viper.Viper, error) {
	return Configure(viper.New(), o...)
}

func Configure(v *viper.Viper, o ...option) (*viper.Viper, error) {
	if v != nil {
		for _, f := range o {
			if err := f(v); err != nil {
				return nil, err
			}
		}
	}

	return v, nil
}
