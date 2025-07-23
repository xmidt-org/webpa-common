package consul

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/v2/adapter"
	"github.com/xmidt-org/webpa-common/v2/service"
	"go.uber.org/zap"
)

func TestNewDatacenterWatcher(t *testing.T) {
	logger := adapter.DefaultLogger().Logger
	envShutdownChan := make(<-chan struct{})

	mockServiceEnvironment := new(service.MockEnvironment)
	mockServiceEnvironment.On("Closed").Return(envShutdownChan)

	tests := []struct {
		description     string
		logger          *zap.Logger
		environment     Environment
		options         Options
		ctx             context.Context
		expectedWatcher *datacenterWatcher
		expectedErr     error
	}{

		{
			description: "Successful Consul Datacenter Watcher",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 10 * time.Second,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 10 * time.Second,
				},
				consulWatchInterval: 10 * time.Second,
			},
		},
		{
			description: "Successful Consul",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 10 * time.Second,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 10 * time.Second,
				},
				consulWatchInterval: 10 * time.Second,
			},
		},
		{
			description: "Success with Default Logger",
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 10 * time.Second,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 10 * time.Second,
				},
				consulWatchInterval: 10 * time.Second,
			},
		},
		{
			description: "Default Consul Watch Interval",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 0,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: defaultWatchInterval,
				},
				consulWatchInterval: defaultWatchInterval,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			w, err := newDatacenterWatcher(tc.logger, tc.environment, tc.options)

			if tc.expectedErr == nil {
				assert.Equal(tc.expectedWatcher.consulWatchInterval, w.consulWatchInterval)
				assert.Equal(tc.expectedWatcher.logger, w.logger)
				assert.Equal(tc.expectedWatcher.environment, w.environment)
				assert.Equal(tc.expectedWatcher.options, w.options)
			} else {
				assert.Equal(tc.expectedErr, err)
			}

		})
	}

}
