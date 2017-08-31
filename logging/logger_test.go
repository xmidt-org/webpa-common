package logging

import (
	"strings"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCallerKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(callerKey, CallerKey())
}

func TestMessageKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(messageKey, MessageKey())
}

func TestErrorKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(errorKey, ErrorKey())
}

func TestTimestampKey(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(timestampKey, TimestampKey())
}

func TestDefaultLogger(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(defaultLogger, DefaultLogger())
}

func TestNew(t *testing.T) {
	assert := assert.New(t)

	assert.NotNil(New(nil))
	assert.NotNil(New(new(Options)))
}

func testNewFilter(t *testing.T, o *Options) {
	var (
		assert = assert.New(t)
		next   = new(mockLogger)
	)

	switch strings.ToUpper(o.level()) {
	case "DEBUG":
		next.On("Log", mock.MatchedBy(matchLevel(level.DebugValue()))).
			Run(expectKeys(assert, MessageKey())).
			Return(nil).
			Once()
		fallthrough

	case "INFO":
		next.On("Log", mock.MatchedBy(matchLevel(level.InfoValue()))).
			Run(expectKeys(assert, MessageKey())).
			Return(nil).
			Once()
		fallthrough

	case "WARN":
		next.On("Log", mock.MatchedBy(matchLevel(level.WarnValue()))).
			Run(expectKeys(assert, MessageKey())).
			Return(nil).
			Once()
		fallthrough

	default:
		next.On("Log", mock.MatchedBy(matchLevel(level.ErrorValue()))).
			Run(expectKeys(assert, MessageKey())).
			Return(nil).
			Once()
	}

	filter := NewFilter(next, o)
	filter.Log(level.Key(), level.DebugValue(), MessageKey(), "debug message")
	filter.Log(level.Key(), level.InfoValue(), MessageKey(), "info message")
	filter.Log(level.Key(), level.WarnValue(), MessageKey(), "warn message")
	filter.Log(level.Key(), level.ErrorValue(), MessageKey(), "error message")

	next.AssertExpectations(t)
}

func TestNewFilter(t *testing.T) {
	t.Run("Nil", func(t *testing.T) { testNewFilter(t, nil) })
	t.Run("Default", func(t *testing.T) { testNewFilter(t, new(Options)) })
	t.Run("Error", func(t *testing.T) { testNewFilter(t, &Options{Level: "error"}) })
	t.Run("Warn", func(t *testing.T) { testNewFilter(t, &Options{Level: "warn"}) })
	t.Run("Info", func(t *testing.T) { testNewFilter(t, &Options{Level: "info"}) })
	t.Run("Debug", func(t *testing.T) { testNewFilter(t, &Options{Level: "debug"}) })
}

func testDefaultCallerSimple(t *testing.T) {
	var (
		assert = assert.New(t)
		next   = new(mockLogger)
	)

	next.On("Log", mock.MatchedBy(func([]interface{}) bool { return true })).
		Run(expectKeys(assert, CallerKey(), MessageKey())).
		Return(expectKeys(assert, MessageKey())).
		Once()

	decorated := DefaultCaller(next)
	decorated.Log(MessageKey(), "message")

	next.AssertExpectations(t)
}

func testDefaultCallerKeyvals(t *testing.T) {
	var (
		assert = assert.New(t)
		next   = new(mockLogger)
	)

	next.On("Log", mock.MatchedBy(func([]interface{}) bool { return true })).
		Run(expectKeys(assert, CallerKey(), MessageKey())).
		Return(expectKeys(assert, MessageKey(), "foo")).
		Once()

	decorated := DefaultCaller(next, "foo", "bar")
	decorated.Log(MessageKey(), "message")

	next.AssertExpectations(t)
}

func TestDefaultCaller(t *testing.T) {
	t.Run("Simple", testDefaultCallerSimple)
	t.Run("Keyvals", testDefaultCallerKeyvals)
}

func testLevelledLogger(t *testing.T, factory func(log.Logger, ...interface{}) log.Logger, expected level.Value) {
	var (
		assert = assert.New(t)
		next   = new(mockLogger)
	)

	next.On("Log", mock.MatchedBy(matchLevel(expected))).
		Run(expectKeys(assert, level.Key(), CallerKey(), MessageKey())).
		Return(nil).
		Once()

	decorated := factory(next)
	decorated.Log(level.Key(), expected, MessageKey(), "message")

	next.AssertExpectations(t)
}

func testLevelledLoggerKeyvals(t *testing.T, factory func(log.Logger, ...interface{}) log.Logger, expected level.Value) {
	var (
		assert = assert.New(t)
		next   = new(mockLogger)
	)

	next.On("Log", mock.MatchedBy(matchLevel(expected))).
		Run(expectKeys(assert, level.Key(), "foo", CallerKey(), MessageKey())).
		Return(nil).
		Once()

	decorated := factory(next, "foo", "bar")
	decorated.Log(MessageKey(), "message")

	next.AssertExpectations(t)
}

func TestError(t *testing.T) {
	t.Run("Simple", func(t *testing.T) { testLevelledLogger(t, Error, level.ErrorValue()) })
	t.Run("Keyvals", func(t *testing.T) { testLevelledLoggerKeyvals(t, Error, level.ErrorValue()) })
}

func TestInfo(t *testing.T) {
	t.Run("Simple", func(t *testing.T) { testLevelledLogger(t, Info, level.InfoValue()) })
	t.Run("Keyvals", func(t *testing.T) { testLevelledLoggerKeyvals(t, Info, level.InfoValue()) })
}

func TestWarn(t *testing.T) {
	t.Run("Simple", func(t *testing.T) { testLevelledLogger(t, Warn, level.WarnValue()) })
	t.Run("Keyvals", func(t *testing.T) { testLevelledLoggerKeyvals(t, Warn, level.WarnValue()) })
}

func TestDebug(t *testing.T) {
	t.Run("Simple", func(t *testing.T) { testLevelledLogger(t, Debug, level.DebugValue()) })
	t.Run("Keyvals", func(t *testing.T) { testLevelledLoggerKeyvals(t, Debug, level.DebugValue()) })
}
