package client

import (
	"reflect"
	"time"
)

type ClientConfig struct {
	TimeOut time.Duration `json: "timeOut,omitEmpty"`
}

func (c *ClientConfig) timeOut() time.Duration {
	if c != nil && c.TimeOut > 0 {
		return c.TimeOut
	}

	return 0
}

func (c *ClientConfig) IsEmpty() bool {
	return !reflect.DeepEqual(c, ClientConfig{})
}
