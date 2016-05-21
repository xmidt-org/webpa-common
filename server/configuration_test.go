package server

import (
	"fmt"
	"github.com/Comcast/webpa-common/types"
	"testing"
	"time"
)

var (
	defaultPrimaryAddress     string = fmt.Sprintf(":%d", DefaultPort)
	defaultHealthCheckAddress string = fmt.Sprintf(":%d", DefaultHealthCheckPort)
	defaultPprofAddress       string = fmt.Sprintf(":%d", DefaultPprofPort)
)

type configurationExpect struct {
	primaryAddress      string
	healthAddress       string
	healthCheckInterval time.Duration
	pprofAddress        string
}

func (expect *configurationExpect) assert(t *testing.T, actual *Configuration) {
	actualPrimaryAddress := actual.PrimaryAddress()
	if expect.primaryAddress != actualPrimaryAddress {
		t.Errorf("Expected primary address %s, but got %s", expect.primaryAddress, actualPrimaryAddress)
	}

	actualHealthAddress := actual.HealthAddress()
	if expect.healthAddress != actualHealthAddress {
		t.Errorf("Expected health address %s, but got %s", expect.healthAddress, actualHealthAddress)
	}

	actualHealthCheckInterval := actual.HealthCheckInterval()
	if expect.healthCheckInterval != actualHealthCheckInterval {
		t.Errorf("Expected health check interval %s, but got %s", expect.healthCheckInterval, actualHealthCheckInterval)
	}

	actualPprofAddress := actual.PprofAddress()
	if expect.pprofAddress != actualPprofAddress {
		t.Errorf("Expected pprof address %s, but got %s", expect.pprofAddress, actualPprofAddress)
	}
}

func TestAccessors(t *testing.T) {
	var testData = []struct {
		configuration Configuration
		expect        configurationExpect
	}{
		{
			Configuration{},
			configurationExpect{
				primaryAddress:      defaultPrimaryAddress,
				healthAddress:       defaultHealthCheckAddress,
				healthCheckInterval: DefaultHealthCheckInterval,
				pprofAddress:        defaultPprofAddress,
			},
		},
		{
			Configuration{
				Port:            58123,
				HealthCheckPort: 120,
				HCInterval:      types.Duration(time.Minute * 73),
				PprofPort:       3241,
			},
			configurationExpect{
				primaryAddress:      fmt.Sprintf(":%d", 58123),
				healthAddress:       fmt.Sprintf(":%d", 120),
				healthCheckInterval: time.Duration(time.Minute * 73),
				pprofAddress:        fmt.Sprintf(":%d", 3241),
			},
		},
	}

	for _, record := range testData {
		record.expect.assert(t, &record.configuration)
	}
}

func TestReadFile(t *testing.T) {
	var testData = []struct {
		filename string
		expected Configuration
	}{
		{
			"test_configuration_blank.json",
			Configuration{},
		},
		{
			"test_configuration_full.json",
			Configuration{
				Port:            8080,
				HealthCheckPort: 8181,
				HCInterval:      types.Duration(time.Second * 30),
				PprofPort:       8282,
				CertificateFile: "cert.file",
				KeyFile:         "key.file",
			},
		},
		{
			"test_configuration_nonsecure.json",
			Configuration{
				Port:            2020,
				HealthCheckPort: 2121,
				HCInterval:      types.Duration(time.Minute * 65),
				PprofPort:       2222,
			},
		},
	}

	for _, record := range testData {
		actual := Configuration{}
		ReadConfigurationFile(record.filename, &actual)
		if record.expected != actual {
			t.Errorf("For file %s, expected configuration %v, but got %v", record.filename, record.expected, actual)
		}
	}
}
