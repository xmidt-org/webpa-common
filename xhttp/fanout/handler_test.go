package fanout

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testHandlerBodyError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("body read error")
		body          = new(mockBody)
		errorEncoder  = new(mockErrorEncoder)

		original = httptest.NewRequest("POST", "/something", body)
		response = httptest.NewRecorder()
		handler  = New(FixedEndpoints{}, WithErrorEncoder(errorEncoder.Encode))
	)

	require.NotNil(handler)
	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Once().Return(0, expectedError)
	errorEncoder.On("Encode", original.Context(), expectedError, response).Once().
		Run(func(arguments mock.Arguments) {
			response := arguments.Get(2).(http.ResponseWriter)
			response.WriteHeader(599)
		})

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
	errorEncoder.AssertExpectations(t)
}

func testHandlerNoEndpoints(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		body         = new(mockBody)
		errorEncoder = new(mockErrorEncoder)

		original = httptest.NewRequest("POST", "/something", body)
		response = httptest.NewRecorder()
		handler  = New(FixedEndpoints{}, WithErrorEncoder(errorEncoder.Encode))
	)

	require.NotNil(handler)
	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Once().Return(0, io.EOF)
	errorEncoder.On("Encode", original.Context(), errNoFanoutEndpoints, response).Once().
		Run(func(arguments mock.Arguments) {
			response := arguments.Get(2).(http.ResponseWriter)
			response.WriteHeader(599)
		})

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
	errorEncoder.AssertExpectations(t)
}

func testHandlerEndpointsError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("endpoints error")
		body          = new(mockBody)
		errorEncoder  = new(mockErrorEncoder)
		endpoints     = new(mockEndpoints)

		original = httptest.NewRequest("POST", "/something", body)
		response = httptest.NewRecorder()
		handler  = New(endpoints, WithErrorEncoder(errorEncoder.Encode))
	)

	require.NotNil(handler)
	body.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Once().Return(0, io.EOF)
	errorEncoder.On("Encode", original.Context(), expectedError, response).Once().
		Run(func(arguments mock.Arguments) {
			response := arguments.Get(2).(http.ResponseWriter)
			response.WriteHeader(599)
		})
	endpoints.On("NewEndpoints", original).Once().Return(nil, expectedError)

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
	errorEncoder.AssertExpectations(t)
}

func TestHandler(t *testing.T) {
	t.Run("BodyError", testHandlerBodyError)
	t.Run("NoEndpoints", testHandlerNoEndpoints)
	t.Run("EndpointsError", testHandlerEndpointsError)
}

func testNewNilEndpoints(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		New(nil)
	})
}

func testNewNilOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		handler = New(FixedEndpoints{},
			WithShouldTerminate(nil),
			WithErrorEncoder(nil),
			WithTransactor(nil),
			WithFanoutBefore(),
			WithClientBefore(),
			WithFanoutAfter(),
		)
	)

	require.NotNil(handler)
	assert.NotNil(handler.shouldTerminate)
	assert.NotNil(handler.errorEncoder)
	assert.NotNil(handler.transactor)
	assert.Empty(handler.before)
	assert.Empty(handler.after)
}

func testNewNoOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		handler = New(FixedEndpoints{})
	)

	require.NotNil(handler)
	assert.NotNil(handler.shouldTerminate)
	assert.NotNil(handler.errorEncoder)
	assert.NotNil(handler.transactor)
	assert.Empty(handler.before)
	assert.Empty(handler.after)
}

func TestNew(t *testing.T) {
	t.Run("NilEndpoints", testNewNilEndpoints)
	t.Run("NilOptions", testNewNilOptions)
	t.Run("NoOptions", testNewNoOptions)
}
