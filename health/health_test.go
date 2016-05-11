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

func setupHealth() *Health {
	h := new(Health)
	h.event = make(chan HealthFunc, 100)
	h.stats = make(Stats)
	h.statDumpInterval = time.Duration(69) * time.Second

	h.log = logging.DefaultLogger{os.Stdout}
	h.wg = &sync.WaitGroup{}
	h.osChecker = testOsChecker{"testOS"}

	h.monitor()
	time.Sleep(time.Duration(1) * time.Second)

	h.commonStats()
	time.Sleep(time.Duration(1) * time.Second)

	return h
}

func setupStats() Stats {
	result := make(Stats)
	result[CurrentMemoryUtilizationActive] = 0
	result[CurrentMemoryUtilizationAlloc] = 0
	result[CurrentMemoryUtilizationHeapSys] = 0
	result[MaxMemoryUtilizationActive] = 0
	result[MaxMemoryUtilizationAlloc] = 0
	result[MaxMemoryUtilizationHeapSys] = 0

	return result
}

func TestAddStatsListener(t *testing.T) {
	h := setupHealth()
	noOfListenersAtStart := len(h.statsListeners)

	h.AddStatsListener(*new(StatsListener))

	h.SendEvent(func(stats Stats) {
		expected := noOfListenersAtStart + 1
		noOfListenersAtEnd := len(h.statsListeners)
		if noOfListenersAtEnd != expected {
			t.Errorf("Failed to correctly add stat listener: Got: %v, Expected: %v", noOfListenersAtEnd, expected)
		}
	})
}

func TestSendEvent(t *testing.T) {
	h := setupHealth()

	done := make(chan bool)
	timer := time.NewTimer(time.Second * 5)
	defer timer.Stop()

	hf := func(s Stats) {
		close(done)
	}

	h.SendEvent(hf)

	select {
	case <-done:
		// test passed
	case <-timer.C:
		// test failed
		close(done) // this might panic, but it's a test failure anyway
		t.Errorf("HealthFunc (hf) was not called")
	}
}

func TestBundle(t *testing.T) {
	done := make(chan bool)
	defer close(done)

	h := setupHealth()

	hf1 := Set("BundleTest1", 111)
	hf2 := Set("BundleTest2", 222)
	hf3 := func(s Stats) {
		done <- true
	}
	h.SendEvent(Bundle(hf1, hf2, hf3))

	select {
	case <-done:
	}

	v1, ok1 := h.stats["BundleTest1"]
	v2, ok2 := h.stats["BundleTest2"]

	if !ok1 ||
		!ok2 ||
		v1 != 111 ||
		v2 != 222 {
		t.Errorf("Bundle test failed. Got: %v, %v, %v, %v.  Expected: true, true, 111, 222", ok1, ok2, v1, v2)
	}
}

func TestInc(t *testing.T) {
	h := setupHealth()
	expected := 4

	h.SendEvent(Inc(CurrentMemoryUtilizationActive, 4))
	time.Sleep(time.Duration(1) * time.Second)

	if h.stats[CurrentMemoryUtilizationActive] != expected {
		t.Errorf("Health Set values do not match.\nGot: %v\nExpected: %v\n", h.stats[CurrentMemoryUtilizationActive], expected)
	}
}

func TestSet(t *testing.T) {
	h := setupHealth()
	expected := 62

	h.SendEvent(Set(CurrentMemoryUtilizationActive, 62))
	time.Sleep(time.Duration(1) * time.Second)

	if h.stats[CurrentMemoryUtilizationActive] != expected {
		t.Errorf("Health Set values do not match.\nGot: %v\nExpected: %v\n", h.stats[CurrentMemoryUtilizationActive], expected)
	}
}

func TestClose(t *testing.T) {
	h := setupHealth()
	h.Close()

	if _, ok := <-h.event; ok {
		t.Error("health event channel was not closed.")
	}
}

func TestNew(t *testing.T) {
	expected := setupHealth()

	dp := 69 * time.Second
	lg := logging.DefaultLogger{os.Stdout}
	wg := &sync.WaitGroup{}

	h := New(dp, lg, wg)
	h.osChecker = testOsChecker{"testOS"}

	if reflect.TypeOf(h) != reflect.TypeOf(expected) {
		t.Error("Newly created health object not correct type: Got:%v, Expected: %v", reflect.TypeOf(h), reflect.TypeOf(expected))
	}

	if h.statDumpInterval != dp {
		t.Error("Health stat dump interval not set correctly.")
	}

	if h.log != lg {
		t.Error("Health logger not set correctly")
	}
}

func TestWaitGroupDone(t *testing.T) {
	dp := 69 * time.Second
	lg := logging.DefaultLogger{os.Stdout}
	wg := &sync.WaitGroup{}

	h := New(dp, lg, wg)
	h.osChecker = testOsChecker{"testOS"}
	h.Close()

	result := make(chan bool, 1)
	defer close(result)
	wg.Wait()
	result <- true

	timer := time.AfterFunc(
		time.Second*5, // pick something reasonable
		func() {
			result <- false
		},
	)
	defer timer.Stop()

	for success := range result {
		if !success {
			t.Errorf("WaitGroup.Done() wasn't called")
		}

		break
	}
}

func TestCommonStats(t *testing.T) {
	h := new(Health)
	h.event = make(chan HealthFunc, 100)
	h.stats = make(Stats)
	h.statDumpInterval = time.Duration(69) * time.Second

	h.log = logging.DefaultLogger{os.Stdout}
	h.wg = &sync.WaitGroup{}

	h.monitor()
	time.Sleep(time.Duration(1) * time.Second)

	h.commonStats()
	time.Sleep(time.Duration(1) * time.Second)

	expected := setupStats()
	if !reflect.DeepEqual(h.stats, expected) {
		t.Errorf("common stats not setup correctly.\n Got: %v\nExpected: %v\n", h.stats, expected)
	}
}

func TestOscheck(t *testing.T) {
	h := setupHealth()
	result := h.oscheck()
	expected := false

	if result != expected {
		t.Error("operating system verification failed. Got: %v, Expected: %v", result, expected)
	}
}

func TestMemory(t *testing.T) {
	h := setupHealth()

	h.memory()
	if h.stats[CurrentMemoryUtilizationAlloc] != 0 ||
		h.stats[CurrentMemoryUtilizationHeapSys] != 0 ||
		h.stats[CurrentMemoryUtilizationActive] != 0 ||
		h.stats[MaxMemoryUtilizationAlloc] != 0 ||
		h.stats[MaxMemoryUtilizationHeapSys] != 0 ||
		h.stats[MaxMemoryUtilizationActive] != 0 {

		t.Error("Bad memory value found: %v", h.stats)
	}
}

func TestMonitor(t *testing.T) {
	h := setupHealth()
	h.monitor()
	h.Close()
}

func TestGetStats(t *testing.T) {
	h := setupHealth()

	expected := setupStats()

	if !reflect.DeepEqual(h.stats, expected) {
		t.Errorf("Newly created stats to not match.\n Got: %v\nExpected: %v\n", h.stats, expected)
	}
}

func setHealthTester(h *Health) {}

func TestShare(t *testing.T) {
	h := setupHealth()
	h.Share(setHealthTester)
}

func TestServeHTTP(t *testing.T) {
	h := setupHealth()
	expected := setupStats()

	req, _ := http.NewRequest("GET", "", nil)
	rw := httptest.NewRecorder()

	h.ServeHTTP(rw, req)

	if rw.Code != 200 {
		t.Error("Status code was not 200.  got: %v", rw.Code)
	}

	result := new(Stats)
	if err := json.Unmarshal(rw.Body.Bytes(), result); err != nil {
		t.Error("json Unmarshal error: %v", err)
	}

	if !reflect.DeepEqual(*result, expected) {
		t.Errorf("ServeHTTP did not return Stats.\n Got: %v\nExpected: %v\n", result, expected)
	}
}

func TestResponseErrorJson(t *testing.T) {
	rw := httptest.NewRecorder()
	err := "Expected test error message"
	code := 2222
	lg := logging.DefaultLogger{os.Stdout}

	responseErrorJson(rw, err, code, lg)

	var js map[string]string
	if json.Unmarshal(rw.Body.Bytes(), &js) != nil {
		t.Errorf("Response error is not JSON: %v", rw.Body)
	}
}
