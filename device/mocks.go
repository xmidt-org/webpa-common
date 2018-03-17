package device

import (
	"net/http"

	"github.com/stretchr/testify/mock"
)

type MockConnector struct {
	mock.Mock
}

var _ Connector = (*MockConnector)(nil)

func (m *MockConnector) Connect(response http.ResponseWriter, request *http.Request, header http.Header) (Interface, error) {
	arguments := m.Called(response, request, header)
	first, _ := arguments.Get(0).(Interface)
	return first, arguments.Error(1)
}

func (m *MockConnector) Disconnect(id ID) bool {
	return m.Called(id).Bool(0)
}

func (m *MockConnector) DisconnectIf(predicate func(ID) bool) int {
	return m.Called(predicate).Int(0)
}

func (m *MockConnector) DisconnectAll() int {
	return m.Called().Int(0)
}
