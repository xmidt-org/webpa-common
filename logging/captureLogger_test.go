package logging

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureLogger(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedMessage = "a message"
		expectedError   = errors.New("an error")

		logger = NewCaptureLogger()
	)

	require.NotNil(logger)
	output := logger.Output()
	require.NotNil(output)

	assert.Panics(func() {
		logger.Log("oops")
	})

	logger.Log(level.Key(), level.ErrorValue(), MessageKey(), expectedMessage, ErrorKey(), expectedError, "count", 12, "name", "foobar")
	m := <-output
	require.NotNil(m)

	assert.Len(m, 5)
	assert.Equal(level.ErrorValue(), m[level.Key()])
	assert.Equal(expectedMessage, m[MessageKey()])
	assert.Equal(expectedError, m[ErrorKey()])
	assert.Equal(12, m["count"])
	assert.Equal("foobar", m["name"])
}
