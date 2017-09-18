package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/go-kit/kit/endpoint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFanoutNilSpanner(t *testing.T) {
	var (
		assert = assert.New(t)
		dummy  = func(context.Context, interface{}) (interface{}, error) {
			assert.Fail("The endpoint should not have been called")
			return nil, nil
		}
	)

	assert.Panics(func() {
		Fanout(nil, map[string]endpoint.Endpoint{"test": dummy})
	})
}

func testFanoutNoConfiguredEndpoints(t *testing.T) {
	assert := assert.New(t)
	for _, empty := range []map[string]endpoint.Endpoint{nil, map[string]endpoint.Endpoint{}} {
		assert.Panics(func() {
			Fanout(tracing.NewSpanner(), empty)
		})
	}
}

func testFanoutSuccessFirst(t *testing.T, serviceCount int) {
	var (
		require             = require.New(t)
		assert              = assert.New(t)
		expectedCtx, cancel = context.WithCancel(
			logging.WithLogger(context.Background(), logging.NewTestLogger(nil, t)),
		)

		expectedRequest  = "expectedRequest"
		expectedResponse = new(tracing.NopMergeable)

		endpoints   = make(map[string]endpoint.Endpoint, serviceCount)
		success     = make(chan string, 1)
		failureGate = make(chan struct{})
	)

	for i := 0; i < serviceCount; i++ {
		if i == 0 {
			endpoints["success"] = func(ctx context.Context, request interface{}) (interface{}, error) {
				assert.Equal(expectedCtx, ctx)
				assert.Equal(expectedRequest, request)
				success <- "success"
				return expectedResponse, nil
			}
		} else {
			endpoints[fmt.Sprintf("failure#%d", i)] = func(ctx context.Context, request interface{}) (interface{}, error) {
				assert.Equal(expectedCtx, ctx)
				assert.Equal(expectedRequest, request)
				<-failureGate
				return nil, fmt.Errorf("expected failure #%d", i)
			}
		}
	}

	fanout := Fanout(tracing.NewSpanner(), endpoints)
	require.NotNil(fanout)

	response, err := fanout(expectedCtx, expectedRequest)
	assert.NoError(err)
	require.NotNil(response)
	assert.Equal("success", <-success)

	assert.Equal(1, len(response.(tracing.Spanned).Spans()))
	close(failureGate)
	cancel()
}

func TestFanout(t *testing.T) {
	t.Run("NoConfiguredEndpoints", testFanoutNoConfiguredEndpoints)
	t.Run("NilSpanner", testFanoutNilSpanner)

	t.Run("SuccessFirst", func(t *testing.T) {
		for c := 1; c <= 5; c++ {
			t.Run(fmt.Sprintf("EndpointCount=%d", c), func(t *testing.T) {
				testFanoutSuccessFirst(t, c)
			})
		}
	})
}
