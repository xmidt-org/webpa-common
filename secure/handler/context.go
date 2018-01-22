package handler

import "context"

//NewContextWithValue returns a context with the specified context values
func NewContextWithValue(ctx context.Context, vals *ContextValues) context.Context {
	return context.WithValue(ctx, handlerValuesKey, vals)
}

//FromContext returns ContextValues type (if any) along with a boolean that indicates whether
//the returned value is of the required/correct type for this package.
func FromContext(ctx context.Context) (*ContextValues, bool) {
	vals, ofType := ctx.Value(handlerValuesKey).(*ContextValues)
	return vals, ofType
}
