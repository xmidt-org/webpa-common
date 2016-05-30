package key

import (
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/store"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFactory(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		purpose  Purpose
		resource string
		valid    bool
	}{
		{
			PurposeVerify, publicKeyPath, true,
		},
		{
			PurposeDecrypt, publicKeyPath, true,
		},
		{
			PurposeVerify, privateKeyPath, false,
		},
		{
			PurposeDecrypt, privateKeyPath, false,
		},
		{
			PurposeSign, privateKeyPath, true,
		},
		{
			PurposeEncrypt, privateKeyPath, true,
		},
		{
			PurposeSign, publicKeyPath, false,
		},
		{
			PurposeEncrypt, publicKeyPath, false,
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		for _, cachePeriod := range []store.CachePeriod{store.CachePeriodForever, store.CachePeriodNever, store.CachePeriod(100000)} {
			factoryJSON := fmt.Sprintf(
				`{"name": "test", "purpose": "%s", "resource": "%s", "cachePeriod": "%s"}`,
				record.purpose,
				record.resource,
				cachePeriod,
			)

			t.Logf("factory JSON: %s", factoryJSON)
			factory := Factory{}
			if !assert.Nil(json.Unmarshal([]byte(factoryJSON), &factory)) {
				continue
			}

			if key, err := factory.NewKey(); err != nil {
				// the value may not be cached, so the error will be upfront
				assert.Nil(key)
				assert.False(record.valid)
			} else if assert.NotNil(key) {
				if value, err := key.Load(); err != nil {
					// this will be the actual, uncached key
					assert.Nil(value)
					assert.False(record.valid)
				} else if assert.NotNil(value) {
					switch value.(type) {
					case *rsa.PublicKey:
						assert.True(factory.Purpose == PurposeVerify || factory.Purpose == PurposeDecrypt)
					case *rsa.PrivateKey:
						assert.True(factory.Purpose == PurposeSign || factory.Purpose == PurposeEncrypt)
					default:
						t.Error("Invalid key type")
					}
				}
			}
		}
	}
}
