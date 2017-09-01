package service

import (
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func TestOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		assert.NotNil(o.logger())
		assert.Equal([]string{DefaultServer}, o.servers())
		assert.Equal(DefaultConnectTimeout, o.connectTimeout())
		assert.Equal(DefaultSessionTimeout, o.sessionTimeout())
		assert.Zero(o.updateDelay())
		assert.Equal(DefaultPath, o.path())
		assert.Equal(DefaultServiceName, o.serviceName())
		assert.Equal(DefaultRegistration, o.registration())
		assert.Equal(DefaultVnodeCount, o.vnodeCount())
	}
}

func TestOptions(t *testing.T) {
	var (
		assert   = assert.New(t)
		logger   = logging.NewTestLogger(nil, t)
		testData = []struct {
			options         *Options
			expectedServers map[string]bool
		}{
			{
				&Options{
					Logger:         logger,
					Servers:        []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
					ConnectTimeout: 16 * time.Minute,
					SessionTimeout: 2 * time.Hour,
					Path:           "/testOptions/workspace",
					ServiceName:    "options",
					Registration:   "https://comcast.net:8080",
					VnodeCount:     67912723,
				},
				map[string]bool{"node1.comcast.net:2181": true, "node2.comcast.net:275": true},
			},
			{
				&Options{
					Logger:         logger,
					Connection:     "foobar.com:1234",
					ConnectTimeout: 45 * time.Minute,
					SessionTimeout: 1 * time.Hour,
					Path:           "/testOptions/workspace",
					ServiceName:    "anotherOptions",
					Registration:   "https://comcast.com:1111",
					VnodeCount:     398312,
				},
				map[string]bool{"foobar.com:1234": true},
			},
			{
				&Options{
					Logger:         logger,
					Connection:     "grover.net:9999,foobar.com:1234",
					ConnectTimeout: 123 * time.Second,
					SessionTimeout: 13 * time.Minute,
					Path:           "/testOptions/anotherone",
					ServiceName:    "anotherOptions",
					Registration:   "https://comcast.com:92",
					VnodeCount:     374,
				},
				map[string]bool{"foobar.com:1234": true, "grover.net:9999": true},
			},
			{
				&Options{
					Logger:         logger,
					Connection:     "grover.net:9999,foobar.com:1234",
					Servers:        []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
					ConnectTimeout: 3847923 * time.Second,
					SessionTimeout: 2 * time.Minute,
					Path:           "/testOptions/anotherone",
					ServiceName:    "anotherOptions",
					Registration:   "https://comcast.com:92",
					VnodeCount:     3812,
				},
				map[string]bool{"node1.comcast.net:2181": true, "node2.comcast.net:275": true, "foobar.com:1234": true, "grover.net:9999": true},
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)
		options := record.options

		actualServers := options.servers()
		assert.Equal(len(record.expectedServers), len(actualServers))
		for _, candidate := range actualServers {
			assert.True(record.expectedServers[candidate])
		}

		assert.Equal(options.Logger, options.logger())
		assert.Equal(options.ConnectTimeout, options.connectTimeout())
		assert.Equal(options.SessionTimeout, options.sessionTimeout())
		assert.Equal(options.Path, options.path())
		assert.Equal(options.ServiceName, options.serviceName())
		assert.Equal(options.Registration, options.registration())
		assert.Equal(int(options.VnodeCount), options.vnodeCount())
	}
}
