package wrpendpoint

import "github.com/stretchr/testify/mock"

type mockService struct {
	mock.Mock
}

func (m *mockService) ServeWRP(r Request) (Response, error) {
	arguments := m.Called(r)
	return arguments.Get(0).(Response), arguments.Error(1)
}
