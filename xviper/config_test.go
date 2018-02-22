package xviper

import (
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddStandardConfigPaths(t *testing.T) {
	configer := new(mockConfiger)
	configer.On("AddConfigPath", "/etc/test").Once()
	configer.On("AddConfigPath", "$HOME/test").Once()
	configer.On("AddConfigPath", ".").Once()

	AddStandardConfigPaths(configer, "test")

	configer.AssertExpectations(t)
}

func testBindConfigNameSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("name", "", "this is the config name")
	require.NoError(flagSet.Parse([]string{"--name", "test"}))

	configer.On("SetConfigName", "test").Once()
	assert.True(BindConfigName(configer, flagSet, "name"))

	configer.AssertExpectations(t)
}

func testBindConfigNameMissing(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("name", "", "this is the config name")
	require.NoError(flagSet.Parse([]string{}))

	assert.False(BindConfigName(configer, flagSet, "name"))
	assert.False(BindConfigName(configer, flagSet, "nosuch"))

	configer.AssertExpectations(t)
}

func TestBindConfigName(t *testing.T) {
	t.Run("Success", testBindConfigNameSuccess)
	t.Run("Missing", testBindConfigNameMissing)
}

func testBindConfigFileSuccess(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("file", "", "this is the config file")
	require.NoError(flagSet.Parse([]string{"--file", "test"}))

	configer.On("SetConfigFile", "test").Once()
	assert.True(BindConfigFile(configer, flagSet, "file"))

	configer.AssertExpectations(t)
}

func testBindConfigFileMissing(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("file", "", "this is the config file")
	require.NoError(flagSet.Parse([]string{}))

	assert.False(BindConfigFile(configer, flagSet, "file"))
	assert.False(BindConfigFile(configer, flagSet, "nosuch"))

	configer.AssertExpectations(t)
}

func TestBindConfigFile(t *testing.T) {
	t.Run("Success", testBindConfigFileSuccess)
	t.Run("Missing", testBindConfigFileMissing)
}

func testBindConfigUsingFile(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("name", "", "this is the config name")
	flagSet.String("file", "", "this is the config file")
	require.NoError(flagSet.Parse([]string{"--file", "test"}))

	configer.On("SetConfigFile", "test").Once()
	assert.True(BindConfig(configer, flagSet, "file", "name"))

	configer.AssertExpectations(t)
}

func testBindConfigUsingName(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("name", "", "this is the config name")
	flagSet.String("file", "", "this is the config file")
	require.NoError(flagSet.Parse([]string{"--name", "test"}))

	configer.On("SetConfigName", "test").Once()
	assert.True(BindConfig(configer, flagSet, "file", "name"))

	configer.AssertExpectations(t)
}

func testBindConfigMissing(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		configer = new(mockConfiger)
		flagSet  = pflag.NewFlagSet("test", pflag.ContinueOnError)
	)

	flagSet.String("name", "", "this is the config name")
	flagSet.String("file", "", "this is the config file")
	require.NoError(flagSet.Parse([]string{}))

	assert.False(BindConfig(configer, flagSet, "file", "name"))
	assert.False(BindConfig(configer, flagSet, "nosuch", "nosuch"))

	configer.AssertExpectations(t)
}

func TestBindConfig(t *testing.T) {
	t.Run("UsingFile", testBindConfigUsingFile)
	t.Run("UsingName", testBindConfigUsingName)
	t.Run("Missing", testBindConfigMissing)
}
