package key

import (
	"errors"
	"testing"
	"time"
)

// testLoader is a Loader implementation designed for testing
// caches.  This Loader always returns the publicKey.key value
// if expectedsLoadKey is true.  Otherwise, it returns an error.
type testLoader struct {
	expectsLoadKey bool
	loadKeyCalled  bool
}

func (loader *testLoader) Name() string {
	return "Test"
}

func (loader *testLoader) Purpose() Purpose {
	return PurposeVerify
}

func (loader *testLoader) LoadKey() (interface{}, error) {
	loader.loadKeyCalled = true
	if loader.expectsLoadKey {
		return publicKey.key, nil
	}

	return nil, errors.New("Unexpected call to LoadKey()")
}

func TestLoaderCaching(t *testing.T) {
	const cachePeriod = time.Duration(1 * time.Hour)
	var testData = []struct {
		loader Loader
	}{
		{&oneTimeLoader{key: publicKey.key, delegate: &testLoader{expectsLoadKey: false}}},
		{&cacheLoader{cachedKey: publicKey.key, cachePeriod: cachePeriod, cacheExpiry: time.Now().Add(cachePeriod), delegate: &testLoader{expectsLoadKey: false}}},
		{&cacheLoader{cachedKey: nil, cachePeriod: cachePeriod, cacheExpiry: time.Now().Add(-cachePeriod), delegate: &testLoader{expectsLoadKey: true}}},
	}

	for _, test := range testData {
		key, err := test.loader.LoadKey()
		if err != nil {
			t.Errorf("Unexpected LoadKey() error: %v", err)
		}

		if key != publicKey.key {
			t.Errorf("Expected key %v but got %v", publicKey.key, key)
		}

		var delegate *testLoader
		switch decorator := test.loader.(type) {
		case *oneTimeLoader:
			delegate = decorator.delegate.(*testLoader)
		case *cacheLoader:
			delegate = decorator.delegate.(*testLoader)
		default:
			t.Fatalf("Unexpected Loader implementation: %v", test.loader)
		}

		if delegate.expectsLoadKey && !delegate.loadKeyCalled {
			t.Errorf("LoadKey() was not called")
		}
	}
}
