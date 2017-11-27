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

func (r *registry) add(d *device) (existing *device, err error) {
	r.lock.Lock()
	existing = r.devices[d.id]

	if existing == nil && uint32(len(r.devices)+1) > r.limit {
		err = fmt.Errorf("Maximum count of devices exceeded [%d]", r.limit)
	} else {
		// if there is an existing device, there's no reason to check the device limit
		// since we're replacing a device
		r.devices[d.id] = d
	}

	r.lock.Unlock()
	return
}

func (r *registry) remove(d *device) {
	r.lock.Lock()
	delete(r.devices, d.id)
	r.lock.Unlock()
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
