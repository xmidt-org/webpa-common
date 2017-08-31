package service

import (
	"bytes"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewOptions(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		logger        = logging.NewTestLogger(nil, t)
		pingCalled    = false
		pingFunc      = func() error { pingCalled = true; return nil }
		configuration = bytes.NewBufferString(`{
			"servers": ["host1:1234", "host2:5678"],
			"connection": "foobar"
		}`)

		v = viper.New()
	)

	v.SetConfigType("json")
	require.Nil(v.ReadConfig(configuration))

	o, err := NewOptions(logger, pingFunc, v)
	require.NotNil(o)
	assert.Nil(err)
	assert.Equal(logger, o.Logger)
	assert.Equal("foobar", o.Connection)
	assert.Equal([]string{"host1:1234", "host2:5678"}, o.Servers)

	require.NotNil(o.PingFunc)
	assert.Nil(o.PingFunc())
	assert.True(pingCalled)
}

func TestNewOptionsNoPingFunc(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		logger        = logging.NewTestLogger(nil, t)
		configuration = bytes.NewBufferString(`{
			"servers": ["host1:1234", "host2:5678"],
			"connection": "foobar"
		}`)

		v = viper.New()
	)

	v.SetConfigType("json")
	require.Nil(v.ReadConfig(configuration))

	o, err := NewOptions(logger, nil, v)
	require.NotNil(o)
	assert.Nil(err)
	assert.Nil(o.PingFunc)
	assert.Equal(logger, o.Logger)
	assert.Equal("foobar", o.Connection)
	assert.Equal([]string{"host1:1234", "host2:5678"}, o.Servers)
}
