package logging

import (
	"io"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// CallerKey is the logging key used to hold the caller for loggers created via this package
	CallerKey = "caller"

	// MessageKey is the logging key for messages
	MessageKey = "msg"

	// TimestampKey is the logging key for timestamps
	TimestampKey = "ts"
)

var (
	defaultLogger = New(nil)
)

// DefaultLogger returns a global singleton default that logs to os.Stdout.
// This returned instance is safe for concurrent access.
func DefaultLogger() log.Logger {
	return defaultLogger
}

// Options stores the configuration of a Logger.  Lumberjack is used for rolling files.
type Options struct {
	// File is the lumberjack Logger file information.  If nil, output is sent to the console.
	File *lumberjack.Logger `json:"file"`

	// JSON is a flag indicating whether JSON logging output is used.  The default is false,
	// meaning that logfmt output is used.
	JSON bool `json:"json"`

	// Level is the error level to output: ERROR, INFO, WARN, or DEBUG.  Any unrecognized string,
	// including the empty string, is equivalent to passing ERROR.
	Level string `json:"level"`
}

func (o *Options) file() *lumberjack.Logger {
	if o != nil {
		return o.File
	}

	return nil
}

func (o *Options) json() bool {
	if o != nil {
		return o.JSON
	}

	return false
}

func (o *Options) output() io.Writer {
	if o != nil && o.File != nil {
		return o.File
	}

	return log.NewSyncWriter(os.Stdout)
}

func (o *Options) loggerFactory() func(io.Writer) log.Logger {
	if o != nil && o.JSON {
		return log.NewJSONLogger
	}

	return log.NewLogfmtLogger
}

func (o *Options) level() string {
	if o != nil {
		return o.Level
	}

	return ""
}

// New creates a go-kit Logger from a set of options.  The options object can be nil,
// in which case a default logger that logs to os.Stdout is returned.  The returned logger
// includes the timestamp in UTC format and will filter according to the Level field.
//
// In order to allow arbitrary decoration, this function does not insert the caller information.
// Use DefaultCaller or Caller in this package to do that.
func New(o *Options) log.Logger {
	logger := log.With(
		// TODO: possibly support terminal output later
		o.loggerFactory()(o.output()),
		TimestampKey, log.DefaultTimestampUTC,
	)

	switch strings.ToUpper(o.level()) {
	case "DEBUG":
		logger = level.NewFilter(logger, level.AllowDebug())

	case "INFO":
		logger = level.NewFilter(logger, level.AllowInfo())

	case "WARN":
		logger = level.NewFilter(logger, level.AllowWarn())

	default:
		logger = level.NewFilter(logger, level.AllowError())
	}

	return logger
}

// DefaultCaller produces a contextual logger as with log.With, but automatically prepends the
// caller under the CallerKey.
//
// The logger returned by this function should not be further decorated.  This will cause the
// callstack to include the decorators, which is pointless.  Instead, decorate the next parameter
// prior to passing it to this function.
func DefaultCaller(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey, log.DefaultCaller}, keyvals...),
	)
}

func Error(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey, log.DefaultCaller, level.Key(), level.ErrorValue()}, keyvals...),
	)
}

func Info(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey, log.DefaultCaller, level.Key(), level.InfoValue()}, keyvals...),
	)
}

func Warn(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey, log.DefaultCaller, level.Key(), level.WarnValue()}, keyvals...),
	)
}

func Debug(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey, log.DefaultCaller, level.Key(), level.DebugValue()}, keyvals...),
	)
}
