package wrpendpoint

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/endpoint"
)

const (
	DefaultWorkers   = 20
	DefaultQueueSize = 100
)

// Service represents a component which processes WRP transactions.
type Service interface {
	// ServeWRP processes a WRP request.  Either the Response or the error
	// returned from this method will be nil, but not both.
	ServeWRP(Request) (Response, error)
}

// ServiceFunc is a function type that implements Service
type ServiceFunc func(Request) (Response, error)

func (sf ServiceFunc) ServeWRP(r Request) (Response, error) {
	return sf(r)
}

// ServiceError gives metadata around the error from a particular service.
// This type itself implements error.
type ServiceError struct {
	// Err is the actual error retrieved from the Service
	Err error

	// Endpoint is the optional name of the endpoint which serviced the request
	Endpoint string

	// Duration is the time taken by the Service to produce the error
	Duration time.Duration
}

func (se *ServiceError) String() string {
	return se.Error()
}

func (se *ServiceError) Error() string {
	return se.Err.Error()
}

// workerResult is message that carries back ServeWRP results
// across a channel or other asynchronous boundary.
type workerResult struct {
	endpoint string
	response Response
	duration time.Duration
	err      error
}

// dispatcherEnvelope wraps a WRP request for transport across
// an asynchronous boundary.  Another goroutine can communicate the
// result by sending a message on the result channel.
type dispatcherEnvelope struct {
	request Request
	result  chan<- workerResult
}

// ServiceDispatcher is a WRP Service implementation that asynchronously invokes another Service
// via a pooled set of worker goroutines.  Obviously, the delegate Service must be safe for concurrent
// invocation.
type ServiceDispatcher struct {
	state     uint32
	envelopes chan dispatcherEnvelope
	delegate  Service

	now   func() time.Time
	since func(time.Time) time.Duration
}

// NewServiceDispatcher constructs and starts a new ServiceDispatcher.
//
// If workers and/or queueSize are nonpositive, they'll be set to sensible defaults
// prior to starting the dispatcher.
func NewServiceDispatcher(workers, queueSize int, delegate Service) *ServiceDispatcher {
	if workers < 1 {
		workers = DefaultWorkers
	}

	if queueSize < 1 {
		queueSize = DefaultQueueSize
	}

	sd := &ServiceDispatcher{
		envelopes: make(chan dispatcherEnvelope, queueSize),
		delegate:  delegate,
		now:       time.Now,
		since:     time.Since,
	}

	for r := 0; r < workers; r++ {
		go sd.worker()
	}

	return sd
}

func (sd *ServiceDispatcher) worker() {
	for e := range sd.envelopes {
		var (
			start = sd.now()
			ctx   = e.request.Context()
		)

		if ctx.Err() != nil {
			e.result <- workerResult{err: ctx.Err(), duration: sd.since(start)}
			continue
		}

		response, err := sd.delegate.ServeWRP(e.request)
		e.result <- workerResult{response: response, err: err, duration: sd.since(start)}
	}
}

func (sd *ServiceDispatcher) Stop() bool {
	if atomic.CompareAndSwapUint32(&sd.state, 0, 1) {
		close(sd.envelopes)
		return true
	}

	return false
}

func (sd *ServiceDispatcher) ServeWRP(request Request) (Response, error) {
	if atomic.LoadUint32(&sd.state) != 0 {
		return nil, errors.New("Dispatcher not running")
	}

	ctx := request.Context()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	result := make(chan workerResult, 1)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case sd.envelopes <- dispatcherEnvelope{request, result}:
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case e := <-result:
		return e.response, e.err
	}
}

// serviceFanout takes a single WRP request and dispatches it concurrently to zero
// or more go-kit endpoints.
type serviceFanout struct {
	endpoints map[string]endpoint.Endpoint

	now   func() time.Time
	since func(time.Time) time.Duration
}

func NewServiceFanout(endpoints map[string]endpoint.Endpoint) Service {
	if len(endpoints) == 0 {
		return ServiceFunc(func(Request) (Response, error) {
			return nil, errors.New("No configured endpoints")
		})
	}

	return &serviceFanout{
		endpoints: endpoints,
		now:       time.Now,
		since:     time.Since,
	}
}

func (sf *serviceFanout) ServeWRP(request Request) (Response, error) {
	ctx := request.Context()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	results := make(chan workerResult, len(sf.endpoints))
	for name, e := range sf.endpoints {
		go func(name string, e endpoint.Endpoint) {
			var (
				start      = sf.now()
				value, err = e(ctx, request)
				duration   = sf.since(start)
			)

			if err != nil {
				results <- workerResult{
					endpoint: name,
					duration: duration,
					err:      &ServiceError{Err: err, Endpoint: name, Duration: duration},
				}
			} else {
				response, _ := value.(Response)
				results <- workerResult{
					endpoint: name,
					response: response,
					duration: duration,
				}
			}
		}(name, e)
	}

	// there can be at most (1) result from each goroutine
	errs := make(map[string]error, len(sf.endpoints))
	for r := 0; r < len(sf.endpoints); r++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case r := <-results:
			if r.err == nil {
				// we have a good response, no more reason to wait
				return r.response, nil
			}

			errs[r.endpoint] = r.err
		}
	}

	// TODO: aggregate errors somehow
	return nil, nil
}
