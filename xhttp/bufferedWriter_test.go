// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package xhttp

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testBufferedWriterClose(t *testing.T) {
	const text = "close test"

	var (
		assert = assert.New(t)
		writer BufferedWriter
	)

	assert.NotNil(writer.Header())
	c, err := writer.Write([]byte(text))
	assert.Equal(len(text), c)
	assert.NoError(err)
	assert.NoError(writer.Close())

	assert.NotNil(writer.Header())
	c, err = writer.Write([]byte(text))
	assert.Zero(c)
	assert.Error(err)
	assert.Error(writer.Close())
}

func testBufferedWriterWriteToEmpty(t *testing.T) {
	var (
		assert = assert.New(t)
		writer BufferedWriter

		response = httptest.NewRecorder()
	)

	c, err := writer.WriteTo(response)
	assert.Zero(c)
	assert.NoError(err)
	assert.Equal(http.StatusOK, response.Code)
	assert.Empty(response.Header())
	assert.Empty(response.Body)
	assert.False(response.Flushed)

	assert.Error(writer.Close())
	c, err = writer.WriteTo(response)
	assert.Zero(c)
	assert.Error(err)
}

func testBufferedWriterWriteToWithContent(t *testing.T) {
	const text = "hello, world!"

	var (
		assert = assert.New(t)
		writer BufferedWriter

		response = httptest.NewRecorder()
	)

	writer.Header().Set("Content-Type", "text/plain")
	writer.Header().Set("X-Custom", "zippidee doo da")
	writer.Header().Set("X-Value", "1")
	writer.Header().Add("X-Value", "2")
	c, err := writer.Write([]byte(text))
	assert.Equal(len(text), c)
	assert.NoError(err)

	c, err = writer.WriteTo(response)
	assert.Equal(len(text), c)
	assert.NoError(err)
	assert.Equal(http.StatusOK, response.Code)
	assert.Equal(
		// nolint: typecheck
		http.Header{
			"Content-Type":   {"text/plain"},
			"X-Custom":       {"zippidee doo da"},
			"X-Value":        {"1", "2"},
			"Content-Length": {strconv.Itoa(len(text))},
		},
		response.Header(),
	)
	assert.Equal(text, response.Body.String())
	assert.False(response.Flushed)

	assert.Error(writer.Close())
	c, err = writer.WriteTo(response)
	assert.Zero(c)
	assert.Error(err)
}

func testBufferedWriterWriteToCustomResponseCode(t *testing.T) {
	const text = "with custom response code!"

	var (
		assert = assert.New(t)
		writer BufferedWriter

		response = httptest.NewRecorder()
	)

	writer.Header().Set("Content-Type", "text/plain")
	writer.Header().Set("X-Custom", "zippidee doo da")
	writer.Header().Set("X-Value", "1")
	writer.Header().Add("X-Value", "2")
	writer.WriteHeader(499)
	c, err := writer.Write([]byte(text))
	assert.Equal(len(text), c)
	assert.NoError(err)
	writer.WriteHeader(333) // should be ignored

	c, err = writer.WriteTo(response)
	assert.Equal(len(text), c)
	assert.NoError(err)
	assert.Equal(499, response.Code)
	assert.Equal(
		// nolint: typecheck
		http.Header{
			"Content-Type":   {"text/plain"},
			"X-Custom":       {"zippidee doo da"},
			"X-Value":        {"1", "2"},
			"Content-Length": {strconv.Itoa(len(text))},
		},
		response.Header(),
	)
	assert.Equal(text, response.Body.String())
	assert.False(response.Flushed)

	assert.Error(writer.Close())
	c, err = writer.WriteTo(response)
	assert.Zero(c)
	assert.Error(err)
}

func testBufferedWriterWriteHeaderBadCode(t *testing.T) {
	var (
		assert = assert.New(t)
		writer BufferedWriter
	)

	assert.Panics(func() {
		writer.WriteHeader(1)
	})
}

func TestBufferedWriter(t *testing.T) {
	t.Run("Close", testBufferedWriterClose)
	t.Run("WriteTo", func(t *testing.T) {
		t.Run("Empty", testBufferedWriterWriteToEmpty)
		t.Run("WithContent", testBufferedWriterWriteToWithContent)
		t.Run("CustomResponseCode", testBufferedWriterWriteToCustomResponseCode)
	})
	t.Run("WriteHeader", func(t *testing.T) {
		t.Run("BadCode", testBufferedWriterWriteHeaderBadCode)
	})
}
