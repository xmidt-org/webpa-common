package client

import (
	"context"
	"crypto/tls"
	"net"
	"reflect"
	"time"
)

type TransportConfig struct {
	Dial                   func(string, string) (net.Conn, error)                  `json: "-"`
	DialContext            func(context.Context, string, string) (net.Conn, error) `json: "-"`
	TLSClientConfig        *tls.Config                                             `json: "-"`
	TLSHandshakeTimeout    time.Duration                                           `json: "tlsHandshakeTimeout,omitempty"`
	DisableKeepAlives      bool                                                    `json: "disableKeepAlives,omitempty"`
	DisableCompression     bool                                                    `json: "disableCompression,omitempty"`
	MaxIdleConns           int                                                     `json: "maxIdleConns,omitempty"`
	MaxIdleConnsPerHost    int                                                     `json: "maxIdleConnsPerHost,omitempty"`
	MaxConnsPerHost        int                                                     `json: "maxConnsPerHost,omitempty"`
	IdleConnTimeOut        time.Duration                                           `json: "idleConnTimeOut,omitempty"`
	ResponseHeaderTimeOut  time.Duration                                           `json: "responseHeaderTimeOut,omitempty"`
	ExpectContinueTimeOut  time.Duration                                           `json: "expectContinueTimeOut,omitempty"`
	MaxResponseHeaderBytes int64                                                   `json: "maxResponseHeaderBytes,omitempty"`
}

func (c *TransportConfig) maxConnsPerHost() int {
	if c != nil && c.MaxConnsPerHost > 0 {
		return c.MaxConnsPerHost
	}

	return 0
}

func (c *TransportConfig) tlsHandShakeTimeout() time.Duration {
	if c != nil && c.TLSHandshakeTimeout > 0 {
		return c.TLSHandshakeTimeout
	}

	return 0
}

func (c *TransportConfig) disableKeepAlives() bool {
	if c != nil && c.DisableKeepAlives != false {
		return c.DisableKeepAlives
	}

	return false
}

func (c *TransportConfig) disableCompression() bool {
	if c != nil && c.DisableCompression != false {
		return c.DisableCompression
	}

	return false
}

func (c *TransportConfig) maxIdleConns() int {
	if c != nil && c.MaxIdleConns != 0 {
		return c.MaxIdleConns
	}

	return 0
}

func (c *TransportConfig) maxIdleConnsPerHost() int {
	if c != nil && c.MaxIdleConnsPerHost != 0 {
		return c.MaxIdleConnsPerHost
	}

	return 0
}

func (c *TransportConfig) idleConnTimeOut() time.Duration {
	if c != nil && c.IdleConnTimeOut > 0 {
		return c.IdleConnTimeOut
	}

	return 0
}

func (c *TransportConfig) responseHeaderTimeOut() time.Duration {
	if c != nil && c.ResponseHeaderTimeOut > 0 {
		return c.ResponseHeaderTimeOut
	}

	return 0
}

func (c *TransportConfig) expectContinueTimeOut() time.Duration {
	if c != nil && c.ExpectContinueTimeOut > 0 {
		return c.ExpectContinueTimeOut
	}

	return 0
}

func (c *TransportConfig) maxResponseHeaderBytes() int64 {
	if c != nil && c.MaxResponseHeaderBytes != 0 {
		return c.MaxResponseHeaderBytes
	}

	return 0
}

func (c *TransportConfig) IsEmpty() bool {
	return reflect.DeepEqual(c, (TransportConfig{}))
}
