package fanout

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

// New produces a go-kit Endpoint which tries all of a set of component endpoints concurrently.  The first component
// to respond successfully causes this endpoint to return with that response immediately, without waiting
// on subsequent endpoints.  If the context is canceled for any reason, ctx.Err() is returned.  Finally,
// if all endpoints fail, an error is returned with a span for each endpoint.
//
// If spanner is nil or endpoints is empty, this function panics.
func New(spanner tracing.Spanner, endpoints Components) endpoint.Endpoint {
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
		ctx = NewContext(ctx, fanoutRequest)

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
