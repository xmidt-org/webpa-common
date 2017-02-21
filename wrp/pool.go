package wrp

import (
	"io"
	"sync"
)

const (
	DefaultInitialBufferSize = 200
)

// EncoderPool represents a pool of Encoder objects that can be used as is
// encode WRP messages.
type EncoderPool struct {
	pool              sync.Pool
	initialBufferSize int
}

// NewEncoderPool returns an EncoderPool for a given format.  The initialBufferSize is
// used when encoding to byte arrays.  If this value is nonpositive, DefaultInitialBufferSize
// is used instead.
func NewEncoderPool(initialBufferSize int, f Format) *EncoderPool {
	if initialBufferSize < 1 {
		initialBufferSize = DefaultInitialBufferSize
	}

	return &EncoderPool{
		pool: sync.Pool{
			New: func() interface{} { return NewEncoder(nil, f) },
		},
		initialBufferSize: initialBufferSize,
	}
}

// Encode uses an Encoder from the pool to encode the source into the destination
func (ep *EncoderPool) Encode(destination io.Writer, source interface{}) error {
	encoder := ep.pool.Get().(Encoder)
	defer ep.pool.Put(encoder)

	encoder.Reset(destination)
	return encoder.Encode(source)
}

// EncodeBytes uses an encoder from the pool to encode the source into a byte array.
// This method attempts to minimize memory allocation overhead by allocating the initialBufferSize
// specified in NewEncoderPool.
func (ep *EncoderPool) EncodeBytes(source interface{}) (data []byte, err error) {
	data = make([]byte, ep.initialBufferSize)
	encoder := ep.pool.Get().(Encoder)
	defer ep.pool.Put(encoder)

	encoder.ResetBytes(&data)
	err = encoder.Encode(source)
	return
}

// DecoderPool is a pool of Decoder instances for a specific format
type DecoderPool struct {
	pool sync.Pool
}

// NewDecoderPool returns a DecoderPool that works with a given Format
func NewDecoderPool(f Format) *DecoderPool {
	return &DecoderPool{
		pool: sync.Pool{
			New: func() interface{} { return NewDecoder(nil, f) },
		},
	}
}

// Decode unmarshals data from the source onto the destination instance, which is
// normally a pointer to some struct (such as *Message).
func (dp *DecoderPool) Decode(destination interface{}, source io.Reader) error {
	decoder := dp.pool.Get().(Decoder)
	defer dp.pool.Put(decoder)

	decoder.Reset(source)
	return decoder.Decode(destination)
}

// DecodeBytes unmarshals data from the source byte slice onto the destination instance.
// The destination is typically a pointer to a struct, such as *Message.
func (dp *DecoderPool) DecodeBytes(destination interface{}, source []byte) error {
	decoder := dp.pool.Get().(Decoder)
	defer dp.pool.Put(decoder)

	decoder.ResetBytes(source)
	return decoder.Decode(destination)
}
