package xhttp

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mustNewRequest invokes http.NewRequest, panicing if an error occurs.  This is important
// for this package to use instead of httptest.NewRequest, as those two NewRequest functions
// are slightly different.  This function ensures that the way code creates requests in
// production is tested.
func mustNewRequest(method, target string, body io.Reader) *http.Request {
	r, err := http.NewRequest(method, target, body)
	if err != nil {
		panic(err)
	}

	return r
}

func TestNopCloser(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedBytes = []byte{9, 12, 74, 125, 22}

		reader = bytes.NewReader(expectedBytes)
	)

	rsc := NopCloser(reader)
	require.NotNil(rsc)
	actualBytes, err := ioutil.ReadAll(rsc)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
	assert.NoError(rsc.Close())

	rsc2 := NopCloser(rsc)
	require.NotNil(rsc2)
	assert.True(rsc == rsc2)
	assert.NoError(rsc2.Close())

	_, err = reader.Seek(0, 0)
	assert.NoError(err)

	actualBytes, err = ioutil.ReadAll(rsc2)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
}

func testNewRewindReadSeeker(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedBytes = []byte{9, 234, 12, 93, 41}

		reader = bytes.NewReader(expectedBytes)
	)

	body, getBody, err := NewRewind(reader)
	assert.NoError(err)
	require.NotNil(body)
	require.NotNil(getBody)

	actualBytes, err := ioutil.ReadAll(body)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)

	body2, err := getBody()
	assert.NoError(err)
	require.NotNil(body2)
	assert.True(body == body2)

	actualBytes, err = ioutil.ReadAll(body2)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
}

func testNewRewindReadError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")

		reader = new(mockReader)
	)

	reader.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	body, getBody, err := NewRewind(reader)
	assert.Nil(body)
	assert.Nil(getBody)
	assert.Error(err)

	reader.AssertExpectations(t)
}

func testNewRewindBuffer(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedBytes = []byte{9, 234, 12, 93, 41}

		buffer = bytes.NewBuffer(expectedBytes)
	)

	body, getBody, err := NewRewind(buffer)
	assert.NoError(err)
	require.NotNil(body)
	require.NotNil(getBody)

	actualBytes, err := ioutil.ReadAll(body)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)

	body2, err := getBody()
	assert.NoError(err)
	require.NotNil(body2)
	assert.True(body == body2)

	actualBytes, err = ioutil.ReadAll(body2)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
}

func TestNewRewind(t *testing.T) {
	t.Run("ReadSeeker", testNewRewindReadSeeker)
	t.Run("ReadError", testNewRewindReadError)
	t.Run("Buffer", testNewRewindBuffer)
}

func TestNewRewindBytes(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedBytes = []byte{7, 234, 12, 9, 100}
	)

	body, getBody := NewRewindBytes(expectedBytes)
	require.NotNil(body)
	require.NotNil(getBody)

	actualBytes, err := ioutil.ReadAll(body)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)

	body2, err := getBody()
	assert.NoError(err)
	require.NotNil(body2)
	assert.True(body == body2)

	actualBytes, err = ioutil.ReadAll(body2)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
}

func testEnsureRewindableNoBody(t *testing.T) {
	var (
		assert = assert.New(t)
		r      = new(http.Request)
	)

	assert.NoError(EnsureRewindable(r))
	assert.Nil(r.Body)
	assert.Nil(r.GetBody)
}

func testEnsureRewindableGetBody(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		getBodyCalled = false
		getBody       = func() (io.ReadCloser, error) {
			getBodyCalled = true
			return nil, nil
		}

		r = &http.Request{
			GetBody: getBody,
		}
	)

	assert.NoError(EnsureRewindable(r))
	assert.Nil(r.Body)
	require.NotNil(r.GetBody)
	r.GetBody()
	assert.True(getBodyCalled)
}

func testEnsureRewindableBodyNotRewindable(t *testing.T) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		expectedContents = []byte{6, 253, 12, 34}

		r = &http.Request{
			Body: ioutil.NopCloser(bytes.NewReader(expectedContents)),
		}
	)

	assert.NoError(EnsureRewindable(r))

	require.NotNil(r.Body)
	actualContents, err := ioutil.ReadAll(r.Body)
	assert.Equal(expectedContents, actualContents)
	assert.NoError(err)

	require.NotNil(r.GetBody)
	actualBuffer, err := r.GetBody()
	require.NoError(err)
	require.NotNil(actualBuffer)
	actualContents, err = ioutil.ReadAll(actualBuffer)
	assert.Equal(expectedContents, actualContents)
	assert.NoError(err)
}

func testEnsureRewindableReadError(t *testing.T) {
	var (
		assert        = assert.New(t)
		contents      = new(mockReader)
		expectedBody  = ioutil.NopCloser(contents)
		expectedError = errors.New("expected")

		r = &http.Request{
			Body: expectedBody,
		}
	)

	contents.On("Read", mock.MatchedBy(func([]byte) bool { return true })).Return(0, expectedError).Once()
	assert.Equal(expectedError, EnsureRewindable(r))
	assert.Nil(r.GetBody)
	assert.True(expectedBody == r.Body)

	contents.AssertExpectations(t)
}

func TestEnsureRewindable(t *testing.T) {
	t.Run("NoBody", testEnsureRewindableNoBody)
	t.Run("GetBody", testEnsureRewindableGetBody)
	t.Run("BodyNotRewindable", testEnsureRewindableBodyNotRewindable)
	t.Run("ReadError", testEnsureRewindableReadError)
}

func testRewindGetBodyError(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")

		getBody = func() (io.ReadCloser, error) {
			return nil, expectedError
		}

		r = &http.Request{
			GetBody: getBody,
		}
	)

	assert.Equal(expectedError, Rewind(r))
}

func testRewindGetBodySuccess(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		expectedBytes = []byte{1, 7, 8, 5, 1, 16, 177}

		getBody = func() (io.ReadCloser, error) {
			return ioutil.NopCloser(bytes.NewReader(expectedBytes)), nil
		}

		r = &http.Request{
			GetBody: getBody,
		}
	)

	assert.NoError(Rewind(r))
	require.NotNil(r.Body)

	actualBytes, err := ioutil.ReadAll(r.Body)
	assert.Equal(expectedBytes, actualBytes)
	assert.NoError(err)
}

func testRewindNoBody(t *testing.T) {
	var (
		assert = assert.New(t)
		r      = new(http.Request)
	)

	assert.NoError(Rewind(r))
	assert.Nil(r.Body)
	assert.Nil(r.GetBody)
}

func testRewindCantRewind(t *testing.T) {
	var (
		assert = assert.New(t)
		r      = httptest.NewRequest("POST", "/", strings.NewReader("hi there"))
	)

	assert.Error(Rewind(r))
	assert.NotNil(r.Body)
	assert.Nil(r.GetBody)
}

func TestRewind(t *testing.T) {
	t.Run("GetBody", func(t *testing.T) {
		t.Run("Error", testRewindGetBodyError)
		t.Run("Success", testRewindGetBodySuccess)
	})

	t.Run("NoBody", testRewindNoBody)
	t.Run("CantRewind", testRewindCantRewind)
}
