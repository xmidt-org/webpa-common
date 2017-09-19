package middleware

import (
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/mock"
)

type mockLoggable struct {
	mock.Mock
}

func (m *mockLoggable) Logger() log.Logger {
	return m.Called().Get(0).(log.Logger)
}
