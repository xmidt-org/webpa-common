// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xcontext

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
)

// ContextAware is an optional mixin implemented by anything with can hold a context
type ContextAware interface {
	// Context *never* returns a nil context
	Context() context.Context
	SetContext(context.Context)
}

type contextAwareResponseWriter struct {
	http.ResponseWriter
	ctx context.Context
}

var _ ContextAware = &contextAwareResponseWriter{}
var _ http.Hijacker = &contextAwareResponseWriter{}
var _ http.Flusher = &contextAwareResponseWriter{}
var _ http.Pusher = &contextAwareResponseWriter{}

func (carw contextAwareResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := carw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, errors.New("hijacker not supported")
}

func (carw contextAwareResponseWriter) Flush() {
	if f, ok := carw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (carw contextAwareResponseWriter) Push(target string, opts *http.PushOptions) error {
	if p, ok := carw.ResponseWriter.(http.Pusher); ok {
		return p.Push(target, opts)
	}

	return errors.New("pusher not supported")
}

func (carw *contextAwareResponseWriter) Context() context.Context {
	// nolint: typecheck
	if carw.ctx != nil {
		return carw.ctx
	}

	return context.Background()
}

func (carw *contextAwareResponseWriter) SetContext(ctx context.Context) {

	if ctx == nil {
		// mimic the behavior of the net/http package
		panic("nil context")
	}

	carw.ctx = ctx
}

func Context(response http.ResponseWriter, request *http.Request) context.Context {
	if ca, ok := response.(ContextAware); ok {
		return ca.Context()
	}

	// fallback to the request's context
	return request.Context()
}

// SetContext associates a context with a response.  Useful for decorated code that needs to communicate
// a context back up the call stack.
//
// Note that since ContextAware is an optional interface, it's possible that the supplied ResponseWriter does
// not implement ContextAware.  This is tolerated, so as to be backward compatible.
//
// The returned ResponseWriter will always be ContextAware.  This writer can be used for subsequent handling code.
func SetContext(response http.ResponseWriter, ctx context.Context) http.ResponseWriter {
	if ca, ok := response.(ContextAware); ok {
		ca.SetContext(ctx)
		return response
	}

	if ctx == nil {
		panic("nil context")
	}

	return &contextAwareResponseWriter{response, ctx}
}

// WithContext associates a context with the response/request pair that can later be accessed via the Context function.
// If response is already ContextAware, it is used and returned as is.
//
// Useful for code that is decorating http handling code in order to establish a context.
func WithContext(response http.ResponseWriter, request *http.Request, ctx context.Context) (http.ResponseWriter, *http.Request) {
	if ca, ok := response.(ContextAware); ok {
		ca.SetContext(ctx)
		return response, request.WithContext(ctx)
	}

	if ctx == nil {
		ctx = context.Background()
	}

	return &contextAwareResponseWriter{response, ctx}, request.WithContext(ctx)
}
