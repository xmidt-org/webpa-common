package store

import (
	"fmt"
	"strconv"
	"time"
)

// CachePeriod is a custom type which describes the amount of time before
// a cached value is considered to have been expired.  This type has
// several reserved values that govern caching behavior.
type CachePeriod time.Duration

const (
	// CachePeriodForever is a special value that means a cache never expires
	CachePeriodForever CachePeriod = 0

	// CachePeriodForeverValue is the string value indicating that a cache never expires
	CachePeriodForeverValue string = "forever"

	// CachePeriodNever is a special value indicating something must never be cached.
	// Any negative value for CachePeriod will be interpreted as this value.
	CachePeriodNever CachePeriod = -1

	// CachePeriodNeverValue is the string value indicating that something should not be cached
	CachePeriodNeverValue string = "never"
)

// String returns a human-readable value for this period.
func (c CachePeriod) String() string {
	if c == CachePeriodForever {
		return CachePeriodForeverValue
	} else if c < 0 {
		return CachePeriodNeverValue
	}

	return time.Duration(c).String()
}

// Next returns time after a given, base time when
// the period has elapsed
func (c CachePeriod) Next(base time.Time) time.Time {
	return base.Add(time.Duration(c))
}

// MarshalJSON provides the custom JSON format for a cache period.
func (c CachePeriod) MarshalJSON() (data []byte, err error) {
	if c == CachePeriodForever {
		data = []byte(`"` + CachePeriodForeverValue + `"`)
	} else if c < 0 {
		data = []byte(`"` + CachePeriodNeverValue + `"`)
	} else {
		data = []byte(`"` + time.Duration(c).String() + `"`)
	}

	return
}

// UnmarshalJSON parses the custom JSON format for a cache period.
// Raw integers are interpreted as seconds.
func (c *CachePeriod) UnmarshalJSON(data []byte) error {
	var value string
	if data[0] == '"' {
		value = string(data[1 : len(data)-1])
	} else {
		value = string(data)
	}

	if value == CachePeriodForeverValue {
		*c = CachePeriodForever
		return nil
	} else if value == CachePeriodNeverValue {
		*c = CachePeriodNever
		return nil
	}

	// first, try to parse the value as a time.Duration
	if duration, err := time.ParseDuration(value); err == nil {
		*c = CachePeriod(duration)
		return nil
	}

	// next, try to parse the value as a number of seconds
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid cache period %s: %v", string(data), err)
	}

	if seconds < 0 {
		*c = CachePeriodNever
	} else if seconds == 0 {
		*c = CachePeriodForever
	} else {
		*c = CachePeriod(time.Second * time.Duration(seconds))
	}

	return nil
}
