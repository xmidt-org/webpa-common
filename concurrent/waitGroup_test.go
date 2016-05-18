package concurrent

import (
	"testing"
	"time"
)

func TestUnwrap(t *testing.T) {
	waitGroup := &WaitGroup{}
	if waitGroup.Unwrap() == nil {
		t.Errorf("Unwrap should return the wrapped sync.WaitGroup")
	}
}

func TestWaitGroupSuccess(t *testing.T) {
	waitGroup := &WaitGroup{}
	waitGroup.Add(1)
	go func() {
		timer := time.NewTimer(time.Millisecond * 500)
		defer timer.Stop()
		<-timer.C
		waitGroup.Done()
	}()

	if !waitGroup.WaitTimeout(time.Millisecond * 1000) {
		t.Errorf("Failed wait within the timeout")
	}
}

func TestWaitGroupFail(t *testing.T) {
	waitGroup := &WaitGroup{}
	waitGroup.Add(1)
	go func() {
		timer := time.NewTimer(time.Second * 3)
		defer timer.Stop()
		<-timer.C
		waitGroup.Done()
	}()

	if waitGroup.WaitTimeout(time.Millisecond * 500) {
		t.Errorf("WaitTimeout() should return false if the timeout elapses without Wait() succeeding")
	}
}
