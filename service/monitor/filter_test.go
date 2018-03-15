package monitor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNopFilter(t *testing.T) {
	assert := assert.New(t)

	assert.NotPanics(func() {
		assert.Equal(
			[]string{"instance"},
			NopFilter([]string{"instance"}),
		)
	})
}

func testFilterNoInstances(t *testing.T, f Filter) {
	assert := assert.New(t)

	assert.Len(f(nil), 0)
	assert.Len(f([]string{}), 0)
}

func testFilterWithInstances(t *testing.T, f Filter) {
	var (
		assert = assert.New(t)

		original = []string{
			" \t   ",
			"localhost:8080",
			"https://foobar.com",
			"http://asdf.net:1234",
			"",
		}
	)

	filtered := f(original)
	assert.Equal(
		[]string{
			"http://asdf.net:1234",
			"https://foobar.com",
			"https://localhost:8080",
		},
		filtered,
	)
}

func TestNewNormalizeFilter(t *testing.T) {
	t.Run("NoInstances", func(t *testing.T) {
		testFilterNoInstances(t, NewNormalizeFilter(""))
	})

	t.Run("WithInstances", func(t *testing.T) {
		testFilterWithInstances(t, NewNormalizeFilter(""))
	})
}

func TestDefaultFilter(t *testing.T) {
	t.Run("NoInstances", func(t *testing.T) {
		testFilterNoInstances(t, DefaultFilter())
	})

	t.Run("WithInstances", func(t *testing.T) {
		testFilterWithInstances(t, DefaultFilter())
	})
}
