package logging

import (
	"io"
	"os"
	"strings"

	lumberjack "gopkg.in/natefinch/lumberjack.v2"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const (
	StdoutFile = "stdout"
)

var (
	defaultLogger = New(&Options{Level: "DEBUG"})

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

// DefaultLogger returns a global singleton default that logs to os.Stdout.
// This returned instance is safe for concurrent access.
func DefaultLogger() log.Logger {
	return defaultLogger
}

// Options stores the configuration of a Logger.  Lumberjack is used for rolling files.
type Options struct {
	// File is the system file path for the log file.  If set to "stdout", this will log to os.Stdout.
	// Otherwise, a lumberjack.Logger is created
	File string `json:"file"`

	// MaxSize is the lumberjack MaxSize
	MaxSize int `json:"maxsize"`

	// MaxAge is the lumberjack MaxAge
	MaxAge int `json:"maxage"`

	// MaxBackups is the lumberjack MaxBackups
	MaxBackups int `json:"maxbackups"`

	// JSON is a flag indicating whether JSON logging output is used.  The default is false,
	// meaning that logfmt output is used.
	JSON bool `json:"json"`

	// Level is the error level to output: ERROR, INFO, WARN, or DEBUG.  Any unrecognized string,
	// including the empty string, is equivalent to passing ERROR.
	Level string `json:"level"`

	// LoggerFactory overrides the JSON field if specified.  This function is used to produce
	// a go-kit Logger from an io.Writer.
	LoggerFactory func(io.Writer) log.Logger
}

func (o *Options) json() bool {
	if o != nil {
		return o.JSON
	}

	return false
}

func (o *Options) output() io.Writer {
	if o != nil && o.File != StdoutFile {
		return &lumberjack.Logger{
			Filename:   o.File,
			MaxSize:    o.MaxSize,
			MaxAge:     o.MaxAge,
			MaxBackups: o.MaxBackups,
		}
	}

	return log.NewSyncWriter(os.Stdout)
}

func (o *Options) loggerFactory() func(io.Writer) log.Logger {
	if o != nil {
		if o.LoggerFactory != nil {
			return o.LoggerFactory
		} else if o.JSON {
			return log.NewJSONLogger
		}
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
	return NewFilter(
		log.With(
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

func Error(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.ErrorValue()}, keyvals...)...,
	)
}

func Info(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.InfoValue()}, keyvals...)...,
	)
}

func Warn(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.WarnValue()}, keyvals...)...,
	)
}

func Debug(next log.Logger, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		next,
		append([]interface{}{CallerKey(), log.DefaultCaller, level.Key(), level.DebugValue()}, keyvals...)...,
	)
}
