package logging

import "github.com/go-kit/kit/log"

// Contextual describes an object which can describe itself with metadata for logging.
// Implementing this interface allows code to carry logging context data across API
// boundaries without compromising encapsulation.
type Contextual interface {
	Metadata() map[string]interface{}
}

// enrich is the helper function that emits contextual information into its logger argument.
func enrich(wither func(log.Logger, ...interface{}) log.Logger, logger log.Logger, objects []interface{}) log.Logger {
	var keyvals []interface{}
	for _, e := range objects {
		switch m := e.(type) {
		case Contextual:
			for k, v := range m.Metadata() {
				keyvals = append(keyvals, k, v)
			}

		case map[string]interface{}:
			for k, v := range m {
				keyvals = append(keyvals, k, v)
			}

		case map[string]string:
			for k, v := range m {
				keyvals = append(keyvals, k, v)
			}
		}
	}

	if len(keyvals) > 0 {
		return wither(logger, keyvals...)
	}

	return logger
}

// Enrich uses log.With to add contextual information to a logger.  The given set of objects are examined to see if they contain
// any metadata.  Objects that do not contain metadata are simply ignored.
//
// An object contains metadata if it implements Contextual, is a map[string]interface{}, or is a map[string]string.  In those cases,
// the key/value pairs are present in the returned logger.
func Enrich(logger log.Logger, objects ...interface{}) log.Logger {
	return enrich(log.With, logger, objects)
}

// EnrichPrefix is like Enrich, except that it uses log.WithPrefix.
func EnrichPrefix(logger log.Logger, objects ...interface{}) log.Logger {
	return enrich(log.WithPrefix, logger, objects)
}
