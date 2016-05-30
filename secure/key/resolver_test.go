package key

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type expectedKey struct {
	name    string
	purpose Purpose
}

func TestResolver(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		resolverFactoryJSON string
		expectedKeys        []expectedKey
		expectedCreateError bool
	}{
		{
			`[]`, []expectedKey{}, false,
		},
		{
			fmt.Sprintf(
				`[`+
					`{"name": "test", "purpose": "verify", "resource": "%s"},`+
					`{"name": "test", "purpose": "sign", "resource": "%s"}`+
					`]`,
				publicKeyPath,
				privateKeyPath,
			),
			[]expectedKey{
				{"test", PurposeVerify},
				{"test", PurposeSign},
			},
			false,
		},
		{
			fmt.Sprintf(
				`[`+
					`{"name": "duplicate", "purpose": "verify", "resource": "%s"},`+
					`{"name": "duplicate", "purpose": "verify", "resource": "%s"}`+
					`]`,
				publicKeyPath,
				privateKeyPath,
			),
			[]expectedKey{},
			true,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		factory := ResolverFactory{}
		if !assert.Nil(json.Unmarshal([]byte(record.resolverFactoryJSON), &factory)) {
			continue
		}

		resolver, err := factory.NewResolver()
		if record.expectedCreateError {
			assert.Nil(resolver)
			assert.NotNil(err)
			continue
		}

		if !assert.NotNil(resolver) || !assert.Nil(err) {
			continue
		}

		for _, expectedKey := range record.expectedKeys {
			key, err := resolver.ResolveKey(expectedKey.name, expectedKey.purpose)
			assert.NotNil(key)
			assert.Nil(err)
		}

		invalid, err := resolver.ResolveKey("this key does not exist", PurposeVerify)
		assert.Nil(invalid)
		assert.NotNil(err)
	}
}
