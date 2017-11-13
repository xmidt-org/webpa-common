package fanout

import "context"

type fanoutRequestKey struct{}

// newContext returns a new Context with the given fanoutRequest.  This function is not
// exported as only the fanout endpoint itself should use it.
func newContext(ctx context.Context, fanoutRequest interface{}) context.Context {
	return context.WithValue(ctx, fanoutRequestKey{}, fanoutRequest)
}

// FromContext produces the originally decoded request object applied to all component fanouts.
// This will be the object returned by the fanout's associated DecodeRequestFunc.
func FromContext(ctx context.Context) interface{} {
	return ctx.Value(fanoutRequestKey{})
}
