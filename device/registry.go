package device

// idMap stores devices keyed by their canonical ID.  Multiple devices are
// allowed to have the same ID.
type idMap map[ID]map[*device]bool

func (m idMap) add(id ID, d *device) {
	if duplicates, ok := m[id]; ok {
		duplicates[d] = true
	} else {
		m[id] = map[*device]bool{d: true}
	}
}

func (m idMap) removeOne(d *device) {
	if duplicates, ok := m[d.id]; ok {
		delete(duplicates, d)
		if len(duplicates) == 0 {
			delete(m, d.id)
		}
	}
}

func (m idMap) removeAll(id ID) (removed []*device) {
	if duplicates, ok := m[id]; ok {
		removed = make([]*device, 0, len(duplicates))
		for d, _ := range duplicates {
			removed = append(removed, d)
		}

		delete(m, id)
	}

	return
}

// keyMap stores devices keyed by their routing Key.  Routing keys are
// unique across a given keyMap.
type keyMap map[Key]*device

func (m keyMap) add(k Key, d *device) error {
	if _, ok := m[k]; ok {
		return NewDuplicateKeyError(k)
	}

	m[k] = d
	return nil
}

func (m keyMap) remove(k Key) bool {
	if _, ok := m[k]; ok {
		delete(m, k)
		return true
	}

	return false
}

// registry is an internal type that stores mappings of devices
// A registry instance is not safe for concurrent access.
type registry struct {
	ids  idMap
	keys keyMap
}

func newRegistry(initialCapacity int) *registry {
	return &registry{
		ids:  make(idMap, initialCapacity),
		keys: make(keyMap, initialCapacity),
	}
}

func (r *registry) visitID(id ID, visitor func(*device)) int {
	if duplicates, ok := r.ids[id]; ok {
		for d, _ := range duplicates {
			visitor(d)
		}

		return len(duplicates)
	}

	return 0
}

func (r *registry) visitKey(k Key, visitor func(*device)) int {
	if d, ok := r.keys[k]; ok {
		visitor(d)
		return 1
	}

	return 0
}

func (r *registry) visitIf(filter func(ID) bool, visitor func(*device)) (count int) {
	for id, duplicates := range r.ids {
		if filter(id) {
			for d, _ := range duplicates {
				visitor(d)
			}

			count += len(duplicates)
		}
	}

	return
}

func (r *registry) visitAll(visitor func(*device)) int {
	for _, d := range r.keys {
		visitor(d)
	}

	return len(r.keys)
}

func (r *registry) add(d *device) error {
	k := d.Key()
	if err := r.keys.add(k, d); err != nil {
		return err
	}

	r.ids.add(d.id, d)
	return nil
}

func (r *registry) removeOne(d *device) bool {
	k := d.Key()
	if !r.keys.remove(k) {
		return false
	}

	r.ids.removeOne(d)
	return true
}

func (r *registry) removeAll(id ID) (removed []*device) {
	removed = r.ids.removeAll(id)
	for _, d := range removed {
		r.keys.remove(d.Key())
	}

	return
}
