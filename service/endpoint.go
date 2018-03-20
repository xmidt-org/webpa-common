package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
)

// Key represents a service key.
type Key interface {
	// Bytes returns the raw byte slice contents of the key.  This is what will be
	// passed to Accessor.Get.
	Bytes() []byte
}

// StringKey is a simple string that implements Key.
type StringKey string

func (sk StringKey) Bytes() []byte {
	return []byte(sk)
}

// KeyParser is a parsing strategy that takes an arbitrary string and produces a service Key.
type KeyParser func(string) (Key, error)

// NewAccessorEndpoint produces a go-kit Endpoint which delegates to an Accessor.
// The returned Endpoint expects a service Key as its request, and returns the instance
// string as the response.
func NewAccessorEndpoint(a Accessor) endpoint.Endpoint {
	if a == nil {
		panic("an Accessor is required")
	}

	return func(ctx context.Context, request interface{}) (interface{}, error) {
		key := request.(Key)
		return a.Get(key.Bytes())
	}
}
