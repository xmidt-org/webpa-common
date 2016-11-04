package device

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

var conveyTestData = []struct {
	encoded      string
	decoded      map[string]interface{}
	valid        bool
	expectedJSON string
}{
	{
		"this is not valid",
		nil,
		false,
		"",
	},
	{
		"eyJmb28iOiAiYmFyIn0=",
		map[string]interface{}{"foo": "bar"},
		true,
		`{"foo": "bar"}`,
	},
	{
		"eyJkZXZpY2UiOiAiMTIzIiwgImF0dHJpYnV0ZXMiOiB7InZhbHVlIjogMTIzLCAiY29ubmVjdGVkIjogdHJ1ZSwgIm5hbWUiOiAiZm9vYmFyIn19",
		map[string]interface{}{"device": "123", "attributes": map[string]interface{}{"value": 123, "connected": true, "name": "foobar"}},
		true,
		`{"device": "123", "attributes": {"value": 123, "connected": true, "name": "foobar"}}`,
	},
}

func TestNewConvey(t *testing.T) {
}

func TestParseConvey(t *testing.T) {
	assert := assert.New(t)

	for _, record := range conveyTestData {
		t.Logf("%v", record)
		convey, err := ParseConvey(record.encoded)
		assert.Equal(record.valid, convey != nil)
		assert.Equal(record.valid, err == nil)

		if convey != nil {
			assert.Equal(record.encoded, convey.Encoded())
			assert.NotEmpty(convey.String())
			t.Logf(convey.String())

			actualJSON, err := json.Marshal(convey)
			if assert.Nil(err, fmt.Sprintf("%s", err)) {
				assert.NotEmpty(actualJSON)
				assert.JSONEq(record.expectedJSON, string(actualJSON))
			}
		}
	}
}
