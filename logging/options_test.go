package logging

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/natefinch/lumberjack.v2"
)

func testOptionsLoggerFactory(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options), &Options{JSON: true}, &Options{JSON: false}} {
		assert.NotNil(o.loggerFactory())
	}
}

func testOptionsOutput(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, &Options{File: StdoutFile}} {
		output := o.output()
		assert.NotNil(output)
		assert.NotPanics(func() {
			_, err := output.Write([]byte("expected output: this shouldn't panic\n"))
			assert.NoError(err)
		})
	}

	var (
		rolling = &Options{
			File:       "foobar.log",
			MaxSize:    689328,
			MaxAge:     9,
			MaxBackups: 454,
		}

		output               = rolling.output()
		lumberjackLogger, ok = output.(*lumberjack.Logger)
	)

	assert.True(ok)
	assert.Equal("foobar.log", lumberjackLogger.Filename)
	assert.Equal(689328, lumberjackLogger.MaxSize)
	assert.Equal(9, lumberjackLogger.MaxAge)
	assert.Equal(454, lumberjackLogger.MaxBackups)
}

func testOptionsLevel(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		assert.Empty(o.level())
	}

	assert.Equal("info", (&Options{Level: "info"}).level())
}

func TestOptions(t *testing.T) {
	t.Run("LoggerFactory", testOptionsLoggerFactory)
	t.Run("Output", testOptionsOutput)
	t.Run("Level", testOptionsLevel)
}
