package logging

import (
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockLogger struct {
	mock.Mock
}

func (m *mockLogger) Log(keyvals ...interface{}) error {
	arguments := m.Called(keyvals)
	first, _ := arguments.Get(0).(error)
	return first
}

// matchLevel returns a mock.MatchedBy function which matches Log methods on an expected log level
func matchLevel(expected level.Value) interface{} {
	return func(keyvals []interface{}) bool {
		for i := 0; i < len(keyvals); i += 2 {
			if keyvals[i] == level.Key() {
				return keyvals[i+1] == expected
			}
		}

		return false
	}
}

// expectKeys produces a mock.Run function which verifies that the given logging keys
// are present in the set of arguments.  Values are not checked by the returned function.
func expectKeys(assert *assert.Assertions, keys ...interface{}) func(mock.Arguments) {
	return func(arguments mock.Arguments) {
		expected := make(map[interface{}]bool)
		for _, k := range keys {
			expected[k] = true
		}

		keyvals := arguments.Get(0).([]interface{})
		for i := 0; i < len(keyvals); i += 2 {
			delete(expected, keyvals[i])
		}

		assert.Empty(expected, "Missing keys: %v", expected)
	}
}
