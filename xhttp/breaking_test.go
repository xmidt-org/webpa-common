package xhttp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBreaking(t *testing.T) {
	assert.Fail(t, "This should break the Travis CI build")
}
