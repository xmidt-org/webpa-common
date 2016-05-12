package health

import (
	"bytes"
	"encoding/json"
	"github.com/Comcast/webpa-common/logging"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
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

// testOsChecker is the test implementation of OsChecker
type testOsChecker struct {
	osName string
}

func (t testOsChecker) OsName() string {
	return t.osName
}

func TestLifecycle(t *testing.T) {
	t.Log("starting TestLifecycle")
	defer t.Log("TestLifecycle complete")
	h := setupHealth()
	h.memory = Memory(h.log, testOsChecker{})

	healthWaitGroup := &sync.WaitGroup{}
	h.Run(healthWaitGroup)

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

	done := make(chan bool)
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

	// verify that the channel has been closed
	h.Close()
	if _, ok := <-h.event; ok {
		t.Errorf("Close() did not close the event channel")
	}

	done = make(chan bool)
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

func TestBundle(t *testing.T) {
	expected := Stats{
		CurrentMemoryUtilizationHeapSys: 0,
		CurrentMemoryUtilizationAlloc:   1,
		CurrentMemoryUtilizationActive:  12,
	}

	actual := Stats{
		CurrentMemoryUtilizationAlloc: 0,
	}

	bundle := Bundle(
		Ensure(CurrentMemoryUtilizationHeapSys),
		Inc(CurrentMemoryUtilizationAlloc, 1),
		Set(CurrentMemoryUtilizationActive, 12),
	)

	bundle(actual)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v, but got %v", expected, actual)
	}
}

func TestInc(t *testing.T) {
	var testData = []struct {
		stat      Stat
		increment int
		initial   Stats
		expected  Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			1,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 1},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			-12,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: -12},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			72,
			Stats{CurrentMemoryUtilizationHeapSys: 0},
			Stats{CurrentMemoryUtilizationHeapSys: 72},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			6,
			Stats{CurrentMemoryUtilizationHeapSys: 45},
			Stats{CurrentMemoryUtilizationHeapSys: 51},
		},
	}

	for _, record := range testData {
		Inc(record.stat, record.increment)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestSet(t *testing.T) {
	var testData = []struct {
		stat     Stat
		newValue int
		initial  Stats
		expected Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			123,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 123},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			37842,
			Stats{CurrentMemoryUtilizationHeapSys: 42734987},
			Stats{CurrentMemoryUtilizationHeapSys: 37842},
		},
	}

	for _, record := range testData {
		Set(record.stat, record.newValue)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestEnsure(t *testing.T) {
	var testData = []struct {
		stat     Stat
		initial  Stats
		expected Stats
	}{
		{
			CurrentMemoryUtilizationHeapSys,
			Stats{},
			Stats{CurrentMemoryUtilizationHeapSys: 0},
		},
		{
			CurrentMemoryUtilizationHeapSys,
			Stats{CurrentMemoryUtilizationHeapSys: -157},
			Stats{CurrentMemoryUtilizationHeapSys: -157},
		},
	}

	for _, record := range testData {
		Ensure(record.stat)(record.initial)

		if !reflect.DeepEqual(record.expected, record.initial) {
			t.Errorf("Expected %v, but got %v", record.expected, record.initial)
		}
	}
}

func TestMemoryNonLinux(t *testing.T) {
	log := &logging.DefaultLogger{os.Stdout}
	memory := Memory(log, testOsChecker{"nonsense"})

	actual := commonStats.Clone()
	memory(actual)
	if !reflect.DeepEqual(commonStats, actual) {
		t.Errorf("On a non-linux platform, Memory should not modify stats")
	}
}

func TestMemoryLinux(t *testing.T) {
	var buffer bytes.Buffer
	log := &logging.DefaultLogger{&buffer}
	memory := Memory(log, testOsChecker{"linux"})

	actual := commonStats.Clone()
	memory(actual)

	if buffer.Len() > 0 && strings.Contains(buffer.String(), "error querying memory information") {
		t.Log("Not on a linux system.  Aborting test of memory statistics update.")
		return
	}

	// if we don't actually get a panic, then verify that the memory did change certain stats
	if actual[CurrentMemoryUtilizationActive] == 0 ||
		actual[MaxMemoryUtilizationActive] == 0 ||
		actual[CurrentMemoryUtilizationAlloc] == 0 ||
		actual[CurrentMemoryUtilizationHeapSys] == 0 ||
		actual[MaxMemoryUtilizationAlloc] == 0 ||
		actual[MaxMemoryUtilizationHeapSys] == 0 {
		t.Errorf("Memory stats not updated")
	}
}

func TestServeHTTP(t *testing.T) {
	h := setupHealth()
	h.Run(&sync.WaitGroup{})
	defer h.Close()

	request, _ := http.NewRequest("GET", "", nil)
	response := httptest.NewRecorder()

	h.ServeHTTP(response, request)

	done := make(chan bool)
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

	result := new(Stats)
	if err := json.Unmarshal(response.Body.Bytes(), result); err != nil {
		t.Errorf("json Unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(commonStats, *result) {
		t.Errorf("ServeHTTP did not return Stats.\n Got: %v\nExpected: %v\n", *result, commonStats)
	}
}
