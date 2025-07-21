// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xfilter

import (
	"net/http"
)

// Interface is essentially a predicate that determines whether a request is allowed.
type Interface interface {
	// Allow tests whether the given request is allowed to execute.  This method can return
	// errors that implement the go-kit interfaces, e.g. StatusCoder.
	Allow(*http.Request) error
}

// Func is the function equivalent of Interface
type Func func(*http.Request) error

func (f Func) Allow(r *http.Request) error {
	return f(r)
}

var allow = Func(func(*http.Request) error {
	return nil
})

// Allow returns an xfilter that always returns a nil error
func Allow() Interface {
	return allow
}

// Reject returns an xfilter that always returns the given error.
// If err == nil, this function is equivalent to Allow.
func Reject(err error) Interface {
	if err == nil {
		return Allow()
	}

	return Func(func(*http.Request) error { return err })
}
