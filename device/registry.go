package device

import (
	"sync"
)

type registry struct {
	lock    sync.RWMutex
	devices map[ID][]*device
}

func newRegistry(initialCapacity uint32) *registry {
	return &registry{
		devices: make(map[ID][]*device, initialCapacity),
	}
}

func (r *registry) add(d *device) error {
	defer r.lock.Unlock()
	r.lock.Lock()

	duplicates := r.devices[d.id]
	for _, candidate := range duplicates {
		if d == candidate {
			return ErrorDuplicateDevice
		}
	}

	r.devices[d.id] = append(duplicates, d)
	return nil
}

func (r *registry) removeAll(id ID) []*device {
	defer r.lock.Unlock()
	r.lock.Lock()

	removedDevices := r.devices[id]
	delete(r.devices, id)
	return removedDevices
}

func (r *registry) removeIf(filter func(ID) bool, visitor func(*device)) int {
	defer r.lock.Unlock()
	r.lock.Lock()

	count := 0
	for id, duplicates := range r.devices {
		if filter(id) {
			count += len(duplicates)
			delete(r.devices, id)

			for _, d := range duplicates {
				visitor(d)
			}
		}
	}

	return count
}

func (r *registry) visitID(id ID, visitor func(*device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	var (
		duplicates = r.devices[id]
		count      = len(duplicates)
	)

	for _, d := range duplicates {
		visitor(d)
	}

	return count
}

func (r *registry) visitAll(visitor func(*device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	count := 0
	for _, duplicates := range r.devices {
		count += len(duplicates)
		for _, d := range duplicates {
			visitor(d)
		}
	}

	return count
}

func (r *registry) visitIf(filter func(ID) bool, visitor func(*device)) int {
	defer r.lock.RUnlock()
	r.lock.RLock()

	count := 0
	for id, duplicates := range r.devices {
		if filter(id) {
			count += len(duplicates)
			for _, d := range duplicates {
				visitor(d)
			}
		}
	}

	return count
}

func (r *registry) getOne(id ID) (*device, error) {
	defer r.lock.RUnlock()
	r.lock.RLock()

	duplicates := r.devices[id]
	switch len(duplicates) {
	case 0:
		return nil, ErrorDeviceNotFound
	case 1:
		return duplicates[0], nil
	default:
		return nil, ErrorNonUniqueID
	}
}
