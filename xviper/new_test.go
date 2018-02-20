package xviper

import (
	"strings"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"

	"github.com/spf13/viper"
)

const config = `
	{
		"test": {
			"name": "Joe",
			"age": 34
		}
	}
`

type TestConfig struct {
	Logger log.Logger
	Name   string
	Age    int
}

func TestViper(t *testing.T) {
	v := viper.New()
	v.SetConfigType("json")
	if err := v.ReadConfig(strings.NewReader(config)); err != nil {
		t.Fatalf("Unable to read config: %s", err)
		return
	}

	v.Set("logger", logging.NewTestLogger(nil, t))
	t.Logf("directly from Viper: %v", v.Get("logger"))

	s := v.Sub("test")
	s.SetDefault("logger", v.Get("logger"))

	var testConfig TestConfig
	if err := s.Unmarshal(&testConfig); err != nil {
		t.Fatalf("Unable to unmarshal config: %s", err)
		return
	}

	t.Logf("unmarshaled: %#v", testConfig)
}
