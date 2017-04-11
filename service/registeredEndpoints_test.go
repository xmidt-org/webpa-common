package service

import (
	"github.com/strava/go.serversets"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRegisteredEndpoints(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			host        string
			port        int
			expectedKey string
		}{
			{
				host:        "node1.comcast.net",
				port:        80,
				expectedKey: "http://node1.comcast.net:80",
			},
			{
				host:        "node1.comcast.net",
				port:        1234,
				expectedKey: "http://node1.comcast.net:1234",
			},
			{
				host:        "https://node1.comcast.net",
				port:        45723,
				expectedKey: "https://node1.comcast.net:45723",
			},
			{
				host:        "unrecognized://node1.comcast.net",
				port:        281,
				expectedKey: "unrecognized://node1.comcast.net:281",
			},
		}
	)

	for _, record := range testData {
		t.Logf("%v", record)

		var (
			registeredEndpoints = make(RegisteredEndpoints)
			expectedValue       = new(serversets.Endpoint)
		)

		registeredEndpoints.AddHostPort(record.host, record.port, expectedValue)
		assert.True(registeredEndpoints.Has(record.expectedKey))
	}
}
