package convey

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func testTranslatorReadFrom(t *testing.T, encoding *base64.Encoding, source io.Reader, expected C) {
	var (
		assert     = assert.New(t)
		require    = require.New(t)
		translator = NewTranslator(encoding)
	)

	actual, err := translator.ReadFrom(source)
	require.NoError(err)
	assert.Equal(expected, actual)
}

func testTranslatorWriteTo(t *testing.T, translatorEncoding, expectedEncoding *base64.Encoding, source C, expected string) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		output     bytes.Buffer
		translator = NewTranslator(translatorEncoding)
	)

	require.NoError(translator.WriteTo(&output, source))

	decoder := base64.NewDecoder(expectedEncoding, &output)
	actual, err := ioutil.ReadAll(decoder)
	require.NoError(err)
	assert.NotEmpty(actual)
	assert.JSONEq(expected, string(actual))
}

func TestTranslator(t *testing.T) {
	encodingLabels := map[*base64.Encoding]string{
		nil:                   "NilEncoding",
		base64.StdEncoding:    "StdEncoding",
		base64.RawStdEncoding: "RawStdEncoding",
		base64.URLEncoding:    "URLEncoding",
		base64.RawURLEncoding: "RawURLEncoding",
	}

	testData := []struct {
		json   string
		convey C
	}{
		{
			`{}`,
			C{},
		},
		{
			`{"foo": "bar"}`,
			C{"foo": "bar"},
		},
		{
			`{"foo": "bar", "nested": {"value": 57234, "name": "syzygy"}}`,
			C{"foo": "bar", "nested": C{"value": uint64(57234), "name": "syzygy"}},
		},
	}

	for encoding, label := range encodingLabels {
		expectedEncoding := encoding
		if expectedEncoding == nil {
			expectedEncoding = base64.StdEncoding
		}

		t.Run(label, func(t *testing.T) {
			t.Run("ReadFrom", func(t *testing.T) {
				for _, record := range testData {
					testTranslatorReadFrom(
						t,
						encoding,
						bytes.NewBufferString(
							expectedEncoding.EncodeToString([]byte(record.json)),
						),
						record.convey,
					)
				}
			})

			t.Run("WriteTo", func(t *testing.T) {
				for _, record := range testData {
					testTranslatorWriteTo(
						t,
						encoding,
						expectedEncoding,
						record.convey,
						record.json,
					)
				}
			})
		})
	}
}

func TestReadString(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")

		translator = new(mockTranslator)
	)

	translator.On("ReadFrom", mock.MatchedBy(func(io.Reader) bool { return true })).
		Return(C{"key": "value"}, expectedError).
		Run(func(arguments mock.Arguments) {
			reader := arguments.Get(0).(io.Reader)
			actual, err := ioutil.ReadAll(reader)
			assert.Equal("expected", string(actual))
			assert.NoError(err)
		}).
		Once()

	actual, actualError := ReadString(translator, "expected")
	assert.Equal(C{"key": "value"}, actual)
	assert.Equal(expectedError, actualError)

	translator.AssertExpectations(t)
}

func TestReadBytes(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedError = errors.New("expected")

		translator = new(mockTranslator)
	)

	translator.On("ReadFrom", mock.MatchedBy(func(io.Reader) bool { return true })).
		Return(C{"key": "value"}, expectedError).
		Run(func(arguments mock.Arguments) {
			reader := arguments.Get(0).(io.Reader)
			actual, err := ioutil.ReadAll(reader)
			assert.Equal("expected", string(actual))
			assert.NoError(err)
		}).
		Once()

	actual, actualError := ReadBytes(translator, []byte("expected"))
	assert.Equal(C{"key": "value"}, actual)
	assert.Equal(expectedError, actualError)

	translator.AssertExpectations(t)
}

func TestWriteString(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedC     = C{"test": true}
		expectedError = errors.New("expected")

		translator = new(mockTranslator)
	)

	translator.On("WriteTo", mock.MatchedBy(func(io.Writer) bool { return true }), expectedC).
		Return(expectedError).
		Run(func(arguments mock.Arguments) {
			arguments.Get(0).(io.Writer).Write([]byte("expected"))
		}).
		Once()

	actual, actualError := WriteString(translator, expectedC)
	assert.Equal("expected", actual)
	assert.Equal(expectedError, actualError)

	translator.AssertExpectations(t)
}

func TestWriteBytes(t *testing.T) {
	var (
		assert        = assert.New(t)
		expectedC     = C{"test": true}
		expectedError = errors.New("expected")

		translator = new(mockTranslator)
	)

	translator.On("WriteTo", mock.MatchedBy(func(io.Writer) bool { return true }), expectedC).
		Return(expectedError).
		Run(func(arguments mock.Arguments) {
			arguments.Get(0).(io.Writer).Write([]byte("expected"))
		}).
		Once()

	actual, actualError := WriteBytes(translator, expectedC)
	assert.Equal("expected", string(actual))
	assert.Equal(expectedError, actualError)

	translator.AssertExpectations(t)
}
