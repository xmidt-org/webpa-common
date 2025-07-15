// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	// nolint: typecheck
	"sync/atomic"
)

var ErrBufferedWriterClosed = errors.New("BufferedWriter has been closed")

// BufferedWriter is a closeable http.ResponseWriter that holds all written response information in memory.
// The zero value of this type is a fully usable writer.
//
// The http.ResponseWriter methods of this type are not safe for concurrent execution.  However, it is safe
// to invoke Close concurrently with the other methods.
//
// Once closed, future Write and WriteTo calls will return errors.  This type is ideal for http.Handler code
// that should be Buffered and optionally discarded depending on other logic.
type BufferedWriter struct {
	closed      uint32
	wroteHeader bool
	code        int
	header      http.Header
	buffer      bytes.Buffer
}

// Close closes this writer.  Once closed, this writer will reject writes with an error.
// This method is idempotent, and will return an error if called more than once on a given writer instance.
func (bw *BufferedWriter) Close() error {
	if atomic.CompareAndSwapUint32(&bw.closed, 0, 1) {
		return nil
	}

	return ErrBufferedWriterClosed
}

// Header returns the HTTP header to write to the response.  This method is unaffected by the close state.
func (bw *BufferedWriter) Header() http.Header {
	// nolint: typecheck
	if bw.header == nil {
		bw.header = make(http.Header)
	}

	return bw.header
}

// Write buffers content for later writing to a real http.ResponseWriter.  If this writer is closed,
// this method returns a count of 0 with an error.
func (bw *BufferedWriter) Write(p []byte) (int, error) {
	if atomic.LoadUint32(&bw.closed) == 1 {
		return 0, ErrBufferedWriterClosed
	}

	if !bw.wroteHeader {
		bw.writeHeader(http.StatusOK)
	}

	return bw.buffer.Write(p)
}

// WriteHeader sets a status code and in most other ways behaves as the standard net/http ResponseWriter.
// This method is idempotent.  Only the first invocation will have an effect.  If this writer is closed, this
// method has no effect.
func (bw *BufferedWriter) WriteHeader(code int) {
	// mimic the current behavior of the stdlib
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("Invalid WriteHeader code %v", code))
	}

	if atomic.LoadUint32(&bw.closed) == 1 || bw.wroteHeader {
		return
	}

	bw.writeHeader(code)
}

func (bw *BufferedWriter) writeHeader(code int) {
	bw.wroteHeader = true
	bw.code = code
}

// WriteTo transfers this writer's state to the given ResponseWriter.  This method will only take effect once.
// Once this method has been invoked successfully, this writer instance is closed and will reject future writes
// (including this method).
func (bw *BufferedWriter) WriteTo(response http.ResponseWriter) (int, error) {
	if atomic.CompareAndSwapUint32(&bw.closed, 0, 1) {
		destination := response.Header()
		for k, v := range bw.header {
			destination[http.CanonicalHeaderKey(k)] = v
		}

		contentLength := bw.buffer.Len()
		if contentLength > 0 {
			destination.Set("Content-Length", strconv.Itoa(contentLength))
		}

		code := bw.code
		if code < 100 {
			code = http.StatusOK
		}

		response.WriteHeader(code)
		c, err := bw.buffer.WriteTo(response)
		return int(c), err
	}

	return 0, ErrBufferedWriterClosed
}
