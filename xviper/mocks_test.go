package xviper

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/mock"
)

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

func (m *mockUnmarshaler) Unmarshal(v interface{}, configOptions ...viper.DecoderConfigOption) error {
	return m.Called(v, configOptions).Error(0)
}

type mockKeyUnmarshaler struct {
	mock.Mock
}

func (m *mockKeyUnmarshaler) UnmarshalKey(k string, v interface{}) error {
	return m.Called(k, v).Error(0)
}
