package client

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
)

func TestNewClient(t *testing.T) {
	var (
		transportConfig = &TransportConfig{
			TLSHandshakeTimeout:    30,
			DisableKeepAlives:      true,
			DisableCompression:     true,
			MaxIdleConns:           4,
			MaxIdleConnsPerHost:    4,
			MaxConnsPerHost:        4,
			IdleConnTimeOut:        30,
			ResponseHeaderTimeOut:  4,
			ExpectContinueTimeOut:  4,
			MaxResponseHeaderBytes: 2,
		}

		retryConfig = &RetryOptionsConfig{
			Retries:  10,
			Interval: 10,
		}

		tlsConfig = &tlsConfig{
			ServerName:         "webpaNode",
			InsecureSkipVerify: true,
			MinVersion:         12,
			MaxVersion:         13,
		}

		clientConfig = &ClientConfig{
			TimeOut: 3,
		}

		config = &HTTPClientConfig{
			TransportConfig:    transportConfig,
			RetryOptionsConfig: retryConfig,
			TLSConfig:          tlsConfig,
			ClientConfig:       clientConfig,
		}
	)

	_, err := config.NewClient()
	if err != nil {
		t.Errorf("Error creating client from:  %v", spew.Sprint(config))
	}
}

func TestNewTransactor(t *testing.T) {
	var (
		transportConfig = &TransportConfig{
			TLSHandshakeTimeout:    30,
			DisableKeepAlives:      true,
			DisableCompression:     true,
			MaxIdleConns:           4,
			MaxIdleConnsPerHost:    4,
			MaxConnsPerHost:        4,
			IdleConnTimeOut:        30,
			ResponseHeaderTimeOut:  4,
			ExpectContinueTimeOut:  4,
			MaxResponseHeaderBytes: 2,
		}

		retryConfig = &RetryOptionsConfig{
			Retries:  10,
			Interval: 10,
		}

		tlsConfig = &tlsConfig{
			ServerName:         "webpaNode",
			InsecureSkipVerify: true,
			MinVersion:         12,
			MaxVersion:         13,
		}

		clientConfig = &ClientConfig{
			TimeOut: 3,
		}

		config = &HTTPClientConfig{
			TransportConfig:    transportConfig,
			RetryOptionsConfig: retryConfig,
			TLSConfig:          tlsConfig,
			ClientConfig:       clientConfig,
		}
	)

	_, err := config.NewTransactor()
	if err != nil {
		t.Errorf("Failed making transactor from: %v", spew.Sprint(config))
	}
}
