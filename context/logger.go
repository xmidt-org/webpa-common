package context

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
