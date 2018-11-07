package semaphore

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func testNewCloseableInvalidCount(t *testing.T) {
	for _, c := range []int{0, -1} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			assert.Panics(t, func() {
				NewCloseable(c)
			})
		})
	}
}

func testNewCloseableValidCount(t *testing.T) {
	for _, c := range []int{1, 2, 5} {
		t.Run(strconv.Itoa(c), func(t *testing.T) {
			s := NewCloseable(c)
			assert.NotNil(t, s)
		})
	}
}

func TestNewCloseable(t *testing.T) {
	t.Run("InvalidCount", testNewCloseableInvalidCount)
	t.Run("ValidCount", testNewCloseableValidCount)
}
