package golog

import (
	"errors"
	"fmt"
	"github.com/ian-kent/go-log/levels"
)

const (
	outOfRangeLogLevelPattern   string = "LogLevel value is out of range: %d"
	invalidLogLevelValuePattern string = "Invalid LogLevel value: %s"
)

// LogLevel is a custom extension of go-log's LogLevel type.
// We use this to provide custom marshalling.
type LogLevel levels.LogLevel

func (l LogLevel) MarshalJSON() ([]byte, error) {
	stringValue, ok := levels.LogLevelsToString[levels.LogLevel(l)]
	if !ok {
		return nil, errors.New(fmt.Sprintf(outOfRangeLogLevelPattern, l))
	}

	return []byte(`"` + stringValue + `"`), nil
}

func (l *LogLevel) UnmarshalJSON(data []byte) error {
	if data[0] == '"' {
		if logLevelValue, ok := levels.StringToLogLevels[string(data[1:len(data)-1])]; ok {
			*l = LogLevel(logLevelValue)
			return nil
		}
	}

	return errors.New(fmt.Sprintf(invalidLogLevelValuePattern, data))
}
