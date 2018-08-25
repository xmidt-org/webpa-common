package xhttp

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	gokithttp "github.com/go-kit/kit/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testErrorState(t *testing.T) {
	var (
		assert    = assert.New(t)
		httpError = &Error{Code: 503, Header: http.Header{"Foo": []string{"Bar"}}, Text: "fubar"}
	)

	assert.Equal(503, httpError.StatusCode())
	assert.Equal(http.Header{"Foo": []string{"Bar"}}, httpError.Headers())
	assert.Equal("fubar", httpError.Error())

	json, err := httpError.MarshalJSON()
	assert.NoError(err)
	assert.JSONEq(
		`{"code": 503, "text": "fubar"}`,
		string(json),
	)
}

func testErrorDefaultEncoding(t *testing.T) {
	var (
		assert    = assert.New(t)
		httpError = &Error{Code: 503, Header: http.Header{"Foo": []string{"Bar"}}, Text: "fubar"}
		response  = httptest.NewRecorder()
	)

	gokithttp.DefaultErrorEncoder(context.Background(), httpError, response)
	assert.Equal(503, httpError.Code)
	assert.Equal("Bar", response.HeaderMap.Get("Foo"))
	assert.JSONEq(
		`{"code": 503, "text": "fubar"}`,
		response.Body.String(),
	)
}

func TestError(t *testing.T) {
	t.Run("State", testErrorState)
	t.Run("DefaultEncoding", testErrorDefaultEncoding)
}

func TestWriteErrorf(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		testData = []struct {
			code         int
			format       string
			parameters   []interface{}
			expectedJSON string
		}{
			{
				http.StatusInternalServerError,
				"some message followed by an int: %d",
				[]interface{}{47},
				`{"code": 500, "message": "some message followed by an int: 47"}`,
			},
			{
				412,
				"this message has no parameters",
				nil,
				`{"code": 412, "message": "this message has no parameters"}`,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)

		var (
			response   = httptest.NewRecorder()
			count, err = WriteErrorf(response, record.code, record.format, record.parameters...)
		)

		assert.True(count > 0)
		assert.NoError(err)
		assert.Equal(record.code, response.Code)
		assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))

		actualJSON, err := ioutil.ReadAll(response.Body)
		require.NoError(err)

		assert.JSONEq(record.expectedJSON, string(actualJSON))
	}
}

func TestWriteError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		testData = []struct {
			code         int
			value        interface{}
			expectedJSON string
		}{
			{
				http.StatusInternalServerError,
				"expected message",
				`{"code": 500, "message": "expected message"}`,
			},
			{
				567,
				"",
				`{"code": 567, "message": ""}`,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)

		var (
			response   = httptest.NewRecorder()
			count, err = WriteError(response, record.code, record.value)
		)

		assert.True(count > 0)
		assert.NoError(err)
		assert.Equal(record.code, response.Code)
		assert.Equal("application/json", response.HeaderMap.Get("Content-Type"))

		actualJSON, err := ioutil.ReadAll(response.Body)
		require.NoError(err)

		assert.JSONEq(record.expectedJSON, string(actualJSON))
	}
}
