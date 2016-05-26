package key

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestNewLoaderFromJSON(t *testing.T) {
	minimumCacheExpiry := time.Now().Add(time.Duration(24 * time.Hour))

	var testData = []struct {
		JSON           string
		expectedLoader Loader
		expectedKey    interface{}
	}{
		// verify with a public key on the local filesystem
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "never"}`, publicKey.fileURL),
			&urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.fileURL},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "forever"}`, publicKey.fileURL),
			&oneTimeLoader{key: publicKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.fileURL}},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s"}`, publicKey.fileURL),
			&oneTimeLoader{key: publicKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.fileURL}},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "24h"}`, publicKey.fileURL),
			&cacheLoader{cachedKey: publicKey.key, cachePeriod: time.Duration(24 * time.Hour), cacheExpiry: minimumCacheExpiry, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.fileURL}},
			publicKey.key,
		},
		// verify with a public key on an HTTP server
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "never"}`, publicKey.httpURL()),
			&urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.httpURL()},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "forever"}`, publicKey.httpURL()),
			&oneTimeLoader{key: publicKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.httpURL()}},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s"}`, publicKey.httpURL()),
			&oneTimeLoader{key: publicKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.httpURL()}},
			publicKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "verify", "url": "%s", "cachePeriod": "24h"}`, publicKey.httpURL()),
			&cacheLoader{cachedKey: publicKey.key, cachePeriod: time.Duration(24 * time.Hour), cacheExpiry: minimumCacheExpiry, delegate: &urlLoader{name: "Test", purpose: PurposeVerify, url: *publicKey.httpURL()}},
			publicKey.key,
		},
		// sign with a private key on the local filesystem
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "never"}`, privateKey.fileURL),
			&urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.fileURL},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "forever"}`, privateKey.fileURL),
			&oneTimeLoader{key: privateKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.fileURL}},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s"}`, privateKey.fileURL),
			&oneTimeLoader{key: privateKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.fileURL}},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "24h"}`, privateKey.fileURL),
			&cacheLoader{cachedKey: privateKey.key, cachePeriod: time.Duration(24 * time.Hour), cacheExpiry: minimumCacheExpiry, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.fileURL}},
			privateKey.key,
		},
		// sign with a private key on an HTTP server
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "never"}`, privateKey.httpURL()),
			&urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.httpURL()},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "forever"}`, privateKey.httpURL()),
			&oneTimeLoader{key: privateKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.httpURL()}},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s"}`, privateKey.httpURL()),
			&oneTimeLoader{key: privateKey.key, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.httpURL()}},
			privateKey.key,
		},
		{
			fmt.Sprintf(`{"name": "Test", "purpose": "sign", "url": "%s", "cachePeriod": "24h"}`, privateKey.httpURL()),
			&cacheLoader{cachedKey: privateKey.key, cachePeriod: time.Duration(24 * time.Hour), cacheExpiry: minimumCacheExpiry, delegate: &urlLoader{name: "Test", purpose: PurposeSign, url: *privateKey.httpURL()}},
			privateKey.key,
		},
	}

	for _, test := range testData {
		builder := &LoaderBuilder{}
		err := json.Unmarshal([]byte(test.JSON), builder)
		if err != nil {
			t.Fatalf("Failed to parse JSON %s: %v", test.JSON, err)
		}

		actualLoader, err := builder.NewLoader()
		if err != nil {
			t.Fatalf("Failed to create new loader: %v", err)
		}

		if err := loaderEqual(actualLoader, test.expectedLoader); err != nil {
			t.Errorf("%v", err)
		}

		actualKey, err := actualLoader.LoadKey()
		if err != nil {
			t.Errorf("LoadKey() failed: %v", err)
		}

		if !reflect.DeepEqual(actualKey, test.expectedKey) {
			t.Errorf("Expected key %v but got %v", test.expectedKey, actualKey)
		}
	}
}
