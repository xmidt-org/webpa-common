package garbagetruck

import (
	"github.com/ian-kent/go-log/logger"
	"sync"
	"testing"
	"time"
)

func setupGarbageTruck() *GarbageTruck {
	tm := time.Duration(30 * time.Second)
	lg := logger.New("TestLog")
	wg := &sync.WaitGroup{}
	
	gt := New(tm, lg, wg)
	gt.Start()
	
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
	lg := logger.New("TestLog")
	gt.SetLog(lg)
	
	if gt.log != lg {
		t.Error("Failed to set log correctly.  expected: %v, got: %v", lg, gt.log)
	}
}

func TestSetWaitGroup(t *testing.T) {
	gt := new(GarbageTruck)
	wg := &sync.WaitGroup{}
	gt.SetWaitGroup(wg)
	
	if gt.wg != wg {
		t.Error("Failed to set sync.WaitGroup correctly.  expected: %v, got: %v", wg, gt.wg)
	}
}

func TestStop(t *testing.T) {
	gt := setupGarbageTruck()
	gt.Stop()
	
	if _, ok := <- gt.stop; ok {
		t.Error("Failed to close channel")
	}
}


func TestStart(t *testing.T) {
	
}
