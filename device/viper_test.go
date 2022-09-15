package device

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	// nolint:staticcheck
	"github.com/xmidt-org/webpa-common/v2/logging"
)

func TestNewOptions(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		logger        = logging.DefaultLogger()
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
		logger        = logging.DefaultLogger()
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
		logger  = logging.DefaultLogger()
	)

	o, err := NewOptions(logger, nil)
	require.NotNil(o)
	assert.Nil(err)

	assert.Equal(Options{Logger: logger}, *o)
}
