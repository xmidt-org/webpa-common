package httppool

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestRequestTask(t *testing.T) {
	assert := assert.New(t)

	if expected, err := http.NewRequest("GET", "http://example.com/", nil); assert.Nil(err) {
		task := RequestTask(expected)
		if actual, err := task(); assert.Nil(err) {
			assert.Equal(expected, actual)
		}
	}
}
