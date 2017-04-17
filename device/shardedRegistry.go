package device

import (
	"hash/fnv"
	"sync"
)

type shard struct {
	lock sync.RWMutex
	data map[ID][]*device
}

type shardedRegistry struct {
	shardCount uint32
	byID       []shard
}

func newShardedRegistry(shards, initialCapacityPerShard int) *shardedRegistry {
	sr := &shardedRegistry{
		shardCount: uint32(shards),
		byID:       make([]shard, shards),
	}

	for i := 0; i < shards; i++ {
		sr.byID[i].data = make(map[ID][]*device, initialCapacityPerShard)
	}

	return sr
}

func (sr *shardedRegistry) shardFor(id ID) *shard {
	hash := fnv.New32a()
	hash.Write(id.Bytes())
	return &sr.byID[hash.Sum32()%sr.shardCount]
}

func (sr *shardedRegistry) add(d *device) {
	s := sr.shardFor(d.id)
	s.lock.Lock()
	s.data[d.id] = append(s.data[d.id], d)
	s.lock.Unlock()
}

func (sr *shardedRegistry) visitID(id ID, visitor func(*device)) int {
	s := sr.shardFor(id)
	s.lock.RLock()
	defer s.lock.RUnlock()

	duplicates := s.data[id]
	for _, duplicate := range duplicates {
		visitor(duplicate)
	}

	return len(duplicates)
}
