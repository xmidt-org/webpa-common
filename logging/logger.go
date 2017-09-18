package logging

import (
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

var (
	defaultLogger = log.NewNopLogger()

	callerKey    interface{} = "caller"
	messageKey   interface{} = "msg"
	errorKey     interface{} = "error"
	timestampKey interface{} = "ts"
)

// CallerKey returns the logging key to be used for the stack location of the logging call
func CallerKey() interface{} {
	return callerKey
}

// MessageKey returns the logging key to be used for the textual message of the log entry
func MessageKey() interface{} {
	return messageKey
}

// ErrorKey returns the logging key to be used for error instances
func ErrorKey() interface{} {
	return errorKey
}

// TimestampKey returns the logging key to be used for the timestamp
func TimestampKey() interface{} {
	return timestampKey
}

// DefaultLogger returns a global singleton NOP logger.
// This returned instance is safe for concurrent access.
func DefaultLogger() log.Logger {
	return defaultLogger
}

// New creates a go-kit Logger from a set of options.  The options object can be nil,
// in which case a default logger that logs to os.Stdout is returned.  The returned logger
// includes the timestamp in UTC format and will filter according to the Level field.
//
// In order to allow arbitrary decoration, this function does not insert the caller information.
// Use either DefaultCaller in this package or the go-kit/kit/log API to add a Caller to the
// returned Logger.
func New(o *Options) log.Logger {
	return NewFilter(
		log.WithPrefix(
			o.loggerFactory()(o.output()),
			TimestampKey(), log.DefaultTimestampUTC,
		),
		o,
	)
}

// NewFilter applies the Options filtering rules in the package to an arbitrary go-kit Logger.
func NewFilter(next log.Logger, o *Options) log.Logger {
	switch strings.ToUpper(o.level()) {
	case "DEBUG":
		return level.NewFilter(next, level.AllowDebug())

	case "INFO":
		return level.NewFilter(next, level.AllowInfo())

	case "WARN":
		return level.NewFilter(next, level.AllowWarn())

	default:
		return level.NewFilter(next, level.AllowError())
	}
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
		append([]interface{}{CallerKey(), log.DefaultCaller}, keyvals...)...,
	)
}

// Error places both the caller and a constant error level into the prefix of the returned logger.
// Additional key value pairs may also be added.
func Error(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.ErrorValue()}, keyvals...)...,
	)
}

// Info places both the caller and a constant info level into the prefix of the returned logger.
// Additional key value pairs may also be added.
func Info(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.InfoValue()}, keyvals...)...,
	)
}

// Warn places both the caller and a constant warn level into the prefix of the returned logger.
// Additional key value pairs may also be added.
func Warn(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.WarnValue()}, keyvals...)...,
	)
}

// Debug places both the caller and a constant debug level into the prefix of the returned logger.
// Additional key value pairs may also be added.
func Debug(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.DebugValue()}, keyvals...)...,
	)
}
