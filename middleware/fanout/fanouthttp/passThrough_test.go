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
			CopyHeader:  http.Header{"Foo": []string{"Bar"}},
			Entity:      []byte{1, 2, 3},
		}
	)

	assert.Empty(original.Spans())

	spanned := original.WithSpans(span).(*PassThrough)
	assert.Equal([]tracing.Span{span}, spanned.Spans())
	assert.False(spanned == &original)
	assert.Equal(original.StatusCode, spanned.StatusCode)
	assert.Equal(original.ContentType, spanned.ContentType)
	assert.Equal(original.CopyHeader, spanned.CopyHeader)
	assert.Equal(original.Entity, spanned.Entity)
}

func testPassThroughHeaders(t *testing.T) {
	assert := assert.New(t)

	{
		pt := &PassThrough{}
		assert.Empty(pt.Headers())
	}

	{
		pt := &PassThrough{CopyHeader: http.Header{}}
		assert.Equal(http.Header{}, pt.Headers())
	}
	{
		pt := &PassThrough{CopyHeader: http.Header{"Foo": []string{"Bar"}}}
		assert.Equal(http.Header{"Foo": []string{"Bar"}}, pt.Headers())
	}
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
	t.Run("Headers", testPassThroughHeaders)
	t.Run("NilEntity", testPassThroughNilEntity)
	t.Run("Entity", testPassThroughEntity)
}

func testDecodePassThroughRequestValid(t *testing.T) {
	var (
		assert   = assert.New(t)
		require  = require.New(t)
		testData = []struct {
			hs             HeaderSet
			originalHeader http.Header
			body           []byte
		}{
			{nil, http.Header{}, []byte{}},
			{nil, http.Header{"X-Test": []string{"value"}}, []byte{}},
			{nil, http.Header{"Content-Type": []string{"application/json"}}, []byte{}},
			{nil, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte{}},
			{HeaderSet{"X-NoSuch"}, http.Header{}, []byte{}},
			{HeaderSet{"X-NoSuch"}, http.Header{"X-Test": []string{"value"}}, []byte{}},
			{HeaderSet{"X-NoSuch"}, http.Header{"Content-Type": []string{"application/json"}}, []byte{}},
			{HeaderSet{"X-NoSuch"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte{}},
			{HeaderSet{"X-Test"}, http.Header{}, []byte{}},
			{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}}, []byte{}},
			{HeaderSet{"X-Test"}, http.Header{"Content-Type": []string{"application/json"}}, []byte{}},
			{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte{}},
			{nil, http.Header{}, []byte("this is a body")},
			{nil, http.Header{"X-Test": []string{"value"}}, []byte("this is a body")},
			{nil, http.Header{"Content-Type": []string{"application/json"}}, []byte("this is a body")},
			{nil, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte("this is a body")},
			{HeaderSet{"X-NoSuch"}, http.Header{}, []byte("this is a body")},
			{HeaderSet{"X-NoSuch"}, http.Header{"X-Test": []string{"value"}}, []byte("this is a body")},
			{HeaderSet{"X-NoSuch"}, http.Header{"Content-Type": []string{"application/json"}}, []byte("this is a body")},
			{HeaderSet{"X-NoSuch"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte("this is a body")},
			{HeaderSet{"X-Test"}, http.Header{}, []byte("this is a body")},
			{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}}, []byte("this is a body")},
			{HeaderSet{"X-Test"}, http.Header{"Content-Type": []string{"application/json"}}, []byte("this is a body")},
			{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"application/json"}}, []byte("this is a body")},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		original := httptest.NewRequest("GET", "/", bytes.NewReader(record.body))
		for n, v := range record.originalHeader {
			original.Header[n] = v
		}

		decoder := DecodePassThroughRequest(record.hs)
		require.NotNil(decoder)

		result, err := decoder(context.Background(), original)
		pt := result.(*PassThrough)
		assert.NoError(err)

		assert.Equal(-1, pt.StatusCode)
		assert.Equal(record.originalHeader.Get("Content-Type"), pt.ContentType)
		assert.Equal(record.hs.Filter(nil, record.originalHeader), pt.CopyHeader)
		assert.Equal(record.body, pt.Entity)
	}
}

func testDecodePassThroughRequestBodyError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("expected")
		badBody       = new(mockReader)
		original      = httptest.NewRequest("GET", "/", badBody)
	)

	badBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	decoder := DecodePassThroughRequest(nil)
	require.NotNil(decoder)

	result, err := decoder(context.Background(), original)
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
		hs              HeaderSet
		componentHeader http.Header
		body            []byte
	}{
		{nil, http.Header{}, []byte{}},
		{nil, http.Header{}, []byte("this is a body")},
		{nil, http.Header{"Content-Type": []string{"text/plain"}}, []byte{}},
		{nil, http.Header{"Content-Type": []string{"text/plain"}}, []byte("this is a body")},
		{nil, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"text/plain"}}, []byte{}},
		{nil, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"text/plain"}}, []byte("this is a body")},
		{HeaderSet{"X-Test"}, http.Header{}, []byte{}},
		{HeaderSet{"X-Test"}, http.Header{}, []byte("this is a body")},
		{HeaderSet{"X-Test"}, http.Header{"Content-Type": []string{"text/plain"}}, []byte{}},
		{HeaderSet{"X-Test"}, http.Header{"Content-Type": []string{"text/plain"}}, []byte("this is a body")},
		{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"text/plain"}}, []byte{}},
		{HeaderSet{"X-Test"}, http.Header{"X-Test": []string{"value"}, "Content-Type": []string{"text/plain"}}, []byte("this is a body")},
	}

	t.Run("GoodResponse", func(t *testing.T) {
		var (
			assert  = assert.New(t)
			require = require.New(t)
		)

		for _, record := range testData {
			t.Logf("%#v", record)

			for _, statusCode := range []int{200, 201, 247, 299, 301, 310, 333} {
				component := &http.Response{
					StatusCode: statusCode,
					Header:     record.componentHeader,
					Body:       ioutil.NopCloser(bytes.NewReader(record.body)),
				}

				decoder := DecodePassThroughResponse(record.hs)
				require.NotNil(decoder)

				result, err := decoder(context.Background(), component)
				pt := result.(*PassThrough)
				assert.NoError(err)

				assert.Equal(statusCode, pt.StatusCode)
				assert.Equal(record.componentHeader.Get("Content-Type"), pt.ContentType)
				assert.Equal(record.hs.Filter(nil, record.componentHeader), pt.CopyHeader)
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
					Header:     record.componentHeader,
					Body:       ioutil.NopCloser(bytes.NewReader(record.body)),
				}

				decoder := DecodePassThroughResponse(record.hs)
				require.NotNil(decoder)

				result, err := decoder(context.Background(), component)
				assert.Nil(result)
				require.Error(err)
				httpError := err.(*httperror.E)

				assert.Equal(statusCode, httpError.Code)
				assert.Equal(record.hs.Filter(nil, record.componentHeader), httpError.Header)
				assert.NotEmpty(httpError.Text)
				assert.Equal(record.body, httpError.Entity)
			}
		}
	})
}

func testDecodePassThroughResponseBodyError(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		expectedError = errors.New("expected")
		badBody       = new(mockReader)
		component     = &http.Response{Body: ioutil.NopCloser(badBody)}
	)

	badBody.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	decoder := DecodePassThroughResponse(nil)
	require.NotNil(decoder)

	result, err := decoder(context.Background(), component)
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
			copyHeader          http.Header
			contentType         string
			body                []byte
			expectedContentType string
		}{
			{http.Header{}, "", []byte{}, ""},
			{http.Header{}, "text/plain", []byte{}, "text/plain"},
			{http.Header{}, "", []byte("this is a body"), ""},
			{http.Header{}, "text/plain", []byte("this is a body"), "text/plain"},
			{http.Header{"X-Test": []string{"value"}}, "", []byte{}, ""},
			{http.Header{"X-Test": []string{"value"}}, "text/plain", []byte{}, "text/plain"},
			{http.Header{"X-Test": []string{"value"}}, "", []byte("this is a body"), ""},
			{http.Header{"X-Test": []string{"value"}}, "text/plain", []byte("this is a body"), "text/plain"},
			{http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "", []byte{}, "application/json"},
			{http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "text/plain", []byte{}, "text/plain"},
			{http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "", []byte("this is a body"), "application/json"},
			{http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "text/plain", []byte("this is a body"), "text/plain"},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			pt = &PassThrough{
				CopyHeader:  record.copyHeader,
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

		assert.Equal(record.expectedContentType, component.Header.Get("Content-Type"))

		for n, v := range record.copyHeader {
			if n != "Content-Type" {
				assert.Equal(v, component.Header[n])
			}
		}
	}
}

func TestEncodePassThroughResponse(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			statusCode          int
			copyHeader          http.Header
			contentType         string
			body                []byte
			expectedStatusCode  int
			expectedContentType string
		}{
			{0, http.Header{}, "", []byte{}, http.StatusOK, "text/plain; charset=utf-8"}, // the implicit content type set by golang
			{0, http.Header{"X-Test": []string{"value"}}, "application/octet-stream", []byte{1, 2, 3}, http.StatusOK, "application/octet-stream"},
			{http.StatusOK, http.Header{}, "", []byte{}, http.StatusOK, ""},
			{231, http.Header{"X-Test": []string{"value"}}, "application/octet-stream", []byte{1, 2, 3}, 231, "application/octet-stream"},
			{0, http.Header{"Content-Type": []string{"application/json"}}, "", []byte{}, http.StatusOK, "application/json"},
			{0, http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "application/octet-stream", []byte{1, 2, 3}, http.StatusOK, "application/octet-stream"},
			{http.StatusOK, http.Header{"Content-Type": []string{"application/json"}}, "", []byte{}, http.StatusOK, "application/json"},
			{231, http.Header{"Content-Type": []string{"application/json"}, "X-Test": []string{"value"}}, "application/octet-stream", []byte{1, 2, 3}, 231, "application/octet-stream"},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			pt = &PassThrough{
				StatusCode:  record.statusCode,
				CopyHeader:  record.copyHeader,
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

		for n, v := range record.copyHeader {
			if n != "Content-Type" {
				assert.Equal(v, original.HeaderMap[n])
			}
		}
	}
}
