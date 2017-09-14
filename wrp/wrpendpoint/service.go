package wrpendpoint

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/go-kit/kit/endpoint"
)

const (
	DefaultServiceDispatcherName = "dispatcher"
	WorkerNameSuffix             = ".worker"

	DefaultServiceFanoutName = "fanout"

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

// workerResult is message that carries back ServeWRP results
// across a channel or other asynchronous boundary.
type workerResult struct {
	span     tracing.Span
	response Response
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
	state      uint32
	name       string
	workerName string
	envelopes  chan dispatcherEnvelope
	spanner    tracing.Spanner
	delegate   Service
}

// NewServiceDispatcher constructs and starts a new ServiceDispatcher.
//
// If workers and/or queueSize are nonpositive, they'll be set to sensible defaults
// prior to starting the dispatcher.
func NewServiceDispatcher(name string, workers, queueSize int, delegate Service) *ServiceDispatcher {
	if len(name) == 0 {
		name = DefaultServiceDispatcherName
	}

	if workers < 1 {
		workers = DefaultWorkers
	}

	if queueSize < 1 {
		queueSize = DefaultQueueSize
	}

	sd := &ServiceDispatcher{
		name:       name,
		workerName: name + WorkerNameSuffix,
		envelopes:  make(chan dispatcherEnvelope, queueSize),
		spanner:    tracing.NewSpanner(),
		delegate:   delegate,
	}

	for r := 0; r < workers; r++ {
		go sd.worker()
	}

	return sd
}

func (sd *ServiceDispatcher) worker() {
	for e := range sd.envelopes {
		var (
			finisher = sd.spanner.Start(sd.workerName)
			ctx      = e.request.Context()
		)

		if ctx.Err() != nil {
			e.result <- workerResult{
				span: finisher(ctx.Err()),
			}

			continue
		}

		response, err := sd.delegate.ServeWRP(e.request)
		e.result <- workerResult{
			span:     finisher(err),
			response: response,
		}
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

	var (
		finisher = sd.spanner.Start(sd.name)
		ctx      = request.Context()
	)

	if ctx.Err() != nil {
		return nil, tracing.SpanError{finisher(ctx.Err())}
	}

	result := make(chan workerResult, 1)
	select {
	case <-ctx.Done():
		return nil, tracing.SpanError{finisher(ctx.Err())}
	case sd.envelopes <- dispatcherEnvelope{request, result}:
	}

	select {
	case <-ctx.Done():
		return nil, tracing.SpanError{finisher(ctx.Err())}
	case r := <-result:
		if r.span.Error() != nil {
			return nil, tracing.SpanError{r.span}
		}

		return r.response.AddSpans(r.span), nil
	}
}

// serviceFanout takes a single WRP request and dispatches it concurrently to zero
// or more go-kit endpoints.
type serviceFanout struct {
	name      string
	endpoints map[string]endpoint.Endpoint
	spanner   tracing.Spanner

	now   func() time.Time
	since func(time.Time) time.Duration
}

func NewServiceFanout(name string, endpoints map[string]endpoint.Endpoint) Service {
	if len(endpoints) == 0 {
		return ServiceFunc(func(Request) (Response, error) {
			return nil, errors.New("No configured endpoints")
		})
	}

	if len(name) == 0 {
		name = DefaultServiceFanoutName
	}

	return &serviceFanout{
		name:      name,
		endpoints: endpoints,
		spanner:   tracing.NewSpanner(),
		now:       time.Now,
		since:     time.Since,
	}
}

func (sf *serviceFanout) ServeWRP(request Request) (Response, error) {
	var (
		finisher = sf.spanner.Start(sf.name)
		ctx      = request.Context()
	)

	if ctx.Err() != nil {
		return nil, tracing.SpanError{finisher(ctx.Err())}
	}

	results := make(chan workerResult, len(sf.endpoints))
	for name, e := range sf.endpoints {
		go func(name string, e endpoint.Endpoint) {
			var (
				finisher    = sf.spanner.Start(name)
				value, err  = e(ctx, request)
				response, _ = value.(Response)
			)

			results <- workerResult{
				span:     finisher(err),
				response: response,
			}
		}(name, e)
	}

	var spanError tracing.SpanError
	for r := 0; r < len(sf.endpoints); r++ {
		select {
		case <-ctx.Done():
			return nil, append(spanError, finisher(ctx.Err()))
		case r := <-results:
			if r.span.Error() != nil {
				spanError = append(spanError, r.span)
			} else {
				return r.response.AddSpans(spanError...), nil
			}
		}
	}

	return nil, spanError
}
