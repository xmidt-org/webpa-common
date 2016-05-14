package server

import (
	"github.com/Comcast/webpa-common/types"
	"testing"
	"time"
)

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
				Port:                8080,
				HealthCheckPort:     8181,
				HealthCheckInterval: types.Duration(time.Second * 30),
				PprofPort:           8282,
				CertificateFile:     "cert.file",
				KeyFile:             "key.file",
			},
		},
		{
			"test_configuration_nonsecure.json",
			Configuration{
				Port:                2020,
				HealthCheckPort:     2121,
				HealthCheckInterval: types.Duration(time.Minute * 65),
				PprofPort:           2222,
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
