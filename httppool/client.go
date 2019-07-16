package httppool

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/logging"
)

const (
	DefaultWorkers   = 10
	DefaultQueueSize = 100
)

var (
	ErrorClosed = errors.New("Dispatcher has been closed")
)

// transactionHandler defines the methods required of something that actually
// handles HTTP transactions.  http.Client satisfies this interface.
type transactionHandler interface {
	// Do synchronously handles the HTTP transaction.  Any type that supplies
	// this method may be used with this infrastructure.
	Do(*http.Request) (*http.Response, error)
}

// Client is factory for asynchronous, pooled HTTP transaction dispatchers.
// An optional Period may be specified which limits the rate at which each worker goroutine
// sends requests.
type Client struct {
	// Name is a human-readable label for dispatchers created via this Client instance.
	// This name shows up in logs to distinguish one pool from another.  If this string
	// has length 0, a default name using the address of this Client instance is generated.
	Name string

	// Handler is any type that has a method with the signature Do(*http.Request) (*http.Response, error)
	// If not supplied, the http.DefaultClient is used.
	Handler transactionHandler

	// Listeners is the slice of Listener instances that will be notified of task events.
	// Each Dispatcher will use a distinct copy created with Start() is called.
	Listeners []Listener

	// Logger is the logging strategy used by this client.  If not supplied, logging is discarded.
	Logger log.Logger

	// QueueSize specifies that maximum number of requests that can be queued.
	// If this value is zero or negative, DefaultQueueSize is used.
	QueueSize int

	// Workers is the number of pooled goroutines that handle tasks.
	// If this value is less than one (1), DefaultWorkers is used.
	Workers int

	// Period is the interval between requests on EACH worker.  If this
	// value is zero or negative, the workers will not be rate-limited.
	Period time.Duration
}

func (client *Client) name() string {
	if len(client.Name) > 0 {
		return client.Name
	}

	return fmt.Sprintf("Pool[%p]", client)
}

func (client *Client) queueSize() int {
	if client.QueueSize > 0 {
		return client.QueueSize
	}

	return DefaultQueueSize
}

func (client *Client) workers() int {
	if client.Workers > 0 {
		return client.Workers
	}

	return DefaultWorkers
}

func (client *Client) logger() log.Logger {
	if client.Logger != nil {
		return client.Logger
	}

	return logging.DefaultLogger()
}

func (client *Client) handler() transactionHandler {
	if client.Handler != nil {
		return client.Handler
	}

	return http.DefaultClient
}

// Start starts the pool of goroutines and returns a DispatchCloser which
// can be used to send tasks and shut down the pool.
func (client *Client) Start() (dispatcher DispatchCloser) {
	name := client.name()
	logger := client.logger()
	logging.Debug(logger).Log(logging.MessageKey(), "client start", "name", name)

	var (
		worker    func(*workerContext)
		listeners []Listener
	)

	if len(client.Listeners) > 0 {
		listeners = make([]Listener, len(client.Listeners))
		copy(listeners, client.Listeners)
	}

	if client.Period > 0 {
		limited := &limitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				name:      name,
				handler:   client.handler(),
				listeners: listeners,
				errorLog:  logging.Error(logger, "name", name),
				debugLog:  logging.Debug(logger, "name", name),
				tasks:     make(chan Task, client.queueSize()),
			},
			period: client.Period,
		}

		worker = limited.worker
		dispatcher = limited
	} else {
		unlimited := &unlimitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				name:      name,
				handler:   client.handler(),
				listeners: listeners,
				errorLog:  logging.Error(logger, "name", name),
				debugLog:  logging.Debug(logger, "name", name),
				tasks:     make(chan Task, client.queueSize()),
			},
		}

		worker = unlimited.worker
		dispatcher = unlimited
	}

	workers := client.workers()
	for workerId := 0; workerId < workers; workerId++ {
		// create a unique context for each worker, especially
		// preallocated buffer for doing HTTP response cleanup.
		go worker(
			&workerContext{
				id:            workerId,
				errorLog:      logging.Error(logger, "name", name, "contextID", workerId),
				debugLog:      logging.Debug(logger, "name", name, "contextID", workerId),
				listeners:     listeners,
				cleanupBuffer: make([]byte, 8*1024),
			},
		)
	}

	return
}

// workerContext defines the contextual information associated
// with each pooled goroutine.  Any data that would be "goroutine-local"
// is stored here.
type workerContext struct {
	id            int
	event         event
	listeners     []Listener
	cleanupBuffer []byte
	errorLog      log.Logger
	debugLog      log.Logger
}

// dispatch handles dispatching an event to any registered listeners.
// This method uses an internal, shared Event instance so as to ease
// pressure on the garbage collector.
func (w *workerContext) dispatch(eventType EventType, eventError error) {
	w.event.eventType = eventType
	w.event.eventError = eventError

	for _, listener := range w.listeners {
		listener.On(&w.event)
	}
}

// pooledDispatcher supplies the common state and logic for all
// Client-based dispatchers
type pooledDispatcher struct {
	state     int32
	name      string
	handler   transactionHandler
	debugLog  log.Logger
	errorLog  log.Logger
	listeners []Listener
	tasks     chan Task
}

// dispatch sends the given event to all configured listeners
func (pooled *pooledDispatcher) dispatch(eventType EventType, eventError error) {
	event := &event{
		eventType:  eventType,
		eventError: eventError,
	}

	for _, listener := range pooled.listeners {
		listener.On(event)
	}
}

// Close shuts down the task channel.  Workers are allowed to finish
// and exit gracefully.
func (pooled *pooledDispatcher) Close() (err error) {
	pooled.debugLog.Log(logging.MessageKey(), "Close")
	defer func() {
		if r := recover(); r != nil {
			err = ErrorClosed
		}
	}()

	close(pooled.tasks)
	return
}

// Send drops the task onto the inbound channel.  This method will block
// if the task queue is full.
//
// This method will return ErrorClosed if the task channel has been closed.
func (pooled *pooledDispatcher) Send(task Task) (err error) {
	pooled.debugLog.Log(logging.MessageKey(), "Send")
	defer func() {
		eventType := EventTypeQueue
		if r := recover(); r != nil {
			eventType = EventTypeReject
			err = ErrorClosed
		}

		pooled.dispatch(eventType, err)
	}()

	pooled.tasks <- task
	return
}

// Offer attempts to send the task via a nonblocking select.
func (pooled *pooledDispatcher) Offer(task Task) (taken bool, err error) {
	pooled.debugLog.Log(logging.MessageKey(), "Offer")
	defer func() {
		eventType := EventTypeQueue
		if r := recover(); r != nil {
			taken = false
			err = ErrorClosed
			eventType = EventTypeReject
		} else if !taken {
			eventType = EventTypeReject
		}

		pooled.dispatch(eventType, err)
	}()

	select {
	case pooled.tasks <- task:
		taken = true
	default:
		taken = false
	}

	return
}

// handleTask takes care of using a task to create the request
// and then sending that request to the handler
func (pooled *pooledDispatcher) handleTask(context *workerContext, task Task) {
	context.debugLog.Log(logging.MessageKey(), "handleTask")
	context.dispatch(EventTypeStart, nil)

	var err error

	defer func() {
		// prevent panics from killing a worker
		if r := recover(); r != nil {
			pooled.errorLog.Log(logging.MessageKey(), "encountered a panic", "contextId", context.id, logging.ErrorKey(), r)
			context.dispatch(EventTypeFinish, fmt.Errorf("%s", r))
		} else {
			context.dispatch(EventTypeFinish, err)
		}
	}()

	request, consumer, err := task()
	if err != nil {
		context.errorLog.Log(logging.MessageKey(), "task error", logging.ErrorKey(), err)
		return
	} else if request == nil {
		context.errorLog.Log(logging.MessageKey(), "nil request")
		return
	}

	response, err := pooled.handler.Do(request)
	if response != nil && response.Body != nil {
		defer func() {
			// if the consumer already cleaned things up, CopyBuffer will return EOF
			// use a canonical cleanup buffer to ease GC pressure
			if _, err := io.CopyBuffer(ioutil.Discard, response.Body, context.cleanupBuffer); err != nil && err != io.EOF {
				context.errorLog.Log(logging.MessageKey(), "error while consuming response body", logging.ErrorKey(), err)
			}

			response.Body.Close()
		}()
	}

	if err != nil {
		context.errorLog.Log(logging.MessageKey(), "HTTP transaction error", logging.ErrorKey(), err)
		return
	}

	if response != nil && consumer != nil {
		consumer(response, request)
	}
}

// unlimitedClientDispatcher is a DispatchCloser that provides
// access to a pool of goroutines that is not rate limited.
type unlimitedClientDispatcher struct {
	pooledDispatcher
}

func (unlimited *unlimitedClientDispatcher) worker(context *workerContext) {
	context.debugLog.Log(logging.MessageKey(), "worker starting", "type", "unlimited")

	for task := range unlimited.tasks {
		unlimited.handleTask(context, task)
	}
}

// limitedClientDispatcher is a DispatchCloser whose pooled goroutines
// send requests on a fixed interval (period).
type limitedClientDispatcher struct {
	pooledDispatcher
	period time.Duration
}

func (limited *limitedClientDispatcher) worker(context *workerContext) {
	context.debugLog.Log(logging.MessageKey(), "worker starting", "type", "limited")
	ticker := time.NewTicker(limited.period)
	defer ticker.Stop()

	for task := range limited.tasks {
		<-ticker.C
		limited.handleTask(context, task)
	}
}
