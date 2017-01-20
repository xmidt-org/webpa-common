package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func TestSubscribeEndWhenWatchClosed(t *testing.T) {
	assert := assert.New(t)
	logger := logging.TestLogger(t)

	expectedUpdates := [][]string{
		[]string{"updated1"},
		[]string{"updated2", "updated3"},
	}

	subscriptionCounter := new(sync.WaitGroup)
	subscriptionCounter.Add(len(expectedUpdates))
	actualUpdates := make(chan []string, len(expectedUpdates))
	subscription := func(update []string) {
		logger.Info("Test subscription called: %v", update)
		defer subscriptionCounter.Done()

		select {
		case actualUpdates <- update:
		default:
			assert.Fail("Subscription was called too many times")
		}
	}

	var (
		watchEvent                        = make(chan struct{})
		receiveWatchEvent <-chan struct{} = watchEvent
		mockWatch                         = new(mockWatch)
	)

	// first update
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedUpdates[0]).Once()

	// second update
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedUpdates[1]).Once()

	// watch is closed
	closeWait := new(sync.WaitGroup)
	closeWait.Add(1)
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Run(func(mock.Arguments) { closeWait.Done() }).Return(true).Once()

	logger.Info("Invoking subscribe, expecting updates: %v\n", expectedUpdates)
	cancelFunc := Subscribe(logger, mockWatch, subscription)
	if !assert.NotNil(cancelFunc) {
		close(watchEvent)
		return
	}

	watchEvent <- struct{}{}
	watchEvent <- struct{}{}
	subscriptionCounter.Wait()

	// simulate a watch event closure ...
	// need to wait, to ensure the other goroutine is finished before
	// we assert expectations
	watchEvent <- struct{}{}
	closeWait.Wait()

	close(actualUpdates)
	position := 0
	for update := range actualUpdates {
		assert.Equal(expectedUpdates[position], update)
		position++
	}

	mockWatch.AssertExpectations(t)
}

func TestSubscribeEndWhenCancelled(t *testing.T) {
	assert := assert.New(t)
	logger := logging.TestLogger(t)

	expectedUpdates := [][]string{
		[]string{"updated1"},
		[]string{"updated2", "updated3"},
	}

	subscriptionCounter := new(sync.WaitGroup)
	subscriptionCounter.Add(len(expectedUpdates))
	actualUpdates := make(chan []string, len(expectedUpdates))
	subscription := func(update []string) {
		logger.Info("Test subscription called: %v", update)
		defer subscriptionCounter.Done()

		select {
		case actualUpdates <- update:
		default:
			assert.Fail("Subscription was called too many times")
		}
	}

	var (
		watchEvent                        = make(chan struct{})
		receiveWatchEvent <-chan struct{} = watchEvent
		mockWatch                         = new(mockWatch)
	)

	// first update
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedUpdates[0]).Once()

	// second update
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedUpdates[1]).Once()

	// watch is cancelled
	mockWatch.On("Event").Return(receiveWatchEvent).Once()

	logger.Info("Invoking subscribe, expecting updates: %v\n", expectedUpdates)
	cancelFunc := Subscribe(logger, mockWatch, subscription)
	if !assert.NotNil(cancelFunc) {
		close(watchEvent)
		return
	}

	watchEvent <- struct{}{}
	watchEvent <- struct{}{}
	subscriptionCounter.Wait()
	cancelFunc()

	// the cancel function should be idempotent
	cancelFunc()

	close(actualUpdates)
	position := 0
	for update := range actualUpdates {
		assert.Equal(expectedUpdates[position], update)
		position++
	}

	mockWatch.AssertExpectations(t)
}

func TestSubscribeEndWhenSubscriptionPanics(t *testing.T) {
	assert := assert.New(t)

	expectedUpdate := []string{"update1", "update2"}
	subscriptionCounter := new(sync.WaitGroup)
	subscriptionCounter.Add(1)
	badSubscription := func(actualUpdate []string) {
		defer subscriptionCounter.Done()
		assert.Equal(expectedUpdate, actualUpdate)
		panic("Expected panic from bad subscription")
	}

	var (
		watchEvent                        = make(chan struct{})
		receiveWatchEvent <-chan struct{} = watchEvent
		mockWatch                         = new(mockWatch)
	)

	// the only update that will be attempted, as the subscription will panic
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedUpdate).Once()

	cancelFunc := Subscribe(nil, mockWatch, badSubscription)
	if !assert.NotNil(cancelFunc) {
		return
	}

	watchEvent <- struct{}{}
	subscriptionCounter.Wait()

	// cancelling any number of times after a panic should be idempotent
	cancelFunc()
	cancelFunc()

	mockWatch.AssertExpectations(t)
}

func TestAccessorSubscription(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	key := []byte("does not matter")
	expectedEndpoints := [][]string{
		[]string{"initial"},
		[]string{"update1"},
		[]string{"update2"},
	}

	var (
		watchEvent                        = make(chan struct{})
		receiveWatchEvent <-chan struct{} = watchEvent
		mockWatch                         = new(mockWatch)
	)

	// initializing the accessor subscription itself
	mockWatch.On("Endpoints").Return(expectedEndpoints[0]).Once()

	// first update
	mockWatch.On("Event").Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedEndpoints[1]).Once()

	// second update
	secondUpdate := new(sync.WaitGroup)
	secondUpdate.Add(1)
	mockWatch.On("Event").Run(func(mock.Arguments) { secondUpdate.Done() }).Return(receiveWatchEvent).Once()
	mockWatch.On("IsClosed").Return(false).Once()
	mockWatch.On("Endpoints").Return(expectedEndpoints[2]).Once()

	// watch is cancelled
	finalUpdate := new(sync.WaitGroup)
	finalUpdate.Add(1)
	mockWatch.On("Event").Run(func(mock.Arguments) { finalUpdate.Done() }).Return(receiveWatchEvent).Once()

	accessorSubscription := NewAccessorSubscription(mockWatch, nil, nil)
	require.NotNil(accessorSubscription)

	hashedEndpoint, err := accessorSubscription.Get(key)
	assert.Equal(expectedEndpoints[0][0], hashedEndpoint)
	assert.NoError(err)

	// after the second update starts, the changes from the first should be visible
	watchEvent <- struct{}{}
	secondUpdate.Wait()
	hashedEndpoint, err = accessorSubscription.Get(key)
	assert.Equal(expectedEndpoints[1][0], hashedEndpoint)
	assert.NoError(err)

	watchEvent <- struct{}{}
	finalUpdate.Wait()
	hashedEndpoint, err = accessorSubscription.Get(key)
	assert.Equal(expectedEndpoints[2][0], hashedEndpoint)
	assert.NoError(err)

	accessorSubscription.Cancel()
	hashedEndpoint, err = accessorSubscription.Get(key)
	assert.Equal(expectedEndpoints[2][0], hashedEndpoint)
	assert.NoError(err)

	// Cancel should be idempotent
	accessorSubscription.Cancel()
	hashedEndpoint, err = accessorSubscription.Get(key)
	assert.Equal(expectedEndpoints[2][0], hashedEndpoint)
	assert.NoError(err)

	mockWatch.AssertExpectations(t)
}
