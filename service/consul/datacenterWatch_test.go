package consul

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/argus/chrysom"
	"github.com/xmidt-org/argus/model"
	"github.com/xmidt-org/webpa-common/service"
	"github.com/xmidt-org/webpa-common/xmetrics/xmetricstest"
)

func TestNewDatacenterWatcher(t *testing.T) {
	logger := log.NewNopLogger()
	p := xmetricstest.NewProvider(nil, chrysom.Metrics)
	envShutdownChan := make(<-chan struct{})

	mockServiceEnvironment := new(service.MockEnvironment)
	mockServiceEnvironment.On("Provider").Return(p, true)
	mockServiceEnvironment.On("Closed").Return(envShutdownChan)

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
				DatacenterWatchInterval: time.Duration(10 * time.Second),
			},
			expectedWatcher: &datacenterWatcher{
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
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: time.Duration(5 * time.Minute),
					ChrysomConfig:           &validChrysomConfig,
				},
				consulDatacenterWatch: &consulDatacenterWatch{
					watchInterval: time.Duration(5 * time.Minute),
				},
				inactiveDatacenters: make(map[string]bool),
				chrysomClient:       &chrysom.Client{},
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
			expectedWatcher: &datacenterWatcher{
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
				chrysomClient: &chrysom.Client{},
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
			expectedWatcher: &datacenterWatcher{
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
			expectedWatcher: &datacenterWatcher{
				logger: logger,
				environment: environment{
					mockServiceEnvironment, new(mockClient),
				},
				options: Options{
					DatacenterWatchInterval: time.Duration(5 * time.Minute),
				},
				consulDatacenterWatch: &consulDatacenterWatch{
					watchInterval: time.Duration(5 * time.Minute),
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
			w, err := NewDatacenterWatcher(testCase.logger, testCase.environment, testCase.options)

			if testCase.expectedErr == nil {
				assert.NotNil(w.inactiveDatacenters)

				assert.NotNil(w.consulDatacenterWatch)
				assert.NotNil(w.consulDatacenterWatch.shutdown)
				assert.Equal(testCase.expectedWatcher.consulDatacenterWatch.watchInterval, w.consulDatacenterWatch.watchInterval)
				testCase.expectedWatcher.consulDatacenterWatch = w.consulDatacenterWatch

				if testCase.expectedWatcher.chrysomClient != nil {
					assert.NotNil(w.chrysomClient)
					testCase.expectedWatcher.chrysomClient = w.chrysomClient
				}

				assert.Equal(testCase.expectedWatcher, w)
			} else {
				assert.Equal(testCase.expectedErr, err)
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
					UUID:       "random-id",
					Identifier: "datacenters",
					Data: map[string]interface{}{
						"name":     "testDC1",
						"inactive": true,
					},
				},
				{
					UUID:       "random-id2",
					Identifier: "datacenters",
					Data: map[string]interface{}{
						"name":     "testDC2",
						"inactive": false,
					},
				},
				{
					UUID:       "random-id3",
					Identifier: "datacenters",
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
