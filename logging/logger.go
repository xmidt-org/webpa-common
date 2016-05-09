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

// Logger defines the expected methods to be provided by logging infrastructure
type Logger interface {
	Debug(parameters ...interface{})
	Info(parameters ...interface{})
	Warn(parameters ...interface{})
	Error(parameters ...interface{})
}

// ErrorWriter adapts a context.Logger so that all output from Write() goes
// to Error(...).  This is useful for HTTP error logs.
type ErrorWriter struct {
	Logger
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

// doWrite mimics the behavior of most logging frameworks, albeit with a simpler implementation.
func (logger DefaultLogger) doWrite(level string, parameters ...interface{}) {
	_, err := logger.Write(
		[]byte(fmt.Sprintf("[%-5.5s] ", level)),
	)

	if err != nil {
		panic(err)
	}

	if len(parameters) > 0 {
		switch head := parameters[0].(type) {
		case fmt.Stringer:
			_, err = logger.Write(
				[]byte(fmt.Sprintf(head.String(), parameters[1:]...)),
			)

		case string:
			_, err = logger.Write(
				[]byte(fmt.Sprintf(head, parameters[1:]...)),
			)
		default:
			panic(
				errors.New(
					fmt.Sprintf(cannotFormatErrorPattern, parameters[0]),
				),
			)
		}

		if err != nil {
			panic(err)
		}

		_, err = logger.Write([]byte("\n"))
		if err != nil {
			panic(err)
		}
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
