package convey

import "context"

type contextKey struct{}

func NewContext(parent context.Context, v C) context.Context {
	return context.WithValue(parent, contextKey{}, v)
}

func FromContext(ctx context.Context) (C, bool) {
	v, ok := ctx.Value(contextKey{}).(C)
	return v, ok
}
