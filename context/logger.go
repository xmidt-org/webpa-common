package context

// Logger defines the expected methods to be provided by logging infrastructure
type Logger interface {
	Debug(parameters ...interface{})
	Info(parameters ...interface{})
	Warn(parameters ...interface{})
	Error(parameters ...interface{})
}
