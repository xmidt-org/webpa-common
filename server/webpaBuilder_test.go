package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/types"
	"testing"
	"time"
)

func TestWebPABuilderConfiguration(t *testing.T) {
	var testData = []struct {
		builder                     WebPABuilder
		expectedPrimaryAddress      string
		expectedHealthAddress       string
		expectedHealthCheckInterval time.Duration
		expectedPprofAddress        string
	}{
		{
			builder:                     WebPABuilder{},
			expectedPrimaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			expectedHealthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			expectedHealthCheckInterval: DefaultHealthCheckInterval,
			expectedPprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{},
			},
			expectedPrimaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			expectedHealthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			expectedHealthCheckInterval: DefaultHealthCheckInterval,
			expectedPprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					Port: 2857,
				},
			},
			expectedPrimaryAddress:      ":2857",
			expectedHealthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			expectedHealthCheckInterval: DefaultHealthCheckInterval,
			expectedPprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					HealthCheckPort: 83,
				},
			},
			expectedPrimaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			expectedHealthAddress:       ":83",
			expectedHealthCheckInterval: DefaultHealthCheckInterval,
			expectedPprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					HealthCheckInterval: types.Duration(time.Hour * 5),
				},
			},
			expectedPrimaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			expectedHealthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			expectedHealthCheckInterval: time.Hour * 5,
			expectedPprofAddress:        fmt.Sprintf(":%d", DefaultPprofPort),
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					PprofPort: 2395,
				},
			},
			expectedPrimaryAddress:      fmt.Sprintf(":%d", DefaultPort),
			expectedHealthAddress:       fmt.Sprintf(":%d", DefaultHealthCheckPort),
			expectedHealthCheckInterval: DefaultHealthCheckInterval,
			expectedPprofAddress:        ":2395",
		},
		{
			builder: WebPABuilder{
				Configuration: &Configuration{
					Port:                1281,
					HealthCheckPort:     56001,
					HealthCheckInterval: types.Duration(time.Minute * 3412),
					PprofPort:           41508,
				},
			},
			expectedPrimaryAddress:      ":1281",
			expectedHealthAddress:       ":56001",
			expectedHealthCheckInterval: time.Minute * 3412,
			expectedPprofAddress:        ":41508",
		},
	}

	for _, record := range testData {
		actualPrimaryAddress := record.builder.PrimaryAddress()
		if record.expectedPrimaryAddress != actualPrimaryAddress {
			t.Errorf("Expected primary address %s, but got %s", record.expectedPrimaryAddress, actualPrimaryAddress)
		}

		actualHealthAddress := record.builder.HealthAddress()
		if record.expectedHealthAddress != actualHealthAddress {
			t.Errorf("Expected health address %s, but got %s", record.expectedHealthAddress, actualHealthAddress)
		}

		actualHealthCheckInterval := record.builder.HealthCheckInterval()
		if record.expectedHealthCheckInterval != actualHealthCheckInterval {
			t.Errorf("Expected health check interval %s, but got %s", record.expectedHealthCheckInterval, actualHealthCheckInterval)
		}

		actualPprofAddress := record.builder.PprofAddress()
		if record.expectedPprofAddress != actualPprofAddress {
			t.Errorf("Expected pprof address %s, but got %s", record.expectedPprofAddress, actualPprofAddress)
		}
	}
}
