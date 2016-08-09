package httppool

import (
	"github.com/stretchr/testify/mock"
	"net/http"
)

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
