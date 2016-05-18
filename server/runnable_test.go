package server

import (
	"errors"
	"github.com/Comcast/webpa-common/concurrent"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunnableSetRun(t *testing.T) {
	var actualRunCount uint32
	var expectedWaitGroup *sync.WaitGroup

	// simulates the successful start of a runnable task
	success := RunnableFunc(func(actualWaitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		atomic.AddUint32(&actualRunCount, 1)
		if expectedWaitGroup != actualWaitGroup {
			t.Errorf("Unexpected wait group passed to Runnable")
		}

		go func() {
			defer actualWaitGroup.Done()
			actualWaitGroup.Add(1)

			// simulates some longrunning task ...
			<-shutdown
		}()

		return nil
	})

	// simulates the aborted start of a runnable task
	fail := RunnableFunc(func(actualWaitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		defer actualWaitGroup.Done()
		atomic.AddUint32(&actualRunCount, 1)
		actualWaitGroup.Add(1)
		if expectedWaitGroup != actualWaitGroup {
			t.Errorf("Unexpected wait group passed to Runnable")
		}

		return errors.New("Expected error")
	})

	var testData = []struct {
		runnable         RunnableSet
		expectedRunCount uint32
	}{
		{nil, 0},
		{RunnableSet{}, 0},
		{RunnableSet{success}, 1},
		{RunnableSet{fail}, 1},
		{RunnableSet{success, success}, 2},
		{RunnableSet{success, fail}, 2},
		{RunnableSet{success, fail, success}, 2},
		{RunnableSet{success, fail, fail}, 2},
		{RunnableSet{success, fail, success}, 2},
		{RunnableSet{success, success, fail, success, success, fail}, 3},
		{RunnableSet{success, success, success, success, fail}, 5},
		{RunnableSet{success, success, success, success, success}, 5},
	}

	for _, record := range testData {
		actualRunCount = 0
		waitGroup := &concurrent.WaitGroup{}
		shutdown := make(chan struct{})
		expectedWaitGroup = waitGroup.Unwrap()
		record.runnable.Run(expectedWaitGroup, shutdown)
		close(shutdown)

		if !waitGroup.WaitTimeout(time.Second * 2) {
			t.Errorf("Blocked on WaitGroup longer than the timeout")
		}

		if record.expectedRunCount != actualRunCount {
			t.Errorf(
				"Expected Run to be called %d time(s), but instead was called %d time(s)",
				record.expectedRunCount,
				actualRunCount,
			)
		}
	}
}
