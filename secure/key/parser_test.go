package key

import (
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

func makeNonKeyPEMBlock() []byte {
	block := pem.Block{
		Type:  "NOT A KEY",
		Bytes: []byte{1, 2, 3, 4, 5},
	}

	return pem.EncodeToMemory(&block)
}

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
		if !assert.Nil(err) && !assert.NotNil(pair) {
			continue
		}

		assert.NotNil(pair.Public())
		assert.Equal(record.expectPrivate, pair.HasPrivate())
		assert.Equal(record.expectPrivate, pair.Private() != nil)
		assert.Equal(record.purpose, pair.Purpose())
	}
}

func TestDefaultParserString(t *testing.T) {
	assert := assert.New(t)
	assert.NotEmpty(fmt.Sprintf("%s", DefaultParser))
}

func TestDefaultParserNoPEM(t *testing.T) {
	assert := assert.New(t)

	notPEM := []byte{9, 9, 9}
	pair, err := DefaultParser.ParseKey(PurposeVerify, notPEM)
	assert.Nil(pair)
	assert.Equal(ErrorPEMRequired, err)
}

func TestDefaultParserInvalidPublicKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeVerify, PurposeDecrypt} {
		t.Logf("%s", purpose)
		pair, err := DefaultParser.ParseKey(purpose, makeNonKeyPEMBlock())
		assert.Nil(pair)
		assert.NotNil(err)
	}
}

func TestDefaultParserInvalidPrivateKey(t *testing.T) {
	assert := assert.New(t)

	for _, purpose := range []Purpose{PurposeSign, PurposeEncrypt} {
		t.Logf("%s", purpose)
		pair, err := DefaultParser.ParseKey(purpose, makeNonKeyPEMBlock())
		assert.Nil(pair)
		assert.Equal(ErrorUnsupportedPrivateKeyFormat, err)
	}
}
