package device

import (
	"bytes"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			HandshakeTimeout: time.Minute + 15*time.Second,
			Logger:           logger,
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
					"deviceNameHeader": {"this": "is not valid"},
					"handshakeTimeout": "this is not a valid duration"
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
