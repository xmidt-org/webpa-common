package httperror

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormatf(t *testing.T) {
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
			count, err = Formatf(response, record.code, record.format, record.parameters...)
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

func TestFormat(t *testing.T) {
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
			count, err = Format(response, record.code, record.value)
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
