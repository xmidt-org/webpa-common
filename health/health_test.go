package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xmidt-org/webpa-common/v2/logging"
)

// setupHealth supplies a Health object with useful test configuration
func setupHealth(t *testing.T) *Health {
	return New(
		time.Duration(69)*time.Second,
		logging.NewTestLogger(nil, t),
	)
}

func TestLifecycle(t *testing.T) {
	var (
		assert = assert.New(t)
		h      = setupHealth(t)

		healthWaitGroup = &sync.WaitGroup{}
		shutdown        = make(chan struct{})
	)

	h.Run(healthWaitGroup, shutdown)

	// verify initial state
	var initialListenerCount int
	testWaitGroup := &sync.WaitGroup{}
	testWaitGroup.Add(1)
	h.SendEvent(func(stats Stats) {
		defer testWaitGroup.Done()
		initialListenerCount = len(h.statsListeners)
		assert.Equal(NewStats(nil), stats)
		assert.Equal(stats, h.stats)
	})

	h.AddStatsListener(StatsListenerFunc(func(Stats) {}))

	testWaitGroup.Add(1)
	h.SendEvent(func(Stats) {
		defer testWaitGroup.Done()
		t.Log("verifying AddStatsListener")
		if len(h.statsListeners) != (initialListenerCount + 1) {
			t.Errorf("Listeners were not updated properly")
		}
	})

	done := make(chan struct{})
	timer := time.NewTimer(time.Second * 10)

	go func() {
		testWaitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Initial state verified")
	case <-timer.C:
		t.Errorf("Failed to verify initial state within the timeout")
		close(done)
	}

	close(shutdown)

	done = make(chan struct{})
	timer.Stop()
	timer = time.NewTimer(time.Second * 10)
	defer timer.Stop()
	go func() {
		healthWaitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Log("Final state verified")
	case <-timer.C:
		t.Errorf("Failed to verify final state within the timeout")
		close(done)
	}
}

func TestServeHTTP(t *testing.T) {
	var (
		assert   = assert.New(t)
		h        = setupHealth(t)
		shutdown = make(chan struct{})

		request  = httptest.NewRequest("GET", "http://something.net", nil)
		response = httptest.NewRecorder()
	)

	h.Run(&sync.WaitGroup{}, shutdown)
	defer close(shutdown)

	h.ServeHTTP(response, request)

	done := make(chan struct{})
	timer := time.NewTimer(time.Second * 15)
	defer timer.Stop()
	h.SendEvent(func(stats Stats) {
		close(done)
	})

	select {
	case <-done:
	case <-timer.C:
		close(done)
		t.Fatalf("Did not receive next event after ServeHTTP in the allotted time")
	}

	assert.Equal(200, response.Code)

	var result Stats
	assert.NoError(json.Unmarshal(response.Body.Bytes(), &result))

	// each key in commonStats should be present in the output
	for _, stat := range memoryStats {
		_, ok := result[stat.(Stat)]
		assert.True(ok)
	}
}

func TestHealthRequestTracker(t *testing.T) {
	var (
		assert   = assert.New(t)
		testData = []struct {
			expectedStatusCode int
			expectedStats      Stats
		}{
			// success codes
			{0, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{100, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{200, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{201, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{202, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{300, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},
			{307, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 1, TotalRequestsDenied: 0}},

			// failure codes
			{400, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 0, TotalRequestsDenied: 1}},
			{404, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 0, TotalRequestsDenied: 1}},
			{500, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 0, TotalRequestsDenied: 1}},
			{523, Stats{TotalRequestsReceived: 1, TotalRequestsSuccessfullyServiced: 0, TotalRequestsDenied: 1}},
		}
	)

	for _, record := range testData {
		t.Logf("%#v", record)

		var (
			monitor  = setupHealth(t)
			shutdown = make(chan struct{})

			handler  = new(mockHandler)
			request  = httptest.NewRequest("GET", "http://something.com", nil)
			response = httptest.NewRecorder()
		)

		monitor.Run(&sync.WaitGroup{}, shutdown)
		defer close(shutdown)

		handler.On("ServeHTTP", mock.MatchedBy(func(*ResponseWriter) bool { return true }), request).
			Once().
			Run(func(arguments mock.Arguments) {
				arguments.Get(0).(http.ResponseWriter).WriteHeader(record.expectedStatusCode)
			})

		compositeHandler := monitor.RequestTracker(handler)
		compositeHandler.ServeHTTP(response, request)
		assert.Equal(record.expectedStatusCode, response.Code)

		assertionWaitGroup := new(sync.WaitGroup)
		assertionWaitGroup.Add(1)
		monitor.SendEvent(
			func(actualStats Stats) {
				defer assertionWaitGroup.Done()
				t.Logf("actual stats: %v", actualStats)
				for stat, value := range record.expectedStats {
					assert.Equal(value, actualStats[stat], fmt.Sprintf("%s should have been %d", stat, value))
				}
			},
		)

		assertionWaitGroup.Wait()
		handler.AssertExpectations(t)
	}
}

func TestHealthRequestTrackerDelegatePanic(t *testing.T) {
	var (
		assert   = assert.New(t)
		monitor  = setupHealth(t)
		shutdown = make(chan struct{})

		handler  = new(mockHandler)
		request  = httptest.NewRequest("GET", "http://something.com", nil)
		response = httptest.NewRecorder()
	)

	monitor.Run(&sync.WaitGroup{}, shutdown)
	defer close(shutdown)

	handler.On("ServeHTTP", mock.MatchedBy(func(*ResponseWriter) bool { return true }), request).
		Once().
		Run(func(mock.Arguments) {
			panic("expected")
		})

	compositeHandler := monitor.RequestTracker(handler)
	compositeHandler.ServeHTTP(response, request)
	assert.Equal(http.StatusInternalServerError, response.Code)

	assertionWaitGroup := new(sync.WaitGroup)
	assertionWaitGroup.Add(1)
	monitor.SendEvent(
		func(actualStats Stats) {
			defer assertionWaitGroup.Done()
			assert.Equal(1, actualStats[TotalRequestsReceived])
			assert.Equal(0, actualStats[TotalRequestsSuccessfullyServiced])
			assert.Equal(1, actualStats[TotalRequestsDenied])
		},
	)

	assertionWaitGroup.Wait()
	handler.AssertExpectations(t)
}
