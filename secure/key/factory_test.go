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

			expectCreateError := !record.valid && cachePeriod == store.CachePeriodForever
			expectLoadError := !record.valid && cachePeriod != store.CachePeriodForever

			// the value may not be cached, so the error will be upfront
			key, err := factory.NewKey()
			if expectCreateError {
				assert.Nil(key)
				assert.NotNil(err)
				continue
			}

			if !assert.NotNil(key) || !assert.Nil(err) {
				continue
			}

			value, err := key.Load()
			if expectLoadError {
				assert.Nil(value)
				assert.NotNil(err)
				continue
			}

			assert.NotNil(value)
			assert.Nil(err)

			if factory.Purpose == PurposeVerify || factory.Purpose == PurposeDecrypt {
				_, ok := value.(*rsa.PublicKey)
				assert.True(ok)
			} else {
				_, ok := value.(*rsa.PrivateKey)
				assert.True(ok)
			}
		}
	}
}
