package wrp

import (
	"io"
)

const (
	DefaultPoolSize          = 100
	DefaultInitialBufferSize = 200
)

// EncoderPool represents a pool of Encoder objects that can be used as is
// encode WRP messages.
type EncoderPool struct {
	pool              chan Encoder
	factory           func() Encoder
	initialBufferSize int
}

// NewEncoderPool returns an EncoderPool for a given format.  The initialBufferSize is
// used when encoding to byte arrays.  If this value is nonpositive, DefaultInitialBufferSize
// is used instead.
func NewEncoderPool(poolSize, initialBufferSize int, f Format) *EncoderPool {
	if poolSize < 1 {
		poolSize = DefaultPoolSize
	}

	if initialBufferSize < 1 {
		initialBufferSize = DefaultInitialBufferSize
	}

	ep := &EncoderPool{
		pool:              make(chan Encoder, poolSize),
		factory:           func() Encoder { return NewEncoder(nil, f) },
		initialBufferSize: initialBufferSize,
	}

	for repeat := 0; repeat < poolSize; repeat++ {
		ep.pool <- ep.factory()
	}

	return ep
}

func (ep *EncoderPool) Get() (encoder Encoder) {
	select {
	case encoder = <-ep.pool:
	default:
		encoder = ep.factory()
	}

	return
}

func (ep *EncoderPool) Put(encoder Encoder) {
	select {
	case ep.pool <- encoder:
	default:
	}
}

// Encode uses an Encoder from the pool to encode the source into the destination
func (ep *EncoderPool) Encode(destination io.Writer, source interface{}) error {
	encoder := ep.Get()
	defer ep.Put(encoder)

	encoder.Reset(destination)
	return encoder.Encode(source)
}

// EncodeBytes uses an encoder from the pool to encode the source into a byte array.
// This method attempts to minimize memory allocation overhead by allocating the initialBufferSize
// specified in NewEncoderPool.
func (ep *EncoderPool) EncodeBytes(source interface{}) (data []byte, err error) {
	data = make([]byte, ep.initialBufferSize)
	encoder := ep.Get()
	defer ep.Put(encoder)

	encoder.ResetBytes(&data)
	err = encoder.Encode(source)
	return
}

// DecoderPool is a pool of Decoder instances for a specific format
type DecoderPool struct {
	pool    chan Decoder
	factory func() Decoder
}

// NewDecoderPool returns a DecoderPool that works with a given Format
func NewDecoderPool(poolSize int, f Format) *DecoderPool {
	if poolSize < 1 {
		poolSize = DefaultPoolSize
	}

	dp := &DecoderPool{
		pool:    make(chan Decoder, poolSize),
		factory: func() Decoder { return NewDecoder(nil, f) },
	}

	for repeat := 0; repeat < poolSize; repeat++ {
		dp.pool <- dp.factory()
	}

	return dp
}

func (dp *DecoderPool) Get() (decoder Decoder) {
	select {
	case decoder = <-dp.pool:
	default:
		decoder = dp.factory()
	}

	return
}

func (dp *DecoderPool) Put(decoder Decoder) {
	select {
	case dp.pool <- decoder:
	default:
	}
}

// Decode unmarshals data from the source onto the destination instance, which is
// normally a pointer to some struct (such as *Message).
func (dp *DecoderPool) Decode(destination interface{}, source io.Reader) error {
	decoder := dp.Get()
	defer dp.Put(decoder)

	decoder.Reset(source)
	return decoder.Decode(destination)
}

// DecodeMessage decodes a WRP message from the source Reader
func (dp *DecoderPool) DecodeMessage(source io.Reader) (message *Message, err error) {
	message = new(Message)
	err = dp.Decode(message, source)
	return
}

// DecodeBytes unmarshals data from the source byte slice onto the destination instance.
// The destination is typically a pointer to a struct, such as *Message.
func (dp *DecoderPool) DecodeBytes(destination interface{}, source []byte) error {
	decoder := dp.Get()
	defer dp.Put(decoder)

	decoder.ResetBytes(source)
	return decoder.Decode(destination)
}

// DecodeMessageBytes unmarshals a WRP message from a source byte slice
func (dp *DecoderPool) DecodeMessageBytes(source []byte) (message *Message, err error) {
	message = new(Message)
	err = dp.DecodeBytes(message, source)
	return
}
