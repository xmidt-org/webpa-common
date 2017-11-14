package fanout

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContext(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		ctx     = NewContext(context.Background(), "fanout request")
	)

	require.NotNil(ctx)
	v, ok := ctx.Value(fanoutRequestKey{}).(string)
	require.True(ok)
	assert.Equal("fanout request", v)
}

func TestFromContext(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
	)

	assert.Nil(FromContext(context.Background()))

	var (
		ctx   = context.WithValue(context.Background(), fanoutRequestKey{}, "fanout request")
		v, ok = FromContext(ctx).(string)
	)

	require.True(ok)
	assert.Equal("fanout request", v)
}

func TestFromContextEntity(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		request = new(mockRequest)
	)

	v, ok := FromContextEntity(context.Background())
	assert.Nil(v)
	assert.False(ok)

	request.On("Entity").Return("entity").Once()
	ctx := NewContext(context.Background(), request)
	require.NotNil(ctx)
	v, ok = FromContextEntity(ctx)
	require.True(ok)
	assert.Equal("entity", v)

	request.AssertExpectations(t)
}
