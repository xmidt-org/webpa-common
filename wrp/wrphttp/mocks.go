package wrphttp

import (
	"github.com/stretchr/testify/mock"
)

type MockHandler struct {
	mock.Mock
}

func (m *MockHandler) ServeWRP(response ResponseWriter, request *Request) {
	m.Called(response, request)
}
