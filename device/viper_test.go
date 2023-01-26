package device

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/sallust"
)

func TestNewOptions(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		logger        = sallust.Default()
		configuration = `{
			"device": {
				"manager": {
					"handshakeTimeout": "1m15s"
				}
			}
		}`

		v = viper.New()
	)

	v.SetConfigType("json")
	require.Nil(v.ReadConfig(bytes.NewBufferString(configuration)))

	o, err := NewOptions(logger, v.Sub(DeviceManagerKey))
	require.NotNil(o)
	assert.Nil(err)

	assert.Equal(
		Options{
			Logger: logger,
		},
		*o,
	)
}

func TestNewOptionsUnmarshalError(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		logger        = sallust.Default()
		configuration = `{
			"device": {
				"manager": {
					"upgrader": "this is not valid"
				}
			}
		}`

		v = viper.New()
	)

	v.SetConfigType("json")
	require.Nil(v.ReadConfig(bytes.NewBufferString(configuration)))

	o, err := NewOptions(logger, v.Sub(DeviceManagerKey))
	require.NotNil(o)
	assert.NotNil(err)

	assert.Equal(logger, o.Logger)
}

func TestNewOptionsNilViper(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = sallust.Default()
	)

	o, err := NewOptions(logger, nil)
	require.NotNil(o)
	assert.Nil(err)

	assert.Equal(Options{Logger: logger}, *o)
}
