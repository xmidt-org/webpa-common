package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewAccessorFactory(t *testing.T) {
	assert := assert.New(t)
	logger := logging.TestLogger(t)
	testData := []struct {
		endpoints    []string
		expectsError bool
	}{
		{
			nil,
			true,
		},
		{
			[]string{},
			true,
		},
		{
			[]string{"http://localhost:0101"},
			false,
		},
		{
			[]string{"https://host1.net:123", "http://host2.com:9293"},
			false,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)

		for _, vnodeCount := range []int{0, 200, 1700} {
			factory := NewAccessorFactory(&Options{Logger: logger, VnodeCount: vnodeCount})
			if !assert.NotNil(factory) {
				continue
			}

			accessor := factory.New(record.endpoints)
			if !assert.NotNil(accessor) {
				continue
			}

			endpoint, err := accessor.Get([]byte("key"))
			if record.expectsError {
				assert.Empty(endpoint)
				assert.Error(err)
			} else {
				assert.NotEmpty(endpoint)
				assert.NoError(err)
			}
		}
	}
}
