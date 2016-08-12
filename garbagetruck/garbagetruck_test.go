package garbagetruck

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type testLogger struct {
	Logger
}

func (l *testLogger) Debug(params ...interface{}) { fmt.Println(params) }
func (l *testLogger) Error(params ...interface{}) { fmt.Println(params) }

func setupGarbageTruck() *GarbageTruck {
	tm := time.Duration(30 * time.Second)
	lg := new(testLogger)
	wg := &sync.WaitGroup{}
	sd := make(chan struct{})

	gt := New(tm, lg, wg, sd)

	return gt
}

func TestSetInterval(t *testing.T) {
	gt := new(GarbageTruck)
	tm := time.Duration(10 * time.Second)
	gt.SetInterval(tm)

	if gt.interval != tm {
		t.Error("Failed to set interval correctly.  expected: %v, got: %v", tm, gt.interval)
	}
}

func TestSetLog(t *testing.T) {
	gt := new(GarbageTruck)
	lg := new(testLogger)
	gt.SetLog(lg)

	if gt.log != lg {
		t.Error("Failed to set log correctly.  expected: %v, got: %v", lg, gt.log)
	}
}

func TestRun(t *testing.T) {

}
