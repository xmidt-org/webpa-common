package middleware

import (
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type mockLoggable struct {
	mock.Mock
}

func (m *mockLoggable) Logger() *zap.Logger {
	return m.Called().Get(0).(*zap.Logger)
}
