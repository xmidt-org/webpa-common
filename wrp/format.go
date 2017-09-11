package wrp

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ugorji/go/codec"
)

//go:generate stringer -type=Format

// Format indicates which format is desired.
// The zero value indicates Msgpack, which means by default other
// infrastructure can assume msgpack-formatted data.
type Format int

const (
	Msgpack Format = iota
	JSON
	lastFormat
)

var (
	jsonHandle = codec.JsonHandle{
		BasicHandle: codec.BasicHandle{
			TypeInfos: codec.NewTypeInfos([]string{"wrp"}),
		},
		IntegerAsString: 'L',
	}

	msgpackHandle = codec.MsgpackHandle{
		BasicHandle: codec.BasicHandle{
			TypeInfos: codec.NewTypeInfos([]string{"wrp"}),
		},
	}
)

// ContentType returns the MIME type associated with this format
func (f Format) ContentType() string {
	switch f {
	case Msgpack:
		return "application/msgpack"
	case JSON:
		return "application/json"
	default:
		return "application/octet-stream"
	}
}

// FormatFromContentType examines the Content-Type value and returns
// the appropriate Format.  This function returns an error if the given
// Content-Type did not map to a WRP format.
func FormatFromContentType(contentType string) (Format, error) {
	if strings.Contains(contentType, "json") {
		return JSON, nil
	} else if strings.Contains(contentType, "msgpack") {
		return Msgpack, nil
	}

	return Format(-1), fmt.Errorf("Invalid WRP content type: %s", contentType)
}

// handle looks up the appropriate codec.Handle for this format constant.
// This method panics if the format is not a valid value.
func (f Format) handle() codec.Handle {
	switch f {
	case Msgpack:
		return &msgpackHandle
	case JSON:
		return &jsonHandle
	}

	panic(fmt.Errorf("Invalid format constant: %d", f))
}

// EncodeListener can be implemented on any type passed to an Encoder in order
// to get notified when an encoding happens.  This interface is useful to set
// mandatory fields, such as message type.
type EncodeListener interface {
	BeforeEncode() error
}

// Encoder represents the underlying ugorji behavior that WRP supports
type Encoder interface {
	Encode(interface{}) error
	Reset(io.Writer)
	ResetBytes(*[]byte)
}

// encoderDecorator wraps a ugorji Encoder and implements the wrp.Encoder interface.
type encoderDecorator struct {
	*codec.Encoder
}

// Encode checks to see if value implements EncoderTo and if it does, uses the
// value.EncodeTo() method.  Otherwise, the value is passed as is to the decorated
// ugorji Encoder.
func (ed *encoderDecorator) Encode(value interface{}) error {
	if listener, ok := value.(EncodeListener); ok {
		if err := listener.BeforeEncode(); err != nil {
			return err
		}
	}

	return ed.Encoder.Encode(value)
}

// Decoder represents the underlying ugorji behavior that WRP supports
type Decoder interface {
	Decode(interface{}) error
	Reset(io.Reader)
	ResetBytes([]byte)
}

// NewEncoder produces a ugorji Encoder using the appropriate WRP configuration
// for the given format
func NewEncoder(output io.Writer, f Format) Encoder {
	return &encoderDecorator{
		codec.NewEncoder(output, f.handle()),
	}
}

// NewEncoderBytes produces a ugorji Encoder using the appropriate WRP configuration
// for the given format
func NewEncoderBytes(output *[]byte, f Format) Encoder {
	return &encoderDecorator{
		codec.NewEncoderBytes(output, f.handle()),
	}
}

// NewDecoder produces a ugorji Decoder using the appropriate WRP configuration
// for the given format
func NewDecoder(input io.Reader, f Format) Decoder {
	return codec.NewDecoder(input, f.handle())
}

// NewDecoderBytes produces a ugorji Decoder using the appropriate WRP configuration
// for the given format
func NewDecoderBytes(input []byte, f Format) Decoder {
	return codec.NewDecoderBytes(input, f.handle())
}

// TranscodeMessage converts a WRP message of any type from one format into another,
// e.g. from JSON into Msgpack.  The intermediate, generic Message used to hold decoded
// values is returned in addition to any error.  If a decode error occurs, this function
// will not perform the encoding step.
func TranscodeMessage(target Encoder, source Decoder) (msg *Message, err error) {
	msg = new(Message)
	if err = source.Decode(msg); err == nil {
		err = target.Encode(msg)
	}

	return
}

// MustEncode is a convenience function that attempts to encode a given message.  A panic
// is raised on any error.  This function is handy for package initialization.
func MustEncode(message interface{}, f Format) []byte {
	var (
		output  bytes.Buffer
		encoder = NewEncoder(&output, f)
	)

	if err := encoder.Encode(message); err != nil {
		panic(err)
	}

	return output.Bytes()
}
