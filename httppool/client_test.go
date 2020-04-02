package httppool

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/logging"
)

const (
	taskCount = 100
)

var (
	taskError        = errors.New("Task error")
	transactionError = errors.New("Transaction error")
)

func TestClientDefaults(t *testing.T) {
	assert := assert.New(t)

	// all these clients should use default values
	var testData = []Client{
		{},
		{
			QueueSize: -1,
			Workers:   -1,
		},
		{
			QueueSize: -34582,
			Workers:   -4582,
			Period:    400 * time.Second,
		},
	}

	for _, client := range testData {
		t.Logf("client: %#v", client)
		assert.NotEmpty(client.name())
		assert.Equal(http.DefaultClient, client.handler())
		assert.Equal(DefaultQueueSize, client.queueSize())
		assert.Equal(DefaultWorkers, client.workers())
		assert.NotNil(client.logger())
	}
}

func TestClientNonDefaults(t *testing.T) {
	assert := assert.New(t)

	expectedName := "expected"
	expectedHandler := &http.Client{}
	expectedLogger := logging.DefaultLogger()

	// none of these clients should use default values
	var testData = []Client{
		{
			Name:      expectedName,
			Handler:   expectedHandler,
			Logger:    expectedLogger,
			Workers:   1234,
			QueueSize: 7983481,
		},
		{
			Name:      expectedName,
			Handler:   expectedHandler,
			Logger:    expectedLogger,
			Workers:   2956,
			QueueSize: 275,
			Period:    1000 * time.Hour,
		},
	}

	for _, client := range testData {
		t.Logf("client: %#v", client)
		assert.Equal(expectedName, client.name())
		assert.Equal(expectedHandler, client.handler())
		assert.Equal(expectedLogger, client.logger())
		assert.Equal(client.QueueSize, client.queueSize())
		assert.Equal(client.Workers, client.workers())
	}
}

func TestClientDispatcherUsingSend(t *testing.T) {
	assert := assert.New(t)
	logger := logging.DefaultLogger()

	var testData = []Client{
		{
			Logger: logger,
		},
		{
			Logger:    logger,
			Workers:   1,
			QueueSize: 5,
		},
		{
			Logger: logger,
			Period: 100 * time.Millisecond,
		},
		{
			Logger:    logger,
			Workers:   12,
			QueueSize: 200,
			Period:    100 * time.Millisecond,
		},
	}

	for _, client := range testData {
		var (
			taskWaitGroup = new(sync.WaitGroup)
			handler       = new(mockTransactionHandler)
			tasks         []Task
		)

		// TODO: Tried several things to be more specific instead of using a stub.  Want to fix this
		// so it expects specific requests.
		handler.On("Do", mock.AnythingOfType("*http.Request")).Return(new(http.Response), (error)(nil))

		taskWaitGroup.Add(taskCount)
		client.Handler = handler

		for taskNumber := 0; taskNumber < taskCount; taskNumber++ {
			var (
				task    Task
				url     = fmt.Sprintf("http://example.com/%d", taskNumber)
				request = httptest.NewRequest("GET", url, nil)
			)

			switch taskNumber % 3 {
			case 0:
				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return request, nil, nil
				})

			case 1:
				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return nil, nil, taskError
				})

			default:
				request := httptest.NewRequest("GET", fmt.Sprintf("http://example.com/%d", taskNumber), nil)

				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return request, nil, nil
				})
			}

			tasks = append(tasks, task)
		}

		dispatcher := client.Start()
		assert.NotNil(dispatcher)

		for _, task := range tasks {
			assert.Nil(dispatcher.Send(task))
		}

		taskWaitGroup.Wait()
		assert.Nil(dispatcher.Close())
		assert.Equal(ErrorClosed, dispatcher.Close())

		assert.Equal(
			ErrorClosed,
			dispatcher.Send(
				Task(func() (*http.Request, Consumer, error) {
					assert.Fail("Task should not have been called after Close()")
					return nil, nil, errors.New("Task should not have been called after Close()")
				}),
			),
		)

		handler.AssertExpectations(t)
	}
}

func TestOffer(t *testing.T) {
	fmt.Println("Start")
	assert := assert.New(t)

	consumerBlocker := make(chan struct{})
	consumerWaitGroup := &sync.WaitGroup{}
	consumerWaitGroup.Add(2)
	consumer := func(*http.Response, *http.Request) {
		defer consumerWaitGroup.Done()
		<-consumerBlocker
	}

	longRunningRequest := MustNewRequest("GET", "http//longrunning.com")
	longRunningResponse := &http.Response{}

	quickRequest := MustNewRequest("GET", "http//quick.com")
	quickResponse := &http.Response{}

	mockTransactionHandler := &mockTransactionHandler{}
	mockTransactionHandler.On("Do", longRunningRequest).Return(longRunningResponse, nil).Once()
	mockTransactionHandler.On("Do", quickRequest).Return(quickResponse, nil).Once()

	dispatcher := (&Client{
		Name:      "TestOffer",
		Handler:   mockTransactionHandler,
		Workers:   1,
		QueueSize: 1,
		Logger:    logging.DefaultLogger(),
	}).Start()

	// first, hold up the dispatcher with a long running request we control
	taken, err := dispatcher.Offer(RequestTask(longRunningRequest, consumer))
	assert.True(taken)
	assert.Nil(err)

	// now, send a quick request, to block until its accepted
	// we know this will be accepted because the queue size is 1 and we have sent 1 task that
	// should now (or will be soon) executing
	err = dispatcher.Send(RequestTask(quickRequest, consumer))
	assert.Nil(err)

	// offer up a random task that should be rejected since the queue is now full
	taken, err = dispatcher.Offer(RequestTask(quickRequest, nil))
	assert.False(taken)
	assert.Nil(err)

	// now let all consumers run
	close(consumerBlocker)

	// wait for our 2 consumers to run
	consumerWaitGroup.Wait()

	// now offer something when closed, which should return an error
	assert.Nil(dispatcher.Close())
	assert.Equal(ErrorClosed, dispatcher.Close())

	taken, err = dispatcher.Offer(RequestTask(quickRequest, nil))
	assert.False(taken)
	assert.Equal(ErrorClosed, err)

	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTaskWhenTaskPanics(t *testing.T) {
	assert := assert.New(t)

	panickingTask := func() (*http.Request, Consumer, error) {
		panic("ow!")
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	assert.NotPanics(func() {
		dispatcher.handleTask(workerContext, panickingTask)
	})

	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTaskWhenTaskReturnsNilRequest(t *testing.T) {
	mockConsumer := &mockConsumer{}

	nilRequestTask := func() (*http.Request, Consumer, error) {
		return nil, mockConsumer.Consumer, nil
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	dispatcher.handleTask(workerContext, nilRequestTask)
	mockConsumer.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTaskWhenTransactionError(t *testing.T) {
	mockConsumer := &mockConsumer{}
	request := MustNewRequest("GET", "http://example.com")

	task := func() (*http.Request, Consumer, error) {
		return request, mockConsumer.Consumer, nil
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	expectedError := errors.New("expected")
	mockTransactionHandler.On("Do", request).Return(nil, expectedError).Once()

	dispatcher.handleTask(workerContext, task)
	mockConsumer.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTaskWhenNonNilResponseAndError(t *testing.T) {
	mockConsumer := &mockConsumer{}
	mockResponseBody := &mockResponseBody{}
	request := MustNewRequest("GET", "http://example.com")
	response := &http.Response{Body: mockResponseBody}

	task := func() (*http.Request, Consumer, error) {
		return request, mockConsumer.Consumer, nil
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	expectedError := errors.New("expected")
	mockTransactionHandler.On("Do", request).Return(response, expectedError).Once()
	mockResponseBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, io.EOF).Once()
	mockResponseBody.On("Close").Return(nil).Once()

	dispatcher.handleTask(workerContext, task)
	mockConsumer.AssertExpectations(t)
	mockResponseBody.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTaskCleanupError(t *testing.T) {
	mockConsumer := &mockConsumer{}
	mockResponseBody := &mockResponseBody{}
	request := MustNewRequest("GET", "http://example.com")
	response := &http.Response{Body: mockResponseBody}

	task := func() (*http.Request, Consumer, error) {
		return request, mockConsumer.Consumer, nil
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	mockTransactionHandler.On("Do", request).Return(response, nil).Once()
	mockConsumer.Expect(response, request)
	mockResponseBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, io.ErrNoProgress).Once()
	mockResponseBody.On("Close").Return(nil).Once()

	mockConsumer.Consumer(response, request)
	dispatcher.handleTask(workerContext, task)
	mockConsumer.AssertExpectations(t)
	mockResponseBody.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestHandleTask(t *testing.T) {
	mockConsumer := &mockConsumer{}
	mockResponseBody := &mockResponseBody{}
	request := MustNewRequest("GET", "http://example.com")
	response := &http.Response{Body: mockResponseBody}

	task := func() (*http.Request, Consumer, error) {
		return request, mockConsumer.Consumer, nil
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(t, 1)
	defer dispatcher.Close()

	mockTransactionHandler.On("Do", request).Return(response, nil).Once()
	mockConsumer.Expect(response, request)
	mockResponseBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, io.EOF).Once()
	mockResponseBody.On("Close").Return(nil).Once()

	mockConsumer.Consumer(response, request)
	dispatcher.handleTask(workerContext, task)
	mockConsumer.AssertExpectations(t)
	mockResponseBody.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestSendUsingListener(t *testing.T) {
	assert := assert.New(t)
	request := MustNewRequest("GET", "http://example.com")
	response := &http.Response{}
	task := func() (*http.Request, Consumer, error) {
		return request, nil, nil
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestSendUsingListener",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockTransactionHandler.On("Do", request).Return(response, nil)
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, nil)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	err := dispatcher.Send(task)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestSendUsingListenerWhenTaskError(t *testing.T) {
	assert := assert.New(t)
	taskError := errors.New("expected")
	task := func() (*http.Request, Consumer, error) {
		return nil, nil, taskError
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestSendUsingListenerWhenTaskError",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, taskError)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	err := dispatcher.Send(task)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestSendUsingListenerWhenTransactionError(t *testing.T) {
	assert := assert.New(t)
	request := MustNewRequest("GET", "http://example.com")
	transactionError := errors.New("expected")
	task := func() (*http.Request, Consumer, error) {
		return request, nil, nil
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestSendUsingListenerWhenTransactionError",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockTransactionHandler.On("Do", request).Return(nil, transactionError)
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, transactionError)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	err := dispatcher.Send(task)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestOfferUsingListener(t *testing.T) {
	assert := assert.New(t)
	request := MustNewRequest("GET", "http://example.com")
	response := &http.Response{}
	task := func() (*http.Request, Consumer, error) {
		return request, nil, nil
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestOfferUsingListener",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockTransactionHandler.On("Do", request).Return(response, nil)
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, nil)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	taken, err := dispatcher.Offer(task)
	assert.True(taken)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestOfferUsingListenerWhenTaskError(t *testing.T) {
	assert := assert.New(t)
	taskError := errors.New("expected")
	task := func() (*http.Request, Consumer, error) {
		return nil, nil, taskError
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestOfferUsingListenerWhenTaskError",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, taskError)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	taken, err := dispatcher.Offer(task)
	assert.True(taken)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}

func TestOfferUsingListenerWhenTransactionError(t *testing.T) {
	assert := assert.New(t)
	request := MustNewRequest("GET", "http://example.com")
	transactionError := errors.New("expected")
	task := func() (*http.Request, Consumer, error) {
		return request, nil, nil
	}

	mockTransactionHandler := &mockTransactionHandler{}
	mockListener := &mockListener{}

	dispatcher := (&Client{
		Name:      "TestOfferUsingListenerWhenTransactionError",
		Handler:   mockTransactionHandler,
		Listeners: []Listener{mockListener},
		Logger:    logging.DefaultLogger(),
		QueueSize: 1,
		Workers:   1,
	}).Start()

	defer dispatcher.Close()

	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	mockListener.Mock.On("On", matchEvent(EventTypeQueue, nil)).Once()
	mockListener.Mock.On("On", matchEvent(EventTypeStart, nil)).Once()
	mockTransactionHandler.On("Do", request).Return(nil, transactionError)
	mockListener.Mock.On("On", matchEvent(EventTypeFinish, transactionError)).
		Run(func(mock.Arguments) { waitGroup.Done() }).
		Once()

	taken, err := dispatcher.Offer(task)
	assert.True(taken)
	assert.Nil(err)

	waitGroup.Wait()
	mockListener.AssertExpectations(t)
	mockTransactionHandler.AssertExpectations(t)
}
