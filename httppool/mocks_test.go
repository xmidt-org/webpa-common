package httppool

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"testing"
)

var testLogger = &logging.LoggerWriter{ioutil.Discard}

func MustNewRequest(method string, url string) *http.Request {
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}

	return request
}

type mockResponseBody struct {
	mock.Mock
}

func (body *mockResponseBody) Read(buffer []byte) (int, error) {
	arguments := body.Called(buffer)
	return arguments.Int(0), arguments.Error(1)
}

func (body *mockResponseBody) Close() error {
	arguments := body.Called()
	return arguments.Error(0)
}

type mockTransactionHandler struct {
	mock.Mock
}

func (handler *mockTransactionHandler) Do(request *http.Request) (*http.Response, error) {
	arguments := handler.Called(request)

	if response, ok := arguments.Get(0).(*http.Response); ok {
		return response, arguments.Error(1)
	} else {
		return nil, arguments.Error(1)
	}
}

// mockConsumer is a bit different from a normal mock:
// Since Consumer is a function type, we can't mock it using testify.
type mockConsumer struct {
	expectsCalled   bool
	expectsRequest  *http.Request
	expectsResponse *http.Response

	wasCalled      bool
	actualResponse *http.Response
	actualRequest  *http.Request
}

func (consumer *mockConsumer) Expect(response *http.Response, request *http.Request) {
	consumer.expectsCalled = true
	consumer.expectsResponse = response
	consumer.expectsRequest = request

	consumer.wasCalled = false
	consumer.actualRequest = nil
	consumer.actualResponse = nil
}

func (consumer *mockConsumer) Consumer(response *http.Response, request *http.Request) {
	consumer.wasCalled = true
	consumer.actualResponse = response
	consumer.actualRequest = request
}

func (consumer *mockConsumer) AssertExpectations(t *testing.T) {
	if consumer.expectsCalled && !consumer.wasCalled {
		t.Error("No call to consumer was made")
	} else if !consumer.expectsCalled && consumer.wasCalled {
		t.Error("Unexpected call to consumer")
	} else if consumer.expectsCalled && consumer.wasCalled {
		assert.Equal(t, consumer.expectsResponse, consumer.actualResponse)
		assert.Equal(t, consumer.expectsRequest, consumer.actualRequest)
	}
}

// newPooledDispatcher creates a pooledDispatcher for testing.  A mockTransactionHandler
// is also returned, which is set as the pooledDispatcher.handler member as well.
func newPooledDispatcher(queueSize int) (*pooledDispatcher, *mockTransactionHandler, *workerContext) {
	handler := &mockTransactionHandler{}
	workerContext := &workerContext{
		id:            999,
		cleanupBuffer: make([]byte, 100),
	}

	return &pooledDispatcher{
		handler: handler,
		logger:  testLogger,
		tasks:   make(chan Task, queueSize),
	}, handler, workerContext
}

type mockRequestFilter struct {
	mock.Mock
}

func (filter *mockRequestFilter) Accept(request *http.Request) bool {
	arguments := filter.Called(request)
	return arguments.Bool(0)
}
