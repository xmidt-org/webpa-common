package handler

import (
	"github.com/Comcast/webpa-common/health"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

type testHealthEventSink struct {
	wasCalled bool
	stats     health.Stats
}

func (sink *testHealthEventSink) SendEvent(event health.HealthFunc) {
	sink.wasCalled = true
	event.Set(sink.stats)
}

func newTestHealthEventSink() *testHealthEventSink {
	return &testHealthEventSink{
		wasCalled: false,
		stats:     make(health.Stats),
	}
}

func TestHealthRequestListenerRequestReceived(t *testing.T) {
	assert := assert.New(t)

	request, err := http.NewRequest("GET", "", nil)
	if !assert.Nil(err) {
		return
	}

	sink := newTestHealthEventSink()
	listener := NewHealthRequestListener(sink)
	if !assert.NotNil(listener) {
		return
	}

	listener.RequestReceived(request)
	assert.True(sink.wasCalled)
	assert.Equal(health.Stats{TotalRequestsReceived: 1}, sink.stats)

	sink.wasCalled = false
	listener.RequestReceived(request)
	assert.True(sink.wasCalled)
	assert.Equal(health.Stats{TotalRequestsReceived: 2}, sink.stats)
}

func TestHealthRequestListenerRequestCompleted(t *testing.T) {
	assert := assert.New(t)

	request, err := http.NewRequest("GET", "", nil)
	if !assert.Nil(err) {
		return
	}

	var testData = []struct {
		sequence []int
		expected health.Stats
	}{
		{
			[]int{200},
			health.Stats{TotalRequestSuccessfullyServiced: 1},
		},
		{
			[]int{200, 200},
			health.Stats{TotalRequestSuccessfullyServiced: 2},
		},
		{
			[]int{404},
			health.Stats{TotalRequestDenied: 1},
		},
		{
			[]int{200, 500},
			health.Stats{TotalRequestSuccessfullyServiced: 1, TotalRequestDenied: 1},
		},
		{
			[]int{100, 200, 202, 400, 404, 500, 200},
			health.Stats{TotalRequestSuccessfullyServiced: 4, TotalRequestDenied: 3},
		},
	}

	for _, record := range testData {
		t.Logf("%#v", record)
		sink := newTestHealthEventSink()
		listener := NewHealthRequestListener(sink)

		for _, statusCode := range record.sequence {
			sink.wasCalled = false
			listener.RequestCompleted(statusCode, request)
			assert.True(sink.wasCalled)
		}

		assert.Equal(record.expected, sink.stats)
	}
}
