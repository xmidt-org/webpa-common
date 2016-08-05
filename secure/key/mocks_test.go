package key

import (
	"github.com/stretchr/testify/mock"
)

type mockResolver struct {
	mock.Mock
}

func (resolver *mockResolver) UsesKeyId() bool {
	args := resolver.Called()
	return args.Bool(0)
}

func (resolver *mockResolver) ResolveKey(keyId string) (interface{}, error) {
	args := resolver.Called(keyId)
	return args.Get(0), args.Error(1)
}

type mockKeyCache struct {
	mock.Mock
}

func (keyCache *mockKeyCache) UsesKeyId() bool {
	args := keyCache.Called()
	return args.Bool(0)
}

func (keyCache *mockKeyCache) ResolveKey(keyId string) (interface{}, error) {
	args := keyCache.Called(keyId)
	return args.Get(0), args.Error(1)
}

func (keyCache *mockKeyCache) UpdateKeys() (int, []error) {
	args := keyCache.Called()
	if errors, ok := args.Get(1).([]error); ok {
		return args.Int(0), errors
	} else {
		return args.Int(0), nil
	}
}
