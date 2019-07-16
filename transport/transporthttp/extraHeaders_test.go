package transporthttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtraHeaders(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			actual   http.Header
			expected http.Header
		}{
			{nil, nil},
			{http.Header{}, http.Header{}},
			{
				http.Header{"Content-Type": []string{"application/json"}},
				http.Header{"Content-Type": []string{"application/json"}},
			},
			{
				http.Header{"content-type": []string{"application/json"}},
				http.Header{"Content-Type": []string{"application/json"}},
			},
			{
				http.Header{"content-type": []string{"application/json"}, "x-something": []string{"abc", "def"}},
				http.Header{"Content-Type": []string{"application/json"}, "X-Something": []string{"abc", "def"}},
			},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			rf      = ExtraHeaders(record.actual)
			ctx     = context.WithValue(context.Background(), "foo", "bar")
			request = httptest.NewRequest("GET", "/", nil)
		)

		for name, _ := range record.expected {
			request.Header[name] = []string{"SHOULD BE OVERWRITTEN"}
		}

		assert.Equal(ctx, rf(ctx, request))
		for name, values := range record.expected {
			assert.Equal(values, request.Header[name])
		}
	}
}
