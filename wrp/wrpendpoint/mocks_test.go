package wrpendpoint

import (
	"context"

	"github.com/stretchr/testify/mock"
)

type mockService struct {
	mock.Mock
}

func (m *mockService) ServeWRP(ctx context.Context, r Request) (Response, error) {
	arguments := m.Called(ctx, r)
	return arguments.Get(0).(Response), arguments.Error(1)
}

type mockReader struct {
	mock.Mock
}

func (m *mockReader) Read(p []byte) (int, error) {
	arguments := m.Called(p)
	return arguments.Int(0), arguments.Error(1)
}
