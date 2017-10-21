package device

import (
	"sync"
)

type registry struct {
	lock    sync.RWMutex
	devices map[ID]*device
}

func newRegistry(initialCapacity uint32) *registry {
	return &registry{
		devices: make(map[ID]*device, initialCapacity),
	}
}

func (r *registry) add(d *device) *device {
	r.lock.Lock()
	existing := r.devices[d.id]
	r.devices[d.id] = d
	r.lock.Unlock()

	return existing
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
