package drain

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics/discard"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics"
)

var (
	ErrAlreadyDraining error = errors.New("Already draining")
	ErrNotDraining     error = errors.New("Not draining")
)

const (
	StateNotDraining uint32 = 0
	StateDraining    uint32 = 1

	MetricNotDraining float64 = 0.0
	MetricDraining    float64 = 1.0

	// disconnectBatchSize is the arbitrary size of batches used when no rate is associated with the drain,
	// i.e. disconnect as fast as possible
	disconnectBatchSize int = 1000
)

type Option func(*drainer)

func WithLogger(l log.Logger) Option {
	return func(dr *drainer) {
		if l != nil {
			dr.logger = l
		} else {
			dr.logger = logging.DefaultLogger()
		}
	}
}

func WithRegistry(r device.Registry) Option {
	return func(dr *drainer) {
		dr.registry = r
	}
}

func WithConnector(c device.Connector) Option {
	return func(dr *drainer) {
		dr.connector = c
	}
}

func WithManager(m device.Manager) Option {
	return func(dr *drainer) {
		dr.registry = m
		dr.connector = m
	}
}

func WithStateGauge(s xmetrics.Setter) Option {
	return func(dr *drainer) {
		if s != nil {
			dr.m.state = s
		} else {
			dr.m.state = discard.NewGauge()
		}
	}
}

func WithDrainCounter(a xmetrics.Adder) Option {
	return func(dr *drainer) {
		if a != nil {
			dr.m.counter = a
		} else {
			dr.m.counter = discard.NewCounter()
		}
	}
}

// Job describes a single execution of the drainer
type Job struct {
	// Count is the total number of devices to disconnect.  If this field is nonpositive,
	// the count of connected devices at the start of job execution is used.
	Count int

	// Rate is the number of devices per tick to disconnect.  If this field is nonpositive,
	// devices are disconnected as fast as possible.
	Rate int

	// Tick is the time unit for the Rate field.  If Rate is set but this field is not set,
	// a tick of 1 second is used as the default.
	Tick time.Duration
}

// Progress describes the current state of a drain job, which includes completed jobs
type Progress struct {
	// Visited is the count of devices visited so far during the drain.  This count refers
	// to the number of disconnection attempts made.
	Visited int32

	// Drained is the count of devices actually disconnected so far.  This number can be less
	// than the Visited field if a device disconnected during the drain job's execution.
	Drained int32

	// Started is the system time at which the job began
	Started time.Time

	// Finished is the system time at which the job completed, which will be the zero time
	// if the job is still running or hasn't been run yet.
	Finished time.Time
}

// Interface describes the behavior of a component which can execute a Job to drain devices.
// Only (1) drain Job is allowed to run at any time.
type Interface interface {
	// Start attempts to begin draining devices.  The supplied Job describes how the drain will proceed.
	Start(Job) error

	// Status returns the current state of this drainer.  The boolean indicates whether a job is currently running.
	// The Job and Progress values describe the job that is currently running, or are the zero values if no job has ever run.
	// This method can be used to query the last completed job, so long as a new job has not been started.
	Status() (bool, Job, Progress)

	// Cancel asynchronously halts any running drain job.  The returned channel can be used to wait for the job to actually exit.
	// If no job is running, an error is returned along with a nil channel.
	Cancel() (<-chan struct{}, error)
}

func defaultNewTicker(d time.Duration) (<-chan time.Time, func()) {
	ticker := time.NewTicker(d)
	return ticker.C, ticker.Stop
}

// New constructs a drainer using the supplied options
func New(options ...Option) Interface {
	dr := &drainer{
		logger:    logging.DefaultLogger(),
		now:       time.Now,
		newTicker: defaultNewTicker,
		m: metrics{
			state:   discard.NewGauge(),
			counter: discard.NewCounter(),
		},
	}

	for _, f := range options {
		f(dr)
	}

	if dr.registry == nil {
		panic("A device.Registry is required")
	}

	if dr.connector == nil {
		panic("A device.Connector is required")
	}

	dr.m.state.Set(MetricNotDraining)
	return dr
}

type metrics struct {
	state   xmetrics.Setter
	counter xmetrics.Adder
}

// drainer is the internal implementation of Interface
type drainer struct {
	logger    log.Logger
	connector device.Connector
	registry  device.Registry
	now       func() time.Time
	newTicker func(time.Duration) (<-chan time.Time, func())
	m         metrics

	controlLock sync.RWMutex
	draining    uint32
	cancel      chan struct{}
	done        chan struct{}
	j           Job
	p           Progress
}

func (dr *drainer) enqueueBatch(batch chan<- device.ID, cancel <-chan struct{}) (success bool) {
	success = true
	dr.registry.VisitAll(func(d device.Interface) bool {
		select {
		case batch <- d.ID():
			return true
		case <-cancel:
			success = false
			return false
		default:
			return false
		}
	})

	return
}

func (dr *drainer) disconnectBatch(batch <-chan device.ID, cancel <-chan struct{}) bool {
	var (
		visited  int32
		drained  int32
		canceled bool
		finished bool
	)

	for !finished {
		select {
		case id := <-batch:
			visited++
			if dr.connector.Disconnect(id) {
				dr.m.counter.Add(1.0)
				drained++
			}
		case <-cancel:
			canceled = true
			finished = true
		default:
			finished = true
		}
	}

	atomic.AddInt32(&dr.p.Visited, visited)
	atomic.AddInt32(&dr.p.Drained, drained)
	return !canceled
}

// jobFinished ensures the right state is set when a drain job completes
func (dr *drainer) jobFinished(done chan<- struct{}) {
	close(done)
	atomic.StoreUint32(&dr.draining, StateNotDraining)

	dr.controlLock.Lock()
	dr.p.Finished = dr.now().UTC()
	dr.controlLock.Unlock()
}

func (dr *drainer) drain(remaining int, rate int, tick time.Duration, cancel <-chan struct{}, done chan<- struct{}) {
	var (
		batch        = make(chan device.ID, rate)
		ticker, stop = dr.newTicker(tick)
	)

	defer func() {
		stop()
		dr.jobFinished(done)
	}()

	dr.m.state.Set(MetricDraining)
	for remaining > 0 {
		select {
		case <-ticker:
			if remaining < rate {
				batch = make(chan device.ID, remaining)
			}

			if !dr.enqueueBatch(batch, cancel) {
				return
			}

			if !dr.disconnectBatch(batch, cancel) {
				return
			}

			remaining -= rate

		case <-cancel:
			return
		}
	}
}

func (dr *drainer) disconnect(remaining int, batchSize int, cancel <-chan struct{}, done chan<- struct{}) {
	defer dr.jobFinished(done)
	batch := make(chan device.ID, batchSize)

	dr.m.state.Set(MetricDraining)
	for remaining > 0 {
		if remaining < batchSize {
			batch = make(chan device.ID, remaining)
		}

		if !dr.enqueueBatch(batch, cancel) {
			return
		}

		if !dr.disconnectBatch(batch, cancel) {
			return
		}

		remaining -= batchSize
	}
}

func (dr *drainer) Start(j Job) error {
	if j.Count < 1 {
		j.Count = dr.registry.Len()
	}

	if j.Rate > 0 && j.Tick <= 0 {
		j.Tick = time.Second
	}

	defer dr.controlLock.Unlock()
	dr.controlLock.Lock()

	if !atomic.CompareAndSwapUint32(&dr.draining, StateNotDraining, StateDraining) {
		return ErrAlreadyDraining
	}

	dr.j = j
	atomic.StoreInt32(&dr.p.Visited, 0)
	atomic.StoreInt32(&dr.p.Drained, 0)
	dr.p.Started = dr.now().UTC()
	dr.p.Finished = time.Time{}
	dr.cancel = make(chan struct{})
	dr.done = make(chan struct{})

	if dr.j.Rate > 0 {
		go dr.drain(dr.j.Count, dr.j.Rate, dr.j.Tick, dr.cancel, dr.done)
	} else {
		go dr.disconnect(dr.j.Count, disconnectBatchSize, dr.cancel, dr.done)
	}

	return nil
}

func (dr *drainer) Status() (bool, Job, Progress) {
	defer dr.controlLock.RUnlock()
	dr.controlLock.RLock()

	return atomic.LoadUint32(&dr.draining) == StateDraining,
		dr.j,
		Progress{
			Visited:  atomic.LoadInt32(&dr.p.Visited),
			Drained:  atomic.LoadInt32(&dr.p.Drained),
			Started:  dr.p.Started,
			Finished: dr.p.Finished,
		}
}

func (dr *drainer) Cancel() (<-chan struct{}, error) {
	defer dr.controlLock.Unlock()
	dr.controlLock.Lock()

	if !atomic.CompareAndSwapUint32(&dr.draining, StateDraining, StateNotDraining) {
		return nil, ErrNotDraining
	}

	close(dr.cancel)
	return dr.done, nil
}
