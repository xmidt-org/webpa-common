package fanout

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xhttp"
	"github.com/Comcast/webpa-common/xhttp/xhttptest"
	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	endpoints.On("FanoutURLs", original).Once().Return(nil, expectedError)

	handler.ServeHTTP(response, original)
	assert.Equal(599, response.Code)

	body.AssertExpectations(t)
}

func testHandlerBadTransactor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		original = httptest.NewRequest("GET", "/api/v2/something", nil).WithContext(ctx)
		response = httptest.NewRecorder()

		endpoints  = generateEndpoints(1)
		transactor = new(xhttptest.MockTransactor)
		complete   = make(chan struct{}, 1)
		handler    = New(endpoints, WithTransactor(transactor.Do))
	)

	require.NotNil(handler)
	transactor.OnDo(
		xhttptest.MatchMethod("GET"),
		xhttptest.MatchURLString(endpoints[0].String()+"/api/v2/something"),
	).Respond(nil, nil).Once().Run(func(mock.Arguments) { complete <- struct{}{} })

	handler.ServeHTTP(response, original)
	assert.Equal(http.StatusInternalServerError, response.Code)

	select {
	case <-complete:
		// passing
	case <-time.After(5 * time.Second):
		assert.Fail("Not all transactors completed")
	}

	transactor.AssertExpectations(t)
}

func testHandlerGet(t *testing.T, expectedResponses []xhttptest.ExpectedResponse, expectedStatusCode int, expectedResponseBody string, expectAfter bool, expectedFailedCalled bool) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger   = logging.NewTestLogger(nil, t)
		ctx      = logging.WithLogger(context.Background(), logger)
		original = httptest.NewRequest("GET", "/api/v2/something", nil).WithContext(ctx)
		response = httptest.NewRecorder()

		fanoutAfterCalled = false
		fanoutAfter       = func(actualCtx context.Context, actualResponse http.ResponseWriter, result Result) context.Context {
			assert.False(fanoutAfterCalled)
			fanoutAfterCalled = true
			assert.Equal(ctx, actualCtx)
			assert.Equal(response, actualResponse)
			if assert.NotNil(result.Response) {
				assert.Equal(expectedStatusCode, result.Response.StatusCode)
			}

			return actualCtx
		}

		clientAfterCalled = false
		clientAfter       = func(actualCtx context.Context, actualResponse *http.Response) context.Context {
			assert.False(clientAfterCalled)
			clientAfterCalled = true
			assert.Equal(ctx, actualCtx)
			assert.Equal(expectedStatusCode, actualResponse.StatusCode)
			return actualCtx
		}

		fanoutFailedCalled = false
		fanoutFail         = func(actualCtx context.Context, actualResponse http.ResponseWriter, result Result) context.Context {
			assert.False(fanoutFailedCalled)
			fanoutFailedCalled = true
			assert.Equal(ctx, actualCtx)
			return ctx
		}

		endpoints  = generateEndpoints(len(expectedResponses))
		transactor = new(xhttptest.MockTransactor)
		complete   = make(chan struct{}, len(expectedResponses))

		handler = New(endpoints,
			WithTransactor(transactor.Do),
			WithClientBefore(gokithttp.SetRequestHeader("X-Test", "foobar")),
			WithFanoutAfter(fanoutAfter),
			WithClientAfter(clientAfter),
			WithFanoutFailure(fanoutFail),
		)
	)

	require.NotNil(handler)
	for i, er := range expectedResponses {
		transactor.OnDo(
			xhttptest.MatchMethod("GET"),
			xhttptest.MatchURLString(endpoints[i].String()+"/api/v2/something"),
			xhttptest.MatchHeader("X-Test", "foobar"),
		).RespondWith(er).Once().Run(func(mock.Arguments) { complete <- struct{}{} })
	}

	handler.ServeHTTP(response, original)
	assert.Equal(expectedStatusCode, response.Code)

	after := time.After(5 * time.Second)
	for i := 0; i < len(expectedResponses); i++ {
		select {
		case <-complete:
			// passing
		case <-after:
			assert.Fail("Not all transactors completed")
			i = len(expectedResponses)
		}
	}

	assert.Equal(expectAfter, clientAfterCalled)
	assert.Equal(expectedFailedCalled, fanoutFailedCalled)
	transactor.AssertExpectations(t)
}

func testHandlerPost(t *testing.T, expectedResponses []xhttptest.ExpectedResponse, expectedStatusCode int, expectedResponseBody string, expectAfter bool, expectedFailedCalled bool) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger              = logging.NewTestLogger(nil, t)
		ctx                 = logging.WithLogger(context.Background(), logger)
		expectedRequestBody = "posted body"
		original            = httptest.NewRequest("POST", "/api/v2/something", strings.NewReader(expectedRequestBody)).WithContext(ctx)
		response            = httptest.NewRecorder()

		fanoutAfterCalled = false
		fanoutAfter       = func(actualCtx context.Context, actualResponse http.ResponseWriter, result Result) context.Context {
			assert.False(fanoutAfterCalled)
			fanoutAfterCalled = true
			assert.Equal(ctx, actualCtx)
			assert.Equal(response, actualResponse)
			if assert.NotNil(result.Response) {
				assert.Equal(expectedStatusCode, result.Response.StatusCode)
			}

			return actualCtx
		}

		clientAfterCalled = false
		clientAfter       = func(actualCtx context.Context, actualResponse *http.Response) context.Context {
			assert.False(clientAfterCalled)
			clientAfterCalled = true
			assert.Equal(ctx, actualCtx)
			assert.Equal(expectedStatusCode, actualResponse.StatusCode)
			return actualCtx
		}
		fanoutFailedCalled = false
		fanoutFail         = func(actualCtx context.Context, actualResponse http.ResponseWriter, result Result) context.Context {
			assert.False(fanoutFailedCalled)
			fanoutFailedCalled = true
			assert.Equal(ctx, actualCtx)
			return ctx
		}

		endpoints  = generateEndpoints(len(expectedResponses))
		transactor = new(xhttptest.MockTransactor)
		complete   = make(chan struct{}, len(expectedResponses))
		handler    = New(endpoints,
			WithTransactor(transactor.Do),
			WithFanoutBefore(ForwardBody(true)),
			WithClientBefore(gokithttp.SetRequestHeader("X-Test", "foobar")),
			WithFanoutAfter(fanoutAfter),
			WithClientAfter(clientAfter),
			WithFanoutFailure(fanoutFail),
		)
	)

	require.NotNil(handler)
	for i, er := range expectedResponses {
		transactor.OnDo(
			xhttptest.MatchMethod("POST"),
			xhttptest.MatchURLString(endpoints[i].String()+"/api/v2/something"),
			xhttptest.MatchHeader("X-Test", "foobar"),
			xhttptest.MatchBodyString(expectedRequestBody),
		).RespondWith(er).Once().Run(func(mock.Arguments) { complete <- struct{}{} })
	}

	handler.ServeHTTP(response, original)
	assert.Equal(expectedStatusCode, response.Code)
	assert.Equal(expectedResponseBody, response.Body.String())
	assert.Equal(expectAfter, clientAfterCalled)
	assert.Equal(expectedFailedCalled, fanoutFailedCalled)

	after := time.After(2 * time.Second)
	for i := 0; i < len(expectedResponses); i++ {
		select {
		case <-complete:
			// passing
		case <-after:
			assert.Fail("Not all transactors completed")
			i = len(expectedResponses)
		}
	}

	transactor.AssertExpectations(t)
}

func testHandlerTimeout(t *testing.T, endpointCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		logger      = logging.NewTestLogger(nil, t)
		ctx, cancel = context.WithCancel(logging.WithLogger(context.Background(), logger))
		original    = httptest.NewRequest("GET", "/api/v2/something", nil).WithContext(ctx)
		response    = httptest.NewRecorder()

		endpoints      = generateEndpoints(endpointCount)
		transactor     = new(xhttptest.MockTransactor)
		transactorWait = make(chan time.Time)
		complete       = make(chan struct{}, endpointCount)
		handlerWait    = make(chan struct{})
		handler        = New(endpoints,
			WithTransactor(transactor.Do),
		)
	)

	require.NotNil(handler)
	for i := 0; i < endpointCount; i++ {
		transactor.OnDo(
			xhttptest.MatchMethod("GET"),
			xhttptest.MatchURLString(endpoints[i].String()+"/api/v2/something"),
		).Respond(nil, nil).Once().WaitUntil(transactorWait).Run(func(mock.Arguments) { complete <- struct{}{} })
	}

	go func() {
		defer close(handlerWait)
		handler.ServeHTTP(response, original)
	}()

	// simulate a context timeout
	cancel()
	select {
	case <-handlerWait:
		assert.Equal(http.StatusGatewayTimeout, response.Code)
	case <-time.After(2 * time.Second):
		assert.Fail("ServeHTTP did not return")
	}

	close(transactorWait)
	after := time.After(2 * time.Second)
	for i := 0; i < endpointCount; i++ {
		select {
		case <-complete:
			// passing
		case <-after:
			assert.Fail("Not all transactors completed")
			i = endpointCount
		}
	}

	transactor.AssertExpectations(t)
}

func TestHandler(t *testing.T) {
	t.Run("BodyError", testHandlerBodyError)
	t.Run("NoEndpoints", testHandlerNoEndpoints)
	t.Run("EndpointsError", testHandlerEndpointsError)
	t.Run("BadTransactor", testHandlerBadTransactor)

	t.Run("Fanout", func(t *testing.T) {
		testData := []struct {
			statusCodes          []xhttptest.ExpectedResponse
			expectedStatusCode   int
			expectedResponseBody string
			expectAfter          bool
			expectedFailedCalled bool
		}{
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 504},
				},
				504,
				"",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 500}, {StatusCode: 501}, {StatusCode: 502}, {StatusCode: 503}, {StatusCode: 504},
				},
				504,
				"",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 504}, {StatusCode: 503}, {StatusCode: 502}, {StatusCode: 501}, {StatusCode: 500},
				},
				504,
				"",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{Err: errors.New("expected")},
				},
				http.StatusServiceUnavailable,
				"expected",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 500}, {Err: errors.New("expected")},
				},
				http.StatusServiceUnavailable,
				"expected",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 599}, {Err: errors.New("expected")},
				},
				599,
				"",
				false,
				true,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 200, Body: []byte("expected body")},
				},
				200,
				"expected body",
				true,
				false,
			},
			{
				[]xhttptest.ExpectedResponse{
					{StatusCode: 404}, {StatusCode: 200, Body: []byte("expected body")}, {StatusCode: 503},
				},
				200,
				"expected body",
				true,
				false,
			},
		}

		t.Run("GET", func(t *testing.T) {
			for _, record := range testData {
				testHandlerGet(t, record.statusCodes, record.expectedStatusCode, record.expectedResponseBody, record.expectAfter, record.expectedFailedCalled)
			}
		})

		t.Run("POST", func(t *testing.T) {
			for _, record := range testData {
				testHandlerPost(t, record.statusCodes, record.expectedStatusCode, record.expectedResponseBody, record.expectAfter, record.expectedFailedCalled)
			}
		})
	})

	t.Run("Timeout", func(t *testing.T) {
		for _, endpointCount := range []int{1, 2, 3, 5} {
			t.Run(fmt.Sprintf("EndpointCount=%d", endpointCount), func(t *testing.T) {
				testHandlerTimeout(t, endpointCount)
			})
		}
	})
}

func testNewNilEndpoints(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		New(nil)
	})
}

func testNewNilConfiguration(t *testing.T) {
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
			WithFanoutFailure(),
			WithClientFailure(),
		)
	)

	require.NotNil(handler)
	assert.NotNil(handler.shouldTerminate)
	assert.NotNil(handler.errorEncoder)
	assert.NotNil(handler.transactor)
	assert.Empty(handler.before)
	assert.Empty(handler.after)
	assert.Empty(handler.failure)
}

func testNewNoConfiguration(t *testing.T) {
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

func testNewShouldTerminate(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		shouldTerminateCalled = false
		shouldTerminate       = func(Result) bool {
			assert.False(shouldTerminateCalled)
			shouldTerminateCalled = true
			return true
		}

		handler = New(FixedEndpoints{}, WithShouldTerminate(shouldTerminate))
	)

	require.NotNil(handler)
	assert.True(handler.shouldTerminate(Result{}))
	assert.True(shouldTerminateCalled)
}

func testNewWithInjectedConfiguration(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedEndpoints = MustParseURLs("http://foobar.com:8080")

		handler = New(
			expectedEndpoints,
			WithConfiguration(Configuration{
				Endpoints:     []string{"localhost:1234"},
				Authorization: "deadbeef",
			}),
		)
	)

	require.NotNil(handler)
	assert.NotNil(handler.transactor)
	assert.Len(handler.before, 1)
	assert.Equal(expectedEndpoints, handler.endpoints)
}

func TestNew(t *testing.T) {
	t.Run("NilEndpoints", testNewNilEndpoints)
	t.Run("NilConfiguration", testNewNilConfiguration)
	t.Run("NoConfiguration", testNewNoConfiguration)
	t.Run("ShouldTerminate", testNewShouldTerminate)
	t.Run("WithInjectedConfiguration", testNewWithInjectedConfiguration)
}
