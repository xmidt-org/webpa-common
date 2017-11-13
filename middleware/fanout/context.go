package fanout

import "context"

type fanoutRequestKey struct{}

// NewContext returns a new Context with the given fanoutRequest.  This function is primarily used by the endpoint
// returned by New to inject the decoded fanout request into the context so that downstream code, such as request functions,
// can access it.
func NewContext(ctx context.Context, fanoutRequest interface{}) context.Context {
	return context.WithValue(ctx, fanoutRequestKey{}, fanoutRequest)
}

// FromContext produces the originally decoded request object applied to all component fanouts.
// This will be the object returned by the fanout's associated DecodeRequestFunc.
func FromContext(ctx context.Context) interface{} {
	return ctx.Value(fanoutRequestKey{})
}
