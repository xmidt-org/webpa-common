package device

import (
	"sync"
)

type registry struct {
	sync.RWMutex
	byID  map[ID][]*device
	byKey map[Key]*device
}

func newRegistry(initialCapacity uint32) *registry {
	return &registry{
		byID:  make(map[ID][]*device, initialCapacity),
		byKey: make(map[Key]*device, initialCapacity),
	}
}

func (r *registry) add(d *device) error {
	key := d.Key()
	r.Lock()
	if _, ok := r.byKey[key]; ok {
		r.Unlock()
		return ErrorDuplicateKey
	}

	duplicates := r.byID[d.id]
	for _, candidate := range duplicates {
		if d == candidate {
			r.Unlock()
			return ErrorDuplicateDevice
		}
	}

	r.byKey[key] = d
	r.byID[d.id] = append(duplicates, d)
	r.Unlock()
	return nil
}

func (r *registry) removeKey(key Key) (d *device) {
	r.Lock()
	if d = r.byKey[key]; d != nil {
		delete(r.byKey, key)
		duplicates := r.byID[d.id]
		for i, candidate := range duplicates {
			if d == candidate {
				duplicates[i] = duplicates[len(duplicates)-1]
				duplicates[len(duplicates)-1] = nil
				duplicates = duplicates[:len(duplicates)-1]

				if len(duplicates) > 0 {
					r.byID[d.id] = duplicates
				} else {
					delete(r.byID, d.id)
				}

				break
			}
		}
	}

	r.Unlock()
	return
}

func (r *registry) removeAll(id ID) (removedDevices []*device) {
	r.Lock()

	removedDevices = r.byID[id]
	delete(r.byID, id)
	for _, d := range removedDevices {
		delete(r.byKey, d.Key())
	}

	r.Unlock()
	return
}

func (r *registry) removeIf(filter func(ID) bool, visitor func(*device)) (count int) {
	r.Lock()

	for id, duplicates := range r.byID {
		if filter(id) {
			count += len(duplicates)
			delete(r.byID, id)

			for _, d := range duplicates {
				delete(r.byKey, d.Key())
				visitor(d)
			}
		}
	}

	r.Unlock()
	return
}

func (r *registry) visitID(id ID, visitor func(*device)) (count int) {
	r.RLock()
	duplicates := r.byID[id]
	count = len(duplicates)
	for _, d := range duplicates {
		visitor(d)
	}

	r.RUnlock()
	return
}

func (r *registry) visitKey(key Key, visitor func(*device)) (count int) {
	r.RLock()
	if d := r.byKey[key]; d != nil {
		count = 1
		visitor(d)
	}

	r.RUnlock()
	return
}

func (r *registry) visitAll(visitor func(*device)) (count int) {
	r.RLock()
	for _, duplicates := range r.byID {
		count += len(duplicates)
		for _, d := range duplicates {
			visitor(d)
		}
	}

	r.RUnlock()
	return
}

func (r *registry) visitIf(filter func(ID) bool, visitor func(*device)) (count int) {
	r.RLock()
	for id, duplicates := range r.byID {
		if filter(id) {
			count += len(duplicates)
			for _, d := range duplicates {
				visitor(d)
			}
		}
	}

	r.RUnlock()
	return
}

func (r *registry) getOne(id ID) (d *device, err error) {
	r.RLock()
	duplicates := r.byID[id]
	switch len(duplicates) {
	case 0:
		err = ErrorDeviceNotFound
	case 1:
		d = duplicates[0]
	default:
		err = ErrorNonUniqueID
	}

	r.RUnlock()
	return
}
