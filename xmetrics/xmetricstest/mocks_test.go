package xmetricstest

import "github.com/stretchr/testify/mock"

type mockTestingT struct {
	mock.Mock
}

func (m *mockTestingT) Errorf(msg string, v ...interface{}) {
	m.Called(msg, v)
}

func AnyMessage(_ string) bool {
	return true
}

func AnyArguments(_ []interface{}) bool {
	return true
}
