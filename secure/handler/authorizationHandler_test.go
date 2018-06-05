package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

func testAuthorizationHandlerNoAuthorization(t *testing.T, expectedStatusCode, configuredStatusCode int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		nextCalled = false
		next       = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})

		validator = new(secure.MockValidator)
		handler   = AuthorizationHandler{
			Logger:              logging.NewTestLogger(nil, t),
			ForbiddenStatusCode: configuredStatusCode,
			Validator:           validator,
		}

		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = handler.Decorate(next)
	)

	require.NotNil(decorated)
	decorated.ServeHTTP(response, request)
	assert.Equal(expectedStatusCode, response.Code)
	assert.False(nextCalled)
	validator.AssertExpectations(t)
}

func testAuthorizationHandlerMalformedAuthorization(t *testing.T, expectedStatusCode, configuredStatusCode int, expectedHeader, configuredHeader string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		nextCalled = false
		next       = http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			nextCalled = true
		})

		validator = new(secure.MockValidator)
		handler   = AuthorizationHandler{
			Logger:              logging.NewTestLogger(nil, t),
			HeaderName:          configuredHeader,
			ForbiddenStatusCode: configuredStatusCode,
			Validator:           validator,
		}

		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = handler.Decorate(next)
	)

	require.NotNil(decorated)
	request.Header.Set(expectedHeader, "there is no way this is a valid authorization header")
	decorated.ServeHTTP(response, request)
	assert.Equal(expectedStatusCode, response.Code)
	assert.False(nextCalled)
	validator.AssertExpectations(t)
}

func testAuthorizationHandlerValid(t *testing.T, expectedHeader, configuredHeader string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		nextCalled = false
		next       = http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			nextCalled = true
			values, ok := FromContext(request.Context())
			require.True(ok)
			require.NotNil(values)

			assert.Equal("x1:webpa-internal:5f0183", values.SatClientID)
			assert.Equal([]string{"comcast"}, values.PartnerIDs)
		})

		validator = new(secure.MockValidator)
		handler   = AuthorizationHandler{
			Logger:     logging.NewTestLogger(nil, t),
			HeaderName: configuredHeader,
			Validator:  validator,
		}

		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = handler.Decorate(next)
	)

	require.NotNil(decorated)
	request.Header.Set(expectedHeader, "Bearer eyJhbGciOiJub25lIiwia2lkIjoidGhlbWlzLTIwMTcwMSIsInR5cCI6IkpXVCJ9.eyJqdGkiOiI4ZjA0MmIyOS03ZDE2LTRjMWYtYjBmOS1mNTJhMGFhZDI5YmMiLCJpc3MiOiJzYXRzLXByb2R1Y3Rpb24iLCJzdWIiOiJ4MTp3ZWJwYS1pbnRlcm5hbDo1ZjAxODMiLCJpYXQiOjE1Mjc3MzAwOTYsIm5iZiI6MTUyNzczMDA5NiwiZXhwIjoxNTI3NzczMjk5LCJ2ZXJzaW9uIjoiMS4wIiwiYWxsb3dlZFJlc291cmNlcyI6eyJhbGxvd2VkUGFydG5lcnMiOlsiY29tY2FzdCJdfSwiY2FwYWJpbGl0aWVzIjpbIngxOndlYnBhOmFwaTouKjphbGwiXSwiYXVkIjpbXX0.")

	validator.On("Validate", mock.MatchedBy(func(context.Context) bool { return true }), mock.MatchedBy(func(*secure.Token) bool { return true })).Return(true, error(nil)).Once()
	decorated.ServeHTTP(response, request)
	assert.Equal(200, response.Code)
	assert.True(nextCalled)
	validator.AssertExpectations(t)
}

func testAuthorizationHandlerInvalid(t *testing.T, expectedStatusCode, configuredStatusCode int, expectedHeader, configuredHeader string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		nextCalled = false
		next       = http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			nextCalled = true
		})

		validator = new(secure.MockValidator)
		handler   = AuthorizationHandler{
			Logger:              logging.NewTestLogger(nil, t),
			HeaderName:          configuredHeader,
			ForbiddenStatusCode: configuredStatusCode,
			Validator:           validator,
		}

		response  = httptest.NewRecorder()
		request   = httptest.NewRequest("GET", "/", nil)
		decorated = handler.Decorate(next)
	)

	require.NotNil(decorated)
	request.Header.Set(expectedHeader, "Basic YWxsYWRpbjpvcGVuc2VzYW1l")

	validator.On("Validate", mock.MatchedBy(func(context.Context) bool { return true }), mock.MatchedBy(func(*secure.Token) bool { return true })).Return(false, error(nil)).Once()
	decorated.ServeHTTP(response, request)
	assert.Equal(expectedStatusCode, response.Code)
	assert.False(nextCalled)
	validator.AssertExpectations(t)
}

func TestAuthorizationHandler(t *testing.T) {
	t.Run("NoDecoration", testAuthorizationHandlerNoDecoration)

	t.Run("NoAuthorization", func(t *testing.T) {
		testData := []struct {
			expectedStatusCode   int
			configuredStatusCode int
		}{
			{http.StatusForbidden, 0},
			{http.StatusForbidden, http.StatusForbidden},
			{599, 599},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testAuthorizationHandlerNoAuthorization(t, record.expectedStatusCode, record.configuredStatusCode)
			})
		}
	})

	t.Run("MalformedAuthorization", func(t *testing.T) {
		testData := []struct {
			expectedStatusCode   int
			configuredStatusCode int
			expectedHeader       string
			configuredHeader     string
		}{
			{http.StatusForbidden, 0, "Authorization", ""},
			{http.StatusForbidden, http.StatusForbidden, "Authorization", "Authorization"},
			{599, 599, "X-Custom", "X-Custom"},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testAuthorizationHandlerMalformedAuthorization(t, record.expectedStatusCode, record.configuredStatusCode, record.expectedHeader, record.configuredHeader)
			})
		}
	})

	t.Run("Valid", func(t *testing.T) {
		testData := []struct {
			expectedHeader   string
			configuredHeader string
		}{
			{"Authorization", ""},
			{"Authorization", "Authorization"},
			{"X-Custom", "X-Custom"},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testAuthorizationHandlerValid(t, record.expectedHeader, record.configuredHeader)
			})
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		testData := []struct {
			expectedStatusCode   int
			configuredStatusCode int
			expectedHeader       string
			configuredHeader     string
		}{
			{http.StatusForbidden, 0, "Authorization", ""},
			{http.StatusForbidden, http.StatusForbidden, "Authorization", "Authorization"},
			{599, 599, "X-Custom", "X-Custom"},
		}

		for i, record := range testData {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				testAuthorizationHandlerInvalid(t, record.expectedStatusCode, record.configuredStatusCode, record.expectedHeader, record.configuredHeader)
			})
		}
	})
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
