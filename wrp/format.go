package wrp

import (
	"github.com/ugorji/go/codec"
	"io"
)

// Format indicates which format is desired
type Format int

const (
	JSON Format = iota
	Msgpack
)

var (
	// handles contains the canonical codec.Handle supported by WRP, in order
	// of Format constants
	handles = []codec.Handle{
		&codec.JsonHandle{
			BasicHandle: codec.BasicHandle{
				TypeInfos: codec.NewTypeInfos([]string{"json"}),
			},
			IntegerAsString: 'L',
		},
		&codec.MsgpackHandle{
			BasicHandle: codec.BasicHandle{
				TypeInfos: codec.NewTypeInfos([]string{"msgpack"}),
			},
		},
	}
)

// handle looks up the appropriate codec.Handle for this format constant.
// This method returns nil if the format value is invalid.
func (f Format) handle() codec.Handle {
	if int(f) < len(handles) {
		return handles[f]
	}

	return nil
}

// Encoder represents the underlying ugorji behavior that WRP supports
type Encoder interface {
	Encode(interface{}) error
	Reset(io.Writer)
	ResetBytes(*[]byte)
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
	return codec.NewEncoder(output, f.handle())
}

// NewEncoderBytes produces a ugorji Encoder using the appropriate WRP configuration
// for the given format
func NewEncoderBytes(output *[]byte, f Format) Encoder {
	return codec.NewEncoderBytes(output, f.handle())
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
