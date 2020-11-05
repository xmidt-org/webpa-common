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
)

func TestNewDatacenterWatcher(t *testing.T) {
	logger := log.NewNopLogger()

	environment := environment{
		new(service.MockEnvironment), new(mockClient),
	}

	chrysomConfig := chrysom.ClientConfig{}

	validOptions := Options{
		DatacenterWatchInterval: time.Duration(10 * time.Second),
		ChrysomConfig:           &chrysomConfig,
	}

	tests := []struct {
		description            string
		logger                 log.Logger
		environment            Environment
		options                Options
		ctx                    context.Context
		expectedWatcher        *DatacenterWatcher
		expectedConsulWatcher  *consulDatacenterWatch
		expectedChrysomWatcher *chrysomDatacenterWatch
		expectedErr            error
	}{
		//Add more test cases:
		// Valid chrysom and consul watch
		// valid chrysom only
		// consul watch only
		// invalid chrysom interval
		//in argus but not in consul catalog

		{
			description: "Success",
			logger:      logger,
			environment: environment,
			options:     validOptions,
			expectedWatcher: &DatacenterWatcher{
				logger:      logger,
				environment: environment,
				options:     validOptions,
			},
		},
		{
			description: "Success with Default Logger",
			environment: environment,
			options:     validOptions,
			expectedWatcher: &DatacenterWatcher{
				logger:      defaultLogger,
				environment: environment,
				options:     validOptions,
			},
		},
		{
			description: "Invalid interval",
			logger:      logger,
			environment: environment,
			options: Options{
				DatacenterWatchInterval: 0,
			},
			expectedErr: errors.New("interval cannot be 0"),
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.description, func(t *testing.T) {
			assert := assert.New(t)
			w, err := NewDatacenterWatcher(testCase.logger, testCase.environment, testCase.options, testCase.ctx)

			if testCase.expectedErr == nil {
				assert.NotNil(w.inactiveDatacenters)

				if testCase.expectedConsulWatcher != nil {
					assert.NotNil(w.consulDatacenterWatch)
					assert.NotNil(w.consulDatacenterWatch.shutdown)
					assert.Equal(testCase.expectedConsulWatcher.watchInterval, w.consulDatacenterWatch.watchInterval)
					testCase.expectedWatcher.consulDatacenterWatch = w.consulDatacenterWatch
				}

				assert.Equal(testCase.expectedWatcher, w)
			} else {
				assert.Equal(testCase.expectedErr, err)
			}

		})
	}

}
