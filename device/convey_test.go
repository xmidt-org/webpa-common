package device

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseConvey(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		value        string
		expectsError bool
		expectedJSON string
	}{
		{
			"this is not valid",
			true,
			"",
		},
		{
			"eyJmb28iOiAiYmFyIn0=",
			false,
			`{"foo": "bar"}`,
		},
		{
			"eyJkZXZpY2UiOiAiMTIzIiwgImF0dHJpYnV0ZXMiOiB7InZhbHVlIjogMTIzLCAiY29ubmVjdGVkIjogdHJ1ZSwgIm5hbWUiOiAiZm9vYmFyIn19",
			false,
			`{"device": "123", "attributes": {"value": 123, "connected": true, "name": "foobar"}}`,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		convey, err := ParseConvey(record.value)
		assert.Equal(record.expectsError, convey == nil)
		assert.Equal(record.expectsError, err != nil)

		if convey != nil {
			assert.Equal(record.value, convey.Encoded())
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
