package handler

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
	"net/http"
	"testing"
)

type testRequestListener struct {
	requestReceived  bool
	requestCompleted *int
}

func (l *testRequestListener) RequestReceived(*http.Request) {
	l.requestReceived = true
}

func (l *testRequestListener) RequestCompleted(statusCode int, request *http.Request) {
	l.requestCompleted = &statusCode
}

func TestListen(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		listeners      []testRequestListener
		contextHandler testContextHandler
	}{
		{
			[]testRequestListener{},
			testContextHandler{
				assert:     assert,
				statusCode: 201,
			},
		},
		{
			[]testRequestListener{
				testRequestListener{},
			},
			testContextHandler{
				assert:     assert,
				statusCode: 202,
			},
		},
		{
			[]testRequestListener{
				testRequestListener{},
				testRequestListener{},
			},
			testContextHandler{
				assert:     assert,
				statusCode: 345,
			},
		},
	}

	for _, record := range testData {
		response, request := dummyHttpOperation()
		requestListeners := make([]RequestListener, len(record.listeners))
		for index := 0; index < len(record.listeners); index++ {
			requestListeners[index] = &record.listeners[index]
		}

		Listen(requestListeners...).ServeHTTP(context.Background(), response, request, &record.contextHandler)
		assert.True(record.contextHandler.wasCalled)

		for _, listener := range record.listeners {
			assert.True(listener.requestReceived)
			if assert.NotNil(listener.requestCompleted) {
				assert.Equal(*listener.requestCompleted, record.contextHandler.statusCode)
			}
		}
	}
}

func TestListenPanic(t *testing.T) {
	assert := assert.New(t)
	var testData = []struct {
		listeners           []testRequestListener
		panicContextHandler panicContextHandler
		expectedStatusCode  int
	}{
		{
			[]testRequestListener{},
			panicContextHandler{value: http.StatusInternalServerError},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{},
			panicContextHandler{value: "ow ow stop the burning!"},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{},
			panicContextHandler{value: NewHttpError(555, "wacky error")},
			555,
		},
		{
			[]testRequestListener{
				testRequestListener{},
			},
			panicContextHandler{value: http.StatusInternalServerError},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{
				testRequestListener{},
			},
			panicContextHandler{value: "ow ow stop the burning!"},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{
				testRequestListener{},
			},
			panicContextHandler{value: NewHttpError(555, "wacky error")},
			555,
		},
		{
			[]testRequestListener{
				testRequestListener{},
				testRequestListener{},
			},
			panicContextHandler{value: http.StatusInternalServerError},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{
				testRequestListener{},
				testRequestListener{},
			},
			panicContextHandler{value: "ow ow stop the burning!"},
			http.StatusInternalServerError,
		},
		{
			[]testRequestListener{
				testRequestListener{},
				testRequestListener{},
			},
			panicContextHandler{value: NewHttpError(555, "wacky error")},
			555,
		},
	}

	for _, record := range testData {
		response, request := dummyHttpOperation()
		requestListeners := make([]RequestListener, len(record.listeners))
		for index := 0; index < len(record.listeners); index++ {
			requestListeners[index] = &record.listeners[index]
		}

		Listen(requestListeners...).ServeHTTP(context.Background(), response, request, &record.panicContextHandler)
		assert.True(record.panicContextHandler.wasCalled)

		for _, listener := range record.listeners {
			assert.True(listener.requestReceived)
			if assert.NotNil(listener.requestCompleted) {
				assert.Equal(*listener.requestCompleted, record.expectedStatusCode)
			}
		}
	}
}
