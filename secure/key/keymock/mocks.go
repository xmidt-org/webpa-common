package keymock

import (
	"github.com/stretchr/testify/mock"
)

// Resolver is a stretchr mock for key.Resolver.
type Resolver struct {
	mock.Mock
}

func (resolver *Resolver) ResolveKey(keyId string) (interface{}, error) {
	args := resolver.Called(keyId)
	return args.Get(0), args.Error(1)
}

// Cache is a stretchr mock for key.Cache
type Cache struct {
	mock.Mock
}

func (cache *Cache) ResolveKey(keyId string) (interface{}, error) {
	args := cache.Called(keyId)
	return args.Get(0), args.Error(1)
}

func (cache *Cache) UpdateKeys() (int, []error) {
	args := cache.Called()
	if errors, ok := args.Get(1).([]error); ok {
		return args.Int(0), errors
	} else {
		return args.Int(0), nil
	}
}
