package device

import (
	"context"
	"net/http"
)

type idKey struct{}

// GetID returns the device ID from a Context.  If no device ID is present, this
// function returns false for the second parameter.
func GetID(ctx context.Context) (id ID, ok bool) {
	id, ok = ctx.Value(idKey{}).(ID)
	return
}

// WithID returns a new Context with the given device ID as a value.
func WithID(id ID, parent context.Context) context.Context {
	return context.WithValue(parent, idKey{}, id)
}

// WithIDRequest returns a new HTTP request with the given device ID in the associated Context.
func WithIDRequest(id ID, original *http.Request) *http.Request {
	return original.WithContext(
		WithID(id, original.Context()),
	)
}
