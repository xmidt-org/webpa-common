package types

import (
	"strconv"
	"time"
)

// Duration is an extension of time.Duration that provides prettier JSON support
type Duration time.Duration

// String delegates to time.Duration.String()
func (d Duration) String() string {
	return time.Duration(d).String()
}

// MarshalJSON produces a formatted string of the form
// produced by time.Duration.String()
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSON permits either: (1) strings of the form accepted by time.ParseDuration(),
// or (2) numeric time values, which are assumed to be nanoseconds.
func (d *Duration) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		unwrappedDuration, err := time.ParseDuration(string(data[1 : len(data)-1]))
		if err != nil {
			return err
		}

		*d = Duration(unwrappedDuration)
	} else {
		nanos, err := strconv.ParseInt(string(data), 10, 64)
		if err != nil {
			return err
		}

		*d = Duration(nanos)
	}

	return nil
}
