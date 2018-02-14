package logging

import (
	"context"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithLogger(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
		ctx     = WithLogger(context.Background(), DefaultLogger())
	)

	require.NotNil(ctx)

	logger, ok := ctx.Value(loggerKey).(log.Logger)
	assert.NotNil(logger)
	assert.True(ok)
}

func testFromContextMissing(t *testing.T) {
	assert := assert.New(t)
	assert.NotNil(FromContext(context.Background()))
}

func testFromContextPresent(t *testing.T) {
	var (
		require = require.New(t)
		assert  = assert.New(t)
		ctx     = WithLogger(context.Background(), New(nil))
	)

	require.NotNil(ctx)
	assert.NotNil(FromContext(ctx))
}

func TestFromContext(t *testing.T) {
	t.Run("Missing", testFromContextMissing)
	t.Run("Present", testFromContextPresent)
}
