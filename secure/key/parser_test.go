package key

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func TestDefaultParser(t *testing.T) {
	assert := assert.New(t)

	var testData = []struct {
		keyFilePath   string
		purpose       Purpose
		expectPrivate bool
	}{
		{publicKeyFilePath, PurposeVerify, false},
		{privateKeyFilePath, PurposeEncrypt, true},
		{privateKeyFilePath, PurposeSign, true},
		{publicKeyFilePath, PurposeDecrypt, false},
	}

	for _, record := range testData {
		t.Logf("%v", record)

		data, err := ioutil.ReadFile(record.keyFilePath)
		if !assert.Nil(err) {
			continue
		}

		pair, err := DefaultParser.ParseKey(record.purpose, data)
		if !assert.NotNil(pair) || !assert.Nil(err) {
			continue
		}

		assert.NotNil(pair.Public())
		assert.Equal(record.expectPrivate, pair.HasPrivate())
		assert.Equal(record.expectPrivate, pair.Private() != nil)
		assert.Equal(record.purpose, pair.Purpose())
	}
}
