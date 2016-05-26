package key

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestNewResolverFromJSON(t *testing.T) {
	// a key name that doesn't occur in the test data
	const nosuchKeyName = "nosuch"

	var validTestData = []struct {
		JSON string

		// Use the internal key as a storage type
		expectedKeys map[loaderKey]interface{}
	}{
		// empty configuration is legal and just results in a Resolver that resolve nothing
		{
			"[]",
			map[loaderKey]interface{}{},
		},
		// A single public key on the file system
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "never"}`, publicKey.fileURL) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// A single cached public key on the file system
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "24h"}`, publicKey.fileURL) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// A single onetime public key on the file system
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "forever"}`, publicKey.fileURL) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// A single public key on an HTTP server
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "never"}`, publicKey.httpURL()) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// A single cached public key on an HTTP server
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "16h"}`, publicKey.httpURL()) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// A single onetime public key on an HTTP server
		{
			`[` +
				fmt.Sprintf(`{"name": "Single", "purpose": "verify", "url": "%s", "cachePeriod": "forever"}`, publicKey.httpURL()) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Single", PurposeVerify}: publicKey.key},
		},
		// Two keys on the filesystem
		{
			`[` +
				fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s", "cachePeriod": "default"},`, publicKey.fileURL) +
				fmt.Sprintf(`{"name": "Private", "purpose": "sign", "url": "%s", "cachePeriod": "never"}`, privateKey.fileURL) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Public", PurposeVerify}: publicKey.key, loaderKey{"Private", PurposeSign}: privateKey.key},
		},
		// Two cached keys on the filesystem
		{
			`[` +
				fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s", "cachePeriod": "forever"},`, publicKey.fileURL) +
				fmt.Sprintf(`{"name": "Private", "purpose": "sign", "url": "%s", "cachePeriod": "default"}`, privateKey.fileURL) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Public", PurposeVerify}: publicKey.key, loaderKey{"Private", PurposeSign}: privateKey.key},
		},
		// Two keys on an HTTP server
		{
			`[` +
				fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s", "cachePeriod": "never"},`, publicKey.httpURL()) +
				fmt.Sprintf(`{"name": "Private", "purpose": "sign", "url": "%s", "cachePeriod": "forever"}`, privateKey.httpURL()) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Public", PurposeVerify}: publicKey.key, loaderKey{"Private", PurposeSign}: privateKey.key},
		},
		// Two cached keys on an HTTP server
		{
			`[` +
				fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s"},`, publicKey.httpURL()) +
				fmt.Sprintf(`{"name": "Private", "purpose": "sign", "url": "%s", "cachePeriod": "3h45m13s"}`, privateKey.httpURL()) +
				`]`,
			map[loaderKey]interface{}{loaderKey{"Public", PurposeVerify}: publicKey.key, loaderKey{"Private", PurposeSign}: privateKey.key},
		},
	}

	var invalidTestData = []string{
		`[` +
			fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s"},`, publicKey.httpURL()) +
			fmt.Sprintf(`{"name": "Public", "purpose": "verify", "url": "%s"}`, publicKey.httpURL()) +
			`]`,
	}

	for _, test := range validTestData {
		var builder ResolverBuilder
		err := json.Unmarshal([]byte(test.JSON), &builder)
		if err != nil {
			t.Fatalf("Failed to parse JSON %s: %v", test.JSON, err)
		}

		resolver, err := builder.NewResolver()
		if err != nil {
			t.Fatalf("Unable to build resolver: %v", err)
		} else if resolver == nil {
			t.Fatalf("Unexpected nil resolver")
		}

		key, err := resolver.ResolveKey(nosuchKeyName, PurposeVerify)
		if key != nil || err != nil {
			t.Errorf("ResolveKey() must return nils when it doesn't not find a key")
		}

		for loaderKey, expectedKey := range test.expectedKeys {
			key, err := resolver.ResolveKey(loaderKey.name, loaderKey.purpose)
			if err != nil {
				t.Errorf("Failed to load key %s with purpose %d: %v", loaderKey.name, loaderKey.purpose, err)
			}

			if !reflect.DeepEqual(key, expectedKey) {
				t.Errorf("Key %s with purpose %d: expected %v but got %v", loaderKey.name, loaderKey.purpose, expectedKey, key)
			}
		}
	}

	for _, invalidConfiguration := range invalidTestData {
		var builder ResolverBuilder
		err := json.Unmarshal([]byte(invalidConfiguration), &builder)
		if err != nil {
			t.Fatalf("Failed to parse JSON %s: %v", invalidConfiguration, err)
		}

		resolver, err := builder.NewResolver()
		if resolver != nil || err == nil {
			t.Errorf("Expected NewResolver() to fail with configuration: %s", invalidConfiguration)
		}
	}
}
