package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultInstancesFilter(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			original []string
			expected []string
		}{
			{nil, []string{}},
			{[]string{"", " abc.com:1212", "\t ", "def.net:8080  "}, []string{"abc.com:1212", "def.net:8080"}},
			{[]string{"qrstuv.net", "", "  abc.com\t", "xyz.foo.net\r\n"}, []string{"abc.com", "qrstuv.net", "xyz.foo.net"}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		assert.Equal(record.expected, DefaultInstancesFilter(record.original))
	}
}

func TestConsistentAccessorFactory(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			vnodeCount int
			instances  []string
		}{
			{123, []string{}},
			{0, []string{"abc.com"}},
			{-47, []string{"abc.com"}},
			{234, []string{"abc.com", "def.com"}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			factory  = ConsistentAccessorFactory(record.vnodeCount)
			accessor = factory(record.instances)
		)

		key, err := accessor.Get([]byte("random key"))
		if len(record.instances) > 0 {
			assert.Contains(record.instances, key)
		} else {
			assert.Error(err)
		}
	}
}
