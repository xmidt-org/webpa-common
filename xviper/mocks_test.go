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
