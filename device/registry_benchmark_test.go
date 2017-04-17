package device

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func benchmarkRegistry(b *testing.B, initialCapacity int) {
	var (
		registry   = newRegistry(initialCapacity)
		lock       sync.RWMutex
		macCounter uint64
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var (
				id  = IntToMAC(atomic.AddUint64(&macCounter, 1))
				key = Key(strconv.FormatUint(atomic.AddUint64(&macCounter, 1), 16))
			)

			lock.Lock()
			registry.add(newDevice(id, key, nil, 1))
			lock.Unlock()

			lock.RLock()
			registry.visitID(id, func(*device) {})
			lock.RUnlock()

			lock.RLock()
			registry.visitKey(key, func(*device) {})
			lock.RUnlock()
		}
	})
}

func BenchmarkRegistry(b *testing.B) {
	for _, initialCapacity := range []int{1, 10, 100, 1000, 10000, 100000} {
		b.Run(strconv.Itoa(initialCapacity), func(b *testing.B) { benchmarkRegistry(b, initialCapacity) })
	}
}

func benchmarkShardedRegistry(b *testing.B, shards, initialCapacity int) {
	var (
		registry   = newShardedRegistry(shards, initialCapacity)
		macCounter uint64
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var (
				id  = IntToMAC(atomic.AddUint64(&macCounter, 1))
				key = Key(strconv.FormatUint(atomic.AddUint64(&macCounter, 1), 16))
			)

			registry.add(newDevice(id, key, nil, 1))
			registry.visitID(id, func(*device) {})
			registry.visitID(id, func(*device) {})
		}
	})
}

func BenchmarkShardedRegistry(b *testing.B) {
	for _, shards := range []int{2, 10, 100, 256, 512} {
		b.Run(fmt.Sprintf("Shards=%d", shards), func(b *testing.B) {
			for _, initialCapacity := range []int{1, 10, 100, 1000, 10000, 100000} {
				b.Run(fmt.Sprintf("InitialCapacityPerShard=%d", initialCapacity), func(b *testing.B) {
					benchmarkShardedRegistry(b, shards, initialCapacity)
				})
			}
		})
	}
}
