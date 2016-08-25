package key

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/secure/key/keymock"
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

func TestBasicCacheStoreAndLoad(t *testing.T) {
	assert := assert.New(t)

	cache := basicCache{}
	assert.Nil(cache.load())
	cache.store(123)
	assert.Equal(123, cache.load())
}

func TestSingleCacheResolveKey(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()

	cache := singleCache{
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

func TestSingleCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(nil, resolveKeyError).Times(routineCount)

	cache := singleCache{
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

func TestSingleCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(oldKeys[0], nil).Once()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	mock.AssertExpectationsForObjects(t, resolver.Mock)
	assert.Equal(1, count)
	assert.Nil(errors)
}

func TestSingleCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(nil, resolveKeyError).Once()

	cache := singleCache{
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

func TestSingleCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(nil, resolveKeyError).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(newKeys[0], nil).Once()

	cache := singleCache{
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

func TestMultiCacheResolveKey(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	for index, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(oldKeys[index], nil).Once()
	}

	cache := multiCache{
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

func TestMultiCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	for _, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(nil, resolveKeyError).Twice()
	}

	cache := multiCache{
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

func TestMultiCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	for index, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(oldKeys[index], nil).Twice()
	}

	cache := multiCache{
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

func TestMultiCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	for _, keyId := range testKeyIds {
		resolver.On("ResolveKey", keyId).Return(nil, resolveKeyError).Once()
	}

	cache := multiCache{
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

func TestMultiCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	resolver := &keymock.Resolver{}
	resolver.On("ResolveKey", testKeyIds[0]).Return(oldKeys[0], nil).Once()
	resolver.On("ResolveKey", testKeyIds[0]).Return(nil, resolveKeyError).Once()
	resolver.On("ResolveKey", testKeyIds[0]).Return(newKeys[0], nil).Once()

	cache := multiCache{
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

	keyCache := &keymock.Cache{}

	var testData = []struct {
		updateInterval time.Duration
		keyCache       Cache
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
		updater := NewUpdater(record.updateInterval, record.keyCache)
		assert.Nil(updater)
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

	keyCache := &keymock.Cache{}
	keyCache.On("UpdateKeys").Return(0, nil).Run(runner)

	if updater := NewUpdater(100*time.Millisecond, keyCache); assert.NotNil(updater) {
		waitGroup := &sync.WaitGroup{}
		shutdown := make(chan struct{})
		updater.Run(waitGroup, shutdown)

		// we only care that the updater called UpdateKeys() at least once
		<-updateKeysCalled
		close(shutdown)
		waitGroup.Wait()
	}
}
