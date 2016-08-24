package secure

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExactMatch(t *testing.T) {
	assert := assert.New(t)

	const tokenValue = "this value should validate"
	token, err := ParseAuthorization(fmt.Sprintf("Basic %s", tokenValue))
	if !assert.NotNil(token) || !assert.Nil(err) {
		assert.FailNow("ParseAuthorization failed: token=%v, err=%s", token, err)
	}

	successValidator := ValidateExactMatch(tokenValue)
	assert.NotNil(successValidator)

	valid, err := successValidator.Validate(token)
	assert.True(valid)
	assert.Nil(err)

	failureValidator := ValidateExactMatch("this should not be valid")
	assert.NotNil(failureValidator)

	valid, err = failureValidator.Validate(token)
	assert.False(valid)
	assert.Nil(err)
}

func TestVerify(t *testing.T) {
	// TODO: Not quite sure how to test this yet
}
