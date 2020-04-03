package logging

import (
	"testing"

	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/mock"
)

func testTestLogger(t *testing.T, o *Options) {
	var testSink = new(mockTestSink)

	testSink.AssertNotCalled(t, "Log", mock.Anything)

	testLogger := NewTestLogger(o, testSink)
	testLogger.Log(level.Key(), level.DebugValue(), MessageKey(), "debug message")
	testLogger.Log(level.Key(), level.InfoValue(), MessageKey(), "info message")
	testLogger.Log(level.Key(), level.WarnValue(), MessageKey(), "warn message")
	testLogger.Log(level.Key(), level.ErrorValue(), MessageKey(), "error message")

	testSink.AssertExpectations(t)
}

func TestNewTestLogger(t *testing.T) {
	t.Run("NilLogsAll", func(t *testing.T) { testTestLogger(t, nil) })
	t.Run("DefaultLogsError", func(t *testing.T) { testTestLogger(t, new(Options)) })
	t.Run("InfoLogsInfoWarnError", func(t *testing.T) { testTestLogger(t, &Options{Level: "info"}) })
}
