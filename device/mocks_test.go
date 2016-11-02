package device

import (
	"github.com/stretchr/testify/mock"
)

// mockRandom provides an io.Reader mock for a source of random bytes
type mockRandom struct {
	mock.Mock
}

func (m *mockRandom) Read(b []byte) (int, error) {
	arguments := m.Called(b)
	return arguments.Int(0), arguments.Error(1)
}
