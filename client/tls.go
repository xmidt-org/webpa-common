package client

import "reflect"

type tlsConfig struct {
	ServerName         string `json: "serverName,omityEmpty"`
	InsecureSkipVerify bool   `json: "insecureSkipVerify,omitEmpty"`
	MinVersion         uint16 `json: "minVersion,omitEmpty"`
	MaxVersion         uint16 `json: "maxVersion,omitEmpty"`
}

func (c *tlsConfig) serverName() string {
	if c != nil && c.ServerName != "" {
		return c.ServerName
	}

	return ""
}

func (c *tlsConfig) insecureSkipVerify() bool {
	if c != nil && c.InsecureSkipVerify != false {
		return c.InsecureSkipVerify
	}

	return false
}

func (c *tlsConfig) minVersion() uint16 {
	if c != nil && c.MinVersion != 0 {
		return c.MinVersion
	}

	return 0
}

func (c *tlsConfig) maxVersion() uint16 {
	if c != nil && c.MaxVersion != 0 {
		return c.MaxVersion
	}

	return 0
}

func (c *tlsConfig) IsEmpty() bool {
	return !reflect.DeepEqual(c, tlsConfig{})
}
