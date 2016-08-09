package httppool

import (
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"os"
	"time"
)

const (
	DefaultWorkers   = 10
	DefaultQueueSize = 100
)

var (
	ErrorAlreadyRunning = errors.New("This client is already running")
)

// Client is an asynchronous, pooled HTTP transaction handler.  This type
// acts as a factory for Dispatcher implementations that manage a pool of
// workers, each handling HTTP transactions.  Support for rate limiting
// is also provided, via the Period member.
type Client struct {
	// Handler is any type that has a method with the signature Do(*http.Request) (*http.Response, error)
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

	// Period is the interval between requests on each worker.  If this
	// value is zero or negative, the workers will not be rate-limited.
	Period time.Duration
}

// Start starts the pool of goroutines and returns a DispatcherCloser which
// can be used to send tasks and shut down the pool.
func (client *Client) Start() (dispatcher DispatchCloser) {
	queueSize := client.QueueSize
	if queueSize < 1 {
		queueSize = DefaultQueueSize
	}

	logger := client.Logger
	if logger == nil {
		logger = &logging.LoggerWriter{os.Stdout}
	}

	var worker func(int)

	if client.Period > 0 {
		limited := &limitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				handler: client.Handler,
				logger:  logger,
				tasks:   make(chan Task, queueSize),
			},
			ticker: time.NewTicker(client.Period),
		}

		worker = limited.worker
		dispatcher = limited
	} else {
		unlimited := &unlimitedClientDispatcher{
			pooledDispatcher: pooledDispatcher{
				handler: client.Handler,
				logger:  logger,
				tasks:   make(chan Task, queueSize),
			},
		}

		worker = unlimited.worker
		dispatcher = unlimited
	}

	workers := client.Workers
	if workers < 1 {
		workers = DefaultWorkers
	}

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

func (pooled *pooledDispatcher) Close() error {
	pooled.logger.Debug("Close()")
	close(pooled.tasks)
	return nil
}

func (pooled *pooledDispatcher) Send(task Task) error {
	pooled.logger.Debug("Send(%v)", task)
	pooled.tasks <- task
	return nil
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
	for task := range unlimited.tasks {
		unlimited.handleTask(workerId, task)
	}
}

// limitedClientDispatcher is a DispatcherCloser whose pooled goroutines
// are limited by a time channel.
type limitedClientDispatcher struct {
	pooledDispatcher
	ticker *time.Ticker
}

func (limited *limitedClientDispatcher) Close() error {
	defer limited.ticker.Stop()
	return limited.pooledDispatcher.Close()
}

func (limited *limitedClientDispatcher) worker(workerId int) {
	for task := range limited.tasks {
		<-limited.ticker.C
		limited.handleTask(workerId, task)
	}
}
