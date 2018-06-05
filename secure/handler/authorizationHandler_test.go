package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	authorizationValue = "Basic dGVzdDp0ZXN0Cg=="
	tokenValue         = "dGVzdDp0ZXN0Cg=="
)

func testAuthorizationHandlerNoDecoration(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		nextCalled = false
		next       = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})

		handler = AuthorizationHandler{
			Logger: logging.NewTestLogger(nil, t),
		}

		decorated = handler.Decorate(next)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	assert.True(nextCalled)
}

/*
func testAuthorizationHandlerNoAuthorizationHeader(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)

		testData = []struct {
			handler            AuthorizationHandler
			expectedStatusCode int
		}{
			{
				handler: AuthorizationHandler{
					Validator: &secure.MockValidator{},
				},
				expectedStatusCode: http.StatusForbidden,
			},
			{
				handler: AuthorizationHandler{
					Validator:           &secure.MockValidator{},
					HeaderName:          "X-Custom-Authorization",
					ForbiddenStatusCode: 512,
					Logger:              logger,
				},
				expectedStatusCode: 512,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		mockHttpHandler := &mockHttpHandler{}
		decorated := record.handler.Decorate(mockHttpHandler)
		assert.NotNil(decorated)

		request, _ := http.NewRequest("GET", "http://test.com/foo", nil)
		response := httptest.NewRecorder()
		decorated.ServeHTTP(response, request)
		assert.Equal(response.Code, record.expectedStatusCode)
		assert.Contains(response.HeaderMap.Get("Content-Type"), "application/json")

		body := response.Body.Bytes()
		t.Logf("response body: %s", body)
		message := make(map[string]interface{})
		jsonError := json.Unmarshal(body, &message)
		assert.NotEmpty(message)
		assert.Nil(jsonError)

		record.handler.Validator.(*secure.MockValidator).AssertExpectations(t)
		mockHttpHandler.AssertExpectations(t)
	}
}

func testAuthorizationHandlerInvalidAuthorizationHeader(t *testing.T) {
	assert := assert.New(t)
	logger := logging.NewTestLogger(nil, t)

	var testData = []struct {
		handler            AuthorizationHandler
		headerName         string
		expectedStatusCode int
	}{
		{
			handler: AuthorizationHandler{
				Validator: &secure.MockValidator{},
			},
			headerName:         secure.AuthorizationHeader,
			expectedStatusCode: http.StatusForbidden,
		},
		{
			handler: AuthorizationHandler{
				Validator:           &secure.MockValidator{},
				HeaderName:          "X-Custom-Authorization",
				ForbiddenStatusCode: 512,
				Logger:              logger,
			},
			headerName:         "X-Custom-Authorization",
			expectedStatusCode: 512,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		mockHttpHandler := &mockHttpHandler{}
		decorated := record.handler.Decorate(mockHttpHandler)
		assert.NotNil(decorated)

		request, _ := http.NewRequest("GET", "http://test.com/foo", nil)
		request.Header.Set(record.headerName, "BadToken 123")
		response := httptest.NewRecorder()
		decorated.ServeHTTP(response, request)
		assert.Equal(response.Code, record.expectedStatusCode)
		assert.Contains(response.HeaderMap.Get("Content-Type"), "application/json")

		body := response.Body.Bytes()
		t.Logf("response body: %s", body)
		message := make(map[string]interface{})
		jsonError := json.Unmarshal(body, &message)
		assert.NotEmpty(message)
		assert.Nil(jsonError)

		record.handler.Validator.(*secure.MockValidator).AssertExpectations(t)
		mockHttpHandler.AssertExpectations(t)
	}
}

func testAuthorizationHandlerSuccess(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)

		testData = []struct {
			handler            AuthorizationHandler
			headerName         string
			expectedStatusCode int
		}{
			{
				handler: AuthorizationHandler{
					Validator: &secure.MockValidator{},
				},
				headerName:         secure.AuthorizationHeader,
				expectedStatusCode: 222,
			},
			{
				handler: AuthorizationHandler{
					Validator:           &secure.MockValidator{},
					HeaderName:          "X-Custom-Authorization",
					ForbiddenStatusCode: 512,
					Logger:              logger,
				},
				headerName:         "X-Custom-Authorization",
				expectedStatusCode: 222,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		mockValidator := record.handler.Validator.(*secure.MockValidator)

		request, _ := http.NewRequest("GET", "http://test.com/foo", nil)
		request.Header.Set(record.headerName, authorizationValue)
		response := httptest.NewRecorder()

		inputCtxValue := &ContextValues{
			SatClientID: "N/A",
			Path:        request.URL.Path,
			Method:      request.Method,
		}

		inputRequest := request.WithContext(request.Context())

		inputCtx := context.WithValue(inputRequest.Context(), contextKey{}, inputCtxValue)

		token, _ := secure.ParseAuthorization(authorizationValue)

		mockValidator.On("Validate", inputCtx, token).Return(true, nil).Once()

		request = request.WithContext(inputCtx) //request has this context after Decorate() is called

		mockHttpHandler := &mockHttpHandler{}
		mockHttpHandler.On("ServeHTTP", response, request).
			Run(func(arguments mock.Arguments) {
				response := arguments.Get(0).(http.ResponseWriter)
				response.WriteHeader(record.expectedStatusCode)
			}).
			Once()

		decorated := record.handler.Decorate(mockHttpHandler)
		assert.NotNil(decorated)
		decorated.ServeHTTP(response, inputRequest)
		assert.Equal(response.Code, record.expectedStatusCode)

		mockValidator.AssertExpectations(t)
		mockHttpHandler.AssertExpectations(t)
	}
}

func testAuthorizationHandlerFailure(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)

		testData = []struct {
			handler            AuthorizationHandler
			headerName         string
			expectedStatusCode int
		}{
			{
				handler: AuthorizationHandler{
					Validator: &secure.MockValidator{},
				},
				headerName:         secure.AuthorizationHeader,
				expectedStatusCode: http.StatusForbidden,
			},
			{
				handler: AuthorizationHandler{
					Validator:           &secure.MockValidator{},
					HeaderName:          "X-Custom-Authorization",
					ForbiddenStatusCode: 512,
					Logger:              logger,
				},
				headerName:         "X-Custom-Authorization",
				expectedStatusCode: 512,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)
		mockValidator := record.handler.Validator.(*secure.MockValidator)

		request, _ := http.NewRequest("GET", "http://test.com/foo", nil)
		request.Header.Set(record.headerName, authorizationValue)

		inputCtxValue := &ContextValues{
			SatClientID: "N/A",
			Path:        request.URL.Path,
			Method:      request.Method,
		}

		inputCtx := context.WithValue(request.Context(), contextKey{}, inputCtxValue)

		token, _ := secure.ParseAuthorization(authorizationValue)

		mockValidator.On("Validate", inputCtx, token).Return(false, errors.New("expected")).Once()

		response := httptest.NewRecorder()
		mockHttpHandler := &mockHttpHandler{}

		decorated := record.handler.Decorate(mockHttpHandler)
		assert.NotNil(decorated)
		decorated.ServeHTTP(response, request)
		assert.Equal(response.Code, record.expectedStatusCode)

		mockValidator.AssertExpectations(t)
		mockHttpHandler.AssertExpectations(t)
	}
}
*/
func TestAuthorizationHandler(t *testing.T) {
	t.Run("NoDecoration", testAuthorizationHandlerNoDecoration)
}

func testPopulateContextValuesNoJWT(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		token, err = secure.ParseAuthorization("Basic abcd==")
	)

	require.NoError(err)
	require.NotNil(token)

	values := new(ContextValues)
	assert.NoError(populateContextValues(token, values))
}

func TestPopulateContextValues(t *testing.T) {
	t.Run("NoJWT", testPopulateContextValuesNoJWT)
}

//A simple verification that a pointer function signature is used
func TestDefineMeasures(t *testing.T) {
	assert := assert.New(t)
	a, m := AuthorizationHandler{}, &secure.JWTValidationMeasures{}
	a.DefineMeasures(m)
	assert.Equal(m, a.measures)
}
