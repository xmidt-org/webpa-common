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
		assert.Equal([]string{DefaultZookeeperServer}, o.zookeeperServers())
		assert.Equal(DefaultZookeeperTimeout, o.zookeeperTimeout())
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
	pingFunc := func() error { return expectedError }

	o := Options{
		Logger:           logger,
		ZookeeperServers: []string{"node1.comcast.net:2181", "node2.comcast.net:275"},
		ZookeeperTimeout: 16 * time.Minute,
		BaseDirectory:    "/testOptions/workspace",
		MemberPrefix:     "testOptions_",
		Environment:      "test-options",
		ServiceName:      "options",
		Registrations:    []Registration{Registration{}, Registration{"https", "comcast.net", 8080}},
		VnodeCount:       67912723,
		PingFunc:         pingFunc,
	}

	assert.Equal(logger, o.logger())
	assert.Equal(o.ZookeeperServers, o.zookeeperServers())
	assert.Equal(o.ZookeeperTimeout, o.zookeeperTimeout())
	assert.Equal(o.BaseDirectory, o.baseDirectory())
	assert.Equal(o.MemberPrefix, o.memberPrefix())
	assert.Equal(serversets.Environment(o.Environment), o.environment())
	assert.Equal(o.ServiceName, o.serviceName())
	assert.Equal(o.Registrations, o.registrations())
	assert.Equal(int(o.VnodeCount), o.vnodeCount())
	assert.Equal(expectedError, o.pingFunc()())
}
