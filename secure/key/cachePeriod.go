package key

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

// CachePeriod is a customization of time.Duration, and adds a few
// extra values.  Positive values are interpreted as cache expiry durations.
// Zero and negative values have some special meanings.
type CachePeriod time.Duration

const (
	// CachePeriodDefault is an indicator that the default cache period should
	// be used.  The default is context-sensitive, and is either the defaultCachePeriod
	// from an enclosing object or CachePeriodForever as a fallback default.
	// This constant is the zero-value for CachePeriod, and can be used to detect
	// when a cache period was omitted.
	CachePeriodDefault      = CachePeriod(0)
	CachePeriodDefaultValue = "default"

	// CachePeriodNever means obtain the key each time it's asked for.
	CachePeriodNever      = CachePeriod(-1)
	CachePeriodNeverValue = "never"

	// CachePeriodForever means to obtain the key exactly once, and hold that
	// key as long as the program is running.
	CachePeriodForever      = CachePeriod(-2)
	CachePeriodForeverValue = "forever"
)

var (
	// purposeUnmarshal is a reverse mapping of the string representations
	// for CachePeriod.  It's principally useful when unmarshalling values.
	periodUnmarshal = map[string]CachePeriod{
		CachePeriodDefaultValue: CachePeriodDefault,
		CachePeriodNeverValue:   CachePeriodNever,
		CachePeriodForeverValue: CachePeriodForever,
	}
)

// String returns a string representation of the this CachePeriod
func (period CachePeriod) String() string {
	switch {
	case period > 0:
		return time.Duration(period).String()
	case period == CachePeriodDefault:
		return CachePeriodDefaultValue
	case period == CachePeriodForever:
		return CachePeriodForeverValue
	case period == CachePeriodNever:
		return CachePeriodNeverValue
	default:
		return CachePeriodForeverValue
	}
}

func (period *CachePeriod) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		value := string(data[1 : len(data)-1])
		reservedValue, ok := periodUnmarshal[value]
		if ok {
			*period = reservedValue
			return nil
		}

		duration, err := time.ParseDuration(value)
		if err != nil {
			return err
		}

		if duration > 0 {
			*period = CachePeriod(duration)
			return nil
		}
	}

	return errors.New(fmt.Sprintf("Invalid cache period: %s", data))
}

func (period CachePeriod) MarshalJSON() ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("\"")
	buffer.WriteString(period.String())
	buffer.WriteString("\"")

	return buffer.Bytes(), nil
}
