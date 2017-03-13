package wrp

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestNewPoolFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	t.Run("NilViper", func(t *testing.T) {
		factory, err := NewPoolFactory(nil)
		require.NotNil(factory)
		require.NoError(err)

		for _, format := range []Format{JSON, Msgpack} {
			t.Run(format.String(), func(t *testing.T) {
				var output []byte
				testEncoderPool(assert, factory.NewEncoderPool(format), &output)
				testDecoderPool(assert, format, factory.NewDecoderPool(format))
			})
		}
	})

	t.Run("WithViper", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("json")
		require.NoError(v.ReadConfig(strings.NewReader(`{
			"wrp": {
				"decoderPoolSize": 131,
				"encoderPoolSize": 67
			}
		}`)))

		factory, err := NewPoolFactory(v.Sub(ViperKey))
		require.NotNil(factory)
		require.NoError(err)

		for _, format := range []Format{JSON, Msgpack} {
			t.Run(format.String(), func(t *testing.T) {
				var output []byte
				testEncoderPool(assert, factory.NewEncoderPool(format), &output)
				testDecoderPool(assert, format, factory.NewDecoderPool(format))
			})
		}
	})

	t.Run("BadConfiguration", func(t *testing.T) {
		v := viper.New()
		v.SetConfigType("json")
		require.NoError(v.ReadConfig(strings.NewReader(`{
			"wrp": {
				"decoderPoolSize": "this is not an integer"
			}
		}`)))

		factory, err := NewPoolFactory(v.Sub(ViperKey))
		assert.NotNil(factory)
		assert.Error(err)
	})
}
