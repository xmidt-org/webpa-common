package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		assert.NotNil(o.logger())
		assert.Equal([]string{DefaultZookeeper}, o.zookeepers())
		assert.Equal(DefaultZookeeperTimeout, o.zookeeperTimeout())
	}
}
