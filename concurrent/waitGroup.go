package concurrent

import (
	"sync"
	"time"
)

// WaitGroup is an extension of sync.WaitGroup that supplies additional behavior
type WaitGroup struct {
	sync.WaitGroup
}

// WaitTimeout waits on this WaitGroup until either the wait succeeds or the
// timeout elapses.  This method returns true if sync.WaitGroup.Wait() returned
// within the timeout, false if the timeout elapsed.
func (wait *WaitGroup) WaitTimeout(timeout time.Duration) bool {
	success := make(chan bool)
	go func() {
		defer func() {
			// swallow any panics, as they'll just be from the channel
			// close if the timeout elapsed
			recover()
		}()
		defer close(success)
		wait.Wait()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-success:
		return true
	case <-timer.C:
		return false
	}
}
