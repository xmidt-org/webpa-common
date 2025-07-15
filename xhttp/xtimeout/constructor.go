// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xtimeout

import (
	"context"
	"net/http"
	"time"

	"github.com/xmidt-org/webpa-common/v2/xhttp"
)

// defaultTimedOut is the default http.Handler used for timeout responses
var defaultTimedOut = xhttp.Constant{Code: http.StatusGatewayTimeout}

// timeoutHandler is the internal decorator handler that handles timeouts in its ServeHTTP method
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

// Options holds the set of configurable options for a timeout constructor.
type Options struct {
	// Timeout is the time allowed for the decorated handler's ServeHTTP method to run.
	// If unset or nonpositive, no decoration is performed.
	Timeout time.Duration

	// TimedOut is the optional http.Handler that is executed with the original http.Request
	// whenever a timeout occurs.  If unset, a default handler is used that simply sets the response
	// code to http.StatusGatewayTimeout.
	TimedOut http.Handler
}

// NewConstructor returns an Alice-style constructor that enforces a timeout for any handler it decorates.
// If timeout is nonpositive, a constructor is returned that does no decoration.
// If timedOut is nil, a default timedOut handler is used that just sets an http.StatusGatewayTimeout response code.
func NewConstructor(o Options) func(http.Handler) http.Handler {
	if o.Timeout <= 0 {
		return xhttp.NilConstructor
	}

	// nolint: typecheck
	if o.TimedOut == nil {
		o.TimedOut = defaultTimedOut
	}

	return func(next http.Handler) http.Handler {
		return &timeoutHandler{
			timeout:  o.Timeout,
			timedOut: o.TimedOut,
			next:     next,
		}
	}
}
