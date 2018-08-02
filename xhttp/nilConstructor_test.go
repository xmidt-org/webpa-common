package xhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNilConstructor(t *testing.T) {
	var (
		assert = assert.New(t)
		next   = Constant{}
	)

	assert.Nil(NilConstructor(nil))
	assert.Equal(next, NilConstructor(next))
}
