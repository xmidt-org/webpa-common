package device

import (
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	conveyEncodings = []struct {
		name     string
		encoding *base64.Encoding
	}{
		{"default", nil},
		{"base64.StdEncoding", base64.StdEncoding},
		{"base64.URLEncoding", base64.URLEncoding},
		{"base64.RawStdEncoding", base64.RawStdEncoding},
		{"base64.RawURLEncoding", base64.RawURLEncoding},
	}
)

var conveyTestData = []struct {
	convey       Convey
	expectedJSON string
}{
	{
		Convey{"foo": "bar"},
		`{"foo": "bar"}`,
	},
	{
		Convey{"device": "123", "attributes": Convey{"value": 123, "connected": true, "name": "foobar"}},
		`{"device": "123", "attributes": {"value": 123, "connected": true, "name": "foobar"}}`,
	},
}

func TestParseConveyInvalid(t *testing.T) {
	assert := assert.New(t)

	for _, conveyEncoding := range conveyEncodings {
		t.Logf("%v", conveyEncoding)
		convey, err := ParseConvey("this is not valid", conveyEncoding.encoding)
		assert.Empty(convey)
		assert.NotNil(err)
	}
}

func TestConvey(t *testing.T) {
	assert := assert.New(t)

	for _, record := range conveyTestData {
		t.Logf("%v", record)

		for _, conveyEncoding := range conveyEncodings {
			t.Logf("%v", conveyEncoding)

			encoded, err := EncodeConvey(record.convey, conveyEncoding.encoding)
			if !assert.Nil(err) {
				continue
			}

			t.Logf("encoded: %s", encoded)
			actualConvey, err := ParseConvey(encoded, conveyEncoding.encoding)
			if !assert.Nil(err) {
				continue
			}

			actualJSON, err := json.Marshal(actualConvey)
			if !assert.Nil(err) {
				continue
			}

			assert.JSONEq(record.expectedJSON, string(actualJSON))
		}
	}
}
