package mocklogging

import (
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
)

// stubErrorReporter redirects the error report to log messages, so that we can see
// the expected errors with verbose turned on but not fail tests spuriously.
type stubErrorReporter struct {
	*testing.T
}

func (s stubErrorReporter) Error(args ...interface{}) {
	s.Log(args...)
}

func (s stubErrorReporter) Errorf(format string, args ...interface{}) {
	s.Logf(format, args...)
}

func TestL(t *testing.T) {
	logger := New()
	OnLog(stubErrorReporter{t}, logger, level.Key(), level.InfoValue()).Return(error(nil)).Once()

	logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "message")
	logger.AssertExpectations(t)
}

func testMOddMatches(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		M(stubErrorReporter{t}, "odd")
	})

	assert.Panics(func() {
		M(stubErrorReporter{t}, "odd", "number", "")
	})
}

func testMNoMatches(t *testing.T) {
	var (
		assert  = assert.New(t)
		matcher = M(stubErrorReporter{t}) // no matches, which means match any even number of key/value pairs
	)

	assert.True(matcher([]interface{}{}))
	assert.True(matcher([]interface{}{"a", "b", "c", "d"}))
	assert.False(matcher([]interface{}{"odd", "number", ""}))
}

func testMShouldMatch(t *testing.T) {
	var (
		assert = assert.New(t)

		value2Called = false
		value3Called = false

		matcher = M(
			stubErrorReporter{t},
			"key1", "value1",
			"key2", func(value interface{}) bool {
				value2Called = true
				assert.Equal("value2", value)
				return true
			},
			"key3", func(key, value interface{}) bool {
				value3Called = true
				assert.Equal("key3", key)
				assert.Equal("value3", value)
				return true
			},
		)
	)

	assert.True(matcher([]interface{}{"key1", "value1", "key2", "value2", "key3", "value3"}))
	assert.True(value2Called)
	assert.True(value3Called)

	value2Called = false
	value3Called = false
	assert.True(matcher([]interface{}{"key1", "value1", "key2", "value2", "key3", "value3", "another key", "some value"}))
	assert.True(value2Called)
	assert.True(value3Called)
}

func testMShouldNotMatch(t *testing.T) {
	var (
		assert = assert.New(t)

		value2Called = false
		value2       = func(value interface{}) bool {
			value2Called = true
			assert.Equal("value2", value)
			return false
		}

		value3Called = false
		value3       = func(key, value interface{}) bool {
			value3Called = true
			assert.Equal("key3", key)
			assert.Equal("value3", value)
			return false
		}
	)

	assert.False(M(stubErrorReporter{t}, "key1", "value1")([]interface{}{}))
	assert.False(M(stubErrorReporter{t}, "key1", "value1")([]interface{}{"key1", "invalid"}))
	assert.False(M(stubErrorReporter{t}, "key1", "value1")([]interface{}{"another key", "another value"}))

	assert.False(M(stubErrorReporter{t}, "key1", "value1", "key2", value2)([]interface{}{"key1", "value1", "key2", "value2"}))
	assert.True(value2Called)

	value2Called = false
	assert.False(M(stubErrorReporter{t}, "key1", "value1", "key3", value3)([]interface{}{"key1", "value1", "key3", "value3"}))
}

func TestM(t *testing.T) {
	t.Run("OddMatches", testMOddMatches)
	t.Run("NoMatches", testMNoMatches)
	t.Run("ShouldMatch", testMShouldMatch)
	t.Run("ShouldNotMatch", testMShouldNotMatch)
}

func TestAnyValue(t *testing.T) {
	assert := assert.New(t)

	assert.True(AnyValue()(nil))
	assert.True(AnyValue()(123451234))
	assert.True(AnyValue()("asdfasdf"))
}
