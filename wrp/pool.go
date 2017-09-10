package wrp

import (
	"io"
	"sync"
)

const (
	DefaultPoolCapacity = 100
)

// EncoderPool represents a pool of Encoder objects that can be used as is
// encode WRP messages.  Unlike a sync.Pool, this pool holds on to its pooled
// encoders across garbage collections.
type EncoderPool struct {
	lock     sync.Mutex
	pool     []Encoder
	capacity int
	format   Format
}

// NewEncoderPool returns an EncoderPool for a given format.  The initialBufferSize is
// used when encoding to byte arrays.  If this value is nonpositive, DefaultInitialBufferSize
// is used instead.
func NewEncoderPool(capacity int, f Format) *EncoderPool {
	if capacity < 1 {
		capacity = DefaultPoolCapacity
	}

	return &EncoderPool{
		pool:     make([]Encoder, 0, capacity),
		capacity: capacity,
		format:   f,
	}
}

// Format returns the wrp format this pool encodes to
func (ep *EncoderPool) Format() Format {
	return ep.format
}

// New simply creates a new Encoder using this pool's configuration.
// This method is used internally to populate and manage the pool, but
// can also be used externally to obtain a new, unpooled instance.
func (ep *EncoderPool) New() Encoder {
	return NewEncoder(nil, ep.format)
}

// Len returns the number of pooled elements available for Get.
func (ep *EncoderPool) Len() int {
	ep.lock.Lock()
	length := len(ep.pool)
	ep.lock.Unlock()
	return length
}

// Cap returns the capacity of the pool, which is fixed at the time of creation.
func (ep *EncoderPool) Cap() int {
	return ep.capacity
}

// Get returns an Encoder from the pool.  If the pool is empty, a new Encoder is
// created using the initial pool configuration.  This method never returns nil.
func (ep *EncoderPool) Get() (encoder Encoder) {
	ep.lock.Lock()

	last := len(ep.pool) - 1
	if last >= 0 {
		encoder, ep.pool[last] = ep.pool[last], nil
		ep.pool = ep.pool[0:last]
	} else {
		encoder = ep.New()
	}

	ep.lock.Unlock()
	return
}

// Put returns an Encoder to the pool.  This method returns true if the encoder
// was returned to the pool, false if the pool was full or encoder was nil.
func (ep *EncoderPool) Put(encoder Encoder) (returned bool) {
	if encoder != nil {
		ep.lock.Lock()

		if len(ep.pool) < ep.capacity {
			ep.pool = append(ep.pool, encoder)
			returned = true
		}

		ep.lock.Unlock()
	}

	return
}

// Encode uses an Encoder from the pool to encode the source into the destination
func (ep *EncoderPool) Encode(destination io.Writer, source interface{}) error {
	encoder := ep.Get()
	defer ep.Put(encoder)

	encoder.Reset(destination)
	return encoder.Encode(source)
}

// EncodeBytes uses an encoder from the pool to encode the source into a byte array.
// The destination pointer will be replaced with a slice sized for the encoded data,
// using a zero-copy approach.  If destination has points to a slice with adequate capacity,
// no new memory allocation is done.
func (ep *EncoderPool) EncodeBytes(destination *[]byte, source interface{}) error {
	encoder := ep.Get()
	defer ep.Put(encoder)

	encoder.ResetBytes(destination)
	return encoder.Encode(source)
}

// DecoderPool is a pool of Decoder instances for a specific format
type DecoderPool struct {
	lock     sync.Mutex
	pool     []Decoder
	capacity int
	format   Format
}

// NewDecoderPool returns a DecoderPool that works with a given Format
func NewDecoderPool(capacity int, f Format) *DecoderPool {
	if capacity < 1 {
		capacity = DefaultPoolCapacity
	}

	return &DecoderPool{
		pool:     make([]Decoder, 0, capacity),
		capacity: capacity,
		format:   f,
	}
}

// Format returns the wrp format this pool decodes from
func (ep *DecoderPool) Format() Format {
	return ep.format
}

// New simply creates a new Decoder using this pool's configuration.
// This method is used internally to populate and manage the pool, but
// can also be used externally to obtain a new, unpooled instance.
func (dp *DecoderPool) New() Decoder {
	return NewDecoder(nil, dp.format)
}

// Len returns the number of pooled elements available for Get.
func (dp *DecoderPool) Len() int {
	dp.lock.Lock()
	length := len(dp.pool)
	dp.lock.Unlock()
	return length
}

// Cap returns the capacity of the pool, which is fixed at the time of creation.
func (dp *DecoderPool) Cap() int {
	return dp.capacity
}

// Get obtains a Decoder from the pool.  If the pool is empty, a new Decoder is
// created using the initial pool configuration.  This method never returns nil.
func (dp *DecoderPool) Get() (decoder Decoder) {
	dp.lock.Lock()

	last := len(dp.pool) - 1
	if last >= 0 {
		decoder, dp.pool[last] = dp.pool[last], nil
		dp.pool = dp.pool[0:last]
	} else {
		decoder = dp.New()
	}

	dp.lock.Unlock()
	return
}

// Put returns a Decoder to the pool.  This method returns true if the decoder
// was returned to the pool, false if the pool was full or decoder was nil.
func (dp *DecoderPool) Put(decoder Decoder) (returned bool) {
	if decoder != nil {
		dp.lock.Lock()

		if len(dp.pool) < cap(dp.pool) {
			dp.pool = append(dp.pool, decoder)
			returned = true
		}

		dp.lock.Unlock()
	}

	return
}

// Decode unmarshals data from the source onto the destination instance, which is
// normally a pointer to some struct (such as *Message).
func (dp *DecoderPool) Decode(destination interface{}, source io.Reader) error {
	decoder := dp.Get()
	defer dp.Put(decoder)

	decoder.Reset(source)
	return decoder.Decode(destination)
}

// DecodeBytes unmarshals data from the source byte slice onto the destination instance.
// The destination is typically a pointer to a struct, such as *Message.
func (dp *DecoderPool) DecodeBytes(destination interface{}, source []byte) error {
	decoder := dp.Get()
	defer dp.Put(decoder)

	decoder.ResetBytes(source)
	return decoder.Decode(destination)
}
