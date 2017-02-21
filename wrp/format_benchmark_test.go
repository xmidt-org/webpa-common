package wrp

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

var (
	testMessage = Message{
		Source:          "http://source.comcast.net:9090/test",
		Destination:     "mac:112233445566",
		TransactionUUID: "as;ldkfjakdljfaskdjfaskdjf",
		Payload:         []byte("ah, some lovely payload here!"),
	}

	encoderPools = []*EncoderPool{
		NewEncoderPool(0, JSON),
		NewEncoderPool(0, Msgpack),
	}
)

func benchmarkCreateEncoderOnTheFly(b *testing.B, format Format, routines int) {
	var (
		startingLine = make(chan struct{})
		waitGroup    = new(sync.WaitGroup)
	)

	waitGroup.Add(routines)
	for spawn := 0; spawn < routines; spawn++ {
		go func() {
			defer waitGroup.Done()
			<-startingLine

			for repeat := 0; repeat < b.N; repeat++ {
				var (
					buffer  = new(bytes.Buffer)
					encoder = NewEncoder(buffer, format)
				)

				encoder.Encode(testMessage)
			}
		}()
	}

	b.ResetTimer()
	close(startingLine)
	waitGroup.Wait()
}

func BenchmarkCreateEncoderOnTheFly(b *testing.B) {
	for _, format := range []Format{Msgpack, JSON} {
		for _, routines := range []int{1, 10, 100} {
			b.Run(
				fmt.Sprintf("%s/routines=%d", format, routines),
				func(b *testing.B) { benchmarkCreateEncoderOnTheFly(b, format, routines) },
			)
		}
	}
}

func benchmarkPooledEncoder(b *testing.B, format Format, routines int) {
	var (
		startingLine = make(chan struct{})
		waitGroup    = new(sync.WaitGroup)
	)

	waitGroup.Add(routines)
	for spawn := 0; spawn < routines; spawn++ {
		go func() {
			defer waitGroup.Done()
			<-startingLine

			for repeat := 0; repeat < b.N; repeat++ {
				encoderPools[format].EncodeBytes(&testMessage)
			}
		}()
	}

	b.ResetTimer()
	close(startingLine)
	waitGroup.Wait()
}

func BenchmarkPooledEncoder(b *testing.B) {
	for _, format := range []Format{Msgpack, JSON} {
		for _, routines := range []int{1, 10, 100} {
			b.Run(
				fmt.Sprintf("%s/routines=%d", format, routines),
				func(b *testing.B) { benchmarkPooledEncoder(b, format, routines) },
			)
		}
	}
}
