package consul

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/v2/service"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
)

func TestNewDatacenterWatcher(t *testing.T) {
	logger := log.NewNopLogger()
	r, err := xmetrics.NewRegistry(nil, Metrics)
	require.Nil(t, err)
	envShutdownChan := make(<-chan struct{})

	mockServiceEnvironment := new(service.MockEnvironment)
	mockServiceEnvironment.On("Provider").Return(r, true)
	mockServiceEnvironment.On("Closed").Return(envShutdownChan)

	noProviderEnv := new(service.MockEnvironment)
	noProviderEnv.On("Provider").Return(nil, false)

	validChrysomConfig := ChrysomConfig{
		BasicClientConfig: chrysom.BasicClientConfig{
			Bucket:  "random-bucket",
			Address: "http://argus:6600",
			Auth: chrysom.Auth{
				Basic: "Basic auth",
			},
		},
		Listen: chrysom.ListenerClientConfig{
			PullInterval: 10 * time.Second,
		},
	}

	nopStop := func(_ context.Context) error { return nil }

	tests := []struct {
		description     string
		logger          log.Logger
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
				inactiveDatacenters: make(map[string]bool),
				consulWatchInterval: 10 * time.Second,
			},
		},
		{
			description: "Empty Chrysom Client Bucket",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				Chrysom: ChrysomConfig{
					BasicClientConfig: chrysom.BasicClientConfig{
						Bucket: "",
					},
				},
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: defaultWatchInterval,
				},
				inactiveDatacenters: make(map[string]bool),
				consulWatchInterval: defaultWatchInterval,
			},
		},
		{
			description: "Successful Chrysom Client",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				Chrysom: validChrysomConfig,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: defaultWatchInterval,
					Chrysom:                 validChrysomConfig,
				},
				consulWatchInterval: defaultWatchInterval,
				inactiveDatacenters: make(map[string]bool),
				stopListener:        nopStop,
			},
		},
		{
			description: "Successful Consul and Chrysom Datacenter Watcher",
			logger:      logger,
			environment: environment{
				mockServiceEnvironment, new(mockClient),
			},
			options: Options{
				DatacenterWatchInterval: 10 * time.Second,
				Chrysom:                 validChrysomConfig,
			},
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 10 * time.Second,
					Chrysom:                 validChrysomConfig,
				},
				inactiveDatacenters: make(map[string]bool),
				consulWatchInterval: 10 * time.Second,
				stopListener:        nopStop,
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
				logger: log.NewNopLogger(),
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: 10 * time.Second,
				},
				consulWatchInterval: 10 * time.Second,
				inactiveDatacenters: make(map[string]bool),
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
				Chrysom: validChrysomConfig,
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
				Chrysom: ChrysomConfig{
					BasicClientConfig: chrysom.BasicClientConfig{
						Bucket:  "random-bucket",
						Address: "http://argus:6600",
						Auth: chrysom.Auth{
							Basic: "Basic auth",
						},
					},
					Listen: chrysom.ListenerClientConfig{
						PullInterval: 0,
					},
				},
			},
			expectedErr: errors.New("chrysom pull interval cannot be 0"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			w, err := newDatacenterWatcher(tc.logger, tc.environment, tc.options)

			if tc.expectedErr == nil {
				assert.NotNil(w.inactiveDatacenters)
				assert.Equal(tc.expectedWatcher.consulWatchInterval, w.consulWatchInterval)
				assert.Equal(tc.expectedWatcher.logger, w.logger)
				assert.Equal(tc.expectedWatcher.environment, w.environment)
				assert.Equal(tc.expectedWatcher.options, w.options)
				if tc.expectedWatcher.stopListener != nil {
					assert.NotNil(w.stopListener)
				}
			} else {
				assert.Equal(tc.expectedErr, err)
			}

		})
	}

}

func TestUpdateInactiveDatacenters(t *testing.T) {
	logger := log.NewNopLogger()

	tests := []struct {
		description                 string
		items                       []model.Item
		currentInactiveDatacenters  map[string]bool
		expectedInactiveDatacenters map[string]bool
		lock                        sync.RWMutex
	}{
		{
			description:                 "Empty database results, empty inactive datacenters",
			items:                       []model.Item{},
			currentInactiveDatacenters:  map[string]bool{},
			expectedInactiveDatacenters: map[string]bool{},
		},
		{
			description: "Empty database results, non-empty inactive datacenters",
			items:       []model.Item{},
			currentInactiveDatacenters: map[string]bool{
				"testDC": true,
			},
			expectedInactiveDatacenters: map[string]bool{},
		},
		{
			description: "Non-Empty Database Results",
			items: []model.Item{
				{
					ID: "random-id",
					Data: map[string]interface{}{
						"name":     "testDC1",
						"inactive": true,
					},
				},
				{
					ID: "random-id2",
					Data: map[string]interface{}{
						"name":     "testDC2",
						"inactive": false,
					},
				},
				{
					ID: "random-id3",
					Data: map[string]interface{}{
						"name":     "testDC3",
						"inactive": true,
					},
				},
			},
			currentInactiveDatacenters: map[string]bool{
				"testDC2": true,
			},
			expectedInactiveDatacenters: map[string]bool{
				"testDC1": true,
				"testDC3": true,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			assert := assert.New(t)
			updateInactiveDatacenters(tc.items, tc.currentInactiveDatacenters, &tc.lock, logger)
			assert.Equal(tc.expectedInactiveDatacenters, tc.currentInactiveDatacenters)

		})
	}
}
