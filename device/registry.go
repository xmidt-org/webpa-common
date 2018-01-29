package device

import (
	"errors"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
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
	logger log.Logger
	lock   sync.RWMutex
	limit  int
	data   map[ID]*device

	count      metrics.Gauge
	connect    metrics.Counter
	disconnect metrics.Counter
	duplicates metrics.Counter
}

func newRegistry(o registryOptions) *registry {
	if o.InitialCapacity < 1 {
		o.InitialCapacity = 10
	}

	return &registry{
		logger:     o.Logger,
		data:       make(map[ID]*device, o.InitialCapacity),
		limit:      o.Limit,
		count:      o.Measures.Device,
		connect:    o.Measures.Connect,
		disconnect: o.Measures.Disconnect,
		duplicates: o.Measures.Duplicates,
	}
}

// add uses a factory function to create a new device atomically with modifying
// the registry
func (r *registry) add(id ID, f func() (*device, error)) (*device, error) {
	defer r.lock.Unlock()
	r.lock.Lock()

	existing := r.data[id]
	if existing == nil && r.limit > 0 && (len(r.data)+1) >= r.limit {
		// adding this would result in exceeding the limit
		return nil, errDeviceLimitReached
	}

	newDevice, err := f()
	if err != nil {
		return nil, err
	}

	// at this point, the various pumps should be up and running for the new device,
	// but the device is not addressable from the outside yet.
	r.data[id] = newDevice

	if existing != nil {
		r.disconnect.Add(1.0)
		r.duplicates.Add(1.0)
		newDevice.Statistics().AddDuplications(existing.Statistics().Duplications() + 1)
		existing.requestClose()
	}

	r.connect.Add(1.0)
	r.count.Set(float64(len(r.data)))
	return newDevice, nil
}

func (r *registry) remove(id ID) (*device, bool) {
	defer r.lock.Unlock()
	r.lock.Lock()

	existing, ok := r.data[id]
	if ok {
		delete(r.data, id)
		r.disconnect.Add(1.0)
		r.count.Add(-1.0)
		existing.requestClose()
	}

	return existing, ok
}

func (r *registry) removeIf(f func(d *device) bool) int {
	defer r.lock.Unlock()
	r.lock.Lock()

	count := 0
	for id, d := range r.data {
		if f(d) {
			delete(r.data, id)
			count++
			d.requestClose()
		}
	}

	if count > 0 {
		r.disconnect.Add(float64(count))
		r.count.Set(float64(len(r.data)))
	}

	return count
}

func (r *registry) visit(f func(d *device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	for _, d := range r.data {
		f(d)
	}

	return len(r.data)
}

func (r *registry) get(id ID) (*device, bool) {
	r.lock.RLock()
	existing, ok := r.data[id]
	r.lock.RUnlock()

	return existing, ok
}
