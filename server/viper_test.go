package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging/golog"
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

func ExampleNew() {
	webPA, _, err := New("example", nil, nil, viper.New())
	if err != nil {
		panic(err)
	}

	fmt.Println(webPA.Address)
	fmt.Println(webPA.CertificateFile)
	fmt.Println(webPA.KeyFile)
	fmt.Println(webPA.LogConnectionState)
	fmt.Println(webPA.HealthAddress)
	fmt.Println(webPA.HealthLogInterval)
	fmt.Println(webPA.PprofAddress)

	// Output:
	// :9000
	// file.cert
	// file.key
	// true
	// :9001
	// 1m0s
	// :9090
}

func ExampleNewDifferentFile() {
	webPA, _, err := New(
		"application",
		[]string{"-f", "example"},
		pflag.NewFlagSet("does not matter", pflag.ContinueOnError),
		viper.New(),
	)

	if err != nil {
		panic(err)
	}

	fmt.Println(webPA.Address)
	fmt.Println(webPA.CertificateFile)
	fmt.Println(webPA.KeyFile)
	fmt.Println(webPA.LogConnectionState)
	fmt.Println(webPA.HealthAddress)
	fmt.Println(webPA.HealthLogInterval)
	fmt.Println(webPA.PprofAddress)

	// Output:
	// :9000
	// file.cert
	// file.key
	// true
	// :9001
	// 1m0s
	// :9090
}

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
			f                  *pflag.FlagSet
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
			actualConfigName, err = ConfigureViper(applicationName, record.f, v)
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
			f                  *pflag.FlagSet
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
			actualConfigName, err = Configure(applicationName, record.arguments, record.f, v)
		)

		assert.Equal(record.expectsError, err != nil)
		assert.Equal(record.expectedConfigName, actualConfigName)
	}
}

func TestNewWebPA(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		badViper = viper.New()
		testData = []struct {
			v            *viper.Viper
			expectsError bool
		}{
			{nil, false},
			{viper.New(), false},
			{badViper, true},
		}
	)

	badViper.SetDefault("address", map[string]string{})

	for _, record := range testData {
		t.Logf("%#v", record)
		webPA, err := NewWebPA(record.v)
		require.NotNil(webPA)
		assert.Equal(record.expectsError, err != nil)
	}
}

func TestNewLoggerFactory(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		badViper = viper.New()
		testData = []struct {
			v            *viper.Viper
			expectsError bool
		}{
			{nil, false},
			{viper.New(), false},
			{badViper, true},
		}
	)

	badViper.SetDefault("file", map[string]string{})

	for _, record := range testData {
		t.Logf("%#v", record)
		loggerFactory, err := NewLoggerFactory(record.v)
		require.NotNil(loggerFactory)
		assert.Equal(record.expectsError, err != nil)

		gologFactory, ok := loggerFactory.(*golog.LoggerFactory)
		require.True(ok)
		assert.Equal(golog.LoggerFactory{}, *gologFactory)
	}
}

func TestNew(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		f       = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		v       = viper.New()

		webPA, loggerFactory, err = New(applicationName, nil, f, v)
	)

	t.Logf("%#v", err)

	require.NotNil(webPA)
	require.NotNil(loggerFactory)
	require.Nil(err)

	assert.Equal(":10001", webPA.Address)
	gologFactory, ok := loggerFactory.(*golog.LoggerFactory)
	require.True(ok)
	assert.Equal(golog.LoggerFactory{}, *gologFactory)
}

func TestNewWhenConfigureError(t *testing.T) {
	var (
		assert = assert.New(t)
		f      = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		v      = viper.New()

		webPA, loggerFactory, err = New(applicationName, []string{"-o", "huh?"}, f, v)
	)

	assert.Nil(webPA)
	assert.Nil(loggerFactory)
	assert.NotNil(err)
}

func TestNewWhenReadInConfigError(t *testing.T) {
	var (
		assert = assert.New(t)
		f      = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		v      = viper.New()

		webPA, loggerFactory, err = New(applicationName, []string{"-f", "nosuchfile"}, f, v)
	)

	assert.Nil(webPA)
	assert.Nil(loggerFactory)
	assert.NotNil(err)
}

func TestNewWhenWebPAError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		f       = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		v       = viper.New()

		webPA, loggerFactory, err = New(applicationName, []string{"-f", "badaddress"}, f, v)
	)

	require.NotNil(webPA)
	assert.Equal(webPA.Name, applicationName)
	assert.Nil(loggerFactory)
	assert.NotNil(err)
}

func TestNewWhenLoggerFactoryError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		f       = pflag.NewFlagSet(flagSetName, pflag.ContinueOnError)
		v       = viper.New()

		webPA, loggerFactory, err = New(applicationName, []string{"-f", "badlogfile"}, f, v)
	)

	require.NotNil(webPA)
	assert.Equal(webPA.Name, applicationName)
	require.NotNil(loggerFactory)
	gologFactory, ok := loggerFactory.(*golog.LoggerFactory)
	require.True(ok)
	assert.Equal(golog.LoggerFactory{}, *gologFactory)
	assert.NotNil(err)
}
