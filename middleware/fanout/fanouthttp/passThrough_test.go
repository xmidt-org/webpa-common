package fanouthttp

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testPassThroughSpans(t *testing.T) {
	var (
		assert = assert.New(t)
		span   = tracing.NewSpanner().Start("one")(nil)

		original = PassThrough{
			StatusCode:  237,
			ContentType: "spplication/something",
			Entity:      []byte{1, 2, 3},
		}
	)

	assert.Empty(original.Spans())

	spanned := original.WithSpans(span).(*PassThrough)
	assert.Equal([]tracing.Span{span}, spanned.Spans())
	assert.False(spanned == &original)
	assert.Equal(original.StatusCode, spanned.StatusCode)
	assert.Equal(original.ContentType, spanned.ContentType)
	assert.Equal(original.Entity, spanned.Entity)
}

func testPassThroughNilEntity(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	pt := PassThrough{}

	rc := pt.ReadCloser()
	require.NotNil(rc)
	contents, err := ioutil.ReadAll(rc)
	assert.Empty(contents)
	assert.NoError(err)
	assert.NoError(rc.Close())

	rc, err = pt.GetBody()
	require.NotNil(rc)
	assert.NoError(err)
	contents, err = ioutil.ReadAll(rc)
	assert.Empty(contents)
	assert.NoError(err)
	assert.NoError(rc.Close())
}

func testPassThroughEntity(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		testData = [][]byte{
			[]byte{},
			[]byte(`this is an entity`),
		}
	)

	for _, entity := range testData {
		pt := PassThrough{Entity: entity}

		rc := pt.ReadCloser()
		require.NotNil(rc)
		contents, err := ioutil.ReadAll(rc)
		assert.Equal(entity, contents)
		assert.NoError(err)
		assert.NoError(rc.Close())

		rc, err = pt.GetBody()
		require.NotNil(rc)
		assert.NoError(err)
		contents, err = ioutil.ReadAll(rc)
		assert.Equal(entity, contents)
		assert.NoError(err)
		assert.NoError(rc.Close())
	}
}

func TestPassThrough(t *testing.T) {
	t.Run("Spans", testPassThroughSpans)
	t.Run("NilEntity", testPassThroughNilEntity)
	t.Run("Entity", testPassThroughEntity)
}

func testDecodePassThroughRequestValid(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			contentType string
			body        []byte
		}{
			{"", []byte{}},
			{"text/plain", []byte{}},
			{"", []byte("this is a body")},
			{"text/plain", []byte("this is a body")},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		original := httptest.NewRequest("GET", "/", bytes.NewReader(record.body))
		original.Header.Set("Content-Type", record.contentType)

		result, err := DecodePassThroughRequest(context.Background(), original)
		pt := result.(*PassThrough)
		assert.NoError(err)

		assert.Equal(-1, pt.StatusCode)
		assert.Equal(record.contentType, pt.ContentType)
		assert.Equal(record.body, pt.Entity)
	}
}

func testDecodePassThroughRequestBodyError(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedError = errors.New("expected")
		badBody       = new(mockReader)
		original      = httptest.NewRequest("GET", "/", badBody)
	)

	badBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()

	result, err := DecodePassThroughRequest(context.Background(), original)
	assert.Nil(result)
	assert.Equal(expectedError, err)

	badBody.AssertExpectations(t)
}

func TestDecodePassThroughRequest(t *testing.T) {
	t.Run("Valid", testDecodePassThroughRequestValid)
	t.Run("BodyError", testDecodePassThroughRequestBodyError)
}

func testDecodePassThroughResponseValid(t *testing.T) {
	testData := []struct {
		contentType string
		body        []byte
	}{
		{"", []byte{}},
		{"", []byte("this is a body")},
		{"application/json", []byte{}},
		{"text/plain", []byte("this is a body")},
	}

	t.Run("GoodResponse", func(t *testing.T) {
		assert := assert.New(t)

		for _, record := range testData {
			t.Logf("%#v", record)

			for _, statusCode := range []int{200, 201, 247, 299, 301, 310, 333} {
				component := &http.Response{
					StatusCode: statusCode,
					Header:     http.Header{"Content-Type": []string{record.contentType}},
					Body:       ioutil.NopCloser(bytes.NewReader(record.body)),
				}

				result, err := DecodePassThroughResponse(context.Background(), component)
				pt := result.(*PassThrough)
				assert.NoError(err)

				assert.Equal(statusCode, pt.StatusCode)
				assert.Equal(record.contentType, pt.ContentType)
				assert.Equal(record.body, pt.Entity)
			}
		}
	})

	t.Run("BadResponse", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)
		)

		for _, record := range testData {
			t.Logf("%#v", record)

			for _, statusCode := range []int{400, 403, 404, 500, 503, 540} {
				component := &http.Response{
					StatusCode: statusCode,
					Body:       ioutil.NopCloser(bytes.NewReader(record.body)),
				}

				result, err := DecodePassThroughResponse(context.Background(), component)
				assert.Nil(result)
				require.Error(err)
				httpError := err.(*httperror.E)

				assert.Equal(statusCode, httpError.Code)
				assert.NotEmpty(httpError.Text)
				assert.Equal(record.body, httpError.Entity)
			}
		}
	})
}

func testDecodePassThroughResponseBodyError(t *testing.T) {
	var (
		assert = assert.New(t)

		expectedError = errors.New("expected")
		badBody       = new(mockReader)
		component     = &http.Response{Body: ioutil.NopCloser(badBody)}
	)

	badBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()

	result, err := DecodePassThroughResponse(context.Background(), component)
	assert.Nil(result)
	assert.Equal(expectedError, err)

	badBody.AssertExpectations(t)
}

func TestDecodePassThroughResponse(t *testing.T) {
	t.Run("Valid", testDecodePassThroughResponseValid)
	t.Run("BodyError", testDecodePassThroughResponseBodyError)
}

func TestEncodePassThroughRequest(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		testData = []struct {
			contentType string
			body        []byte
		}{
			{"", []byte{}},
			{"text/plain", []byte{}},
			{"", []byte("this is a body")},
			{"text/plain", []byte("this is a body")},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			pt = &PassThrough{
				ContentType: record.contentType,
				Entity:      record.body,
			}

			component = httptest.NewRequest("GET", "/", bytes.NewReader(record.body))
		)

		assert.NoError(EncodePassThroughRequest(context.Background(), component, pt))

		contents, err := ioutil.ReadAll(component.Body)
		assert.Equal(record.body, contents)
		assert.NoError(err)

		require.NotNil(component.GetBody)
		rc, err := component.GetBody()
		require.NotNil(rc)
		require.NoError(err)
		contents, err = ioutil.ReadAll(rc)
		assert.Equal(record.body, contents)
		assert.NoError(err)

		assert.Equal(record.contentType, component.Header.Get("Content-Type"))
	}
}

func TestEncodePassThroughResponse(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			statusCode          int
			contentType         string
			body                []byte
			expectedStatusCode  int
			expectedContentType string
		}{
			{0, "", []byte{}, http.StatusOK, "text/plain; charset=utf-8"}, // the implicit content type set by golang
			{http.StatusOK, "", []byte{}, http.StatusOK, ""},
			{201, "application/json", []byte{}, 201, "application/json"},
			{0, "", []byte("this is a body"), http.StatusOK, "text/plain; charset=utf-8"}, // the implicit content type set by golang
			{http.StatusOK, "", []byte("this is a body"), http.StatusOK, ""},
			{201, "application/json", []byte("this is a body"), 201, "application/json"},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			pt = &PassThrough{
				StatusCode:  record.statusCode,
				ContentType: record.contentType,
				Entity:      record.body,
			}

			original = httptest.NewRecorder()
			buffer   = bytes.NewBuffer([]byte{})
		)

		original.Body = buffer
		assert.NoError(EncodePassThroughResponse(context.Background(), original, pt))

		assert.Equal(record.expectedStatusCode, original.Code)
		assert.Equal(record.expectedContentType, original.HeaderMap.Get("Content-Type"))
		assert.Equal(record.body, buffer.Bytes())
	}
}
