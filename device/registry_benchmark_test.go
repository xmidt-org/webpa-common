package device

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
)

func benchmarkRegistry(b *testing.B, shards, initialCapacity uint32) {
	var (
		registry   = newRegistry(shards, initialCapacity)
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
	for _, shards := range []uint32{2, 10, 256, 512} {
		b.Run(fmt.Sprintf("Shards=%d", shards), func(b *testing.B) {
			for _, initialCapacity := range []uint32{10, 100, 1000} {
				b.Run(fmt.Sprintf("InitialCapacityPerShard=%d", initialCapacity), func(b *testing.B) {
					benchmarkRegistry(b, shards, initialCapacity)
				})
			}
		})
	}
}
