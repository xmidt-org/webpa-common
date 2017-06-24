package device

import (
	"bytes"
	"encoding/base64"
	"reflect"

	"github.com/ugorji/go/codec"
)

var (
	conveyHandle codec.Handle = &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				MapType: reflect.TypeOf(map[string]interface{}(nil)),
			},
		},
		IntegerAsString: 'L',
	}
)

// Convey represents an arbitrary block of JSON that should be transmitted
// in HTTP requests related to devices.  It is typically sent via a header
// as base64-encoded JSON.
type Convey map[string]interface{}

// ParseConvey decodes a value using the supplied encoding and then unmarshals
// the result as a Convey map.  If encoding is nil, base64.StdEncoding is used.
func ParseConvey(value string, encoding *base64.Encoding) (Convey, error) {
	if encoding == nil {
		encoding = base64.StdEncoding
	}

	input := bytes.NewBufferString(value)
	decoder := codec.NewDecoder(
		base64.NewDecoder(encoding, input),
		conveyHandle,
	)

	var convey Convey
	if err := decoder.Decode(&convey); err != nil {
		return nil, err
	}

	return convey, nil
}

// EncodeConvey transforms a Convey map into its on-the-wire representation,
// using the supplied encoding.  If encoding == nil, base64.StdEncoding is used.
func EncodeConvey(convey Convey, encoding *base64.Encoding) (string, error) {
	if encoding == nil {
		encoding = base64.StdEncoding
	}

	output := new(bytes.Buffer)
	base64 := base64.NewEncoder(encoding, output)
	encoder := codec.NewEncoder(base64, conveyHandle)
	if err := encoder.Encode(convey); err != nil {
		return "", err
	}

	base64.Close()
	return output.String(), nil
}

// MustEncodeConvey works as EncodeConvey, except that this function panics if
// there is any error.
func MustEncodeConvey(convey Convey, encoding *base64.Encoding) string {
	if encodedConvey, err := EncodeConvey(convey, encoding); err != nil {
		panic(err)
	} else {
		return encodedConvey
	}
}
