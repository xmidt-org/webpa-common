package httppool

import (
	"errors"
	"github.com/Comcast/webpa-common/logging"
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

// Start starts the pool of goroutines and returns a DispatcherCloser which
// can be used to send tasks and shut down the pool.
func (client *Client) Start() (dispatcher DispatchCloser) {
	logger := client.logger()
	logger.Debug("Start()")

	var worker func(int)
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
		go worker(workerId)
	}

	return
}

// pooledDispatcher supplies the common state and logic for all
// Client-based dispatchers
type pooledDispatcher struct {
	handler transactionHandler
	logger  logging.Logger
	tasks   chan Task
}

// Close halts the consumption of tasks for all worker goroutines.
// Any remaining tasks are abandoned.
func (pooled *pooledDispatcher) Close() error {
	pooled.logger.Debug("Close()")
	close(pooled.tasks)
	return nil
}

// Send drops the task onto the inbound channel.  This method will block
// if the task queue is full.
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

// handleTask takes care of using a task to create the request
// and then sending that request to the handler
func (pooled *pooledDispatcher) handleTask(workerId int, task Task) {
	pooled.logger.Debug("handleTask(%d, %v)", workerId, task)
	if request, err := task(); err == nil {
		if response, err := pooled.handler.Do(request); err == nil {
			pooled.logger.Debug("response: %v", response)
		} else {
			pooled.logger.Error("HTTP transaction failed: %s", err)
		}
	} else {
		pooled.logger.Error("Unable to create request: %s", err)
	}
}

// unlimitedClientDispatcher is a DispatcherCloser that provides
// access to a pool of goroutines that is not rate limited.
type unlimitedClientDispatcher struct {
	pooledDispatcher
}

func (unlimited *unlimitedClientDispatcher) worker(workerId int) {
	unlimited.logger.Debug("Unlimited Worker %d starting", workerId)

	for task := range unlimited.tasks {
		unlimited.handleTask(workerId, task)
	}
}

// limitedClientDispatcher is a DispatcherCloser whose pooled goroutines
// send requests on a fixed interval (period).
type limitedClientDispatcher struct {
	pooledDispatcher
	period time.Duration
}

func (limited *limitedClientDispatcher) worker(workerId int) {
	limited.logger.Debug("Rate-limited Worker %d starting", workerId)
	ticker := time.NewTicker(limited.period)
	defer ticker.Stop()

	for task := range limited.tasks {
		<-ticker.C
		limited.handleTask(workerId, task)
	}
}
