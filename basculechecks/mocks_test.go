package basculechecks

import "github.com/stretchr/testify/mock"

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Log(keyvals ...interface{}) error {
	arguments := m.Called(keyvals)
	first, _ := arguments.Get(0).(error)
	return first
}
