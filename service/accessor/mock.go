package accessor

import "github.com/stretchr/testify/mock"

// MockAccessor is a mocked Accessor
type MockAccessor struct {
	mock.Mock
}

var _ Accessor = (*MockAccessor)(nil)

func (m *MockAccessor) Get(v []byte) (string, error) {
	arguments := m.Called(v)
	return arguments.String(0), arguments.Error(1)
}
