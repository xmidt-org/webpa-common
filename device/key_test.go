package device

import (
	"bytes"
	"encoding/base64"
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"testing"
)

var (
	uuidEncodings = []struct {
		actualEncoding   *base64.Encoding
		expectedEncoding *base64.Encoding
	}{
		{nil, base64.RawURLEncoding}, // the default
		{base64.StdEncoding, base64.StdEncoding},
		{base64.URLEncoding, base64.URLEncoding},
		{base64.RawStdEncoding, base64.RawStdEncoding},
		{base64.RawURLEncoding, base64.RawURLEncoding},
	}
)

func uuidKeyReadMatcher(b []byte) bool {
	return len(b) == 16
}

func assertKeyEncoding(assert *assert.Assertions, key Key, expectedEncoding *base64.Encoding) {
	decoder := base64.NewDecoder(expectedEncoding, bytes.NewBufferString(string(key)))
	decoded, err := ioutil.ReadAll(decoder)
	if !assert.NotEmpty(decoded) || !assert.Nil(err) {
		return
	}

	// we modify the input bytes to conform to type 4 UUIDs, so all that's important is
	// that the decoded byte array has the correct length
	assert.Equal(16, len(decoded))
}

func TestUUIDKeyFunc(t *testing.T) {
	assert := assert.New(t)
	const randomBytes = "FEDCBA9876543210"

	copyExpectedKey := func(arguments mock.Arguments) {
		copy(arguments.Get(0).([]byte), []byte(randomBytes))
	}

	for _, record := range uuidEncodings {
		t.Logf("%v", record)

		mockRandom := new(mockReader)
		mockRandom.
			On("Read", mock.MatchedBy(uuidKeyReadMatcher)).
			Run(copyExpectedKey).
			Return(16, nil).
			Once()

		keyFunc := UUIDKeyFunc(mockRandom, record.actualEncoding)
		if !assert.NotNil(keyFunc) {
			continue
		}

		key, err := keyFunc(ID("expected"), nil, nil)
		if !assert.NotEmpty(key) || !assert.Nil(err) {
			continue
		}

		assertKeyEncoding(assert, key, record.expectedEncoding)
		mockRandom.AssertExpectations(t)
	}
}

func TestUUIDKeyFuncSourceError(t *testing.T) {
	assert := assert.New(t)
	sourceError := errors.New("expected error")

	for _, record := range uuidEncodings {
		t.Logf("%v", record)

		mockRandom := new(mockReader)
		mockRandom.
			On("Read", mock.MatchedBy(uuidKeyReadMatcher)).
			Return(0, sourceError).
			Once()

		keyFunc := UUIDKeyFunc(mockRandom, record.actualEncoding)
		if !assert.NotNil(keyFunc) {
			continue
		}

		key, err := keyFunc(ID("expected"), nil, nil)
		assert.Equal(invalidKey, key)
		assert.Equal(sourceError, err)

		mockRandom.AssertExpectations(t)
	}
}

func TestUUIDKeyFuncDefaultSource(t *testing.T) {
	assert := assert.New(t)

	for _, record := range uuidEncodings {
		t.Logf("%v", record)

		keyFunc := UUIDKeyFunc(nil, record.actualEncoding)
		if !assert.NotNil(keyFunc) {
			continue
		}

		key, err := keyFunc(ID("expected"), nil, nil)
		if !assert.NotEmpty(key) || !assert.Nil(err) {
			continue
		}

		assertKeyEncoding(assert, key, record.expectedEncoding)
	}
}
