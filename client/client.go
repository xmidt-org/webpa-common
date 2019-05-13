package client

import (
	"crypto/tls"
	"net"
	"net/http"
	"reflect"
)

type HTTPClientConfig struct {
	TransportConfig      *TransportConfig                           `json:"-"`
	RetryOptionsConfig   *RetryOptionsConfig                        `json:"-"`
	TLSConfig            *tlsConfig                                 `json:"-"`
	ClientConfig         *ClientConfig                              `json:"-"`
	RedirectPolicyConfig *RedirectPolicyConfig                      `json:"-"`
	DialerConfig         *DialerConfig                              `json:"-"`
	CheckRedirect        func(*http.Request, *[]http.Request) error `json:"-"`
}

func (c *HTTPClientConfig) NewClient() *http.Client {
	return createHTTPClient(c)
}

/*
func (c *HTTPClientConfig) NewTransactor(om OutboundMeasures, or OutboundMetricOptions) (func(*http.Request) (*http.Response, error), error) {
	ci := createHTTPClient(c)
	return ci.Do, nil
}
/*
*/
func createHTTPClient(c *HTTPClientConfig) *http.Client {
	client := new(http.Client)

	ok := c.ClientConfig.IsEmpty()
	if !ok {
		client.Timeout = c.ClientConfig.timeOut()
	}

	ok = c.RedirectPolicyConfig.IsEmpty()
	if !ok {
		client.CheckRedirect = c.RedirectPolicyConfig.checkRedirect()
	}

	ok = c.TransportConfig.IsEmpty()
	if !ok {
		transport := &http.Transport{
			TLSHandshakeTimeout:    c.TransportConfig.tlsHandShakeTimeout(),
			DisableKeepAlives:      c.TransportConfig.disableKeepAlives(),
			DisableCompression:     c.TransportConfig.disableCompression(),
			MaxIdleConns:           c.TransportConfig.maxIdleConns(),
			MaxIdleConnsPerHost:    c.TransportConfig.maxIdleConnsPerHost(),
			MaxConnsPerHost:        c.TransportConfig.maxConnsPerHost(),
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

		ok = c.DialerConfig.IsEmpty()
		if !ok {
			dialer := (&net.Dialer{
				Timeout:       c.DialerConfig.timeOut(),
				FallbackDelay: c.DialerConfig.fallbackDelay(),
				KeepAlive:     c.DialerConfig.keepAlive(),
			}).Dial

			transport.Dial = dialer
		}

		client.Transport = http.RoundTripper(transport)
	}

	return client
}

func (c *HTTPClientConfig) IsEmpty() bool {
	return reflect.DeepEqual(c, (ClientConfig{}))
}
