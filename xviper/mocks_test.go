package xviper

import "github.com/stretchr/testify/mock"

type mockConfiger struct {
	mock.Mock
}

func (m *mockConfiger) AddConfigPath(v string) {
	m.Called(v)
}

func (m *mockConfiger) SetConfigName(v string) {
	m.Called(v)
}

func (m *mockConfiger) SetConfigFile(v string) {
	m.Called(v)
}

type mockUnmarshaler struct {
	mock.Mock
}

func (m *mockUnmarshaler) Unmarshal(v interface{}) error {
	return m.Called(v).Error(0)
}
