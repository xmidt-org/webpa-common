package server

import (
	"fmt"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func ExampleInitialize() {
	_, _, webPA, err := Initialize("example", nil, nil, viper.New())
	if err != nil {
		panic(err)
	}

	fmt.Println(webPA.Primary.Name)
	fmt.Println(webPA.Primary.Address)
	fmt.Println(webPA.Primary.LogConnectionState)

	fmt.Println(webPA.Alternate.Name)
	fmt.Println(webPA.Alternate.Address)
	fmt.Println(webPA.Alternate.LogConnectionState)

	fmt.Println(webPA.Health.Name)
	fmt.Println(webPA.Health.Address)
	fmt.Println(webPA.Health.LogInterval)
	fmt.Println(webPA.Health.Options)

	// Output:
	// example
	// localhost:10010
	// true
	// example.alternate
	// :10011
	// false
	// example.health
	// :9001
	// 45s
	// [TotalRequests TotalResponses SomeOtherStat]
}

func ExampleInitializeWithFlags() {
	var (
		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		// simulates passing `-f example` on the command line
		_, _, webPA, err = Initialize("applicationName", []string{"-f", "example"}, f, v)
	)

	if err != nil {
		panic(err)
	}

	fmt.Println(webPA.Primary.Name)
	fmt.Println(webPA.Primary.Address)
	fmt.Println(webPA.Primary.LogConnectionState)

	fmt.Println(webPA.Alternate.Name)
	fmt.Println(webPA.Alternate.Address)
	fmt.Println(webPA.Alternate.LogConnectionState)

	fmt.Println(webPA.Health.Name)
	fmt.Println(webPA.Health.Address)
	fmt.Println(webPA.Health.LogInterval)
	fmt.Println(webPA.Health.Options)

	// Output:
	// applicationName
	// localhost:10010
	// true
	// applicationName.alternate
	// :10011
	// false
	// applicationName.health
	// :9001
	// 45s
	// [TotalRequests TotalResponses SomeOtherStat]
}

func TestConfigureWhenParseError(t *testing.T) {
	var (
		assert = assert.New(t)

		f   = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v   = viper.New()
		err = Configure("applicationName", []string{"-unknown"}, f, v)
	)

	assert.NotNil(err)
}

func TestInitializeWhenConfigureError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		logger, registry, webPA, err = Initialize("applicationName", []string{"-unknown"}, f, v)
	)

	assert.NotNil(logger)
	assert.Nil(registry)
	assert.Nil(webPA)
	assert.NotNil(err)
}

func TestInitializeWhenReadInConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		logger, registry, webPA, err = Initialize("applicationName", []string{"-f", "thisfiledoesnotexist"}, f, v)
	)

	assert.NotNil(logger)
	assert.Nil(registry)
	assert.Nil(webPA)
	assert.NotNil(err)
}

func TestInitializeWhenWebPAUnmarshalError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("invalidWebPA", pflag.ContinueOnError)
		v = viper.New()

		logger, registry, webPA, err = Initialize("invalidWebPA", nil, f, v)
	)

	assert.NotNil(logger)
	assert.Nil(registry)
	assert.Nil(webPA)
	assert.NotNil(err)
}

func TestInitializeWhenWebPANewLoggerError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("invalidLog", pflag.ContinueOnError)
		v = viper.New()

		logger, registry, webPA, err = Initialize("invalidLog", nil, f, v)
	)

	assert.NotNil(logger)
	assert.NotNil(registry)
	assert.NotNil(webPA)
	assert.Nil(err)
}
