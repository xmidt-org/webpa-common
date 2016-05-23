package convey

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

// payloadTestData can be generated with:
// https://play.golang.org/p/T0ymv6RE4l
var payloadTestData = []struct {
	encoding *base64.Encoding
	value    string
	expected Payload
}{
	{
		encoding: base64.StdEncoding,
		value:    "eyAicGFyYW1ldGVycyI6IFsgeyAibmFtZSI6ICJEZXZpY2UuRGV2aWNlSW5mby5XZWJwYS5YX0NPTUNBU1QtQ09NX0NJRCIsICJ2YWx1ZSI6ICIwIiwgImRhdGFUeXBlIjogMCB9LCB7ICJuYW1lIjogIkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ01DIiwgInZhbHVlIjogIjI2OSIsICJkYXRhVHlwZSI6IDIgfSBdIH0K",
		expected: Payload{
			"parameters": []Payload{
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CID",
					"value":    "0",
					"dataType": 0,
				},
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CMC",
					"value":    "269",
					"dataType": 2,
				},
			},
		},
	},
	{
		encoding: base64.RawURLEncoding,
		value:    "eyJwYXJhbWV0ZXJzIjpbeyJkYXRhVHlwZSI6MCwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ0lEIiwidmFsdWUiOiIwIn0seyJkYXRhVHlwZSI6MiwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ01DIiwidmFsdWUiOiIyNjkifV19",
		expected: Payload{
			"parameters": []Payload{
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CID",
					"value":    "0",
					"dataType": 0,
				},
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CMC",
					"value":    "269",
					"dataType": 2,
				},
			},
		},
	},
	{
		encoding: base64.URLEncoding,
		value:    "eyJwYXJhbWV0ZXJzIjpbeyJkYXRhVHlwZSI6MCwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ0lEIiwidmFsdWUiOiIwIn0seyJkYXRhVHlwZSI6MiwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ01DIiwidmFsdWUiOiIyNjkifV19",
		expected: Payload{
			"parameters": []Payload{
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CID",
					"value":    "0",
					"dataType": 0,
				},
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CMC",
					"value":    "269",
					"dataType": 2,
				},
			},
		},
	},
	{
		encoding: base64.RawStdEncoding,
		value:    "eyJwYXJhbWV0ZXJzIjpbeyJkYXRhVHlwZSI6MCwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ0lEIiwidmFsdWUiOiIwIn0seyJkYXRhVHlwZSI6MiwibmFtZSI6IkRldmljZS5EZXZpY2VJbmZvLldlYnBhLlhfQ09NQ0FTVC1DT01fQ01DIiwidmFsdWUiOiIyNjkifV19",
		expected: Payload{
			"parameters": []Payload{
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CID",
					"value":    "0",
					"dataType": 0,
				},
				Payload{
					"name":     "Device.DeviceInfo.Webpa.X_COMCAST-COM_CMC",
					"value":    "269",
					"dataType": 2,
				},
			},
		},
	},
}

func TestDecodeBase64(t *testing.T) {
	assertions := assert.New(t)
	for _, record := range payloadTestData {
		var actual Payload
		if err := actual.DecodeBase64(record.encoding, record.value); err != nil {
			t.Errorf("DecodeBase64 failed: %v", err)
		}

		expectedJson, err := json.Marshal(record.expected)
		if err != nil {
			t.Fatalf("Unable to marshal expected JSON: %v", err)
		}

		actualJson, err := json.Marshal(actual)
		if err != nil {
			t.Fatalf("Unable to marshal actual JSON: %v", err)
		}

		assertions.JSONEq(string(expectedJson), string(actualJson))
	}
}

func TestEncodeBase64(t *testing.T) {
	assertions := assert.New(t)
	for _, record := range payloadTestData {
		// perform the reverse test: use the expected as our actual JSON
		actualEncoded, err := record.expected.EncodeBase64(record.encoding)
		if err != nil {
			t.Fatalf("Unable to encode: %v", err)
		}

		// decode the actual value, to get JSON that was can compare against
		actualInput := bytes.NewBufferString(actualEncoded)
		actualDecoder := base64.NewDecoder(record.encoding, actualInput)
		actualJson, err := ioutil.ReadAll(actualDecoder)
		if err != nil {
			t.Fatalf("Unable to decode the output of EncodeBase64: %v", err)
		}

		expectedInput := bytes.NewBufferString(record.value)
		expectedDecoder := base64.NewDecoder(record.encoding, expectedInput)
		expectedJson, err := ioutil.ReadAll(expectedDecoder)
		if err != nil {
			t.Fatalf("Unable to decode expected value: %v", err)
		}

		assertions.JSONEq(string(expectedJson), string(actualJson))
	}
}

func TestParsePayload(t *testing.T) {
	assertions := assert.New(t)
	for _, record := range payloadTestData {
		actualPayload, err := ParsePayload(record.encoding, record.value)
		if err != nil {
			t.Fatalf("ParsePayload failed: %v", err)
		}

		actualJson, err := json.Marshal(actualPayload)
		if err != nil {
			t.Fatalf("Failed to marshal actual payload: %v", err)
		}

		expectedJson, err := json.Marshal(record.expected)
		if err != nil {
			t.Fatalf("Failed to marshal expected JSON: %v", err)
		}

		assertions.JSONEq(string(expectedJson), string(actualJson))
	}
}
