package client

import (
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/viper"
)

// TestInitializationDefualts test two cases when building up webPAClient
// 1. Sufficient fields are filled with a configuration
// 2. Sufficient fields (defaults) are filled with out a configuration
func TestInitializationDefaults(t *testing.T) {
	t.Run("testNewClientWithConfiguration", testNewClientWithConfiguration)
	t.Run("testNewClientWithOutConfiguration", testNewClientWithOutConfiguration)
}

func testNewClientWithConfiguration(t *testing.T) {
	var (
		v    = viper.New()
		s, _ = os.Getwd()
	)

	v.SetConfigName("config-example")
	v.AddConfigPath(s)
	_ = v.ReadInConfig()

	var (
		expectedClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: nil,
				/*
					TLSClientConfig: &tls.Config{
						serverName:
						insecureSkipVerify:
						minVersion
						maxVersion:
					},
				*/
				TLSHandshakeTimeout:    0,
				DisableKeepAlives:      false,
				MaxIdleConns:           0,
				MaxIdleConnsPerHost:    0,
				MaxConnsPerHost:        0,
				IdleConnTimeout:        0,
				ResponseHeaderTimeout:  0,
				ExpectContinueTimeout:  0,
				MaxResponseHeaderBytes: 0,
			},
			CheckRedirect: nil,
			Timeout:       0,
		}
	)

	clientConfig, _ := viperToHTTPClientConfig(v)
	actualClient := clientConfig.NewClient()

	if ok := reflect.DeepEqual(actualClient, expectedClient); !ok {
		t.Fatalf("\n\nActual: %v\n, Expected: %v\n", spew.Sdump(actualClient), spew.Sdump(expectedClient))
	}
}

func testNewClientWithOutConfiguration(t *testing.T) {
	var (
		v    = viper.New()
		s, _ = os.Getwd()
	)

	v.SetConfigName("config-example-defaults")
	v.AddConfigPath(s)
	_ = v.ReadInConfig()

	var (
		expectedClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: nil,
				/*
					TLSClientConfig: &tls.Config{
						serverName:
						insecureSkipVerify:
						minVersion
						maxVersion:
					},
				*/
				TLSHandshakeTimeout:    0,
				DisableKeepAlives:      false,
				MaxIdleConns:           0,
				MaxIdleConnsPerHost:    0,
				MaxConnsPerHost:        0,
				IdleConnTimeout:        0,
				ResponseHeaderTimeout:  0,
				ExpectContinueTimeout:  0,
				MaxResponseHeaderBytes: 0,
			},
			CheckRedirect: nil,
			Timeout:       0,
		}
	)

	clientConfig, _ := viperToHTTPClientConfig(v)
	actualClient := clientConfig.NewClient()

	if ok := reflect.DeepEqual(actualClient, expectedClient); !ok {
		t.Fatalf("\n\nActual: %v\n, Expected: %v\n", spew.Sdump(actualClient), spew.Sdump(expectedClient))
	}
}
