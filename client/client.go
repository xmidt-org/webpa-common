package client

import (
	"crypto/tls"
	"net/http"
)

type HTTPClientConfig struct {
	TransportConfig    *TransportConfig                           `json:"-"`
	RetryOptionsConfig *RetryOptionsConfig                        `json:"-"`
	TLSConfig          *tlsConfig                                 `json:"-"`
	ClientConfig       *ClientConfig                              `json:"-"`
	CheckRedirect      func(*http.Request, *[]http.Request) error `json:"-"`
}

func (c *HTTPClientConfig) NewClient() (*http.Client, error) {
	client := new(http.Client)

	ok := c.ClientConfig.IsEmpty()
	if !ok {
		client.Timeout = c.ClientConfig.timeOut()
	}

	ok = c.TransportConfig.IsEmpty()
	if !ok {
		transport := &http.Transport{
			TLSHandshakeTimeout:    c.TransportConfig.tlsHandShakeTimeout(),
			DisableKeepAlives:      c.TransportConfig.disableKeepAlives(),
			DisableCompression:     c.TransportConfig.disableCompression(),
			MaxIdleConns:           c.TransportConfig.maxIdleConns(),
			MaxIdleConnsPerHost:    c.TransportConfig.maxIdleConnsPerHost(),
			IdleConnTimeout:        c.TransportConfig.idleConnTimeOut(),
			ResponseHeaderTimeout:  c.TransportConfig.responseHeaderTimeOut(),
			ExpectContinueTimeout:  c.TransportConfig.expectContinueTimeOut(),
			MaxResponseHeaderBytes: c.TransportConfig.maxResponseHeaderBytes(),
		}

		ok = c.TLSConfig.IsEmpty()
		if !ok {
			tls := &tls.Config{
				ServerName:         c.TLSConfig.serverName(),
				InsecureSkipVerify: c.TLSConfig.insecureSkipVerify(),
				MinVersion:         c.TLSConfig.minVersion(),
				MaxVersion:         c.TLSConfig.maxVersion(),
			}

			transport.TLSClientConfig = tls
		}

		client.Transport = http.RoundTripper(transport)

		return client, nil
	}

	return client, nil
}

func (c *HTTPClientConfig) NewTransactor() (func(*http.Request) (*http.Response, error), error) {
	client := new(http.Client)

	ok := c.ClientConfig.IsEmpty()
	if !ok {
		client.Timeout = c.ClientConfig.timeOut()
	}

	ok = c.TransportConfig.IsEmpty()
	if !ok {
		transport := &http.Transport{
			TLSHandshakeTimeout:    c.TransportConfig.tlsHandShakeTimeout(),
			DisableKeepAlives:      c.TransportConfig.disableKeepAlives(),
			DisableCompression:     c.TransportConfig.disableCompression(),
			MaxIdleConns:           c.TransportConfig.maxIdleConns(),
			MaxIdleConnsPerHost:    c.TransportConfig.maxIdleConnsPerHost(),
			IdleConnTimeout:        c.TransportConfig.idleConnTimeOut(),
			ResponseHeaderTimeout:  c.TransportConfig.responseHeaderTimeOut(),
			ExpectContinueTimeout:  c.TransportConfig.expectContinueTimeOut(),
			MaxResponseHeaderBytes: c.TransportConfig.maxResponseHeaderBytes(),
		}

		ok = c.TLSConfig.IsEmpty()
		if !ok {
			tls := &tls.Config{
				ServerName:         c.TLSConfig.serverName(),
				InsecureSkipVerify: c.TLSConfig.insecureSkipVerify(),
				MinVersion:         c.TLSConfig.minVersion(),
				MaxVersion:         c.TLSConfig.maxVersion(),
			}

			transport.TLSClientConfig = tls
		}

		client.Transport = http.RoundTripper(transport)
	}

	return client.Do, nil
}
