package service

import (
	"testing"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
)

func testOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Logf("%#v", o)

		assert.NotNil(o.logger())
		assert.Equal([]string{DefaultServer}, o.servers())
		assert.Equal(DefaultConnectTimeout, o.connectTimeout())
		assert.Equal(DefaultSessionTimeout, o.sessionTimeout())
		assert.Zero(o.updateDelay())
		assert.Equal(DefaultPath, o.path())
		assert.Equal(DefaultServiceName, o.serviceName())
		assert.Equal(DefaultRegistration, o.registration())
		assert.Equal(DefaultVnodeCount, o.vnodeCount())
		assert.NotNil(o.instancesFilter())
		assert.NotNil(o.accessorFactory())
		assert.NotNil(o.after())
		assert.NotEmpty(o.String())
	}
}

func testOptionsCustom(t *testing.T) {
	var (
		assert = assert.New(t)
		logger = logging.NewTestLogger(nil, t)

		customInstancesFilterCalled bool
		customInstancesFilter       = func(i []string) []string { customInstancesFilterCalled = true; return i }

		customAccessorFactoryCalled bool
		customAccessorFactory       = func([]string) Accessor { customAccessorFactoryCalled = true; return nil }

		customAfterCalled bool
		customAfter       = func(time.Duration) <-chan time.Time { customAfterCalled = true; return nil }

		testData = []struct {
			options         *Options
			expectedServers map[string]bool
		}{
			{
				&Options{
					Logger:          logger,
					Servers:         []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
					ConnectTimeout:  16 * time.Minute,
					SessionTimeout:  2 * time.Hour,
					UpdateDelay:     3 * time.Minute,
					Path:            "/testOptions/workspace",
					ServiceName:     "options",
					Registration:    "https://comcast.net:8080",
					VnodeCount:      67912723,
					InstancesFilter: customInstancesFilter,
					AccessorFactory: customAccessorFactory,
					After:           customAfter,
				},
				map[string]bool{"node1.comcast.net:2181": true, "node2.comcast.net:275": true},
			},
			{
				&Options{
					Logger:          logger,
					Connection:      "foobar.com:1234",
					ConnectTimeout:  45 * time.Minute,
					SessionTimeout:  1 * time.Hour,
					UpdateDelay:     67 * time.Hour,
					Path:            "/testOptions/workspace",
					ServiceName:     "anotherOptions",
					Registration:    "https://comcast.com:1111",
					VnodeCount:      398312,
					InstancesFilter: customInstancesFilter,
					AccessorFactory: customAccessorFactory,
					After:           customAfter,
				},
				map[string]bool{"foobar.com:1234": true},
			},
			{
				&Options{
					Logger:          logger,
					Connection:      "grover.net:9999,foobar.com:1234",
					ConnectTimeout:  123 * time.Second,
					SessionTimeout:  13 * time.Minute,
					UpdateDelay:     0,
					Path:            "/testOptions/anotherone",
					ServiceName:     "anotherOptions",
					Registration:    "https://comcast.com:92",
					VnodeCount:      374,
					InstancesFilter: customInstancesFilter,
					AccessorFactory: customAccessorFactory,
					After:           customAfter,
				},
				map[string]bool{"foobar.com:1234": true, "grover.net:9999": true},
			},
			{
				&Options{
					Logger:          logger,
					Connection:      "grover.net:9999,foobar.com:1234",
					Servers:         []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
					ConnectTimeout:  3847923 * time.Second,
					SessionTimeout:  2 * time.Minute,
					UpdateDelay:     17 * time.Second,
					Path:            "/testOptions/anotherone",
					ServiceName:     "anotherOptions",
					Registration:    "https://comcast.com:92",
					VnodeCount:      3812,
					InstancesFilter: customInstancesFilter,
					AccessorFactory: customAccessorFactory,
					After:           customAfter,
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
		assert.Equal(options.UpdateDelay, options.updateDelay())
		assert.Equal(options.Path, options.path())
		assert.Equal(options.ServiceName, options.serviceName())
		assert.Equal(options.Registration, options.registration())
		assert.Equal(int(options.VnodeCount), options.vnodeCount())
		assert.NotEmpty(options.String())

		customInstancesFilterCalled = false
		options.instancesFilter()([]string{})
		assert.True(customInstancesFilterCalled)

		customAccessorFactoryCalled = false
		options.accessorFactory()([]string{})
		assert.True(customAccessorFactoryCalled)

		customAfterCalled = false
		options.after()(time.Minute)
		assert.True(customAfterCalled)
	}
}

func TestOptions(t *testing.T) {
	t.Run("Default", testOptionsDefault)
	t.Run("Custom", testOptionsCustom)
}
