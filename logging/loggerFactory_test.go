package logging

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDefaultLoggerFactory(t *testing.T) {
	assert := assert.New(t)

	loggerFactory := DefaultLoggerFactory{}
	product, err := loggerFactory.NewLogger("test")
	if assert.NotNil(product) {
		_, ok := product.(*LoggerWriter)
		assert.True(ok)
	}

	assert.Nil(err)
}

func TestDefaultLoggerFactoryCustomWriter(t *testing.T) {
	assert := assert.New(t)

	var output bytes.Buffer
	loggerFactory := DefaultLoggerFactory{&output}
	product, err := loggerFactory.NewLogger("test")
	if assert.NotNil(product) {
		logger, ok := product.(*LoggerWriter)
		assert.True(ok)

		logger.Debug("test output")
		assert.NotEmpty(output.Bytes())
	}

	assert.Nil(err)
}
