package gate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func testNewConstructorNilGate(t *testing.T) {
	assert := assert.New(t)
	assert.Panics(func() {
		NewConstructor(nil)
	})
}

func TestNewConstructor(t *testing.T) {
	t.Run("NilGate", testNewConstructorNilGate)
}
