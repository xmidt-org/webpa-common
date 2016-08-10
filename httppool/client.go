package httppool

import (
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"
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
	// Handler is any type that has a method with the signature Do(*http.Request) (*http.Response, error)
	// If not supplied, the http.DefaultClient is used.
	Handler transactionHandler

	// Logger is the logging strategy used by this client.  If not supplied, all output will
	// go to the console.
	Logger logging.Logger

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

func (client *Client) logger() logging.Logger {
	if client.Logger != nil {
		return client.Logger
	}

	return &logging.LoggerWriter{os.Stdout}
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
	logger := client.logger()
	logger.Debug("Start()")

	var worker func(*workerContext)
	if client.Period > 0 {
		limited := &limitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				handler: client.handler(),
				logger:  logger,
				tasks:   make(chan Task, client.queueSize()),
			},
			period: client.Period,
		}

		worker = limited.worker
		dispatcher = limited
	} else {
		unlimited := &unlimitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				handler: client.handler(),
				logger:  logger,
				tasks:   make(chan Task, client.queueSize()),
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
	cleanupBuffer []byte
}

// pooledDispatcher supplies the common state and logic for all
// Client-based dispatchers
type pooledDispatcher struct {
	handler transactionHandler
	logger  logging.Logger
	tasks   chan Task
}

// Close shuts down the task channel.  Workers are allowed to finish
// and exit gracefully.
func (pooled *pooledDispatcher) Close() (err error) {
	pooled.logger.Debug("Close()")
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
	pooled.logger.Debug("Send(%v)", task)
	defer func() {
		if r := recover(); r != nil {
			err = ErrorClosed
		}
	}()

	pooled.tasks <- task
	return
}

// Offer attempts to send the task via a nonblocking select.
func (pooled *pooledDispatcher) Offer(task Task) (taken bool, err error) {
	pooled.logger.Debug("Offer(%v)", task)
	defer func() {
		if r := recover(); r != nil {
			taken = false
			err = ErrorClosed
		}
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
	pooled.logger.Debug("handleTask(%d, %v)", context.id, task)

	// prevent panics from killing a worker
	defer func() {
		if r := recover(); r != nil {
			pooled.logger.Error("Worker %d encountered a panic: %s", context.id, r)
		}
	}()

	request, consumer, err := task()
	if err != nil {
		pooled.logger.Error("Worker %d received an error from a task: %s", context.id, err)
		return
	} else if request == nil {
		pooled.logger.Error("Worker %d received a nil request", context.id)
		return
	}

	response, err := pooled.handler.Do(request)
	if response != nil && response.Body != nil {
		defer func() {
			// if the consumer already cleaned things up, CopyBuffer will return EOF
			// use a canonical cleanup buffer to ease GC pressure
			if _, err := io.CopyBuffer(ioutil.Discard, response.Body, context.cleanupBuffer); err != nil && err != io.EOF {
				pooled.logger.Error("Worker %d encountered an error while consuming the response body: %s", err)
			}

			response.Body.Close()
		}()
	}

	if err != nil {
		pooled.logger.Error("HTTP transaction resulted in error: %s", err)
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
	unlimited.logger.Debug("Unlimited Worker %d starting", context.id)

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
	limited.logger.Debug("Rate-limited Worker %d starting", context.id)
	ticker := time.NewTicker(limited.period)
	defer ticker.Stop()

	for task := range limited.tasks {
		<-ticker.C
		limited.handleTask(context, task)
	}
}
