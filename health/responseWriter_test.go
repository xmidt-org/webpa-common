package health

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestResponseWriter(t *testing.T) {
	var (
		assert    = assert.New(t)
		delegate  = new(mockResponseWriter)
		composite = Wrap(delegate)
	)

	delegate.On("WriteHeader", 200).Once()

	assert.Equal(0, composite.StatusCode())
	composite.WriteHeader(200)
	assert.Equal(200, composite.StatusCode())

	delegate.AssertExpectations(t)
}
