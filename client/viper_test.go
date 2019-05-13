package client

import (
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/viper"
)

// Tests if configurations are correctly built with viper in two cases:
// 1. With a filled configuraiton
// 2. With no configuration at all
func TestViperConfiguration(t *testing.T) {
	t.Run("TestWithConfiguration", testViperToClientConfig)
	t.Run("TestWithNoConfiguration", testViperToClientConfigDefaults)
}

func testViperToClientConfig(t *testing.T) {
	var (
		v    = viper.New()
		s, _ = os.Getwd()
	)

	v.SetConfigName("config-example")
	v.AddConfigPath(s)

	if err := v.ReadInConfig(); err != nil {
		t.Errorf(err.Error())
	}

	t.Log("1: Testing clientConfig")
	clientConfig, err := viperToHTTPClientConfig(v)
	if err != nil {
		t.Errorf("Failed to create config file from viper: %v", spew.Sprint(clientConfig))
	}

	t.Log("2: Testing retryOptionsConfig")
	if ok := clientConfig.RetryOptionsConfig.IsEmpty(); ok {
		t.Errorf("Failed to create RetryOptionsConfig: %v", spew.Sprint(clientConfig.RetryOptionsConfig))
	}

	t.Log("3: Testing transportConfig")
	if ok := clientConfig.TransportConfig.IsEmpty(); ok {
		t.Errorf("Failed to create TransportConfig: %v", spew.Sprint(clientConfig.TransportConfig))
	}

	t.Log("4: Testing tlsConfig")
	if ok := clientConfig.TLSConfig.IsEmpty(); ok {
		t.Errorf("Failed to create tlsConfig: %v", spew.Sprint(clientConfig.TLSConfig))
	}

	t.Log("5: Testing redirectPolicyConfig")
	if ok := clientConfig.RedirectPolicyConfig.IsEmpty(); ok {
		t.Errorf("Failed to create redirectPolicyConfig: %v", spew.Sprint(clientConfig.RedirectPolicyConfig))
	}
}

func testViperToClientConfigDefaults(t *testing.T) {
	var (
		v    = viper.New()
		s, _ = os.Getwd()
	)

	v.SetConfigName("config-example-defaults")
	v.AddConfigPath(s)

	if err := v.ReadInConfig(); err != nil {
		t.Errorf(err.Error())
	}

	t.Log("1: Testing clientConfig")
	clientConfig, err := viperToHTTPClientConfig(v)
	if err != nil {
		t.Errorf("Failed to create config from from defaults: %v", spew.Sprint(clientConfig))
	}

	t.Log("Testing if clientConfig has values it shouldn't")
	if ok := reflect.ValueOf(clientConfig).IsNil(); ok {
		t.Errorf("ClientConfiguration struct: %v, should be nil", spew.Sprint(clientConfig))
	}
}
