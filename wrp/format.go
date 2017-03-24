package wrp

import (
	"fmt"
	"github.com/ugorji/go/codec"
	"io"
)

// Format indicates which format is desired
type Format int

const (
	JSON Format = iota
	Msgpack

	InvalidFormatString = "!!INVALID!!"
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

func (f Format) String() string {
	switch f {
	case JSON:
		return "JSON"
	case Msgpack:
		return "Msgpack"
	}

	return InvalidFormatString
}

// handle looks up the appropriate codec.Handle for this format constant.
// This method panics if the format is not a valid value.
func (f Format) handle() codec.Handle {
	switch f {
	case JSON:
		return &jsonHandle
	case Msgpack:
		return &msgpackHandle
	}

	panic(fmt.Errorf("Invalid format constant: %d", f))
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
	if encoderTo, ok := value.(EncoderTo); ok {
		return encoderTo.EncodeTo(ed.Encoder)
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
