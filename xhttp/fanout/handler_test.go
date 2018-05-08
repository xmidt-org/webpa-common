package fanout

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/xhttptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testHandlerBodyError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = &xhttp.Error{Code: 599, Text: "body read error"}
		body          = new(xhttptest.MockBody)
		logger        = logging.NewTestLogger(nil, t)
		ctx           = logging.WithLogger(context.Background(), logger)
		original      = httptest.NewRequest("POST", "/something", body).WithContext(ctx)
		response      = httptest.NewRecorder()

		handler = New(FixedEndpoints{})
	)

	require.NotNil(handler)
	body.OnReadError(expectedError).Once()

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
}

func testHandlerNoEndpoints(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		body     = new(xhttptest.MockBody)
		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		original = httptest.NewRequest("POST", "/something", body).WithContext(ctx)
		response = httptest.NewRecorder()

		handler = New(FixedEndpoints{}, WithErrorEncoder(func(_ context.Context, err error, response http.ResponseWriter) {
			response.WriteHeader(599)
		}))
	)

	require.NotNil(handler)
	body.OnReadError(io.EOF).Once()

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
}

func testHandlerEndpointsError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("endpoints error")
		body          = new(xhttptest.MockBody)
		endpoints     = new(mockEndpoints)

		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		original = httptest.NewRequest("POST", "/something", body).WithContext(ctx)
		response = httptest.NewRecorder()

		handler = New(endpoints, WithErrorEncoder(func(_ context.Context, err error, response http.ResponseWriter) {
			response.WriteHeader(599)
		}))
	)

	require.NotNil(handler)
	body.OnReadError(io.EOF).Once()
	endpoints.On("NewEndpoints", original).Once().Return(nil, expectedError)

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
}

func testHandlerGet(t *testing.T, expectedResponses []xhttptest.ExpectedResponse, expectedStatusCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		original = httptest.NewRequest("GET", "/api/v2/something", nil).WithContext(ctx)
		response = httptest.NewRecorder()

		endpoints  = generateEndpoints(len(expectedResponses))
		transactor = new(xhttptest.MockTransactor)
		handler    = New(endpoints, WithTransactor(transactor.Do))
	)

	require.NotNil(handler)
	for i, er := range expectedResponses {
		transactor.OnDo(
			xhttptest.MatchMethod("GET"),
			xhttptest.MatchURLString(endpoints[i].String()+"/api/v2/something"),
		).Respond(er).Once()
	}

	handler.ServeHTTP(response, original)
	assert.Equal(expectedStatusCode, response.Code)

	transactor.AssertExpectations(t)
}

func TestHandler(t *testing.T) {
	t.Run("BodyError", testHandlerBodyError)
	t.Run("NoEndpoints", testHandlerNoEndpoints)
	t.Run("EndpointsError", testHandlerEndpointsError)

	testData := []struct {
		expectedResponses  []xhttptest.ExpectedResponse
		expectedStatusCode int
	}{
		{
			[]xhttptest.ExpectedResponse{
				{Err: errors.New("expected")},
			},
			http.StatusServiceUnavailable,
		},
		{
			[]xhttptest.ExpectedResponse{
				{Response: xhttptest.NewResponse(504, nil)},
			},
			504,
		},
	}

	t.Run("GET", func(t *testing.T) {
		for _, record := range testData {
			testHandlerGet(t, record.expectedResponses, record.expectedStatusCode)
		}
	})
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
