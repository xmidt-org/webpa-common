package convey

import (
	"bytes"
	"encoding/base64"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
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

func testTranslatorWriteTo(t *testing.T, encoding *base64.Encoding, source C, expected string) {
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
		})
	}
}
