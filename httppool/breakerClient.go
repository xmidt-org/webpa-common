package httppool

import (
	"net/http"
	"time"
	
	"github.com/Comcast/webpa-common/logging"
	"github.com/rubyist/circuitbreaker"
)

func BreakerClient(timeout time.Duration, threshold int64, log logging.Logger, delegate *http.Client) *circuit.HTTPClient {
	client := circuit.NewHostBasedHTTPClient(timeout, threshold, delegate)
	
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
			switch panelEvent.Event {
			case circuit.BreakerTripped:
				log.Debug("breaker event (tripped): %s", panelEvent.Name)
			case circuit.BreakerReset:
				log.Debug("breaker event (reset): %s", panelEvent.Name)
			case circuit.BreakerFail:
				log.Debug("breaker event (fail): %s", panelEvent.Name)
			case circuit.BreakerReady:
				log.Debug("breaker event (ready): %s", panelEvent.Name)
			}
		}
	}()
	
	return client
}

