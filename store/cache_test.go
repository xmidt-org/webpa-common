package store

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/concurrent"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewCacheInvalidCachePeriod(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		invalidCachePeriod CachePeriod
	}{
		{
			CachePeriodNever,
		},
		{
			CachePeriodForever,
		},
		{
			CachePeriod(0),
		},
		{
			CachePeriod(-1),
		},
		{
			CachePeriod(-234971),
		},
	}

	for _, record := range testData {
		source := &testValue{}
		cache, err := NewCache(source, record.invalidCachePeriod)
		assert.Nil(cache)
		assert.NotNil(err)
	}
}

func TestCacheRefresh(t *testing.T) {
	assert := assert.New(t)
	source := &testValue{
		value: "success",
	}

	cache, err := NewCache(source, CachePeriod(1*time.Hour))
	if !assert.NotNil(cache) || !assert.Nil(err) {
		return
	}

	t.Log("Initializing cache ...")
	value, err := cache.Load()
	assert.True(source.wasCalled)
	assert.Equal(value, "success")
	assert.Nil(err)

	t.Log("Refresh() should update the cache upon a source success")
	source.wasCalled = false
	source.value = "success again"
	value, err = cache.Refresh()
	assert.True(source.wasCalled)
	assert.Equal(value, "success again")
	assert.Nil(err)

	t.Log("Load() should return the new value from Refresh()")
	source.wasCalled = false
	value, err = cache.Load()
	assert.False(source.wasCalled)
	assert.Equal(value, "success again")
	assert.Nil(err)

	t.Log("Refresh() should not update the cache on a source error")
	source.wasCalled = false
	source.value = nil
	source.err = errors.New("failure")
	value, err = cache.Refresh()
	assert.True(source.wasCalled)
	assert.Nil(value)
	assert.Equal(source.err, err)

	t.Log("Load() should return the old value after Refresh fails")
	source.wasCalled = false
	value, err = cache.Load()
	assert.False(source.wasCalled)
	assert.Equal(value, "success again")
	assert.Nil(err)
}

func TestCacheLoad(t *testing.T) {
	assert := assert.New(t)
	source := &testValue{
		value: "success",
	}

	cache, err := NewCache(source, CachePeriod(1*time.Hour))
	if !assert.NotNil(cache) || !assert.Nil(err) {
		return
	}

	t.Log("Initializing cache ...")
	value, err := cache.Load()
	assert.True(source.wasCalled)
	assert.Equal(value, "success")
	assert.Nil(err)

	t.Log("Value should actually be cached")
	source.wasCalled = false
	value, err = cache.Load()
	assert.False(source.wasCalled)
	assert.Equal(value, "success")
	assert.Nil(err)

	t.Log("Simulating cache expiration")
	source.wasCalled = false
	source.value = "success again"
	cache.cache.Store(&cacheEntry{
		expiry: time.Now().Add(-2 * time.Hour),
	})
	value, err = cache.Load()
	assert.True(source.wasCalled)
	assert.Equal(value, "success again")
	assert.Nil(err)

	t.Log("New value should actually be cached")
	source.wasCalled = false
	value, err = cache.Load()
	assert.False(source.wasCalled)
	assert.Equal(value, "success again")
	assert.Nil(err)

	t.Log("Upon an error, Load() should not update the cache")
	source.wasCalled = false
	source.value = nil
	source.err = errors.New("failure")
	cache.cache.Store(&cacheEntry{
		value:  "old success",
		expiry: time.Now().Add(-2 * time.Hour),
	})
	value, err = cache.Load()
	assert.True(source.wasCalled)
	assert.Equal(value, "old success")
	assert.Nil(err)

	t.Log("After an error, the new value should be cached normally")
	source.wasCalled = false
	value, err = cache.Load()
	assert.False(source.wasCalled)
	assert.Equal(value, "old success")
	assert.Nil(err)
}

func TestCacheLoadFirstTimeFails(t *testing.T) {
	assert := assert.New(t)
	source := &testValue{
		err: errors.New("failure"),
	}

	cache, err := NewCache(source, CachePeriod(1*time.Hour))
	if !assert.NotNil(cache) || !assert.Nil(err) {
		return
	}

	t.Log("The first time Load() is called and there is a source error, it should return that error")
	value, err := cache.Load()
	assert.True(source.wasCalled)
	assert.Nil(value)
	assert.Equal(source.err, err)
}

func TestCacheLoadConcurrency(t *testing.T) {
	assert := assert.New(t)

	var currentValue int32
	source := ValueFunc(func() (interface{}, error) {
		return atomic.AddInt32(&currentValue, 1), nil
	})

	cache, err := NewCache(source, CachePeriod(100*time.Millisecond))
	if !assert.NotNil(cache) || !assert.Nil(err) {
		return
	}

	const loadCount int = 3
	const refreshCount int = 2
	shutdown := make(chan struct{})
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(loadCount + refreshCount)

	for repeat := 0; repeat < loadCount; repeat++ {
		go func() {
			defer waitGroup.Done()
			ticker := time.NewTicker(27 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-shutdown:
					return
				case <-ticker.C:
					value, err := cache.Load()
					assert.Nil(err)
					assert.NotNil(value)
				}
			}
		}()
	}

	for repeat := 0; repeat < refreshCount; repeat++ {
		go func() {
			defer waitGroup.Done()
			ticker := time.NewTicker(62 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-shutdown:
					return
				case <-ticker.C:
					value, err := cache.Refresh()
					assert.Nil(err)
					assert.NotNil(value)
				}
			}
		}()
	}

	time.Sleep(1 * time.Second)
	close(shutdown)

	if !concurrent.WaitTimeout(waitGroup, 500*time.Millisecond) {
		t.Fatal("Cache routines did not complete within the timeout")
	}
}
