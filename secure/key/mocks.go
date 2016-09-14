package key

import (
	"github.com/stretchr/testify/mock"
)

// MockResolver is a stretchr mock for Resolver.  It's exposed for other package tests.
type MockResolver struct {
	mock.Mock
}

func (resolver *MockResolver) ResolveKey(keyId string) (Pair, error) {
	arguments := resolver.Called(keyId)
	if pair, ok := arguments.Get(0).(Pair); ok {
		return pair, arguments.Error(1)
	} else {
		return nil, arguments.Error(1)
	}
}

// MockCache is a stretchr mock for Cache.  It's exposed for other package tests.
type MockCache struct {
	mock.Mock
}

func (cache *MockCache) ResolveKey(keyId string) (Pair, error) {
	arguments := cache.Called(keyId)
	if pair, ok := arguments.Get(0).(Pair); ok {
		return pair, arguments.Error(1)
	} else {
		return nil, arguments.Error(1)
	}
}

func (cache *MockCache) UpdateKeys() (int, []error) {
	arguments := cache.Called()
	if errors, ok := arguments.Get(1).([]error); ok {
		return arguments.Int(0), errors
	} else {
		return arguments.Int(0), nil
	}
}

// MockPair is a stretchr mock for Pair.  It's exposed for other package tests.
type MockPair struct {
	mock.Mock
}

func (pair *MockPair) Purpose() Purpose {
	arguments := pair.Called()
	return arguments.Get(0).(Purpose)
}

func (pair *MockPair) Public() interface{} {
	arguments := pair.Called()
	return arguments.Get(0)
}

func (pair *MockPair) HasPrivate() bool {
	arguments := pair.Called()
	return arguments.Bool(0)
}

func (pair *MockPair) Private() interface{} {
	arguments := pair.Called()
	return arguments.Get(0)
}

type MockParser struct {
	mock.Mock
}

func (parser *MockParser) ParseKey(purpose Purpose, data []byte) (Pair, error) {
	arguments := parser.Called(purpose, data)
	if pair, ok := arguments.Get(0).(Pair); ok {
		return pair, arguments.Error(1)
	}

	return nil, arguments.Error(1)
}
