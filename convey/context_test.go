package convey

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewContext(t *testing.T) {
	var (
		assert = assert.New(t)
		ctx    = NewContext(context.Background(), C{"foo": "bar"})
	)

	assert.Equal(C{"foo": "bar"}, ctx.Value(contextKey{}))
}

func TestFromContext(t *testing.T) {
	var (
		assert = assert.New(t)
		ctx    = context.Background()
	)

	v, ok := FromContext(ctx)
	assert.Empty(v)
	assert.False(ok)

	ctx = context.WithValue(ctx, contextKey{}, C{"foo": "bar"})
	v, ok = FromContext(ctx)
	assert.Equal(C{"foo": "bar"}, v)
	assert.True(ok)
}
