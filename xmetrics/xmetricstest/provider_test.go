package xmetricstest

import (
	"testing"

	"github.com/go-kit/kit/metrics/generic"
)

func Test(t *testing.T) {
	c := generic.NewCounter("foo")
	c.With("code", "200").Add(1.0)
	t.Logf("value: %f", c.Value())
}
