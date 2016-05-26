package secure

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExactMatch(t *testing.T) {
	assert := assert.New(t)
	token := &Token{Basic, "dXNlcjpwYXNzd29yZA=="}

	{
		shouldMatch := ExactMatch("dXNlcjpwYXNzd29yZA==")
		matched, err := shouldMatch.Validate(token)
		assert.Equal(token, matched)
		assert.Nil(err)
	}

	{
		shouldNotMatch := ExactMatch("huh?")
		matched, err := shouldNotMatch.Validate(token)
		assert.Equal(token, matched)
		assert.NotNil(err)
	}
}

func TestVerify(t *testing.T) {
	// TODO: Not quite sure how to test this yet
}
