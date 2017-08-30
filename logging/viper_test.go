package logging

import (
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSub(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		v       = viper.New()
	)

	assert.Nil(Sub(nil))
	assert.Nil(Sub(v))

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(strings.NewReader(`
		{"log": {
			"file": "foobar.log"
		}}
	`)))

	child := Sub(v)
	require.NotNil(child)
	assert.Equal("foobar.log", child.GetString("file"))
}

func testFromViperNil(t *testing.T) {
	var (
		assert = assert.New(t)
		o, err = FromViper(nil)
	)

	assert.NotNil(o)
	assert.NoError(err)

}

func testFromViperMissing(t *testing.T) {
	var (
		assert = assert.New(t)
		o, err = FromViper(viper.New())
	)

	assert.NotNil(o)
	assert.NoError(err)

}

func testFromViperError(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		badConfiguration = `
			{"maxage": "this is not a valid integer"}
		`

		v = viper.New()
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(strings.NewReader(badConfiguration)))

	o, err := FromViper(v)
	assert.Nil(o)
	assert.Error(err)
}

func testFromViperUnmarshal(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		configuration = `
			{
				"file": "foobar.log",
				"maxsize": 459234098,
				"maxage": 52,
				"maxbackups": 452,
				"json": true,
				"level": "info"
			}
		`

		v = viper.New()
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(strings.NewReader(configuration)))

	o, err := FromViper(v)
	require.NotNil(o)
	require.Nil(err)

	assert.Equal("foobar.log", o.File)
	assert.Equal(459234098, o.MaxSize)
	assert.Equal(52, o.MaxAge)
	assert.Equal(452, o.MaxBackups)
	assert.True(o.JSON)
	assert.Equal("info", o.Level)
}

func TestFromViper(t *testing.T) {
	t.Run("Nil", testFromViperNil)
	t.Run("Missing", testFromViperMissing)
	t.Run("Error", testFromViperError)
	t.Run("Unmarshal", testFromViperUnmarshal)
}
