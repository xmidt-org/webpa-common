package device

import (
	"fmt"
	"math"
	"sync"
)

// registry is the internal lookup map for devices.  it is bounded by an optional maximum number
// of connected devices.
type registry struct {
	lock    sync.RWMutex
	limit   uint32
	devices map[ID]*device
}

func newRegistry(initialCapacity, maxDevices uint32) *registry {
	if maxDevices == 0 {
		maxDevices = math.MaxUint32
	}

	return &registry{
		devices: make(map[ID]*device, initialCapacity),
		limit:   maxDevices,
	}
}

func (r *registry) maxDevices() uint32 {
	return r.limit
}

func (r *registry) add(d *device) (*device, int, error) {
	r.lock.Lock()

	var (
		existing = r.devices[d.id]
		failed   = false
		err      error
	)

	// if there is an existing device, it will be replaced so the count won't go up
	// if there is NOT an existing device, the count will go up by one and that must be within the limit
	if existing != nil || uint32(len(r.devices)+1) <= r.limit {
		r.devices[d.id] = d
	} else {
		failed = true
	}

	deviceCount := len(r.devices)
	r.lock.Unlock()

	if failed {
		err = fmt.Errorf("Maximum count of devices exceeded [%d]", r.limit)
	}

	return existing, deviceCount, err
}

func (r *registry) remove(d *device) int {
	r.lock.Lock()
	delete(r.devices, d.id)
	deviceCount := len(r.devices)
	r.lock.Unlock()

	return deviceCount
}

func (r *registry) removeID(id ID) (*device, bool) {
	r.lock.Lock()
	existing, ok := r.devices[id]
	delete(r.devices, id)
	r.lock.Unlock()

	return existing, ok
}

func (r *registry) removeIf(filter func(ID) bool, visitor func(*device)) int {
	defer r.lock.Unlock()
	r.lock.Lock()

	count := 0
	for id, candidate := range r.devices {
		if filter(id) {
			count++
			delete(r.devices, id)
			visitor(candidate)
		}
	}

	return count
}

func (r *registry) visitAll(visitor func(*device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	for _, d := range r.devices {
		visitor(d)
	}

	return len(r.devices)
}

func (r *registry) visitIf(filter func(ID) bool, visitor func(*device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	count := 0
	for id, candidate := range r.devices {
		if filter(id) {
			count++
			visitor(candidate)
		}
	}

	return count
}

func (r *registry) get(id ID) (*device, bool) {
	r.lock.RLock()
	existing, ok := r.devices[id]
	r.lock.RUnlock()

	return existing, ok
}
