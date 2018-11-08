package semaphore

import (
	"sync"
	"sync/atomic"
	"testing"
)

func benchmarkAtomic(b *testing.B) {
	var value int32

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			atomic.AddInt32(&value, 1)
		}
	})
}

func benchmarkSyncMutex(b *testing.B) {
	var (
		value int
		lock  sync.Mutex
	)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			lock.Lock()
			value++
			lock.Unlock()
		}
	})
}

func benchmarkBinarySemaphore(b *testing.B) {
	var (
		value int
		m     = Mutex()
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Acquire()
			value++
			m.Release()
		}
	})
}

func benchmarkCloseableBinarySemaphore(b *testing.B) {
	var (
		value int
		m     = CloseableMutex()
	)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Acquire()
			value++
			m.Release()
		}
	})
}

func benchmarkChannel(b *testing.B) {
	var (
		value   int
		updates = make(chan chan struct{}, 1)
	)

	go func() {
		for u := range updates {
			value++
			close(u)
		}
	}()

	defer close(updates)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			result := make(chan struct{})
			updates <- result
			<-result
		}
	})
}

func BenchmarkSingleResource(b *testing.B) {
	b.Run("atomic", benchmarkAtomic)
	b.Run("sync.Mutex", benchmarkSyncMutex)
	b.Run("semaphore", func(b *testing.B) {
		b.Run("binary", benchmarkBinarySemaphore)
		b.Run("closeable", benchmarkCloseableBinarySemaphore)
	})
	b.Run("channel", benchmarkChannel)
}
