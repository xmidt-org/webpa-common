package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/mock"
)

const (
	authorizationValue = "Basic dGVzdDp0ZXN0Cg=="
	tokenValue         = "dGVzdDp0ZXN0Cg=="
)

func tokenMatcher(token *secure.Token) bool {
	return token.Type() == secure.Basic && token.Value() == tokenValue
}

type mockHttpHandler struct {
	mock.Mock
}

func (h *mockHttpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h.Called(response, request)
}

func ExampleBasicAuthorization() {
	// typical usage: just take the defaults for header and code
	authorizationHandler := AuthorizationHandler{
		Logger:    logging.DefaultLogger(),
		Validator: secure.ExactMatchValidator(tokenValue),
	}

	myHandler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		fmt.Println("Authorized!")
	})

	decorated := authorizationHandler.Decorate(myHandler)

	validRequest, _ := http.NewRequest("GET", "http://example.org/basic/auth", nil)
	validRequest.Header.Set(secure.AuthorizationHeader, authorizationValue)
	validResponse := httptest.NewRecorder()
	decorated.ServeHTTP(validResponse, validRequest)
	fmt.Println(validResponse.Code)

	rejectedRequest, _ := http.NewRequest("GET", "http://example.org/basic/auth/rejected", nil)
	rejectedRequest.Header.Set(secure.AuthorizationHeader, "Basic cmVqZWN0bWU6cmVqZWN0ZWQK")
	rejectedResponse := httptest.NewRecorder()
	decorated.ServeHTTP(rejectedResponse, rejectedRequest)
	fmt.Println(rejectedResponse.Code)

	// Output:
	// Authorized!
	// 200
	// 403
}

func TestAuthorizationHandlerNoDecoration(t *testing.T) {
	assert := assert.New(t)
	mockHttpHandler := &mockHttpHandler{}

	handler := AuthorizationHandler{
		Logger: logging.NewTestLogger(nil, t),
	}

	decorated := handler.Decorate(mockHttpHandler)
	assert.Equal(decorated, mockHttpHandler)

	mockHttpHandler.AssertExpectations(t)
}

func TestAuthorizationHandlerNoAuthorizationHeader(t *testing.T) {
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

func TestAuthorizationHandlerInvalidAuthorizationHeader(t *testing.T) {
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

func TestAuthorizationHandlerSuccess(t *testing.T) {
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

		inputCtx := context.WithValue(inputRequest.Context(), handlerValuesKey, inputCtxValue)

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

func TestAuthorizationHandlerFailure(t *testing.T) {
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

		inputCtx := context.WithValue(request.Context(), handlerValuesKey, inputCtxValue)

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

func TestExtractSatClientID(t *testing.T) {

	t.Run("JWT Type", func(t *testing.T) {
		assert := assert.New(t)
		token, errParse := secure.ParseAuthorization("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ0ZXN0LXN1YnNjcmliZXIiLCIiOiJKb2huIERvZSIsImFkbWluIjp0cnVlfQ.IXIs63ofkXSeZmPMKGs5zxREksHoLMS33LRqw1NMrCA")

		if errParse != nil {
			t.FailNow()
		}

		assert.EqualValues("test-subscriber", extractSatClientID(token, logging.DefaultLogger()))
	})

	t.Run("Non-JWT Type", func(t *testing.T) {
		assert := assert.New(t)
		token, errParse := secure.ParseAuthorization("Basic abcd==")

		if errParse != nil {
			t.FailNow()
		}

		assert.EqualValues("N/A", extractSatClientID(token, logging.DefaultLogger()))
	})

}
