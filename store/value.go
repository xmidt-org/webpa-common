package store

// Value fronts an arbitrary object resulting from some nontrivial operation.
// For example, the parsed object resulting from a JSON message, or a secure
// key resulting from parsing a PEM block.
//
// This type is subtley different from atomic.Value.  This Value type should be
// assumed to use an external resource or some complex algorithm in order to obtain its values.
// That's why Load() returns an error secondary parameter.
type Value interface {
	// Load returns the value object.  The mechanism used to obtain the value can
	// be arbitrarily complex and might fail.  For example, an external resource
	// such as an http server might be consulted.
	Load() (interface{}, error)
}

// ValueFunc is a function type that implements Value
type ValueFunc func() (interface{}, error)

func (v ValueFunc) Load() (interface{}, error) {
	return v()
}

// singleton is an internal Value implementation that returns the same
// value all the time.  It's primarily used when NewValue() is called
// with CachePeriodForever.
type singleton struct {
	value interface{}
}

func (s *singleton) Load() (interface{}, error) {
	return s.value, nil
}

// NewValue creates a dynamic implementation of Value based on the period
// parameter.
//
// If period is 0 (CachePeriodForever), this function will immediately attempt
// to invoke source.Load() and return a singleton Value.  If source.Load() fails,
// this function returns nil plus that error.
func NewValue(source Value, period CachePeriod) (Value, error) {
	if period == CachePeriodForever {
		if once, err := source.Load(); err != nil {
			return nil, err
		} else {
			return &singleton{once}, nil
		}
	} else if period < 0 {
		// never cache ... so just return the source
		return source, nil
	}

	return NewCache(source, period)
}
