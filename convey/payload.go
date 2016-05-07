package convey

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
)

// Payload represents the decoded payload of the convey header
type Payload map[string]interface{}

// DecodeBase64 assumes that the value parameter is Base64-encoded JSON
func (payload *Payload) DecodeBase64(encoding *base64.Encoding, value string) error {
	input := bytes.NewBufferString(value)
	decoder := base64.NewDecoder(encoding, input)
	decodedValue, err := ioutil.ReadAll(decoder)
	if err != nil {
		return err
	}

	return json.Unmarshal(decodedValue, payload)
}

// EncodeBase64 returns the Base64-encoded JSON representation of this payload.
// This method is the inverse of DecodeBase64, but will not necessarily yield the
// same value.  The act of unmarshalling followed by marshalling will most often
// result in the same JSON structure but with different field ordering.
func (payload *Payload) EncodeBase64(encoding *base64.Encoding) (encoded string, err error) {
	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return
	}

	output := &bytes.Buffer{}
	encoder := base64.NewEncoder(encoding, output)
	if _, err = encoder.Write(payloadJson); err != nil {
		return
	}

	encoder.Close()
	encoded = output.String()
	return
}
