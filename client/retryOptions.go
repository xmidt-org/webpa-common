package client

import (
	"log"
	"reflect"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/go-kit/kit/metrics"
)

type RetryOptionsConfig struct {
	Logger      log.Logger            `json: "-"`
	ShouldRetry xhttp.ShouldRetryFunc `json: "-"`
	Counter     metrics.Counter       `json: "-"`
	Retries     int                   `json: "retries,omitempty"`
	Interval    time.Duration         `json: "interval,omitempty"`
}

func (c *RetryOptionsConfig) retries() int {
	if c != nil && c.Retries > 0 {
		return c.Retries
	}

	return 0
}

func (c *RetryOptionsConfig) interval() time.Duration {
	if c != nil && c.Interval > 0 {
		return c.Interval
	}

	return 0
}

func (c *RetryOptionsConfig) IsEmpty() bool {
	return !reflect.DeepEqual(c, RetryOptionsConfig{})
}
