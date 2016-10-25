package httppool

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
	
	"github.com/Comcast/webpa-common/logging"
	"github.com/rubyist/circuitbreaker"
)

// DispatherWithBreaker creates a Dispatcher which uses a circuit.HTTPClient for its Handler
func DispatcherWithBreaker(workers int, queueSize int, log logging.Logger, timeout time.Duration, threshold int64) Dispatcher {
	return (&Client{
		Workers: workers,
		QueueSize: queueSize,
		Logger: log,
		Handler: circuitBreakerClient(timeout, threshold, log),
	}).Start()
}

func circuitBreakerClient(timeout time.Duration, threshold int64, log logging.Logger) *circuit.HTTPClient {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		Timeout: timeout,
	}	
	
	breaker := circuit.NewConsecutiveBreaker(threshold)
	brclient := circuit.NewHTTPClientWithBreaker(breaker, timeout, client)
	
	brclient.BreakerTripped = func() {
		log.Debug("breaker was tripped.")
	}
	brclient.BreakerReset = func() {
		log.Debug("breaker was reset.")
	}
	
	brclient.BreakerLookup = func(c *circuit.HTTPClient, val interface{}) *circuit.Breaker {
		rawURL := val.(string)
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			breaker, _ := c.Panel.Get("_default")
			return breaker
		}
		host := parsedURL.Host
		
		cb, ok := c.Panel.Get(host)
		if !ok {
			cb = circuit.NewConsecutiveBreaker(threshold)
			c.Panel.Add(host, cb)
			
			events := cb.Subscribe()
			go func() {
				for {
					event := <- events
					switch event {
					case circuit.BreakerTripped:
						log.Debug("breaker event (tripped): %+v, breaker: %+v", host, cb)
//						brclient.BreakerTripped()
					case circuit.BreakerReset:
						log.Debug("breaker event (reset): %+v, breaker: %+v", host, cb)
//						brclient.BreakerReset()
					case circuit.BreakerFail:
						log.Debug("breaker event (fail): %+v, breaker: %+v", host, cb)
					case circuit.BreakerReady:
						log.Debug("breaker event (ready): %+v, breaker: %+v", host, cb)
					}
				}
			}()
		}
		
		return cb
	}
	
	return brclient
}