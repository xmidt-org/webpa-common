package handler

import "context"

type contextKey struct{}

//ContextValues contains the values shared under the satClientIDKey from this package
type ContextValues struct {
	SatClientID string
	Method      string
	Path        string
	PartnerIDs  []string
}

//NewContextWithValue returns a context with the specified context values
func NewContextWithValue(ctx context.Context, vals *ContextValues) context.Context {
	return context.WithValue(ctx, contextKey{}, vals)
}

//FromContext returns ContextValues type (if any) along with a boolean that indicates whether
//the returned value is of the required/correct type for this package.
func FromContext(ctx context.Context) (*ContextValues, bool) {
	vals, ofType := ctx.Value(contextKey{}).(*ContextValues)
	return vals, ofType
}
