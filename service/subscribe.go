package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/strava/go.serversets"
)

// Watch is the subset of methods required by this package that *serversets.Watch implements
type Watch interface {
	IsClosed() bool
	Event() <-chan struct{}
	Endpoints() []string
}

var _ Watch = (*serversets.Watch)(nil)

// Subscribe consumes watch events and invokes a subscription function with the endpoints.
// The returned function can be called to cancel the subscription.  This returned cancellation
// function is idempotent.
func Subscribe(logger logging.Logger, watch Watch, subscription func([]string)) func() {
	if logger == nil {
		logger = logging.DefaultLogger()
	}

	logger.Debug("Creating subscription for %#v", watch)
	cancel := make(chan struct{})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Subscription ending due to panic: %s", r)
			}
		}()

		for {
			select {
			case <-cancel:
				logger.Info("Subscription cancel event received")
				return
			case <-watch.Event():
				logger.Debug("Watch event received")
				if watch.IsClosed() {
					logger.Info("Watch closed.  Subscription ending.")
					return
				}

				endpoints := watch.Endpoints()
				logger.Info("Updated endpoints: %v", endpoints)
				subscription(endpoints)
			}
		}
	}()

	return func() {
		defer func() {
			recover()
		}()

		logger.Debug("Subscription cancellation function called")
		close(cancel)
	}
}
