package sessionid

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	assert := assert.New(t)
	now := time.Now()
	id := GenerateIDWithTime(now)
	assert.NotEmpty(id)

	ts, err := ParseID(id)
	assert.NoError(err)
	assert.Equal(ts.Unix(), now.Unix())

	_, err = ParseID("")
	assert.Error(err)

	_, err = ParseID("XjS1ECGCZU8WP18PmmIdc")
	assert.Error(err)
}
