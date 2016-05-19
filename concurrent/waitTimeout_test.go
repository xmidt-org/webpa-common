package concurrent

import (
	"sync"
	"testing"
	"time"
)

func TestWaitTimeoutSuccess(t *testing.T) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		timer := time.NewTimer(time.Millisecond * 500)
		defer timer.Stop()
		<-timer.C
		waitGroup.Done()
	}()

	if !WaitTimeout(waitGroup, time.Millisecond*1000) {
		t.Errorf("Failed wait within the timeout")
	}
}

func TestWaitTimeoutFail(t *testing.T) {
	waitGroup := &sync.WaitGroup{}
	waitGroup.Add(1)
	go func() {
		timer := time.NewTimer(time.Second * 3)
		defer timer.Stop()
		<-timer.C
		waitGroup.Done()
	}()

	if WaitTimeout(waitGroup, time.Millisecond*500) {
		t.Errorf("WaitTimeout() should return false if the timeout elapses without Wait() succeeding")
	}
}
