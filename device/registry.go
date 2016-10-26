package device

import (
	"fmt"
)

type Key string
type keyMap map[Key]*device

type registry struct {
	byID  map[ID]keyMap
	byKey keyMap
}

func newRegistry(initialSize int) *registry {
	return &registry{
		byID:  make(map[ID]keyMap, initialSize),
		byKey: make(keyMap, initialSize),
	}
}

func (r *registry) add(d *device) error {
	if _, ok := r.byKey[d.key]; ok {
		return fmt.Errorf("Duplicate device key: %s", d.key)
	}

	r.byKey[d.key] = d
	if devices, ok := r.byID[d.id]; ok {
		devices[d.key] = d
	} else {
		r.byID[d.id] = keyMap{d.key: d}
	}

	return nil
}

func (r *registry) removeOne(id ID, k Key) *device {
	if deleted, ok := r.byKey[k]; ok {
		delete(r.byKey, k)
		if devices, ok := r.byID[id]; ok {
			delete(devices, k)
			if len(devices) == 1 {
				delete(r.byID, id)
			}
		}

		return deleted
	}

	return nil
}

func (r *registry) removeAll(id ID) keyMap {
	if devices, ok := r.byID[id]; ok {
		delete(r.byID, id)
		for _, d := range devices {
			delete(r.byKey, d.key)
		}

		return devices
	}

	return nil
}

func (r *registry) removeIf(filter func(ID) bool) (removedDevices []*device) {
	for id, devices := range r.byID {
		if filter(id) {
			delete(r.byID, id)
			for _, d := range devices {
				delete(r.byKey, d.key)
				removedDevices = append(removedDevices, d)
			}
		}
	}

	return
}

func (r *registry) visitAll(visitor func(Interface)) int {
	for _, device := range r.byKey {
		visitor(device)
	}

	return len(r.byKey)
}

func (r *registry) visitID(id ID, visitor func(Interface)) int {
	devices := r.byID[id]
	for _, d := range devices {
		visitor(d)
	}

	return len(devices)
}
