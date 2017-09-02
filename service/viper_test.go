package service

import (
	"strings"
	"testing"
	"time"

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
		{"service": {
			"path": "/foo/bar"
		}}
	`)))

	child := Sub(v)
	require.NotNil(child)
	assert.Equal("/foo/bar", child.GetString("path"))
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
			{"connectTimeout": "this is not a valid timeout"}
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
				"connection": "host1:2181,host2:2181",
				"connectTimeout": "12m",
				"sessionTimeout": "1h0m",
				"updateDelay": "5m",
				"path": "/foo/bar",
				"serviceName": "fantastical",
				"registration": "https://foobar.com:8080",
				"vnodeCount": 567829
			}
		`

		v = viper.New()
	)

	v.SetConfigType("json")
	require.NoError(v.ReadConfig(strings.NewReader(configuration)))

	o, err := FromViper(v)
	require.NotNil(o)
	require.Nil(err)

	assert.Equal("host1:2181,host2:2181", o.Connection)
	assert.Equal(12*time.Minute, o.ConnectTimeout)
	assert.Equal(1*time.Hour, o.SessionTimeout)
	assert.Equal(5*time.Minute, o.UpdateDelay)
	assert.Equal("/foo/bar", o.Path)
	assert.Equal("fantastical", o.ServiceName)
	assert.Equal("https://foobar.com:8080", o.Registration)
	assert.Equal(uint(567829), o.VnodeCount)
}

func TestFromViper(t *testing.T) {
	t.Run("Nil", testFromViperNil)
	t.Run("Missing", testFromViperMissing)
	t.Run("Error", testFromViperError)
	t.Run("Unmarshal", testFromViperUnmarshal)
}
