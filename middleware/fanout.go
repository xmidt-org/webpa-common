package middleware

import (
	"context"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log/level"
)

// fanoutResponse is the internal tuple used to communicate the results of an asynchronously
// invoked endpoint
type fanoutResponse struct {
	name     string
	span     tracing.Span
	response interface{}
	err      error
}

type fanoutRequestKey struct{}

// FanoutRequestFromContext produces the originally decoded request object applied to all component fanouts.
// This will be the object returned by the fanout's associated DecodeRequestFunc.
func FanoutRequestFromContext(ctx context.Context) interface{} {
	return ctx.Value(fanoutRequestKey{})
}

// ComponentEndpoints holds the component endpoint objects which will be concurrently invoked by a fanout.
type ComponentEndpoints map[string]endpoint.Endpoint

// Apply produces a new ComponentEndpoints with each endpoint decorated by the given middleware.  To apply
// multiple middleware in one shot, pass the result of endpoint.Chain to this method.
func (c ComponentEndpoints) Apply(m endpoint.Middleware) ComponentEndpoints {
	decorated := make(ComponentEndpoints, len(c))
	for k, v := range c {
		decorated[k] = m(v)
	}

	return decorated
}

// Fanout produces a go-kit Endpoint which tries all of a set of endpoints concurrently.  The first endpoint
// to respond successfully causes this endpoint to return with that response immediately, without waiting
// on subsequent endpoints.  If the context is canceled for any reason, ctx.Err() is returned.  Finally,
// if all endpoints fail, an error is returned with a span for each endpoint.
//
// If spanner is nil or endpoints is empty, this function panics.
func Fanout(spanner tracing.Spanner, endpoints ComponentEndpoints) endpoint.Endpoint {
	if spanner == nil {
		panic("No spanner supplied")
	}

	if len(endpoints) == 0 {
		panic("No endpoints supplied")
	}

	// use a copy of the endpoints map, for concurrent safety
	copyOf := make(map[string]endpoint.Endpoint, len(endpoints))
	for k, v := range endpoints {
		copyOf[k] = v
	}

	endpoints = copyOf
	return func(ctx context.Context, fanoutRequest interface{}) (interface{}, error) {
		ctx = context.WithValue(ctx, fanoutRequestKey{}, fanoutRequest)

		var (
			logger  = logging.Logger(ctx)
			results = make(chan fanoutResponse, len(endpoints))
		)

		for name, e := range endpoints {
			go func(name string, e endpoint.Endpoint) {
				var (
					finisher      = spanner.Start(name)
					response, err = e(ctx, fanoutRequest)
				)

				results <- fanoutResponse{
					name:     name,
					span:     finisher(err),
					response: response,
					err:      err,
				}
			}(name, e)
		}

		var (
			lastError error
			spans     []tracing.Span
		)

		for r := 0; r < len(endpoints); r++ {
			select {
			case <-ctx.Done():
				logger.Log(level.Key(), level.WarnValue(), logging.ErrorKey(), ctx.Err(), logging.MessageKey(), "timed out")
				return nil, tracing.NewSpanError(ctx.Err(), spans...)
			case fr := <-results:
				spans = append(spans, fr.span)
				if fr.err != nil {
					lastError = fr.err
					logger.Log(level.Key(), level.DebugValue(), "service", fr.name, logging.ErrorKey(), fr.err, logging.MessageKey(), "failed")
				} else {
					logger.Log(level.Key(), level.DebugValue(), "service", fr.name, logging.MessageKey(), "success")
					response, _ := tracing.MergeSpans(fr.response, spans)
					return response, nil
				}
			}
		}

		logger.Log(level.Key(), level.ErrorValue(), logging.ErrorKey(), lastError, logging.MessageKey(), "all endpoints failed")
		return nil, tracing.NewSpanError(lastError, spans...)
	}
}
