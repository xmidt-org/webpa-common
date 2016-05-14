package logging

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
)

const (
	cannotFormatErrorPattern string = "Cannot format log statement: unrecognized parameter %#v"
)

// ErrorLogger provides the interface for outputting errors to a log sink
type ErrorLogger interface {
	// Error will result in complaints by go vet if used with a format string.
	// Use Errorf to avoid those.
	Error(parameters ...interface{})

	// Errorf is provided to get around go vet problems.
	Errorf(parameters ...interface{})
}

// FatalLogger provides the interface for outputting fatal errors.  Implementations
// are free to panic or exit the current process, so use with caution.
type FatalLogger interface {
	Fatal(parameters ...interface{})
}

// Logger defines the expected methods to be provided by logging infrastructure
type Logger interface {
	ErrorLogger
	FatalLogger
	Debug(parameters ...interface{})
	Info(parameters ...interface{})
	Warn(parameters ...interface{})
}

// ErrorWriter adapts a context.Logger so that all output from Write() goes
// to Error(...).  This is useful for HTTP error logs.
type ErrorWriter struct {
	ErrorLogger
}

func (e *ErrorWriter) Write(data []byte) (int, error) {
	e.Error(string(data))
	return len(data), nil
}

var _ io.Writer = (*ErrorWriter)(nil)

// DefaultLogger embeds an io.Writer and sends all output to that writer.  This type
// is primarily intended for testing.
type DefaultLogger struct {
	io.Writer
}

var _ Logger = DefaultLogger{}

// doWrite mimics the behavior of most logging frameworks, albeit with a simpler implementation.
func (logger DefaultLogger) doWrite(level string, parameters ...interface{}) {
	if _, err := fmt.Fprintf(logger, "[%-5.5s] ", level); err != nil {
		panic(err)
	}

	if len(parameters) > 0 {
		switch head := parameters[0].(type) {
		case fmt.Stringer:
			if _, err := fmt.Fprintf(logger, head.String(), parameters[1:]...); err != nil {
				panic(err)
			}

		case string:
			if _, err := fmt.Fprintf(logger, head, parameters[1:]...); err != nil {
				panic(err)
			}

		default:
			panic(
				errors.New(
					fmt.Sprintf(cannotFormatErrorPattern, parameters[0]),
				),
			)
		}
	}

	if _, err := fmt.Fprintln(logger); err != nil {
		panic(err)
	}
}

func (logger DefaultLogger) Debug(parameters ...interface{}) {
	logger.doWrite("DEBUG", parameters...)
}

func (logger DefaultLogger) Info(parameters ...interface{}) {
	logger.doWrite("INFO", parameters...)
}

func (logger DefaultLogger) Warn(parameters ...interface{}) {
	logger.doWrite("WARN", parameters...)
}

func (logger DefaultLogger) Error(parameters ...interface{}) {
	logger.doWrite("ERROR", parameters...)
}

func (logger DefaultLogger) Errorf(parameters ...interface{}) {
	logger.doWrite("ERROR", parameters...)
}

func (logger DefaultLogger) Fatal(parameters ...interface{}) {
	logger.doWrite("FATAL", parameters...)
}

// NewErrorLog creates a new log.Logger appropriate for http.Server.ErrorLog
func NewErrorLog(logger Logger, serverName string) *log.Logger {
	return log.New(&ErrorWriter{logger}, serverName, log.LstdFlags|log.LUTC)
}

// NewConnectionStateLogger produces a function appropriate for http.Server.ConnState.
// The returned function will log debug statements for each state change.
func NewConnectionStateLogger(logger Logger, serverName string) func(net.Conn, http.ConnState) {
	return func(connection net.Conn, connectionState http.ConnState) {
		logger.Debug(
			"[%s] [%s] -> %s",
			serverName,
			connection.LocalAddr().String(),
			connectionState,
		)
	}
}
