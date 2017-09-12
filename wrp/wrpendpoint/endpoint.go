package wrpendpoint

import (
	"context"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

const DefaultTimeout = 30 * time.Second

// New constructs a go-kit endpoint for the given WRP service.  This endpoint enforces
// the constraint that ctx must be the context associated with the Request.
func New(s Service) endpoint.Endpoint {
	return func(ctx context.Context, value interface{}) (interface{}, error) {
		request := value.(*request)
		request.ctx = ctx

		return s.ServeWRP(request)
	}
}

// Timeout applies the given timeout to all WRP Requests.  The context's cancellation
// function is always called.
func Timeout(timeout time.Duration) endpoint.Middleware {
	if timeout < 1 {
		timeout = DefaultTimeout
	}

	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, value interface{}) (interface{}, error) {
			var (
				timeoutCtx, cancel = context.WithTimeout(ctx, timeout)
				request            = value.(*request)
			)

			request.ctx = timeoutCtx
			defer cancel()
			return next(timeoutCtx, request)
		}
	}
}

// Logging provides a Middleware instance that logs WRP requests and responses.
func Logging(logger log.Logger) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, value interface{}) (interface{}, error) {
			request := value.(Request)

			logger.Log(
				level.Key(), level.InfoValue(),
				logging.MessageKey(), "WRP request",
				"destination", request.Destination(),
				"transactionID", request.TransactionID(),
			)

			result, err := next(ctx, value)
			if err != nil {
				logger.Log(
					level.Key(), level.ErrorValue(),
					logging.MessageKey(), "WRP error",
					"destination", request.Destination(),
					"transactionID", request.TransactionID(),
					logging.ErrorKey(), err,
				)
			} else {
				response := result.(Response)
				logger.Log(
					level.Key(), level.InfoValue(),
					logging.MessageKey(), "WRP response",
					"destination", response.Destination(),
					"transactionID", request.TransactionID(),
				)
			}

			return result, err
		}
	}
}
