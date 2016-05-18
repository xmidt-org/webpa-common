package logging

import (
	"io"
	"os"
)

// LoggerFactory represents the behavior of a type which can create a Logger.
// Integrations must supply an implementation of this interface, typically
// configured through JSON.
type LoggerFactory interface {
	// Returns a new, distinct Logger instance using this factory's configuration
	NewLogger(name string) (Logger, error)
}

// DefaultLoggerFactory provides a simple way to create Logger instances adapted
// onto log.Logger.
type DefaultLoggerFactory struct {
	Output io.Writer
}

func (factory *DefaultLoggerFactory) NewLogger(name string) (Logger, error) {
	output := factory.Output
	if output == nil {
		output = os.Stdout
	}

	return &LoggerWriter{output}, nil
}
