package health

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"
)

// setupHealth supplies a Health object with useful test configuration
func setupHealth() *Health {
	return New(
		time.Duration(69)*time.Second,
		logging.DefaultLogger{os.Stdout},
	)
}

func TestLifecycle(t *testing.T) {
	t.Log("starting TestLifecycle")
	defer t.Log("TestLifecycle complete")
	h := setupHealth()

	healthWaitGroup := &sync.WaitGroup{}
	shutdown := make(chan struct{})
	h.Run(healthWaitGroup, shutdown)

	// verify initial state
	var initialListenerCount int
	testWaitGroup := &sync.WaitGroup{}
	testWaitGroup.Add(1)
	h.SendEvent(func(stats Stats) {
		defer testWaitGroup.Done()
		t.Log("verifying initial state")
		initialListenerCount = len(h.statsListeners)
		if !reflect.DeepEqual(commonStats, h.stats) {
			t.Errorf("Initial stats not set properly.  Expected %v, but got %v", commonStats, h.stats)
		}

		if !reflect.DeepEqual(commonStats, stats) {
			t.Errorf("Stats not copied properly.  Expected %v, but got %v", commonStats, h.stats)
		}
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
	h := setupHealth()
	shutdown := make(chan struct{})
	h.Run(&sync.WaitGroup{}, shutdown)
	defer close(shutdown)

	request, _ := http.NewRequest("GET", "", nil)
	response := httptest.NewRecorder()

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

	if response.Code != 200 {
		t.Error("Status code was not 200.  got: %v", response.Code)
	}

	var result Stats
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatalf("json Unmarshal error: %v", err)
	}

	// each key in commonStats should be present in the output
	for key, _ := range commonStats {
		if _, ok := result[key]; !ok {
			t.Errorf("Key %s not present in ServeHTTP results", key)
		}
	}
}
