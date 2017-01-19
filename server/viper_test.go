package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func ExampleNew() {
	webPA, err := New("example", nil, nil, viper.New())
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

func ExampleNewWithFlags() {
	var (
		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		// simulates passing `-f example` on the command line
		webPA, err = New("applicationName", []string{"-f", "example"}, f, v)
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

func TestNewWhenConfigureError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		webPA, err = New("applicationName", []string{"-unknown"}, f, v)
	)

	assert.Nil(webPA)
	assert.NotNil(err)
}

func TestNewWhenReadInConfigError(t *testing.T) {
	var (
		assert = assert.New(t)

		f = pflag.NewFlagSet("applicationName", pflag.ContinueOnError)
		v = viper.New()

		webPA, err = New("applicationName", []string{"-f", "thisfiledoesnotexist"}, f, v)
	)

	assert.Nil(webPA)
	assert.NotNil(err)
}
