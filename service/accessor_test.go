package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		logger  = logging.TestLogger(t)

		testData = []struct {
			endpoints        []string
			expectedBaseURLs []string
			expectsError     bool
		}{
			{
				endpoints:        nil,
				expectedBaseURLs: make([]string, 0),
				expectsError:     true,
			},
			{
				endpoints:        []string{},
				expectedBaseURLs: make([]string, 0),
				expectsError:     true,
			},
			{
				endpoints:        []string{"[http://localhost]:7500"},
				expectedBaseURLs: []string{"http://localhost:7500"},
				expectsError:     false,
			},
			{
				endpoints:        []string{"[https://host1.net]:123", "[http://host2.com]:9293"},
				expectedBaseURLs: []string{"http://host2.com:9293", "https://host1.net:123"},
				expectsError:     false,
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)

		for _, vnodeCount := range []uint{0, 200, 1700} {
			factory := NewAccessorFactory(&Options{Logger: logger, VnodeCount: vnodeCount})
			if !assert.NotNil(factory) {
				continue
			}

			accessor, actualBaseURLs := factory.New(record.endpoints)
			require.NotNil(accessor)
			assert.Equal(record.expectedBaseURLs, actualBaseURLs)

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
