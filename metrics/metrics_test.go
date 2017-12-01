package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCounter(t *testing.T) {

	t.Run("DefaultGathererCounter", func(t *testing.T) {
		assert := assert.New(t)
		provider := &Provider{DefaultGathererInUse: true}
		counter := provider.GetCounter("name", "help", []string{})
		assert.NotNil(counter)
	})

	t.Run("CustomGathererCounter", func(t *testing.T) {
		assert := assert.New(t)
		provider := &Provider{DefaultGathererInUse: false} //setting value to be explicit``
		counter := provider.GetCounter("name", "help", []string{})
		assert.NotNil(counter)
	})
}
