package httppool

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

const (
	requestCount = 100
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

	// all these clients should use real values, not the defaults
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
		Client{},
		Client{
			Workers:   1,
			QueueSize: 5,
		},
		Client{
			Period: 100 * time.Millisecond,
		},
		Client{
			Workers:   12,
			QueueSize: 200,
			Period:    100 * time.Millisecond,
		},
	}

	for _, client := range testData {
		taskWaitGroup := &sync.WaitGroup{}
		taskWaitGroup.Add(requestCount)

		handler := &mockTransactionHandler{}
		client.Handler = handler

		tasks := make([]Task, 0, requestCount)

		for requestNumber := 0; requestNumber < requestCount; requestNumber++ {
			request, err := http.NewRequest("GET", fmt.Sprintf("http://example.com/%d", requestNumber), nil)
			assert.NotNil(request)
			assert.Nil(err)

			var task Task

			switch requestNumber % 3 {
			case 0:
				task = Task(func() (*http.Request, error) {
					defer taskWaitGroup.Done()
					return request, nil
				})

				handler.On("Do", request).Return(&http.Response{}, nil)

			case 1:
				task = Task(func() (*http.Request, error) {
					defer taskWaitGroup.Done()
					return nil, taskError
				})

			default:
				task = Task(func() (*http.Request, error) {
					defer taskWaitGroup.Done()
					return request, nil
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
				Task(func() (*http.Request, error) {
					assert.Fail("Task should not have been called after Close()")
					return nil, errors.New("Task should not have been called after Close()")
				}),
			),
		)

		handler.AssertExpectations(t)
	}
}
