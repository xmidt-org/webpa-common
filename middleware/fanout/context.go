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

// Request is the interface that transport-specific fanout requests can implement to expose.
// the processed entity.  Implementing this request is optional for the fanout, but is required
// if response encoders are to be able to access decoded request entities.
type Request interface {
	// Entity returns the decoded entity associated with this fanout request
	Entity() interface{}
}

// FromContextEntity returns the entity decoded by the request decoder.  If no such entity
// is in the context (including if the fanout request did not supply one), this method returns false.
func FromContextEntity(ctx context.Context) (interface{}, bool) {
	r, ok := FromContext(ctx).(Request)
	if !ok {
		return nil, false
	}

	return r.Entity(), true
}
