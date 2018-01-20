package contextcommon

import "context"

//NewContextWithValue returns a context with the specified value
func NewContextWithValue(ctx context.Context, key interface{}, val interface{}) context.Context {
	return context.WithValue(ctx, key, val)
}

//FromContext retrieves a generic value (if any) from the given context
func FromContext(ctx context.Context, key interface{}) interface{} {
	return ctx.Value(key)
}
