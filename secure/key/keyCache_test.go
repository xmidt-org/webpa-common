package key

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"sync"
	"testing"
	"time"
)

const (
	routineCount = 3
)

var (
	resolveKeyError error = errors.New("ResolveKey failed!")
	testKeyIds      []string
	oldKeys         []interface{}
	newKeys         []interface{}
)

func init() {
	// create one test key id and "key" for each routine
	// this is primarily for testing multi key caching
	testKeyIds = make([]string, routineCount)
	oldKeys = make([]interface{}, routineCount)
	newKeys = make([]interface{}, routineCount)

	for index := 0; index < routineCount; index++ {
		testKeyIds[index] = fmt.Sprintf("key%d", index)
		oldKeys[index] = fmt.Sprintf("this is an old key #%d", index)
		newKeys[index] = fmt.Sprintf("this is the new key #%d", index)
	}
}

func TestBasicCacheUsesKeyId(t *testing.T) {
	assert := assert.New(t)

	for _, expected := range []bool{true, false} {
		resolver := &mockResolver{}
		resolver.On("UsesKeyId").Return(expected).Once()

		cache := &basicCache{
			delegate: resolver,
		}

		assert.Equal(expected, cache.UsesKeyId())
		mock.AssertExpectationsForObjects(t, resolver.Mock)
	}
}

func TestBasicCacheStoreAndLoad(t *testing.T) {
	assert := assert.New(t)

	cache := basicCache{}
	assert.Nil(cache.load())
	cache.store(123)
	assert.Equal(123, cache.load())
}

func TestSingleKeyCacheResolveKey(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()

	cache := singleKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(routineCount)
	barrier := make(chan struct{})

	for repeat := 0; repeat < routineCount; repeat++ {
		go func() {
			defer waitGroup.Done()
			<-barrier
			key, err := cache.ResolveKey(testKeyIds[0])
			assert.Equal(oldKeys[0], key)
			assert.Nil(err)
		}()
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver.Mock)
	assert.Equal(oldKeys[0], cache.load())
}

func TestSingleKeyCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(nil, resolveKeyError).Times(routineCount)

	cache := singleKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(routineCount)
	barrier := make(chan struct{})

	for repeat := 0; repeat < routineCount; repeat++ {
		go func() {
			defer waitGroup.Done()
			<-barrier
			key, err := cache.ResolveKey(testKeyIds[0])
			assert.Nil(key)
			assert.Equal(resolveKeyError, err)
		}()
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver.Mock)
	assert.Nil(cache.load())
}

func TestSingleKeyCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(oldKeys[0], nil).Once()

	cache := singleKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	mock.AssertExpectationsForObjects(t, resolver.Mock)
	assert.Equal(1, count)
	assert.Nil(errors)
}

func TestSingleKeyCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(nil, resolveKeyError).Once()

	cache := singleKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	mock.AssertExpectationsForObjects(t, resolver.Mock)
	assert.Equal(1, count)
	assert.Equal([]error{resolveKeyError}, errors)

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestSingleKeyCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(nil, resolveKeyError).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(newKeys[0], nil).Once()

	cache := singleKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	key, err := cache.ResolveKey(testKeyIds[0])
	assert.Equal(oldKeys[0], key)
	assert.Nil(err)

	count, errors := cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Equal([]error{resolveKeyError}, errors)

	// resolving should pull the key from the cache
	key, err = cache.ResolveKey(testKeyIds[0])
	assert.Equal(oldKeys[0], key)
	assert.Nil(err)

	// this time, the mock will succeed
	count, errors = cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Len(errors, 0)

	// resolving should pull the *new* key from the cache
	key, err = cache.ResolveKey(testKeyIds[0])
	assert.Equal(newKeys[0], key)
	assert.Nil(err)

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestMultiKeyCacheResolveKey(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	for index, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(oldKeys[index], nil).Once()
	}

	cache := multiKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	// spawn twice the number of routines as keys so
	// that we test concurrently resolving keys from the cache
	// and from the delegate
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2 * routineCount)
	barrier := make(chan struct{})

	tester := func(keyId string, expectedKey interface{}) {
		defer waitGroup.Done()
		<-barrier
		key, err := cache.ResolveKey(keyId)
		assert.Equal(expectedKey, key)
		assert.Nil(err)
	}

	for repeat := 0; repeat < (2 * routineCount); repeat++ {
		go tester(testKeyIds[repeat%routineCount], oldKeys[repeat%routineCount])
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestMultiKeyCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	for _, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(nil, resolveKeyError).Twice()
	}

	cache := multiKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	// spawn twice the number of routines as keys so
	// that we test concurrently resolving keys from the cache
	// and from the delegate
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2 * routineCount)
	barrier := make(chan struct{})

	tester := func(keyId string, expectedKey interface{}) {
		defer waitGroup.Done()
		<-barrier
		key, err := cache.ResolveKey(keyId)
		assert.Nil(key)
		assert.Equal(resolveKeyError, err)
	}

	for repeat := 0; repeat < (2 * routineCount); repeat++ {
		go tester(testKeyIds[repeat%routineCount], oldKeys[repeat%routineCount])
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestMultiKeyCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	for index, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(oldKeys[index], nil).Twice()
	}

	cache := multiKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	for index, keyId := range testKeyIds {
		key, err := cache.ResolveKey(keyId)
		assert.Equal(oldKeys[index], key)
		assert.Nil(err)
	}

	count, errors = cache.UpdateKeys()
	assert.Equal(len(testKeyIds), count)
	assert.Len(errors, 0)

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestMultiKeyCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	for _, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(nil, resolveKeyError).Once()
	}

	cache := multiKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	for _, keyId := range testKeyIds {
		key, err := cache.ResolveKey(keyId)
		assert.Nil(key)
		assert.Equal(resolveKeyError, err)
	}

	// UpdateKeys should still not do anything, because
	// nothing should be in the cache due to errors
	count, errors = cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestMultiKeyCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	resolver := &mockResolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()
	resolver.On("ResolveKey", testKeyIds[0]).Return(nil, resolveKeyError).Once()
	resolver.On("ResolveKey", testKeyIds[0]).Return(newKeys[0], nil).Once()

	cache := multiKeyCache{
		basicCache{
			delegate: resolver,
		},
	}

	key, err := cache.ResolveKey(testKeyIds[0])
	assert.Equal(oldKeys[0], key)
	assert.Nil(err)

	// an error should leave the existing key alone
	count, errors := cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Equal([]error{resolveKeyError}, errors)

	// the key should resolve to the old key from the cache
	key, err = cache.ResolveKey(testKeyIds[0])
	assert.Equal(oldKeys[0], key)
	assert.Nil(err)

	// again, this time the mock will succeed
	count, errors = cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Len(errors, 0)

	// resolving a key should show the new value now
	key, err = cache.ResolveKey(testKeyIds[0])
	assert.Equal(newKeys[0], key)
	assert.Nil(err)

	mock.AssertExpectationsForObjects(t, resolver.Mock)
}

func TestNewUpdaterNoRunnable(t *testing.T) {
	assert := assert.New(t)

	keyCache := &mockKeyCache{}

	var testData = []struct {
		updateInterval time.Duration
		keyCache       KeyCache
	}{
		{
			updateInterval: -1,
			keyCache:       keyCache,
		},
		{
			updateInterval: 0,
			keyCache:       keyCache,
		},
		{
			updateInterval: 1232354,
			keyCache:       nil,
		},
	}

	for _, record := range testData {
		t.Log(record)
		updater, ok := NewUpdater(record.updateInterval, record.keyCache)
		assert.Nil(updater)
		assert.False(ok)
	}

	mock.AssertExpectationsForObjects(t, keyCache.Mock)
}

func TestNewUpdater(t *testing.T) {
	assert := assert.New(t)

	updateKeysCalled := make(chan struct{})
	runner := func(mock.Arguments) {
		defer func() {
			recover() // ignore panics from multiple closes
		}()

		close(updateKeysCalled)
	}

	keyCache := &mockKeyCache{}
	keyCache.On("UpdateKeys").Return(0, nil).Run(runner)

	if updater, ok := NewUpdater(100*time.Millisecond, keyCache); assert.NotNil(updater) && assert.True(ok) {
		waitGroup := &sync.WaitGroup{}
		shutdown := make(chan struct{})
		updater.Run(waitGroup, shutdown)

		// we only care that the updater called UpdateKeys() at least once
		<-updateKeysCalled
		close(shutdown)
		waitGroup.Wait()
	}
}
