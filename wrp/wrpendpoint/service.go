package wrpendpoint

import (
	"context"
)

// Service represents a component which processes WRP transactions.
type Service interface {
	// ServeWRP processes a WRP request.  Either the Response or the error
	// returned from this method will be nil, but not both.
	ServeWRP(context.Context, Request) (Response, error)
}

// ServiceFunc is a function type that implements Service
type ServiceFunc func(context.Context, Request) (Response, error)

func (sf ServiceFunc) ServeWRP(ctx context.Context, r Request) (Response, error) {
	return sf(ctx, r)
}
