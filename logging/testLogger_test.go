package logging

import (
	"strings"
	"testing"

	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewTestWriter(t *testing.T) {
	const expected = "expected"

	var (
		assert   = assert.New(t)
		testSink = new(mockTestSink)

		testWriter = NewTestWriter(testSink)
	)

	testSink.On("Log", mock.MatchedBy(func(values []interface{}) bool {
		return values[0].(string) == expected
	})).Once()

	count, err := testWriter.Write([]byte(expected))
	assert.Equal(len(expected), count)
	assert.NoError(err)

	testSink.AssertExpectations(t)
}

func testTestLogger(t *testing.T, o *Options) {
	var (
		testSink        = new(mockTestSink)
		invocationCount = 0
		configuredLevel = strings.ToUpper(o.level())
	)

	if o == nil {
		// for test loggers, we turn on debug by default
		configuredLevel = "DEBUG"
	}

	switch configuredLevel {
	case "DEBUG":
		invocationCount++
		fallthrough

	case "INFO":
		invocationCount++
		fallthrough

	case "WARN":
		invocationCount++
		fallthrough

	default:
		invocationCount++
	}

	testSink.On("Log", mock.MatchedBy(func([]interface{}) bool { return true })).Times(invocationCount)

	testLogger := NewTestLogger(o, testSink)
	testLogger.Log(level.Key(), level.DebugValue(), MessageKey(), "debug message")
	testLogger.Log(level.Key(), level.InfoValue(), MessageKey(), "info message")
	testLogger.Log(level.Key(), level.WarnValue(), MessageKey(), "warn message")
	testLogger.Log(level.Key(), level.ErrorValue(), MessageKey(), "error message")

	testSink.AssertExpectations(t)
}

func TestNewTestLogger(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { testTestLogger(t, nil) })
	t.Run("Default", func(t *testing.T) { testTestLogger(t, new(Options)) })
	t.Run("Error", func(t *testing.T) { testTestLogger(t, &Options{Level: "error"}) })
	t.Run("Warn", func(t *testing.T) { testTestLogger(t, &Options{Level: "warn"}) })
	t.Run("Info", func(t *testing.T) { testTestLogger(t, &Options{Level: "info"}) })
	t.Run("Debug", func(t *testing.T) { testTestLogger(t, &Options{Level: "debug"}) })
}
