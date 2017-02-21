package wrp

import (
	"bytes"
	"sync"
	"testing"
)

type pooledEncoder struct {
	encoder Encoder
	buffer  *bytes.Buffer
}

func pooledEncoderFactory(format Format) func() interface{} {
	return func() interface{} {
		return &pooledEncoder{
			encoder: NewEncoder(nil, format),
			buffer:  new(bytes.Buffer),
		}
	}
}

var (
	testMessage = Message{
		Source:          "http://source.comcast.net:9090/test",
		Destination:     "mac:112233445566",
		TransactionUUID: "as;ldkfjakdljfaskdjfaskdjf",
		Payload:         []byte("ah, some lovely payload here!"),
	}

	encoderPools = []sync.Pool{
		sync.Pool{
			New: pooledEncoderFactory(JSON),
		},
		sync.Pool{
			New: pooledEncoderFactory(Msgpack),
		},
	}
)

func benchmarkCreateEncoderOnTheFly(b *testing.B, format Format) {
	for repeat := 0; repeat < b.N; repeat++ {
		var (
			buffer  = new(bytes.Buffer)
			encoder = NewEncoder(buffer, format)
		)

		encoder.Encode(testMessage)
	}
}

func BenchmarkCreateEncoderOnTheFly(b *testing.B) {
	for _, format := range []Format{Msgpack, JSON} {
		b.Run(
			format.String(),
			func(b *testing.B) {
				benchmarkCreateEncoderOnTheFly(b, format)
			},
		)
	}
}

func benchmarkPooledEncoder(b *testing.B, format Format) {
	for repeat := 0; repeat < b.N; repeat++ {
		pooled := encoderPools[format].Get().(*pooledEncoder)
		pooled.buffer.Reset()
		pooled.encoder.Reset(pooled.buffer)
		pooled.encoder.Encode(testMessage)

		encoderPools[format].Put(pooled)
	}
}

func BenchmarkPooledEncoder(b *testing.B) {
	for _, format := range []Format{Msgpack, JSON} {
		b.Run(
			format.String(),
			func(b *testing.B) {
				benchmarkPooledEncoder(b, format)
			},
		)
	}
}
