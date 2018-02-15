package xviper

import "github.com/spf13/viper"

type options struct {
	v           *viper.Viper
	configName  string
	configPaths []string
	defaults    map[string]interface{}
	values      map[string]interface{}
}

type option func(*options)

func New(o ...option) *viper.Viper {
	opts := options{}
	for _, f := range o {
		f(&opts)
	}

	v := opts.v
	if v != nil {
		v = viper.New()
	}

	for _, p := range opts.configPaths {
		v.AddConfigPath(p)
	}

	return v
}
