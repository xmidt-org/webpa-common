package convey

import (
	"bytes"
	"encoding/base64"
	"io"
	"reflect"
	"strings"

	"github.com/ugorji/go/codec"
)

var (
	// conveyHandle is the internal package singleton used to parse Convey JSON
	conveyHandle codec.Handle = &codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			DecodeOptions: codec.DecodeOptions{
				MapType: reflect.TypeOf((C)(nil)),
			},
		},
		PreferFloat:     false,
		IntegerAsString: 'L',
	}
)

// C represents an arbitrary block of JSON which base64-encoded and typically
// transmitted as an HTTP header.  This is the central type used by this package.
type C map[string]interface{}

// Translator provides translation between the on-the-wire representation of a convey map
// and its runtime representation.  Instances of Translator are safe for concurrent usage.
type Translator interface {
	// ReadFrom extracts base64-encoded JSON from the supplied reader and produces a convey map.
	// Any error in either base64 decoding or JSON unmarshaling results in an error.
	ReadFrom(io.Reader) (C, error)

	// WriteTo encodes the given convey map into its on-the-wire repesentation, which is base64-encoded
	// JSON.  Any error in either base64 encoding or JSON marhsaling results in an error.
	WriteTo(io.Writer, C) error
}

// translator is the internal Translator implementation
type translator struct {
	encoding *base64.Encoding
}

// NewTranslator produces a Translator which uses the specified base64 encoding.  If
// the encoding is nil, base64.StdEncoding is used.
func NewTranslator(encoding *base64.Encoding) Translator {
	if encoding == nil {
		encoding = base64.StdEncoding
	}

	return &translator{
		encoding: encoding,
	}
}

func (t *translator) ReadFrom(source io.Reader) (C, error) {
	decoder := codec.NewDecoder(
		base64.NewDecoder(t.encoding, source),
		conveyHandle,
	)

	var convey C
	err := decoder.Decode(&convey)
	return convey, err
}

func (t *translator) WriteTo(destination io.Writer, source C) error {
	encoder := base64.NewEncoder(t.encoding, destination)
	err := codec.NewEncoder(
		encoder,
		conveyHandle,
	).Encode(source)

	encoder.Close()
	return err
}

// ReadString uses the supplied Translator to extract a C instance from an arbitrary string
func ReadString(t Translator, v string) (C, error) {
	return t.ReadFrom(
		strings.NewReader(v),
	)
}

// ReadBytes is like ReadString, but with a byte slice
func ReadBytes(t Translator, v []byte) (C, error) {
	return t.ReadFrom(
		bytes.NewReader(v),
	)
}

// WriteString uses the given Translator to turn a C into a string
func WriteString(t Translator, v C) (string, error) {
	var output bytes.Buffer
	err := t.WriteTo(&output, v)
	return output.String(), err
}

// WriteBytes uses the given Translator to turn a C into a byte slice
func WriteBytes(t Translator, v C) ([]byte, error) {
	var output bytes.Buffer
	err := t.WriteTo(&output, v)
	return output.Bytes(), err
}
