package convey

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/ugorji/go/codec"
	"io/ioutil"
	"net/http"
)

const (
	// DefaultPayloadHeaderName is used when no header is configured
	DefaultPayloadHeaderName = "X-Webpa-Convey"
)

var (
	// codecHandle is the ugorji JSON handle for creating encoders and decoders
	codecHandle codec.Handle = &codec.JsonHandle{
		IntegerAsString: 'L',
	}
)

// Payload represents the decoded payload of the convey header.  Payloads are encoded
// as base64 JSON strings.
type Payload map[string]interface{}

// Factory supplies various ways to obtain Payload instances.  A Factory is always
// safe for concurrent usage.
//
// Typically, applications will create (1) Factory for repeated use in an http.Handler.
type Factory interface {
	// FromValue accepts an encoded value and returns the resulting Payload
	FromValue(string) (Payload, error)

	// FromRequest examines an HTTP request to find a Payload.  If no header was found
	// in the request, an error is returned with a nil Payload.
	FromRequest(*http.Request) (Payload, error)
}

// NewFactory creates a Factory for Payload objects using the supplied configuration.
// If headerName is empty, DefaultPayloadHeaderName is sued.  If encoding is nil,
// then base64.StdEncoding is used.
func NewFactory(headerName string, encoding *base64.Encoding) Factory {
	if len(headerName) == 0 {
		headerName = DefaultPayloadHeaderName
	}

	if encoding == nil {
		encoding = base64.StdEncoding
	}

	return &factory{
		headerName: headerName,
		missingHeaderError: httperror.New(
			fmt.Sprintf("Missing header: %s", headerName),
			http.StatusBadRequest,
			nil,
		),
		encoding: encoding,
	}
}

type factory struct {
	headerName         string
	missingHeaderError error
	encoding           *base64.Encoding
}

func (f *factory) FromValue(value string) (payload Payload, err error) {
	input := bytes.NewBufferString(value)
	decoder := codec.NewDecoder(
		base64.NewDecoder(f.encoding, input),
		codecHandle,
	)

	err = decoder.Decode(&payload)
	return
}

func (f *factory) FromRequest(request *http.Request) (Payload, error) {
	value := request.Header.Get(f.headerName)
	if len(value) == 0 {
		return nil, f.missingHeaderError
	}

	return f.FromValue(value)
}

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

	if err = encoder.Close(); err != nil {
		return
	}

	encoded = output.String()
	return
}

// ParsePayload leverages DecodeBase64 to produce a fully initialized Payload.
// The value parameter is expected to be Base64-encoded JSON, exactly as would come
// from a convey header.
func ParsePayload(encoding *base64.Encoding, value string) (payload Payload, err error) {
	err = payload.DecodeBase64(encoding, value)
	return
}
