package server

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	flagSetName     = "flagSet"
	applicationName = "applicationName"
)

func TestConfigureFlagSet(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		f       = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
	)

	require.Nil(f.Lookup(FileFlagName))
	ConfigureFlagSet(applicationName, f)

	fileFlag := f.Lookup(FileFlagName)
	require.NotNil(fileFlag)

	assert.Equal(FileFlagName, fileFlag.Name)
	assert.Equal(FileFlagShorthand, fileFlag.Shorthand)
	assert.NotEmpty(fileFlag.Usage)

	require.NotNil(fileFlag.Value)
	assert.Equal(applicationName, fileFlag.Value.String())
}

func TestConfigureViper(t *testing.T) {
	var (
		assert          = assert.New(t)
		require         = require.New(t)
		flagSetWithFile = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		testData        = []struct {
			flagSet            *pflag.FlagSet
			expectedConfigName string
		}{
			{nil, applicationName},
			{pflag.NewFlagSet(flagSetName, pflag.ContinueOnError), applicationName},
			{flagSetWithFile, "different"},
		}
	)

	ConfigureFlagSet("different", flagSetWithFile)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			v                     = viper.New()
			actualConfigName, err = ConfigureViper(applicationName, record.flagSet, v)
		)

		require.Nil(err)
		assert.Equal(record.expectedConfigName, actualConfigName)
	}
}

func TestConfigure(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			arguments          []string
			flagSet            *pflag.FlagSet
			expectedConfigName string
			expectsError       bool
		}{
			// no flagset provided
			{nil, nil, applicationName, false},
			{[]string{"-o", "-q"}, nil, applicationName, false},

			// valid flagset provided
			{nil, pflag.NewFlagSet(flagSetName, pflag.ContinueOnError), applicationName, false},
			{[]string{"-o", "-q"}, pflag.NewFlagSet(flagSetName, pflag.ContinueOnError), "", true},
			{[]string{"-f", "somefile"}, pflag.NewFlagSet(flagSetName, pflag.ContinueOnError), "somefile", false},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			v                     = viper.New()
			actualConfigName, err = Configure(applicationName, record.arguments, record.flagSet, v)
		)

		assert.Equal(record.expectsError, err != nil)
		assert.Equal(record.expectedConfigName, actualConfigName)
	}
}

func TestNewWebPA(t *testing.T) {
	var (
		require    = require.New(t)
		v          = viper.New()
		webPA, err = NewWebPA(v)
	)

	require.NotNil(webPA)
	require.Nil(err)
}

func TestNewLoggerFactory(t *testing.T) {
	var (
		require            = require.New(t)
		v                  = viper.New()
		loggerFactory, err = NewLoggerFactory(v)
	)

	require.NotNil(loggerFactory)
	require.Nil(err)
}
