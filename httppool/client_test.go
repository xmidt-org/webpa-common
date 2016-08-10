package httppool

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
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
		Client{},
		Client{
			QueueSize: -1,
			Workers:   -1,
		},
		Client{
			QueueSize: -34582,
			Workers:   -4582,
			Period:    400 * time.Second,
		},
	}

	for _, client := range testData {
		t.Logf("client: %#v", client)
		assert.Equal(http.DefaultClient, client.handler())
		assert.Equal(DefaultQueueSize, client.queueSize())
		assert.Equal(DefaultWorkers, client.workers())

		_, ok := client.logger().(*logging.LoggerWriter)
		assert.True(ok)
	}
}

func TestClientNonDefaults(t *testing.T) {
	assert := assert.New(t)

	expectedHandler := &http.Client{}
	expectedLogger := &logging.LoggerWriter{os.Stderr}

	// none of these clients should use default values
	var testData = []Client{
		Client{
			Handler:   expectedHandler,
			Logger:    expectedLogger,
			Workers:   1234,
			QueueSize: 7983481,
		},
		Client{
			Handler:   expectedHandler,
			Logger:    expectedLogger,
			Workers:   2956,
			QueueSize: 275,
			Period:    1000 * time.Hour,
		},
	}

	for _, client := range testData {
		t.Logf("client: %#v", client)
		assert.Equal(expectedHandler, client.handler())
		assert.Equal(expectedLogger, client.logger())
		assert.Equal(client.QueueSize, client.queueSize())
		assert.Equal(client.Workers, client.workers())
	}
}

func TestClientDispatcher(t *testing.T) {
	assert := assert.New(t)

	var testData = []Client{
		Client{
			Logger: testLogger,
		},
		Client{
			Logger:    testLogger,
			Workers:   1,
			QueueSize: 5,
		},
		Client{
			Logger: testLogger,
			Period: 100 * time.Millisecond,
		},
		Client{
			Logger:    testLogger,
			Workers:   12,
			QueueSize: 200,
			Period:    100 * time.Millisecond,
		},
	}

	for _, client := range testData {
		taskWaitGroup := &sync.WaitGroup{}
		taskWaitGroup.Add(taskCount)

		handler := &mockTransactionHandler{}
		client.Handler = handler

		tasks := make([]Task, 0, taskCount)

		for taskNumber := 0; taskNumber < taskCount; taskNumber++ {
			var task Task

			switch taskNumber % 3 {
			case 0:
				request := MustNewRequest("GET", fmt.Sprintf("http://example.com/%d", taskNumber))

				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return request, nil, nil
				})

				handler.On("Do", request).Return(&http.Response{}, nil)

			case 1:
				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return nil, nil, taskError
				})

			default:
				request := MustNewRequest("GET", fmt.Sprintf("http://example.com/%d", taskNumber))

				task = Task(func() (*http.Request, Consumer, error) {
					defer taskWaitGroup.Done()
					return request, nil, nil
				})

				handler.On("Do", request).Return(nil, transactionError)
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

func TestHandleTaskWhenTaskPanics(t *testing.T) {
	assert := assert.New(t)

	panickingTask := func() (*http.Request, Consumer, error) {
		panic("ow!")
	}

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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

	dispatcher, mockTransactionHandler, workerContext := newPooledDispatcher(1)
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
