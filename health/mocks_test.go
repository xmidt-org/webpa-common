package health

import (
	"github.com/stretchr/testify/mock"
	"net/http"
)

type mockHandler struct {
	mock.Mock
}

func (m *mockHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	m.Called(response, request)
}

type mockResponseWriter struct {
	mock.Mock
}

func (m *mockResponseWriter) Header() http.Header {
	return m.Called().Get(0).(http.Header)
}

func (m *mockResponseWriter) Write(data []byte) (int, error) {
	arguments := m.Called(data)
	return arguments.Int(0), arguments.Error(1)
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.Called(statusCode)
}
