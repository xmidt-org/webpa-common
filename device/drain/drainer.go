package drain

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/discard"

	"github.com/Comcast/webpa-common/device"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics"
)

var (
	ErrActive    error = errors.New("A drain operation is already running")
	ErrNotActive error = errors.New("No drain operation is running")
)

const (
	StateNotActive uint32 = 0
	StateActive    uint32 = 1

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
	if r == nil {
		panic("A device.Registry is required")
	}

	return func(dr *drainer) {
		dr.registry = r
	}
}

func WithConnector(c device.Connector) Option {
	if c == nil {
		panic("A device.Connector is required")
	}

	return func(dr *drainer) {
		dr.connector = c
	}
}

func WithManager(m device.Manager) Option {
	if m == nil {
		panic("A device.Manager is required")
	}

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

type Job struct {
	// Count is the total number of devices to disconnect.  If this field is nonpositive and percent is unset,
	// the count of connected devices at the start of job execution is used.  If Percent is set, this field's
	// original value is ignored and it is set to that percentage of total devices connected at the time the
	// job starts.
	Count int `json:"count" schema:"count"`

	// Percent is the fraction of devices to drain.  If this field is set, Count's original value is ignored
	// and set to the computed percentage of connected devices at the time the job starts.
	Percent int `json:"percent,omitempty" schema:"percent"`

	// Rate is the number of devices per tick to disconnect.  If this field is nonpositive,
	// devices are disconnected as fast as possible.
	Rate int `json:"rate,omitempty" schema:"rate"`

	// Tick is the time unit for the Rate field.  If Rate is set but this field is not set,
	// a tick of 1 second is used as the default.
	Tick time.Duration `json:"tick,omitempty" schema:"tick"`
}

// ToMap returns a map representation of this Job appropriate for marshaling to formats like JSON.
// This method makes things a bit prettier, like the Tick.
func (j Job) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"count": j.Count,
	}

	if j.Percent > 0 {
		m["percent"] = j.Percent
	}

	if j.Rate > 0 {
		m["rate"] = j.Rate
	}

	if j.Tick > 0 {
		m["tick"] = j.Tick.String()
	}

	return m
}

// normalize applies some basic logic to interpret defaults and set values appropriately for a given device count
func (j *Job) normalize(deviceCount int) {
	if j.Percent > 0 {
		j.Count = int((float64(deviceCount) / 100.0) * float64(j.Percent))
	} else if j.Count <= 0 {
		j.Count = deviceCount
	}

	if j.Rate > 0 {
		if j.Tick <= 0 {
			j.Tick = time.Second
		}
	} else {
		j.Rate = 0
		j.Tick = 0
	}
}

// Interface describes the behavior of a component which can execute a Job to drain devices.
// Only (1) drain Job is allowed to run at any time.
type Interface interface {
	// Start attempts to begin draining devices.  The supplied Job describes how the drain will proceed.
	// The returned channel can be used to wait for the drain job to complete.  The returned Job will be
	// the result of applying defaults and will represent the actual Job being executed.  For example, if Job.Rate
	// is set but Job.Tick is not, the returned Job will reflect the default of 1 second for Job.Tick.
	Start(Job) (<-chan struct{}, Job, error)

	// Status returns information about the current drain job, if any.  The boolean return indicates whether
	// the job is currently active, while the returned Job describes the actual options used in starting the drainer.
	// This returned Job instance will not necessarily be the same as that passed to Start, as certain fields
	// may be computed or defaulted.
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

// jobContext stores all the runtime information for a drain job
type jobContext struct {
	id        uint32
	logger    log.Logger
	t         *tracker
	j         Job
	batchSize int
	ticker    <-chan time.Time
	stop      func()
	cancel    chan struct{}
	done      chan struct{}
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
	active      uint32
	currentID   uint32
	current     atomic.Value
}

// nextBatch grabs a batch of devices, bounded by the size of the supplied batch channel, and attempts
// to disconnect each of them.  This method is sensitive to the jc.cancel channel.  If cancelled, or if
// no more devices are available, this method returns false.
func (dr *drainer) nextBatch(jc jobContext, batch chan device.ID) (more bool, visited int) {
	jc.logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "nextBatch starting")

	more = true
	dr.registry.VisitAll(func(d device.Interface) bool {
		select {
		case batch <- d.ID():
			return true
		case <-jc.cancel:
			jc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "job cancelled")
			more = false
			return false
		default:
			return false
		}
	})

	visited = len(batch)
	if !more {
		return
	}

	if visited > 0 {
		jc.logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "nextBatch", "visited", visited)
		drained := 0
		for finished := false; more && !finished; {
			select {
			case id := <-batch:
				if dr.connector.Disconnect(id) {
					drained++
				}
			case <-jc.cancel:
				jc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "job cancelled")
				more = false
			default:
				finished = true
			}
		}

		jc.logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "nextBatch", "visited", visited, "drained", drained)
		jc.t.addVisited(visited)
		jc.t.addDrained(drained)
	} else {
		// if no devices were visited (or enqueued), then we must be done.
		// either a cancellation occurred or no devices are left
		dr.logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "no devices visited")
		more = false
	}

	return
}

func (dr *drainer) jobFinished(jc jobContext) {
	if jc.stop != nil {
		jc.stop()
	}

	jc.t.done(dr.now().UTC())

	// we need to contend on the control lock to avoid clobbering state from Start/Cancel code
	dr.controlLock.Lock()
	if jc.id == dr.currentID && atomic.CompareAndSwapUint32(&dr.active, StateActive, StateNotActive) {
		dr.m.state.Set(MetricNotDraining)
	}

	dr.controlLock.Unlock()

	// only close the done channel when all cleanup is complete
	close(jc.done)

	p := jc.t.Progress()
	jc.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "drain complete", "visited", p.Visited, "drained", p.Drained)
}

// drain is run as a goroutine to drain devices at a particular rate
func (dr *drainer) drain(jc jobContext) {
	defer dr.jobFinished(jc)
	jc.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "drain starting", "count", jc.j.Count, "rate", jc.j.Rate, "tick", jc.j.Tick)

	var (
		remaining = jc.j.Count
		visited   = 0
		more      = true
		batch     = make(chan device.ID, jc.j.Rate)
	)

	for more && remaining > 0 {
		if remaining < jc.j.Rate {
			batch = make(chan device.ID, remaining)
		}

		select {
		case <-jc.ticker:
			more, visited = dr.nextBatch(jc, batch)
			remaining -= visited
		case <-jc.cancel:
			jc.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "job cancelled")
			more = false
		}
	}
}

// disconnect is run as a goroutine to drain devices without a rate, i.e. as fast as possible
func (dr *drainer) disconnect(jc jobContext) {
	defer dr.jobFinished(jc)
	jc.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "drain starting", "count", jc.j.Count)

	var (
		remaining = jc.j.Count
		visited   = 0
		more      = true
		batch     = make(chan device.ID, jc.batchSize)
	)

	for more && remaining > 0 {
		if remaining < jc.batchSize {
			batch = make(chan device.ID, remaining)
		}

		more, visited = dr.nextBatch(jc, batch)
		remaining -= visited
	}
}

func (dr *drainer) Start(j Job) (<-chan struct{}, Job, error) {
	j.normalize(dr.registry.Len())

	defer dr.controlLock.Unlock()
	dr.controlLock.Lock()

	if !atomic.CompareAndSwapUint32(&dr.active, StateNotActive, StateActive) {
		return nil, Job{}, ErrActive
	}

	dr.currentID++
	jc := jobContext{
		id:     dr.currentID,
		logger: log.With(dr.logger, "id", dr.currentID),
		t: &tracker{
			started: dr.now().UTC(),
			counter: dr.m.counter,
		},
		j:      j,
		cancel: make(chan struct{}),
		done:   make(chan struct{}),
	}

	if jc.j.Rate > 0 {
		jc.ticker, jc.stop = dr.newTicker(j.Tick)
		go dr.drain(jc)
	} else {
		jc.batchSize = disconnectBatchSize
		go dr.disconnect(jc)
	}

	dr.m.state.Set(MetricDraining)
	dr.current.Store(jc)
	return jc.done, jc.j, nil
}

func (dr *drainer) Status() (bool, Job, Progress) {
	defer dr.controlLock.RUnlock()
	dr.controlLock.RLock()

	if jc, ok := dr.current.Load().(jobContext); ok {
		return atomic.LoadUint32(&dr.active) == StateActive,
			jc.j,
			jc.t.Progress()
	}

	// if the job has never run, this result will be returned
	return false, Job{}, Progress{}
}

func (dr *drainer) Cancel() (<-chan struct{}, error) {
	defer dr.controlLock.Unlock()
	dr.controlLock.Lock()

	if !atomic.CompareAndSwapUint32(&dr.active, StateActive, StateNotActive) {
		return nil, ErrNotActive
	}

	dr.m.state.Set(MetricNotDraining)
	jc := dr.current.Load().(jobContext)
	close(jc.cancel)
	return jc.done, nil
}
