package health

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/logging"
	"github.com/stretchr/testify/assert"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

// setupHealth supplies a Health object with useful test configuration
func setupHealth() *Health {
	return New(
		time.Duration(69)*time.Second,
		&logging.LoggerWriter{os.Stdout},
	)
}

func TestLifecycle(t *testing.T) {
	var (
		assert = assert.New(t)
		h      = setupHealth()

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

	if _, ok := <-h.events; ok {
		t.Errorf("Close() did not close the event channel")
	}
}

func TestServeHTTP(t *testing.T) {
	var (
		assert   = assert.New(t)
		h        = setupHealth()
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
