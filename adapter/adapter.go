package adapter

import (
	"go.uber.org/zap"
)

type Logger struct {
	*zap.Logger
}

// this method makes Adapter implement log.Logger
func (l Logger) Log(keyvals ...interface{}) error {
	fields := make([]zap.Field, 0, len(keyvals))
	for i := 0; i < len(keyvals); i += 2 {
		fields = append(fields, zap.Any(keyvals[i].(string), keyvals[i+1]))
	}

	// ignore the case where there's an odd number of keyvals ... that would be a bug
	// and we're deprecating webpa-common anyway

	l.Logger.Info("", fields...)
	return nil
}
