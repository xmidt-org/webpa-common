package device

import (
	"errors"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/xmidt-org/webpa-common/xmetrics"
)

var errDeviceLimitReached = errors.New("Device limit reached")

type registryOptions struct {
	Logger          log.Logger
	Limit           int
	InitialCapacity int
	Measures        Measures
}

// registry is the internal lookup map for devices.  it is bounded by an optional maximum number
// of connected devices.
type registry struct {
	logger          log.Logger
	lock            sync.RWMutex
	limit           int
	initialCapacity int
	data            map[ID]*device

	count        xmetrics.Setter
	limitReached xmetrics.Incrementer
	connect      xmetrics.Incrementer
	disconnect   xmetrics.Adder
	duplicates   xmetrics.Incrementer
}

func newRegistry(o registryOptions) *registry {
	if o.InitialCapacity < 1 {
		o.InitialCapacity = 10
	}

	return &registry{
		logger:          o.Logger,
		initialCapacity: o.InitialCapacity,
		data:            make(map[ID]*device, o.InitialCapacity),
		limit:           o.Limit,
		count:           o.Measures.Device,
		limitReached:    o.Measures.LimitReached,
		connect:         o.Measures.Connect,
		disconnect:      o.Measures.Disconnect,
		duplicates:      o.Measures.Duplicates,
	}
}

// len returns the size of this registry
func (r *registry) len() int {
	r.lock.RLock()
	l := len(r.data)
	r.lock.RUnlock()

	return l
}

// add uses a factory function to create a new device atomically with modifying
// the registry
func (r *registry) add(newDevice *device) error {
	id := newDevice.ID()
	r.lock.Lock()

	existing := r.data[id]
	if existing == nil && r.limit > 0 && (len(r.data)+1) > r.limit {
		// adding this would result in exceeding the limit
		r.lock.Unlock()
		r.limitReached.Inc()
		r.disconnect.Add(1.0)
		newDevice.requestClose(CloseReason{Err: errDeviceLimitReached, Text: "device-limit-reached"})
		return errDeviceLimitReached
	}

	if existing != nil {
		r.disconnect.Add(1.0)
		r.duplicates.Inc()
		newDevice.Statistics().AddDuplications(existing.Statistics().Duplications() + 1)
		r.remove(existing.id, CloseReason{Text: "duplication"})
		// existing.requestClose(CloseReason{Text: "duplicate"})
	}

	// this will either leave the count the same or add 1 to it ...
	r.data[id] = newDevice
	r.count.Set(float64(len(r.data)))
	r.lock.Unlock()

	r.connect.Inc()
	return nil
}

func (r *registry) remove(id ID, reason CloseReason) (*device, bool) {
	r.lock.Lock()
	existing, ok := r.data[id]
	if ok {
		delete(r.data, id)
	}

	r.count.Set(float64(len(r.data)))
	r.lock.Unlock()

	if existing != nil {
		r.disconnect.Add(1.0)
		existing.requestClose(reason)
	}

	return existing, ok
}

func (r *registry) removeIf(f func(d *device) (CloseReason, bool)) int {
	// first, gather up all the devices that match the predicate
	matched := make([]*device, 0, 100)
	reasons := make([]CloseReason, 0, 100)

	r.lock.RLock()
	for _, d := range r.data {
		if reason, ok := f(d); ok {
			matched = append(matched, d)
			reasons = append(reasons, reason)
		}
	}

	r.lock.RUnlock()

	if len(matched) == 0 {
		return 0
	}

	// now, remove each device one at a time, releasing the write
	// lock in between
	count := 0
	for i, d := range matched {
		r.lock.Lock()

		// allow for barging
		_, ok := r.data[d.ID()]
		if ok {
			delete(r.data, d.ID())
			r.count.Set(float64(len(r.data)))
		}

		r.lock.Unlock()

		if ok {
			count++
			d.requestClose(reasons[i])
		}
	}

	if count > 0 {
		r.disconnect.Add(float64(count))
	}

	return count
}

func (r *registry) removeAll(reason CloseReason) int {
	r.lock.Lock()
	original := r.data
	r.data = make(map[ID]*device, r.initialCapacity)
	r.count.Set(0.0)
	r.lock.Unlock()

	count := len(original)
	for _, d := range original {
		d.requestClose(reason)
	}

	r.disconnect.Add(float64(count))
	return count
}

func (r *registry) visit(f func(d *device) bool) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	visited := 0
	for _, d := range r.data {
		visited++
		if !f(d) {
			break
		}
	}

	return visited
}

func (r *registry) get(id ID) (*device, bool) {
	r.lock.RLock()
	existing, ok := r.data[id]
	r.lock.RUnlock()

	return existing, ok
}
