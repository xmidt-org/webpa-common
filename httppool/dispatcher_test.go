package httppool

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestRequestTask(t *testing.T) {
	assert := assert.New(t)

	if expectedRequest, err := http.NewRequest("GET", "http://example.com/", nil); assert.Nil(err) {
		consumerCalled := false
		expectedConsumer := Consumer(func(*http.Response, *http.Request) {
			consumerCalled = true
		})

		task := RequestTask(expectedRequest, expectedConsumer)
		if actualRequest, actualConsumer, err := task(); assert.Nil(err) {
			assert.Equal(expectedRequest, actualRequest)
			assert.NotNil(actualConsumer)
			actualConsumer(nil, nil)
			assert.True(consumerCalled)
		}
	}
}
