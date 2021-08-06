package conveyhttp

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/webpa-common/v2/convey"
)

func testHeaderTranslatorFromHeader(t *testing.T, actualHeaderName, expectedHeaderName string, actualTranslator, expectedTranslator convey.Translator) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		header           = make(http.Header)
		headerTranslator = NewHeaderTranslator(actualHeaderName, actualTranslator)
	)

	c, err := headerTranslator.FromHeader(header)
	assert.Empty(c)
	assert.Error(err)

	value, err := convey.WriteString(expectedTranslator, convey.C{"foo": "bar"})
	require.NotEmpty(value)
	require.NoError(err)

	header.Set(expectedHeaderName, value)
	c, err = headerTranslator.FromHeader(header)
	assert.Equal(convey.C{"foo": "bar"}, c)
	assert.NoError(err)

	header.Add(expectedHeaderName, "something invalid")
	c, err = headerTranslator.FromHeader(header)
	assert.Equal(convey.C{"foo": "bar"}, c)
	assert.NoError(err)
}

func testHeaderTranslatorToHeader(t *testing.T, actualHeaderName, expectedHeaderName string, actualTranslator, expectedTranslator convey.Translator) {
	var (
		assert           = assert.New(t)
		require          = require.New(t)
		header           = make(http.Header)
		headerTranslator = NewHeaderTranslator(actualHeaderName, actualTranslator)
	)

	require.NoError(headerTranslator.ToHeader(header, convey.C{"foo": "bar"}))

	value := header.Get(expectedHeaderName)
	require.NotEmpty(value)

	c, err := convey.ReadString(expectedTranslator, value)
	assert.Equal(convey.C{"foo": "bar"}, c)
	assert.NoError(err)
}

func TestHeaderTranslator(t *testing.T) {
	t.Run("FromHeader", func(t *testing.T) {
		testHeaderTranslatorFromHeader(
			t,
			"",
			DefaultHeaderName,
			nil,
			convey.NewTranslator(nil),
		)

		testHeaderTranslatorFromHeader(
			t,
			"Some-Header",
			"Some-Header",
			nil,
			convey.NewTranslator(nil),
		)

		testHeaderTranslatorFromHeader(
			t,
			"",
			DefaultHeaderName,
			convey.NewTranslator(base64.RawURLEncoding),
			convey.NewTranslator(base64.RawURLEncoding),
		)

		testHeaderTranslatorFromHeader(
			t,
			"Another-Header",
			"Another-Header",
			convey.NewTranslator(base64.URLEncoding),
			convey.NewTranslator(base64.URLEncoding),
		)
	})

	t.Run("ToHeader", func(t *testing.T) {
		testHeaderTranslatorToHeader(
			t,
			"",
			DefaultHeaderName,
			nil,
			convey.NewTranslator(nil),
		)

		testHeaderTranslatorToHeader(
			t,
			"Some-Header",
			"Some-Header",
			nil,
			convey.NewTranslator(nil),
		)

		testHeaderTranslatorToHeader(
			t,
			"",
			DefaultHeaderName,
			convey.NewTranslator(base64.RawURLEncoding),
			convey.NewTranslator(base64.RawURLEncoding),
		)

		testHeaderTranslatorToHeader(
			t,
			"Another-Header",
			"Another-Header",
			convey.NewTranslator(base64.URLEncoding),
			convey.NewTranslator(base64.URLEncoding),
		)
	})
}
