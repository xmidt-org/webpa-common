package xhttptest

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
)

// NewResponse provides a convenient way of synthesizing a client response, similar to httptest.NewRequest.
// This function initializes most members to useful values for testing with.
func NewResponse(statusCode int, body []byte) *http.Response {
	return &http.Response{
		Status:        strconv.Itoa(statusCode),
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
}
