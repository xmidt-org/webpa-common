package xcontext

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Comcast/webpa-common/xhttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPopulateNoDecoration(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		next http.Handler = xhttp.Constant{}

		constructor = Populate(-1)
	)

	require.NotNil(constructor)
	assert.Equal(next, constructor(next))
}

func testPopulate(t *testing.T, funcCount int) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		funcCalled = make([]bool, funcCount)
		funcs      = make([]ContextFunc, funcCount)

		nextCalled = false
		next       = http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			nextCalled = true
			deadline, ok := request.Context().Deadline()
			assert.False(deadline.IsZero())
			assert.True(ok)

			for i := 0; i < funcCount; i++ {
				assert.Equal(fmt.Sprintf("value-%d", i), request.Context().Value(fmt.Sprintf("key-%d", i)))
			}
		})

		response = httptest.NewRecorder()
		request  = httptest.NewRequest("GET", "/", nil)
	)

	for i := 0; i < funcCount; i++ {
		i := i
		funcs[i] = func(ctx context.Context, actual *http.Request) context.Context {
			funcCalled[i] = true
			assert.Equal(request, actual)
			return context.WithValue(ctx, fmt.Sprintf("key-%d", i), fmt.Sprintf("value-%d", i))
		}
	}

	constructor := Populate(100*time.Second, funcs...)
	require.NotNil(constructor)
	decorated := constructor(next)
	require.NotNil(decorated)

	decorated.ServeHTTP(response, request)
	assert.True(nextCalled)
}

func TestPopulate(t *testing.T) {
	t.Run("NoDecoration", testPopulateNoDecoration)

	for _, funcCount := range []int{0, 1, 2, 5} {
		t.Run(fmt.Sprintf("FuncCount=%d", funcCount), func(t *testing.T) {
			testPopulate(t, funcCount)
		})
	}
}
