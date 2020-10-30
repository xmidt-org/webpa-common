package consul

import (
	"errors"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/xmidt-org/webpa-common/service"
)

func TestNewDatacenterrWatcher(t *testing.T) {
	logger := log.NewNopLogger()

	environment := environment{
		new(service.MockEnvironment), new(mockClient),
	}

	validOptions := Options{
		DatacenterWatchInterval: time.Duration(10 * time.Second),
	}

	tests := []struct {
		description     string
		logger          log.Logger
		environment     Environment
		options         Options
		expectedWatcher *DatacenterWatcher
		expectedErr     error
	}{
		{
			description: "Success",
			logger:      logger,
			environment: environment,
			options:     validOptions,
			expectedWatcher: &DatacenterWatcher{
				watchInterval: validOptions.DatacenterWatchInterval,
				logger:        logger,
				environment:   environment,
				options:       validOptions,
			},
		},
		{
			description: "Success with Default Logger",
			environment: environment,
			options:     validOptions,
			expectedWatcher: &DatacenterWatcher{
				watchInterval: validOptions.DatacenterWatchInterval,
				logger:        defaultLogger,
				environment:   environment,
				options:       validOptions,
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
			w, err := newDatacenterWatcher(testCase.logger, testCase.environment, testCase.options)

			if testCase.expectedErr == nil {
				assert.NotNil(w.shutdown)
				testCase.expectedWatcher.shutdown = w.shutdown
				assert.Equal(testCase.expectedWatcher, w)
			} else {
				assert.Equal(testCase.expectedErr, err)
			}

		})
	}

}
