package consul

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
)

func TestNewDatacenterWatcher(t *testing.T) {
	logger := log.NewNopLogger()
	p := xmetricstest.NewProvider(nil, chrysom.Metrics)

	mockServiceEnvironment := new(service.MockEnvironment)
	mockServiceEnvironment.On("Provider").Return(p, true)

	noProviderEnv := new(service.MockEnvironment)
	noProviderEnv.On("Provider").Return(nil, false)

	validChrysomConfig := chrysom.ClientConfig{
		Bucket:       "random-bucket",
		PullInterval: time.Duration(10 * time.Second),
		Address:      "http://argus:6600",
		AdminToken:   "admin-token",
		Auth: chrysom.Auth{
			Basic: "Basic auth",
		},
		Logger: logger,
	}

	tests := []struct {
		description     string
		logger          log.Logger
		environment     Environment
		options         Options
		ctx             context.Context
		expectedWatcher *DatacenterWatcher
		expectedErr     error
	}{

		{
			description: "Successful Consul Datacenter Watcher",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: time.Duration(10 * time.Second),
			},
			expectedWatcher: &DatacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: time.Duration(10 * time.Second),
				},
				inactiveDatacenters: make(map[string]bool),
				consulDatacenterWatch: &consulDatacenterWatch{
					watchInterval: time.Duration(10 * time.Second),
				},
			},
		},
		{
			description: "Successful Chrysom Datacenter Watcher",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				ChrysomConfig: &validChrysomConfig,
			},
			expectedWatcher: &DatacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					ChrysomConfig: &validChrysomConfig,
				},
				inactiveDatacenters:    make(map[string]bool),
				chrysomDatacenterWatch: &chrysomDatacenterWatch{},
			},
		},
		{
			description: "Successful Consul and Chrysom Datacenter Watcher",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: time.Duration(10 * time.Second),
				ChrysomConfig:           &validChrysomConfig,
			},
			expectedWatcher: &DatacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: time.Duration(10 * time.Second),
					ChrysomConfig:           &validChrysomConfig,
				},
				inactiveDatacenters: make(map[string]bool),
				consulDatacenterWatch: &consulDatacenterWatch{
					watchInterval: time.Duration(10 * time.Second),
				},
				chrysomDatacenterWatch: &chrysomDatacenterWatch{},
			},
		},
		{
			description: "Success with Default Logger",
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: time.Duration(10 * time.Second),
			},
			expectedWatcher: &DatacenterWatcher{
				logger: defaultLogger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: time.Duration(10 * time.Second),
				},
				consulDatacenterWatch: &consulDatacenterWatch{
					watchInterval: time.Duration(10 * time.Second),
				},
				inactiveDatacenters: make(map[string]bool),
			},
		},
		{
			description: "0 Datacenter Watch Interval",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 0,
			},
			expectedWatcher: &DatacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 0,
				},
				inactiveDatacenters: make(map[string]bool),
			},
		},
		{
			description: "No Provider",
			logger:      logger,
			environment: environment{
				noProviderEnv, new(mockClient),
			},
			options: Options{
				ChrysomConfig: &validChrysomConfig,
			},
			expectedErr: errors.New("must pass in a metrics provider"),
		},
		{
			description: "Invalid chrysom watcher interval",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				ChrysomConfig: &chrysom.ClientConfig{
					Bucket:       "random-bucket",
					PullInterval: 0,
					Address:      "http://argus:6600",
					Auth: chrysom.Auth{
						Basic: "Basic auth",
					},
					Logger: logger,
				},
			},
			expectedErr: errors.New("chrysom pull interval cannot be 0"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.description, func(t *testing.T) {
			assert := assert.New(t)
			w, err := NewDatacenterWatcher(testCase.logger, testCase.environment, testCase.options, testCase.ctx)

			if testCase.expectedErr == nil {
				assert.NotNil(w.inactiveDatacenters)

				if testCase.expectedWatcher.consulDatacenterWatch != nil {
					assert.NotNil(w.consulDatacenterWatch)
					assert.NotNil(w.consulDatacenterWatch.shutdown)
					assert.Equal(testCase.expectedWatcher.consulDatacenterWatch.watchInterval, w.consulDatacenterWatch.watchInterval)
					testCase.expectedWatcher.consulDatacenterWatch = w.consulDatacenterWatch
				}

				if testCase.expectedWatcher.chrysomDatacenterWatch != nil {
					assert.NotNil(w.chrysomDatacenterWatch)
					assert.NotNil(w.chrysomDatacenterWatch.chrysomClient)
					assert.NotNil(w.chrysomDatacenterWatch.ctx)
					testCase.expectedWatcher.chrysomDatacenterWatch = w.chrysomDatacenterWatch
				}

				assert.Equal(testCase.expectedWatcher, w)
			} else {
				assert.Equal(testCase.expectedErr, err)
			}

		})
	}

}

func TestDatacentersListener(t *testing.T) {

}
