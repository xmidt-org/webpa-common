package device

import (
	"hash/fnv"
	"sync"
)

// idShard represents a single shard of mappings for devices keyed by ID.  Each idShard
// permits duplicate devices mapped to the same ID.
type idShard struct {
	sync.RWMutex
	data map[ID][]*device
}

func (is *idShard) add(id ID, d *device) {
	is.Lock()
	defer is.Unlock()
	duplicates := is.data[id]

	// just do a simple linear search, as the number of duplicate devices
	// for a given ID is likely to be very small
	for _, candidate := range duplicates {
		if d == candidate {
			// this device is already here
			return
		}
	}

	is.data[id] = append(duplicates, d)
}

func (is *idShard) removeOne(id ID, d *device) bool {
	is.Lock()
	defer is.Unlock()

	duplicates := is.data[id]
	for i, candidate := range duplicates {
		if d == candidate {
			last := len(duplicates) - 1

			duplicates[i] = duplicates[last]
			duplicates[last] = nil
			duplicates = duplicates[:last]

			if len(duplicates) > 0 {
				is.data[id] = duplicates
			} else {
				delete(is.data, id)
			}

			return true
		}
	}

	return false
}

func (is *idShard) removeAll(id ID) []*device {
	is.Lock()
	defer is.Unlock()
	duplicates := is.data[id]
	delete(is.data, id)
	return duplicates
}

func (is *idShard) visitID(id ID, visitor func(*device)) int {
	is.RLock()
	defer is.RUnlock()

	duplicates := is.data[id]
	for _, d := range duplicates {
		visitor(d)
	}

	return len(duplicates)
}

func (is *idShard) visitIf(filter func(ID) bool, visitor func(*device)) (count int) {
	is.RLock()
	defer is.RUnlock()

	for id, duplicates := range is.data {
		if filter(id) {
			count += len(duplicates)
			for _, d := range duplicates {
				visitor(d)
			}
		}
	}

	return
}

func (is *idShard) removeIf(filter func(ID) bool, visitor func(*device)) (count int) {
	is.RLock()
	defer is.RUnlock()

	for id, duplicates := range is.data {
		if filter(id) {
			delete(is.data, id)
			count += len(duplicates)

			for _, d := range duplicates {
				visitor(d)
			}
		}
	}

	return
}

// keyShard represents a single shard of mappings for devices keyed by their unique Keys.
type keyShard struct {
	sync.RWMutex
	data map[Key]*device
}

func (ks *keyShard) add(key Key, d *device) error {
	ks.Lock()
	defer ks.Unlock()
	if _, ok := ks.data[key]; ok {
		return ErrorDuplicateKey
	}

	ks.data[key] = d
	return nil
}

func (ks *keyShard) remove(key Key) *device {
	ks.Lock()
	defer ks.Unlock()
	d := ks.data[key]
	delete(ks.data, key)
	return d
}

func (ks *keyShard) visitKey(key Key, visitor func(*device)) int {
	ks.RLock()
	defer ks.RUnlock()

	if d, ok := ks.data[key]; ok {
		visitor(d)
		return 1
	}

	return 0
}

func (ks *keyShard) visitAll(visitor func(*device)) int {
	ks.RLock()
	defer ks.RUnlock()

	for _, d := range ks.data {
		visitor(d)
	}

	return len(ks.data)
}

// registry is a fully sharded concurrent-safe mapping of connected devices.  Devices are mapped by both
// ID and Key.  A registry allows duplicate devices for the same ID, but only (1) device may be mapped to
// a given Key.
type registry struct {
	byID  []idShard
	byKey []keyShard
}

func newRegistry(shards, initialCapacity uint32) *registry {
	r := &registry{
		byID:  make([]idShard, shards),
		byKey: make([]keyShard, shards),
	}

	for i := uint32(0); i < shards; i++ {
		r.byID[i].data = make(map[ID][]*device, initialCapacity)
		r.byKey[i].data = make(map[Key]*device, initialCapacity)
	}

	return r
}

func (r *registry) idShardFor(id ID) *idShard {
	hasher := fnv.New32a()
	hasher.Write(id.Bytes())
	return &r.byID[hasher.Sum32()%uint32(len(r.byID))]
}

func (r *registry) keyShardFor(key Key) *keyShard {
	hasher := fnv.New32a()
	hasher.Write([]byte(key))
	return &r.byKey[hasher.Sum32()%uint32(len(r.byID))]
}

func (r *registry) add(d *device) error {
	key := d.Key()
	if err := r.keyShardFor(key).add(key, d); err != nil {
		return err
	}

	r.idShardFor(d.id).add(d.id, d)
	return nil
}

func (r *registry) removeKey(key Key) *device {
	return r.keyShardFor(key).remove(key)
}

func (r *registry) removeOne(d *device) (removed bool) {
	if removed = r.idShardFor(d.id).removeOne(d.id, d); removed {
		key := d.Key()
		r.keyShardFor(key).remove(key)
	}

	return
}

func (r *registry) removeAll(id ID) (removedDevices []*device) {
	removedDevices = r.idShardFor(id).removeAll(id)
	for _, d := range removedDevices {
		key := d.Key()
		r.keyShardFor(key).remove(key)
	}

	return
}

func (r *registry) removeIf(filter func(ID) bool, visitor func(*device)) (count int) {
	removed := make([]*device, 0, 10)
	for i := 0; i < len(r.byID); i++ {
		removed = removed[:0]
		count += r.byID[i].removeIf(filter, func(d *device) {
			// don't update the key shards here, as we're under the id shard lock ...
			removed = append(removed, d)
		})

		// handle deletion of the keys outside the id shard lock
		for _, d := range removed {
			key := d.Key()
			r.keyShardFor(key).remove(key)
			visitor(d)
		}
	}

	return
}

func (r *registry) visitID(id ID, visitor func(*device)) int {
	return r.idShardFor(id).visitID(id, visitor)
}

func (r *registry) visitKey(key Key, visitor func(*device)) int {
	return r.keyShardFor(key).visitKey(key, visitor)
}

func (r *registry) visitAll(visitor func(*device)) (count int) {
	for i := 0; i < len(r.byKey); i++ {
		count += r.byKey[i].visitAll(visitor)
	}

	return
}

func (r *registry) visitIf(filter func(ID) bool, visitor func(*device)) (count int) {
	for i := 0; i < len(r.byID); i++ {
		count += r.byID[i].visitIf(filter, visitor)
	}

	return
}
