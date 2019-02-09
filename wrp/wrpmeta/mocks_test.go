package wrpmeta

import "github.com/stretchr/testify/mock"

type mockSource struct {
	mock.Mock
}

func (m *mockSource) ExpectPresent(key, value string) *mock.Call {
	return m.On("GetString", key).Return(value, true)
}

func (m *mockSource) ExpectAbsent(key string) *mock.Call {
	return m.On("GetString", key).Return("", true)
}

func (m *mockSource) GetString(key string) (string, bool) {
	arguments := m.Called(key)
	return arguments.String(0), arguments.Bool(1)
}
