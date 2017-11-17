package fanouthttp

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/Comcast/webpa-common/middleware/fanout"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyHeaders(t *testing.T) {
	var (
		assert        = assert.New(t)
		require       = require.New(t)
		original      = httptest.NewRequest("GET", "/foo/bar", nil)
		fanoutRequest = &fanoutRequest{
			original: original,
		}

		component   = httptest.NewRequest("GET", "/", nil)
		copyHeaders = CopyHeaders("X-Scalar", "x-multi")
	)

	require.NotNil(copyHeaders)

	original.Header.Set("X-NotCopied", "something")
	original.Header.Set("X-Scalar", "1234")
	original.Header.Add("X-Multi", "value1")
	original.Header.Add("X-Multi", "value2")

	ctx := fanout.NewContext(context.Background(), fanoutRequest)
	assert.Equal(ctx, copyHeaders(ctx, component))

	assert.Empty(component.Header.Get("X-NotCopied"))
	assert.Equal("1234", component.Header.Get("X-Scalar"))
	assert.Equal([]string{"value1", "value2"}, component.Header["X-Multi"])
}
