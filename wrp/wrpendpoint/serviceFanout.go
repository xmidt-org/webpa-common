package wrpendpoint

import (
	"context"
	"errors"

	"github.com/Comcast/webpa-common/tracing"
)

// serviceFanout takes a single WRP request and dispatches it concurrently to zero
// or more go-kit endpoints.
type serviceFanout struct {
	services map[string]Service
	spanner  tracing.Spanner
}

// fanoutResponse is the internal tuple used to communicate the results of an asynchronously
// invoked service
type fanoutResponse struct {
	spans    []tracing.Span
	response Response
	err      error
}

// NewServiceFanout produces a WRP service which invokes each of a set of services concurrently for each WRP request.
// The first service which returns a valid response becomes the response of the fanout service.  If the context is
// cancelled, then ctx.Err() is returned.  Finally, if all services fail then a tracing.SpanError is returned with
// the last error set as the causal error.
func NewServiceFanout(services map[string]Service) Service {
	if len(services) == 0 {
		return ServiceFunc(func(context.Context, Request) (Response, error) {
			return nil, errors.New("No configured services")
		})
	}

	copyOf := make(map[string]Service, len(services))
	for k, v := range services {
		copyOf[k] = v
	}

	return &serviceFanout{
		services: copyOf,
		spanner:  tracing.NewSpanner(),
	}
}

func (sf *serviceFanout) ServeWRP(ctx context.Context, request Request) (Response, error) {
	results := make(chan fanoutResponse, len(sf.services))
	for name, s := range sf.services {
		go func(name string, s Service) {
			var (
				finisher      = sf.spanner.Start(name)
				response, err = s.ServeWRP(ctx, request)
				span          = finisher(err)
			)

			results <- fanoutResponse{
				spans:    []tracing.Span{span},
				response: response.AddSpans(span),
				err:      err,
			}
		}(name, s)
	}

	var (
		lastError error
		spans     []tracing.Span
	)

	for r := 0; r < len(sf.services); r++ {
		select {
		case <-ctx.Done():
			return nil, tracing.NewSpanError(ctx.Err(), spans...)
		case fr := <-results:
			spans = append(spans, fr.spans...)
			if fr.err != nil {
				lastError = fr.err
			} else {
				return fr.response.AddSpans(spans...), nil
			}
		}
	}

	// use the last error as the causal error
	return nil, tracing.NewSpanError(lastError, spans...)
}
