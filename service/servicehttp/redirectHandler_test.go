package servicehttp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xmidt-org/webpa-common/service"
	"github.com/stretchr/testify/assert"
)

func testRedirectHandlerKeyFuncError(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedError = errors.New("expected")
		keyFunc       = func(*http.Request) ([]byte, error) { return nil, expectedError }
		accessor      = new(service.MockAccessor)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		handler = RedirectHandler{
			KeyFunc:      keyFunc,
			Accessor:     accessor,
			RedirectCode: http.StatusTemporaryRedirect,
		}
	)

	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusBadRequest, response.Code)
	accessor.AssertExpectations(t)
}

func testRedirectHandlerAccessorError(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedKey   = []byte("34589lkdjasd")
		keyFunc       = func(*http.Request) ([]byte, error) { return expectedKey, nil }
		expectedError = errors.New("expected")
		accessor      = new(service.MockAccessor)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		handler = RedirectHandler{
			KeyFunc:      keyFunc,
			Accessor:     accessor,
			RedirectCode: http.StatusTemporaryRedirect,
		}
	)

	accessor.On("Get", expectedKey).Return("", expectedError).Once()
	handler.ServeHTTP(response, request)

	assert.Equal(http.StatusInternalServerError, response.Code)
	accessor.AssertExpectations(t)
}

func testRedirectHandlerSuccess(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedKey      = []byte("asdfqwer")
		expectedInstance = "https://ahost123.com:324"
		keyFunc          = func(*http.Request) ([]byte, error) { return expectedKey, nil }
		accessor         = new(service.MockAccessor)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)

		handler = RedirectHandler{
			KeyFunc:      keyFunc,
			Accessor:     accessor,
			RedirectCode: http.StatusTemporaryRedirect,
		}
	)

	accessor.On("Get", expectedKey).Return(expectedInstance, error(nil)).Once()
	handler.ServeHTTP(response, request)

	assert.Equal(handler.RedirectCode, response.Code)
	assert.Equal(expectedInstance, response.HeaderMap.Get("Location"))
	accessor.AssertExpectations(t)
}

func testRedirectHandlerSuccessWithPath(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedKey         = []byte("asdfqwer")
		expectedInstance    = "https://ahost123.com:324"
		requestURI          = "/this/awesome/path"
		expectedRedirectURL = expectedInstance + requestURI
		keyFunc             = func(*http.Request) ([]byte, error) { return expectedKey, nil }
		accessor            = new(service.MockAccessor)

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "https://someIrrelevantHost.com"+requestURI, nil)

		handler = RedirectHandler{
			KeyFunc:      keyFunc,
			Accessor:     accessor,
			RedirectCode: http.StatusTemporaryRedirect,
		}
	)

	//setting this manually as we assume the net client would provide it
	request.RequestURI = requestURI

	accessor.On("Get", expectedKey).Return(expectedInstance, error(nil)).Once()
	handler.ServeHTTP(response, request)

	assert.Equal(handler.RedirectCode, response.Code)
	assert.Equal(expectedRedirectURL, response.HeaderMap.Get("Location"))
	accessor.AssertExpectations(t)
}

func TestRedirectHandler(t *testing.T) {
	t.Run("KeyFuncError", testRedirectHandlerKeyFuncError)
	t.Run("AccessorError", testRedirectHandlerAccessorError)
	t.Run("Success", testRedirectHandlerSuccess)
	t.Run("SuccessPath", testRedirectHandlerSuccessWithPath)
}
