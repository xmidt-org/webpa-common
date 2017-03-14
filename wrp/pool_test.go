package wrp

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testEncoderPool(assert *assert.Assertions, encoderPool *EncoderPool, output *[]byte) {
	var (
		initialSize = len(encoderPool.pool)

		testMessage = SimpleEvent{
			Destination: "foobar.com/test",
			Source:      "mac:11112222333",
			Payload:     []byte("testEncoderPool"),
		}

		buffer bytes.Buffer
	)

	assert.True(initialSize > 0)

	assert.NoError(encoderPool.Encode(&buffer, &testMessage))
	assert.Equal(initialSize, len(encoderPool.pool))
	assert.True(buffer.Len() > 0)

	err := encoderPool.EncodeBytes(output, &testMessage)
	assert.Equal(initialSize, len(encoderPool.pool))
	assert.NotEmpty(*output)
	assert.NoError(err)
	assert.Equal(*output, buffer.Bytes())

	for len(encoderPool.pool) > 0 {
		assert.NotNil(encoderPool.Get())
	}

	// an exhausted pool should still give out encoders
	assert.NotNil(encoderPool.Get())

	for len(encoderPool.pool) < initialSize {
		encoderPool.Put(encoderPool.factory())
	}

	// a full pool should silently reject Puts
	encoderPool.Put(encoderPool.factory())
	assert.Equal(initialSize, len(encoderPool.pool))
}

func TestEncoderPool(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			poolSize          int
			initialBufferSize int
			format            Format
		}{
			{0, 0, Msgpack},
			{10, 10, Msgpack},
			{0, 0, JSON},
			{10, 10, JSON},
			{0, 1000, JSON},
			{10, 1000, JSON},
		}
	)

	for _, record := range testData {
		t.Run(
			fmt.Sprintf("%s/poolSize=%d/initialBufferSize=%d", record.format, record.poolSize, record.initialBufferSize),
			func(t *testing.T) {
				output := make([]byte, record.initialBufferSize)
				testEncoderPool(assert, NewEncoderPool(record.poolSize, record.format), &output)
			},
		)
	}
}

func testDecoderPool(assert *assert.Assertions, format Format, decoderPool *DecoderPool) {
	var (
		initialSize = len(decoderPool.pool)

		originalMessage = SimpleEvent{
			Destination: "foobar.com/test",
			Source:      "mac:11112222333",
			Payload:     []byte("testDecoderPool"),
		}

		testMessage *SimpleEvent
		encoded     []byte
	)

	if !assert.NoError(NewEncoderBytes(&encoded, format).Encode(&originalMessage)) {
		return
	}

	assert.True(initialSize > 0)

	testMessage = new(SimpleEvent)
	assert.NoError(decoderPool.Decode(testMessage, bytes.NewReader(encoded)))
	assert.Equal(initialSize, len(decoderPool.pool))
	assert.Equal(originalMessage, *testMessage)

	assert.NoError(decoderPool.DecodeBytes(testMessage, encoded))
	assert.Equal(initialSize, len(decoderPool.pool))
	assert.Equal(originalMessage, *testMessage)

	for len(decoderPool.pool) > 0 {
		assert.NotNil(decoderPool.Get())
	}

	// an exhausted pool should still give out encoders
	assert.NotNil(decoderPool.Get())

	for len(decoderPool.pool) < initialSize {
		decoderPool.Put(decoderPool.factory())
	}

	// a full pool should silently reject Puts
	decoderPool.Put(decoderPool.factory())
	assert.Equal(initialSize, len(decoderPool.pool))
}

func TestDecoderPool(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			poolSize int
			format   Format
		}{
			{0, Msgpack},
			{10, Msgpack},
			{0, JSON},
			{10, JSON},
		}
	)

	for _, record := range testData {
		t.Run(
			fmt.Sprintf("%s/poolSize=%d", record.format, record.poolSize),
			func(t *testing.T) {
				testDecoderPool(assert, record.format, NewDecoderPool(record.poolSize, record.format))
			},
		)
	}
}
