package logging

import (
	"context"

	"github.com/go-kit/kit/log"
	"go.uber.org/zap"
)

type contextKey uint32

const loggerKey contextKey = 1

// WithLogger adds the given Logger to the context so that it can be retrieved with Logger
func WithLogger(parent context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(parent, loggerKey, logger)
}

// GetLogger retrieves the go-kit logger associated with the context.  If no logger is
// present in the context, DefaultLogger is returned instead.
func GetLogger(ctx context.Context) log.Logger {
	if logger, ok := ctx.Value(loggerKey).(log.Logger); ok {
		return logger
	}

	return DefaultLogger()
}
