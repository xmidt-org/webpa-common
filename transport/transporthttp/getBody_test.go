package transporthttp

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xmidt-org/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testGetBody(t *testing.T, expected []byte) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		ctx     = context.WithValue(context.Background(), "foo", "bar")
		getBody = GetBody(logging.NewTestLogger(nil, t))
		request = httptest.NewRequest("GET", "/", bytes.NewReader(expected))
	)

	require.NotNil(getBody)
	assert.Equal(ctx, getBody(ctx, request))

	require.NotNil(request.Body)
	reread, err := ioutil.ReadAll(request.Body)
	require.NoError(err)
	assert.Equal(expected, reread)
	reread, err = ioutil.ReadAll(request.Body)
	require.NoError(err)
	assert.Empty(reread)

	require.NotNil(request.GetBody)
	for repeat := 0; repeat < 2; repeat++ {
		reader, err := request.GetBody()
		require.NotNil(reader)
		require.NoError(err)
		data, err := ioutil.ReadAll(reader)
		require.NoError(err)
		assert.Equal(expected, data)
	}
}

func testGetBodyNilRequest(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		GetBody(logging.NewTestLogger(nil, t))(context.Background(), nil)
	})
}

func testGetBodyNilRequestBody(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() {
		GetBody(logging.NewTestLogger(nil, t))(context.Background(), new(http.Request))
	})
}

func TestGetBody(t *testing.T) {
	t.Run("EmptyBody", func(t *testing.T) {
		testGetBody(t, []byte{})
	})

	t.Run("NonemptyBody", func(t *testing.T) {
		testGetBody(t, []byte(`a;slkdfjas;dkjfqweo84yu6tphdfkahsep5t837546987ydfjkghsdkfj`))
	})

	t.Run("NilRequest", testGetBodyNilRequest)
	t.Run("NilRequestBody", testGetBodyNilRequestBody)
}
