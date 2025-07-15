// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

// StatsListener receives Stats on regular intervals.
type StatsListener interface {
	// OnStats is called with a copy of the health's stats map
	// at regular intervals.
	OnStats(Stats)
}

// StatsListenerFunc is a function type that implements StatsListener.
type StatsListenerFunc func(Stats)

func (f StatsListenerFunc) OnStats(stats Stats) {
	f(stats)
}

// Dispatcher represents a sink for Health events
type Dispatcher interface {
	SendEvent(HealthFunc)
}

// Monitor is the basic interface implemented by health event sinks
type Monitor interface {
	Dispatcher

	// HACK HACK HACK
	// This should be moved to another package
	ServeHTTP(http.ResponseWriter, *http.Request)
}

// Health is the central type of this package.  It defines and endpoint for tracking
// and updating various statistics.  It also dispatches events to one or more StatsListeners
// at regular intervals.
type Health struct {
	lock             sync.Mutex
	stats            Stats
	statDumpInterval time.Duration
	logger           *zap.Logger
	statsListeners   []StatsListener
	memInfoReader    *MemInfoReader
	once             sync.Once
}

var _ Monitor = (*Health)(nil)

// RequestTracker is an Alice-style constructor that wraps the given delegate in request-tracking
// code.
func (h *Health) RequestTracker(delegate http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		h.SendEvent(Inc(TotalRequestsReceived, 1))
		wrappedResponse := Wrap(response)

		defer func() {
			if r := recover(); r != nil {
				h.logger.Error("Delegate handler panicked", zap.Any("error", r))

				// TODO: Probably need an error stat instead of just "denied"
				h.SendEvent(Inc(TotalRequestsDenied, 1))

				if wrappedResponse.StatusCode() == 0 {
					// only write the header if one has not been written yet
					wrappedResponse.WriteHeader(http.StatusInternalServerError)
				}
			} else if wrappedResponse.StatusCode() < 400 {
				h.SendEvent(Inc(TotalRequestsSuccessfullyServiced, 1))
			} else {
				h.SendEvent(Inc(TotalRequestsDenied, 1))
			}
		}()

		delegate.ServeHTTP(wrappedResponse, request)
	})
}

// AddStatsListener adds a new listener to this Health.  This method
// is asynchronous.  The listener will eventually receive events, but callers
// should not assume events will be dispatched immediately after this method call.
func (h *Health) AddStatsListener(listener StatsListener) {
	h.SendEvent(func(stat Stats) {
		h.statsListeners = append(h.statsListeners, listener)
	})
}

// SendEvent dispatches a HealthFunc to the internal event queue
func (h *Health) SendEvent(healthFunc HealthFunc) {
	h.lock.Lock()
	healthFunc(h.stats)
	h.lock.Unlock()
}

// New creates a Health object with the given statistics.
func New(interval time.Duration, logger *zap.Logger, options ...Option) *Health {
	initialStats := NewStats(options)

	return &Health{
		stats:            initialStats,
		statDumpInterval: interval,
		logger:           logger,
		memInfoReader:    &MemInfoReader{},
	}
}

// Run executes this Health object.  This method is idempotent:  once a
// Health object is Run, it cannot be Run again.
func (h *Health) Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
	h.once.Do(func() {
		h.logger.Info("Health Monitor Started")

		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			ticker := time.NewTicker(h.statDumpInterval)
			defer ticker.Stop()
			defer h.logger.Info("Health Monitor Stopped")

			for {
				select {
				case <-shutdown:
					return

				case <-ticker.C:
					h.lock.Lock()
					h.stats.UpdateMemory(h.memInfoReader)
					dispatchStats := h.stats.Clone()
					h.lock.Unlock()
					for _, statsListener := range h.statsListeners {
						statsListener.OnStats(dispatchStats)
					}
				}
			}
		}()
	})

	return nil
}

func (h *Health) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	var (
		data []byte
		err  error
	)

	h.SendEvent(func(stats Stats) {
		stats.UpdateMemory(h.memInfoReader)
		data, err = json.Marshal(stats)
	})

	response.Header().Set("Content-Type", "application/json")
	if err != nil {
		h.logger.Error("Could not marshal stats", zap.Error(err))
		response.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(response, `{"message": "%s"}\n`, err.Error())
	} else {
		fmt.Fprintf(response, "%s", data)
	}
}
