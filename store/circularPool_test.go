package store

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

const (
	poolSize    = 3
	workerCount = 5
	resource    = "I am a pooled resource!"
	timeout     = time.Duration(10 * time.Millisecond)
)

func resourceFunc() interface{} {
	return resource
}

func ExampleResourcePool() {
	pool, err := NewCircularPool(poolSize, poolSize, resourceFunc)
	if err != nil {
		fmt.Println(err)
		return
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(workerCount)

	for repeat := 0; repeat < workerCount; repeat++ {
		go func() {
			defer waitGroup.Done()

			// check out a resource
			sharedResource := pool.Get().(string)

			// use it
			fmt.Println(sharedResource)

			// return it to the pool
			pool.Put(sharedResource)
		}()
	}

	waitGroup.Wait()

	// Output:
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
	// I am a pooled resource!
}

func TestNewCircularPoolInvalidInitialSize(t *testing.T) {
	assert := assert.New(t)

	pool, err := NewCircularPool(10, 5, resourceFunc)
	assert.Nil(pool)
	assert.Equal(ErrorInvalidInitialSize, err)
}

func TestNewCircularPoolInvalidMaxSize(t *testing.T) {
	assert := assert.New(t)

	pool, err := NewCircularPool(0, 0, resourceFunc)
	assert.Nil(pool)
	assert.Equal(ErrorInvalidMaxSize, err)

	pool, err = NewCircularPool(0, -1, resourceFunc)
	assert.Nil(pool)
	assert.Equal(ErrorInvalidMaxSize, err)
}

func TestNewCircularPoolNilNew(t *testing.T) {
	assert := assert.New(t)

	pool, err := NewCircularPool(poolSize, poolSize, nil)
	assert.Nil(pool)
	assert.Equal(ErrorNewRequired, err)
}

func TestCircularPoolInitiallyEmpty(t *testing.T) {
	assert := assert.New(t)
	pool, err := NewCircularPool(0, poolSize, resourceFunc)
	assert.NotNil(pool)
	assert.Nil(err)

	// The pool is empty, so we should get nothing
	value, ok := pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)

	// GetOrNew should return a new resource
	value = pool.GetOrNew()
	assert.Equal(resource, value)

	// TryGet and GetTimeout should still get nothing
	value, ok = pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		value := pool.Get().(string)
		assert.Equal(resource, value)
		pool.Put(value)
	}()

	pool.Put(resource)
	waitGroup.Wait()

	// There should now be a resource on the pool
	value, ok = pool.TryGet()
	assert.Equal(resource, value)
	assert.True(ok)

	pool.Put(value)
	value, ok = pool.GetTimeout(timeout)
	assert.Equal(resource, value)
	assert.True(ok)

	// GetTimeout should fallback to TryGet for nonpositive timeouts
	value, ok = pool.GetTimeout(0)
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(-1)
	assert.Nil(value)
	assert.False(ok)
}

func TestCircularPoolGetPreallocated(t *testing.T) {
	assert := assert.New(t)
	pool, err := NewCircularPool(poolSize, poolSize, resourceFunc)
	assert.NotNil(pool)
	assert.Nil(err)

	// Put on a full pool should not alter its size
	pool.Put(resource)

	// Get should succeed poolSize times
	for repeat := 0; repeat < poolSize; repeat++ {
		value := pool.Get()
		assert.Equal(resource, value)
	}

	// the pool should be empty
	value, ok := pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)
}

func TestCircularPoolTryGetPreallocated(t *testing.T) {
	assert := assert.New(t)
	pool, err := NewCircularPool(poolSize, poolSize, resourceFunc)
	assert.NotNil(pool)
	assert.Nil(err)

	// Put on a full pool should not alter its size
	pool.Put(resource)

	// TryGet should succeed poolSize times
	for repeat := 0; repeat < poolSize; repeat++ {
		value, ok := pool.TryGet()
		assert.Equal(resource, value)
		assert.True(ok)
	}

	// the pool should be empty
	value, ok := pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)
}

func TestCircularPoolTryGetTimeoutPreallocated(t *testing.T) {
	assert := assert.New(t)
	pool, err := NewCircularPool(poolSize, poolSize, resourceFunc)
	assert.NotNil(pool)
	assert.Nil(err)

	// Put on a full pool should not alter its size
	pool.Put(resource)

	// GetTimeout should succeed poolSize times
	for repeat := 0; repeat < poolSize; repeat++ {
		value, ok := pool.GetTimeout(timeout)
		assert.Equal(resource, value)
		assert.True(ok)
	}

	// the pool should be empty
	value, ok := pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)
}

func TestCircularPoolTryGetOrNewPreallocated(t *testing.T) {
	assert := assert.New(t)
	pool, err := NewCircularPool(poolSize, poolSize, resourceFunc)
	assert.NotNil(pool)
	assert.Nil(err)

	// Put on a full pool should not alter its size
	pool.Put(resource)

	// GetOrNew should drain the pool poolSize times
	for repeat := 0; repeat < poolSize; repeat++ {
		value := pool.GetOrNew()
		assert.Equal(resource, value)
	}

	// the pool should be empty
	value, ok := pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)

	// GetOrNew should return a new object, but the pool should be empty afterward
	value = pool.GetOrNew()
	assert.Equal(resource, value)

	value, ok = pool.TryGet()
	assert.Nil(value)
	assert.False(ok)

	value, ok = pool.GetTimeout(timeout)
	assert.Nil(value)
	assert.False(ok)
}
