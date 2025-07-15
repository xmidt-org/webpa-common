// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"github.com/go-kit/kit/sd"
)

type ContextualInstancer struct {
	sd.Instancer
	m map[string]interface{}
}

func (ci ContextualInstancer) Metadata() map[string]interface{} {
	return ci.m
}

// NewContextualInstancer returns an sd.Instancer that has been enriched with metadata.
// This metadata allows infrastructure to carry configuration information about the instancer
// across API boundaries so that it can be logged or otherwise processed.
//
// If m is empty, i is returned as is.
func NewContextualInstancer(i sd.Instancer, m map[string]interface{}) sd.Instancer {
	if len(m) == 0 {
		return i
	}

	return ContextualInstancer{i, m}
}

// Instancers is a collection of sd.Instancer objects, keyed by arbitrary strings.
type Instancers map[string]sd.Instancer

func (is Instancers) Len() int {
	return len(is)
}

func (is Instancers) Has(key string) bool {
	_, ok := is[key]
	return ok
}

func (is Instancers) Get(key string) (sd.Instancer, bool) {
	v, ok := is[key]
	return v, ok
}

func (is *Instancers) Set(key string, i sd.Instancer) {
	if *is == nil {
		*is = make(Instancers)
	}

	(*is)[key] = i
}

func (is Instancers) Copy() Instancers {
	if len(is) > 0 {
		clone := make(Instancers, len(is))
		for k, v := range is {
			clone[k] = v
		}

		return clone
	}

	return nil
}

func (is Instancers) Stop() {
	for _, v := range is {
		v.Stop()
	}
}
