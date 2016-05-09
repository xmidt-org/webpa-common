package logging

// LoggerFactory represents the behavior of a type which can create a Logger.
// Integrations must supply an implementation of this interface, typically
// configured through JSON.
type LoggerFactory interface {
	// Returns a new, distinct Logger instance using this factory's configuration
	NewLogger() (Logger, error)
}
