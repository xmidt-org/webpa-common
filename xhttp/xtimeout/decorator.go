package xtimeout

import (
	"context"
	"net/http"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
)

var defaultTimedOut = xhttp.Constant{Code: http.StatusGatewayTimeout}

type timeoutHandler struct {
	timeout  time.Duration
	timedOut http.Handler
	next     http.Handler
}

func (th *timeoutHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	var (
		done        = make(chan struct{})
		panics      = make(chan interface{}, 1)
		writer      xhttp.BufferedWriter
		ctx, cancel = context.WithTimeout(request.Context(), th.timeout)
	)

	defer cancel()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				panics <- p
			}
		}()

		th.next.ServeHTTP(&writer, request.WithContext(ctx))
		close(done)
	}()

	select {
	case p := <-panics:
		panic(p) // mimic the behavior of net/http TimeoutHandler
	case <-done:
		writer.WriteTo(response)
	case <-ctx.Done():
		writer.Close()
		th.timedOut.ServeHTTP(response, request)
	}
}

// NewConstructor returns an Alice-style constructor that enforces a timeout for any handler it decorates.
// If timeout is nonpositive, a constructor is returned that does no decoration.
// If timedOut is nil, a default timedOut handler is used that just sets an http.StatusGatewayTimeout response code.
func NewConstructor(timeout time.Duration, timedOut http.Handler) func(http.Handler) http.Handler {
	if timeout <= 0 {
		return xhttp.NilConstructor
	}

	if timedOut == nil {
		timedOut = defaultTimedOut
	}

	return func(next http.Handler) http.Handler {
		return &timeoutHandler{
			timeout:  timeout,
			timedOut: timedOut,
			next:     next,
		}
	}
}
