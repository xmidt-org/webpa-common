package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"os"
	"testing"

	"github.com/Comcast/webpa-common/xhttp"
	DE "github.com/go-test/deep"
	"github.com/spf13/viper"
)

// TestNewClient tests the two cases when building a http.Client from a ClientConfig struct
// 1. Sufficient fields are filled with a configuration
// 2. Sufficient fields (defaults) are filled with out a configuration
func TestNewClient(t *testing.T) {
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
		checkRedirect = xhttp.CheckRedirect(xhttp.RedirectPolicy{
			MaxRedirects:   5,
			ExcludeHeaders: []string{"test1", "test2", "test3"},
		})

		expectedClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					ServerName:         "wes",
					InsecureSkipVerify: true,
					MinVersion:         0x0300,
					MaxVersion:         0x0304,
				},
				Dial: (&net.Dialer{
					Timeout:       5,
					FallbackDelay: 5,
					KeepAlive:     5,
				}).Dial,
				TLSHandshakeTimeout:    5,
				DisableKeepAlives:      true,
				DisableCompression:     true,
				MaxIdleConns:           5,
				MaxIdleConnsPerHost:    5,
				MaxConnsPerHost:        5,
				IdleConnTimeout:        5,
				ResponseHeaderTimeout:  5,
				ExpectContinueTimeout:  5,
				MaxResponseHeaderBytes: 5,
			},
			Timeout:       5,
			CheckRedirect: checkRedirect,
		}
	)

	clientConfig, _ := viperToHTTPClientConfig(v)
	actualClient := clientConfig.NewClient()

	if diff := DE.Equal(actualClient, expectedClient); diff != nil {
		t.Error(diff)
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
				TLSClientConfig: &tls.Config{
					ServerName:         "",
					InsecureSkipVerify: false,
					MinVersion:         0,
					MaxVersion:         0,
				},
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

	if diff := DE.Equal(actualClient, expectedClient); diff != nil {
		t.Error(diff)
	}
}
