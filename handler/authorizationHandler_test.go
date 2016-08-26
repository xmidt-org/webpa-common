package handler

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/secure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"net/http"
	"testing"
)

type mockValidator struct {
	mock.Mock
}

func (v *mockValidator) Validate(token *secure.Token) (bool, error) {
	arguments := v.Called(token)
	return arguments.Bool(0), arguments.Error(1)
}

type mockHttpHandler struct {
	mock.Mock
}

func (h *mockHttpHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	h.Called(response, request)
}

func TestAuthorizationHandlerNoDecoration(t *testing.T) {
	assert := assert.New(t)
	mockHttpHandler := &mockHttpHandler{}

	handler := AuthorizationHandler{
		Logger: &logging.LoggerWriter{ioutil.Discard},
	}

	decorated := handler.Decorate(mockHttpHandler)
	assert.Equal(decorated, mockHttpHandler)

	mockHttpHandler.AssertExpectations(t)
}
