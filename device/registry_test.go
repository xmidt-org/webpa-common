package device

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testRegistryConcurrentAddAndVisit(t *testing.T, r *registry) {
	var (
		assert      = assert.New(t)
		addGate     = new(sync.WaitGroup)
		addWait     = new(sync.WaitGroup)
		expectedIDs = map[ID]bool{
			ID("1"): true,
			ID("2"): true,
			ID("3"): true,
			ID("4"): true,
			ID("5"): true,
		}
	)

	addGate.Add(1)
	addWait.Add(len(expectedIDs))
	for id := range expectedIDs {
		go func(id ID) {
			defer addWait.Done()
			addGate.Wait()

			var (
				first  = &device{id: id}
				second = &device{id: id}
			)

			assert.Nil(r.add(first))
			existing, ok := r.get(id)
			assert.True(first == existing)
			assert.Equal(id, existing.id)
			assert.True(ok)

			assert.True(first == r.add(second))
			existing, ok = r.get(id)
			assert.True(second == existing)
			assert.Equal(id, existing.id)
			assert.True(ok)
		}(id)
	}

	addGate.Done()
	addWait.Wait()

	var (
		visitGate = new(sync.WaitGroup)
		visitWait = new(sync.WaitGroup)
	)

	visitGate.Add(1)
	visitWait.Add(len(expectedIDs) + 1) // the extra goroutine is for visitAll
	for id := range expectedIDs {
		go func(expected ID) {
			defer visitWait.Done()
			visitGate.Wait()

			assert.Equal(
				1,
				r.visitIf(
					func(actual ID) bool { return actual == expected },
					func(actual *device) {
						assert.NotNil(actual)
						assert.Equal(expected, actual.id)
					},
				),
			)
		}(id)
	}

	go func() {
		defer visitWait.Done()
		visitGate.Wait()

		visitedIDs := make(map[ID]bool, len(expectedIDs))
		assert.Equal(
			len(expectedIDs),
			r.visitAll(func(actual *device) {
				assert.NotNil(actual)
				visitedIDs[actual.id] = true
			}),
		)

		assert.Equal(expectedIDs, visitedIDs)
	}()

	visitGate.Done()
	visitWait.Wait()
}

func testRegistryConcurrentAddAndRemove(t *testing.T, r *registry) {
	var (
		assert           = assert.New(t)
		addAndRemoveGate = new(sync.WaitGroup)
		addAndRemoveWait = new(sync.WaitGroup)
		expectedIDs      = map[ID]bool{
			ID("1"): true,
			ID("2"): true,
			ID("3"): true,
			ID("4"): true,
			ID("5"): true,
		}
	)

	addAndRemoveGate.Add(1)
	addAndRemoveWait.Add(len(expectedIDs))
	for id := range expectedIDs {
		go func(id ID) {
			defer addAndRemoveWait.Done()
			addAndRemoveGate.Wait()

			d := &device{id: id}
			assert.Nil(r.add(d))
			r.remove(d)

			existing, ok := r.get(id)
			assert.Nil(existing)
			assert.False(ok)

			assert.Nil(r.add(d))
			removed, ok := r.removeID(id)
			assert.True(d == removed)
			assert.True(ok)

			existing, ok = r.get(id)
			assert.Nil(existing)
			assert.False(ok)

			assert.Nil(r.add(d))

			assert.Equal(
				1,
				r.removeIf(
					func(actual ID) bool { return actual == id },
					func(actual *device) {
						assert.True(d == actual)
					},
				),
			)
		}(id)
	}
}

func TestRegistry(t *testing.T) {
	t.Run("ConcurrentAddAndVisit", func(t *testing.T) {
		testRegistryConcurrentAddAndVisit(t, newRegistry(0))
		testRegistryConcurrentAddAndVisit(t, newRegistry(1))
		testRegistryConcurrentAddAndVisit(t, newRegistry(100))
	})

	t.Run("ConcurrentAddAndRemove", func(t *testing.T) {
		testRegistryConcurrentAddAndRemove(t, newRegistry(0))
		testRegistryConcurrentAddAndRemove(t, newRegistry(1))
		testRegistryConcurrentAddAndRemove(t, newRegistry(100))
	})
}
