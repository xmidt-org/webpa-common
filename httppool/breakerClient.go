package httppool

import (
	"net/http"
	"time"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/rubyist/circuitbreaker"
)

func breakerEventString(e circuit.BreakerEvent) string {
	switch e {
	case circuit.BreakerTripped:
		return "tripped"
	case circuit.BreakerReset:
		return "reset"
	case circuit.BreakerFail:
		return "fail"
	case circuit.BreakerReady:
		return "ready"
	default:
		return "unknown"
	}
}

func BreakerClient(timeout time.Duration, threshold int64, logger log.Logger, delegate *http.Client) *circuit.HTTPClient {
	var (
		client   = circuit.NewHostBasedHTTPClient(timeout, threshold, delegate)
		debugLog = logging.Debug(logger, "timeout", timeout, "threshold", threshold)
	)

	// alter the breaker that's returned:
	delegateLookup := client.BreakerLookup
	tripFunc := circuit.ConsecutiveTripFunc(threshold)
	client.BreakerLookup = func(c *circuit.HTTPClient, val interface{}) *circuit.Breaker {
		breaker := delegateLookup(c, val)
		breaker.ShouldTrip = tripFunc
		return breaker
	}

	subscription := client.Panel.Subscribe()
	go func() {
		for panelEvent := range subscription {
			debugLog.Log("event", breakerEventString(panelEvent.Event), "name", panelEvent.Name)
		}
	}()

	return client
}
