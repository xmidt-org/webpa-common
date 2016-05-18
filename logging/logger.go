package logging

import (
	"bytes"
	"fmt"
	"io"
)

// ErrorLogger provides the interface for outputting errors to a log sink
type ErrorLogger interface {
	Error(parameters ...interface{})
}

// Logger defines the expected methods to be provided by logging infrastructure.
// This interface uses different idioms than those used by golang's stdlib Logger.
type Logger interface {
	ErrorLogger

	Trace(parameters ...interface{})
	Debug(parameters ...interface{})
	Info(parameters ...interface{})
	Warn(parameters ...interface{})

	// Printf is a synonym for Info, though Printf only permits a string as
	// its first parameter.  This method is supplied so that this Logger implements
	// common interfaces used by other libraries.
	Printf(format string, parameters ...interface{})
}

const (
	traceLevel string = "[TRACE] "
	debugLevel string = "[DEBUG] "
	infoLevel  string = "[INFO]  "
	warnLevel  string = "[WARN]  "
	errorLevel string = "[ERROR] "
	fatalLevel string = "[FATAL] "
)

// LoggerWriter is a default, built-in logging type that simply writes output
// to an embedded io.Writer.  This is a "poor man's" Logger.  It should normally
// only be used in utilities and tests.
//
// This logger will panic if any io errors occur.
type LoggerWriter struct {
	io.Writer
}

func (l *LoggerWriter) logf(level, format string, parameters []interface{}) {
	var buffer bytes.Buffer
	buffer.WriteString(level)

	if _, err := fmt.Fprintf(&buffer, format, parameters...); err != nil {
		panic(err)
	}

	buffer.WriteRune('\n')
	if _, err := l.Write(buffer.Bytes()); err != nil {
		panic(err)
	}
}

func (l *LoggerWriter) formatf(level string, parameters []interface{}) {
	if len(parameters) > 0 {
		format, ok := parameters[0].(string)
		if !ok {
			if stringer, ok := parameters[0].(fmt.Stringer); ok {
				format = stringer.String()
			} else {
				format = fmt.Sprintf("%v", parameters[0])
			}
		}

		l.logf(level, format, parameters[1:])
	} else {
		l.logf(level, "", parameters)
	}
}

func (l *LoggerWriter) Trace(parameters ...interface{}) { l.formatf(traceLevel, parameters) }
func (l *LoggerWriter) Debug(parameters ...interface{}) { l.formatf(debugLevel, parameters) }
func (l *LoggerWriter) Info(parameters ...interface{})  { l.formatf(infoLevel, parameters) }
func (l *LoggerWriter) Warn(parameters ...interface{})  { l.formatf(warnLevel, parameters) }
func (l *LoggerWriter) Error(parameters ...interface{}) { l.formatf(errorLevel, parameters) }

func (l *LoggerWriter) Printf(format string, parameters ...interface{}) {
	l.logf(infoLevel, format, parameters)
}
