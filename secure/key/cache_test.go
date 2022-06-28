package key

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeExpectedPairs(count int) (expectedKeyIDs []string, expectedPairs map[string]Pair) {
	expectedPairs = make(map[string]Pair, count)
	for index := 0; index < count; index++ {
		keyID := fmt.Sprintf("key#%d", index)
		expectedKeyIDs = append(expectedKeyIDs, keyID)
		expectedPairs[keyID] = &MockPair{}
	}

	return
}

func assertExpectationsForPairs(t *testing.T, pairs map[string]Pair) {
	for _, pair := range pairs {
		if mockPair, ok := pair.(*MockPair); ok {
			mock.AssertExpectationsForObjects(t, mockPair)
		}
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

	const keyID = "TestSingleCacheResolveKey"
	expectedPair := &MockPair{}
	resolver := &MockResolver{}
	resolver.On("ResolveKey", keyID).Return(expectedPair, nil).Once()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)
	barrier := make(chan struct{})

	for repeat := 0; repeat < 2; repeat++ {
		go func() {
			defer waitGroup.Done()
			<-barrier
			actualPair, err := cache.ResolveKey(keyID)
			assert.Equal(expectedPair, actualPair)
			assert.Nil(err)
		}()
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, expectedPair)
	mock.AssertExpectationsForObjects(t, resolver)
	assert.Equal(expectedPair, cache.load())
}

func TestSingleCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	const keyID = "TestSingleCacheResolveKeyError"
	expectedError := errors.New("TestSingleCacheResolveKeyError")
	resolver := &MockResolver{}
	resolver.On("ResolveKey", keyID).Return(nil, expectedError).Twice()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(2)
	barrier := make(chan struct{})

	for repeat := 0; repeat < 2; repeat++ {
		go func() {
			defer waitGroup.Done()
			<-barrier
			pair, err := cache.ResolveKey(keyID)
			assert.Nil(pair)
			assert.Equal(expectedError, err)
		}()
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver)
	assert.Nil(cache.load())
}

func TestSingleCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	expectedPair := &MockPair{}
	resolver := &MockResolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(expectedPair, nil).Once()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	mock.AssertExpectationsForObjects(t, expectedPair)
	mock.AssertExpectationsForObjects(t, resolver)
	assert.Equal(1, count)
	assert.Nil(errors)
}

func TestSingleCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	expectedError := errors.New("TestSingleCacheUpdateKeysError")
	resolver := &MockResolver{}
	resolver.On("ResolveKey", dummyKeyId).Return(nil, expectedError).Once()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	mock.AssertExpectationsForObjects(t, resolver)
	assert.Equal(1, count)
	assert.Equal([]error{expectedError}, errors)

	mock.AssertExpectationsForObjects(t, resolver)
}

func TestSingleCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	const keyID = "TestSingleCacheUpdateKeysSequence"
	expectedError := errors.New("TestSingleCacheUpdateKeysSequence")
	oldPair := &MockPair{}
	newPair := &MockPair{}
	resolver := &MockResolver{}
	resolver.On("ResolveKey", keyID).Return(oldPair, nil).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(nil, expectedError).Once()
	resolver.On("ResolveKey", dummyKeyId).Return(newPair, nil).Once()

	cache := singleCache{
		basicCache{
			delegate: resolver,
		},
	}

	firstPair, err := cache.ResolveKey(keyID)
	assert.Equal(oldPair, firstPair)
	assert.Nil(err)

	count, errors := cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Equal([]error{expectedError}, errors)

	// resolving should pull the key from the cache
	firstPair, err = cache.ResolveKey(keyID)
	assert.Equal(oldPair, firstPair)
	assert.Nil(err)

	// this time, the mock will succeed
	count, errors = cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Len(errors, 0)

	// resolving should pull the *new* key from the cache
	secondPair, err := cache.ResolveKey(keyID)
	assert.Equal(newPair, secondPair)
	assert.Nil(err)

	mock.AssertExpectationsForObjects(t, resolver)
}

func TestMultiCacheResolveKey(t *testing.T) {
	assert := assert.New(t)

	expectedKeyIDs, expectedPairs := makeExpectedPairs(2)
	resolver := &MockResolver{}
	for _, keyID := range expectedKeyIDs {
		resolver.On("ResolveKey", keyID).Return(expectedPairs[keyID], nil).Once()
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
	waitGroup.Add(5 * len(expectedKeyIDs))
	barrier := make(chan struct{})

	for repeat := 0; repeat < 5; repeat++ {
		for _, keyID := range expectedKeyIDs {
			go func(keyID string, expectedPair Pair) {
				t.Logf("keyID=%s", keyID)
				defer waitGroup.Done()
				<-barrier
				pair, err := cache.ResolveKey(keyID)
				assert.Equal(expectedPair, pair)
				assert.Nil(err)
			}(keyID, expectedPairs[keyID])
		}
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver)
	assertExpectationsForPairs(t, expectedPairs)
}

func TestMultiCacheResolveKeyError(t *testing.T) {
	assert := assert.New(t)

	expectedError := errors.New("TestMultiCacheResolveKeyError")
	expectedKeyIDs, _ := makeExpectedPairs(2)
	resolver := &MockResolver{}
	for _, keyID := range expectedKeyIDs {
		resolver.On("ResolveKey", keyID).Return(nil, expectedError).Twice()
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
	waitGroup.Add(2 * len(expectedKeyIDs))
	barrier := make(chan struct{})

	for repeat := 0; repeat < 2; repeat++ {
		for _, keyID := range expectedKeyIDs {
			go func(keyID string) {
				defer waitGroup.Done()
				<-barrier
				key, err := cache.ResolveKey(keyID)
				assert.Nil(key)
				assert.Equal(expectedError, err)
			}(keyID)
		}
	}

	close(barrier)
	waitGroup.Wait()

	mock.AssertExpectationsForObjects(t, resolver)
}

func TestMultiCacheUpdateKeys(t *testing.T) {
	assert := assert.New(t)

	resolver := &MockResolver{}
	expectedKeyIDs, expectedPairs := makeExpectedPairs(2)
	t.Logf("expectedKeyIDs: %s", expectedKeyIDs)

	for _, keyID := range expectedKeyIDs {
		resolver.On("ResolveKey", keyID).Return(expectedPairs[keyID], nil).Twice()
	}

	cache := multiCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	for _, keyID := range expectedKeyIDs {
		pair, err := cache.ResolveKey(keyID)
		assert.Equal(expectedPairs[keyID], pair)
		assert.Nil(err)
	}

	count, errors = cache.UpdateKeys()
	assert.Equal(len(expectedKeyIDs), count)
	assert.Len(errors, 0)

	mock.AssertExpectationsForObjects(t, resolver)
	assertExpectationsForPairs(t, expectedPairs)
}

func TestMultiCacheUpdateKeysError(t *testing.T) {
	assert := assert.New(t)

	expectedError := errors.New("TestMultiCacheUpdateKeysError")
	expectedKeyIDs, _ := makeExpectedPairs(2)
	resolver := &MockResolver{}
	for _, keyID := range expectedKeyIDs {
		resolver.On("ResolveKey", keyID).Return(nil, expectedError).Once()
	}

	cache := multiCache{
		basicCache{
			delegate: resolver,
		},
	}

	count, errors := cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	for _, keyID := range expectedKeyIDs {
		key, err := cache.ResolveKey(keyID)
		assert.Nil(key)
		assert.Equal(expectedError, err)
	}

	// UpdateKeys should still not do anything, because
	// nothing should be in the cache due to errors
	count, errors = cache.UpdateKeys()
	assert.Equal(0, count)
	assert.Len(errors, 0)

	mock.AssertExpectationsForObjects(t, resolver)
}

func TestMultiCacheUpdateKeysSequence(t *testing.T) {
	assert := assert.New(t)

	const keyID = "TestMultiCacheUpdateKeysSequence"
	expectedError := errors.New("TestMultiCacheUpdateKeysSequence")
	oldPair := &MockPair{}
	newPair := &MockPair{}

	resolver := &MockResolver{}
	resolver.On("ResolveKey", keyID).Return(oldPair, nil).Once()
	resolver.On("ResolveKey", keyID).Return(nil, expectedError).Once()
	resolver.On("ResolveKey", keyID).Return(newPair, nil).Once()

	cache := multiCache{
		basicCache{
			delegate: resolver,
		},
	}

	pair, err := cache.ResolveKey(keyID)
	assert.Equal(oldPair, pair)
	assert.Nil(err)

	// an error should leave the existing key alone
	count, errors := cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Equal([]error{expectedError}, errors)

	// the key should resolve to the old key from the cache
	pair, err = cache.ResolveKey(keyID)
	assert.Equal(oldPair, pair)
	assert.Nil(err)

	// again, this time the mock will succeed
	count, errors = cache.UpdateKeys()
	assert.Equal(1, count)
	assert.Len(errors, 0)

	// resolving a key should show the new value now
	pair, err = cache.ResolveKey(keyID)
	assert.Equal(newPair, pair)
	assert.Nil(err)

	mock.AssertExpectationsForObjects(t, resolver, oldPair, newPair)
}

func TestNewUpdaterNoRunnable(t *testing.T) {
	assert := assert.New(t)

	keyCache := &MockCache{}

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

	mock.AssertExpectationsForObjects(t, keyCache)
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

	keyCache := &MockCache{}
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
