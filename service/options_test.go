package service

import (
	"errors"
	"github.com/Comcast/webpa-common/logging"
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestRegistrationDefault(t *testing.T) {
	assert := assert.New(t)

	for _, r := range []*Registration{nil, new(Registration)} {
		t.Log(r)

		assert.Equal(DefaultScheme, r.scheme())
		assert.Equal(DefaultHost, r.host())
		assert.Equal(defaultPorts[DefaultScheme], r.port())
	}
}

func TestRegistration(t *testing.T) {
	assert := assert.New(t)
	testData := []struct {
		registration   *Registration
		expectedScheme string
		expectedHost   string
		expectedPort   uint16
	}{
		{
			&Registration{Scheme: "unrecognized", Host: "comcast.net"},
			"unrecognized",
			"comcast.net",
			0,
		},
		{
			&Registration{Scheme: "unrecognized", Host: "comcast.net", Port: 4721},
			"unrecognized",
			"comcast.net",
			4721,
		},
		{
			&Registration{Host: "comcast.net"},
			"http",
			"comcast.net",
			80,
		},
		{
			&Registration{Scheme: "http", Host: "comcast.net"},
			"http",
			"comcast.net",
			80,
		},
		{
			&Registration{Scheme: "https", Host: "comcast.net"},
			"https",
			"comcast.net",
			443,
		},
		{
			&Registration{Scheme: "https", Host: "comcast.net", Port: 8080},
			"https",
			"comcast.net",
			8080,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)

		assert.Equal(record.expectedScheme, record.registration.scheme())
		assert.Equal(record.expectedHost, record.registration.host())
		assert.Equal(record.expectedPort, record.registration.port())
	}
}

func TestOptionsDefault(t *testing.T) {
	assert := assert.New(t)

	for _, o := range []*Options{nil, new(Options)} {
		t.Log(o)

		assert.NotNil(o.logger())
		assert.Equal([]string{DefaultServer}, o.servers())
		assert.Equal(DefaultTimeout, o.timeout())
		assert.Equal(DefaultBaseDirectory, o.baseDirectory())
		assert.Equal(DefaultMemberPrefix, o.memberPrefix())
		assert.Equal(DefaultEnvironment, o.environment())
		assert.Equal(DefaultServiceName, o.serviceName())
		assert.Empty(o.registrations())
		assert.Equal(DefaultVnodeCount, o.vnodeCount())
		assert.Nil(o.pingFunc())
	}
}

func TestOptions(t *testing.T) {
	assert := assert.New(t)
	logger := logging.TestLogger(t)
	expectedError := errors.New("TestOptions expected error")
	testData := []struct {
		options         *Options
		expectedServers []string
		expectedTimeout time.Duration
	}{
		{
			&Options{
				Logger:        logger,
				Servers:       []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
				Timeout:       16 * time.Minute,
				BaseDirectory: "/testOptions/workspace",
				MemberPrefix:  "testOptions_",
				Environment:   "test-options",
				ServiceName:   "options",
				Registrations: []Registration{Registration{}, Registration{"https", "comcast.net", 8080}},
				VnodeCount:    67912723,
				PingFunc:      nil,
			},
			[]string{"node1.comcast.net:2181", "node2.comcast.net:275"},
			16 * time.Minute,
		},
		{
			&Options{
				Connection:    "test1.comcast.net:2181",
				Timeout:       -5 * time.Minute,
				BaseDirectory: "/testOptions/workspace",
				MemberPrefix:  "testOptions_",
				Environment:   "test-options",
				ServiceName:   "options",
				VnodeCount:    34572,
				PingFunc:      func() error { return expectedError },
			},
			[]string{"test1.comcast.net:2181"},
			DefaultTimeout,
		},
		{
			&Options{
				Connection:    "test1.comcast.net:2181, test2.foobar.com:9999   \t",
				Servers:       []string{"node1.qbert.net"},
				Timeout:       7 * time.Hour,
				BaseDirectory: "/testOptions/workspace",
				MemberPrefix:  "testOptions_",
				Environment:   "test-options",
				ServiceName:   "options",
				VnodeCount:    34572,
				PingFunc:      func() error { return expectedError },
			},
			[]string{"test1.comcast.net:2181", "test2.foobar.com:9999", "node1.qbert.net"},
			7 * time.Hour,
		},
	}

	for _, record := range testData {
		t.Logf("%v", record)
		options := record.options

		if options.Logger != nil {
			assert.Equal(options.Logger, options.logger())
		} else {
			assert.NotNil(options.logger())
		}

		assert.Equal(record.expectedServers, options.servers())
		assert.Equal(record.expectedTimeout, options.timeout())
		assert.Equal(options.BaseDirectory, options.baseDirectory())
		assert.Equal(options.MemberPrefix, options.memberPrefix())
		assert.Equal(serversets.Environment(options.Environment), options.environment())
		assert.Equal(options.ServiceName, options.serviceName())
		assert.Equal(options.Registrations, options.registrations())
		assert.Equal(int(options.VnodeCount), options.vnodeCount())

		if options.PingFunc != nil {
			assert.Equal(expectedError, options.pingFunc()())
		} else {
			assert.Nil(options.pingFunc())
		}
	}
}
