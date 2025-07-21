// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package concurrent

import (
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// success returns a closure that simulates a successfully started task
func success(t *testing.T, runCount *uint32) Runnable {
	return RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		atomic.AddUint32(runCount, 1)
		waitGroup.Add(1)

		// simulates some longrunning task ...
		go func() {
			defer waitGroup.Done()
			<-shutdown
		}()

		return nil
	})
}

// fail returns a closure that simulates a task that failed to start
func fail(t *testing.T, runCount *uint32) Runnable {
	return RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		atomic.AddUint32(runCount, 1)
		return errors.New("Expected error")
	})
}

func TestRunnableSetRun(t *testing.T) {
	var actualRunCount uint32
	success := success(t, &actualRunCount)
	fail := fail(t, &actualRunCount)

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
		waitGroup := &sync.WaitGroup{}
		shutdown := make(chan struct{})
		record.runnable.Run(waitGroup, shutdown)
		close(shutdown)

		if !WaitTimeout(waitGroup, time.Second*2) {
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

func TestExecuteSuccess(t *testing.T) {
	var actualRunCount uint32
	success := success(t, &actualRunCount)
	waitGroup, shutdown, err := Execute(success)
	if actualRunCount != 1 {
		t.Error("Execute() did not invoke Run()")
	}

	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	if waitGroup == nil {
		t.Fatal("Execute() returned a nil WaitGroup")
	}

	if shutdown == nil {
		t.Fatal("Execute() returned a nil shutdown channel")
	}

	close(shutdown)

	if !WaitTimeout(waitGroup, time.Second*2) {
		t.Errorf("Blocked on WaitGroup longer than the timeout")
	}
}

func TestExecuteFail(t *testing.T) {
	var actualRunCount uint32
	fail := fail(t, &actualRunCount)
	waitGroup, shutdown, err := Execute(fail)
	if actualRunCount != 1 {
		t.Error("Execute() did not invoke Run()")
	}

	if err == nil {
		t.Error("Execute() should have returned an error")
	}

	if waitGroup == nil {
		t.Fatal("Execute() returned a nil WaitGroup")
	}

	if shutdown == nil {
		t.Fatal("Execute() returned a nil shutdown channel")
	}

	close(shutdown)

	if !WaitTimeout(waitGroup, time.Second*2) {
		t.Errorf("Blocked on WaitGroup longer than the timeout")
	}
}

func TestAwaitSuccess(t *testing.T) {
	testWaitGroup := &sync.WaitGroup{}
	testWaitGroup.Add(1)
	success := RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			defer testWaitGroup.Done()
			<-shutdown
		}()

		return nil
	})

	signals := make(chan os.Signal, 1)
	go func() {
		Await(success, signals)
	}()

	// simulate a ctrl+c
	signals <- os.Interrupt

	if !WaitTimeout(testWaitGroup, time.Second*2) {
		t.Errorf("Blocked on WaitGroup longer than the timeout")
	}
}

func TestAwaitFail(t *testing.T) {
	testWaitGroup := &sync.WaitGroup{}
	testWaitGroup.Add(1)
	fail := RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
		defer testWaitGroup.Done()
		return errors.New("Expected error")
	})

	signals := make(chan os.Signal, 1)
	go func() {
		Await(fail, signals)
	}()

	// simulate a ctrl+c
	signals <- os.Interrupt

	if !WaitTimeout(testWaitGroup, time.Second*2) {
		t.Errorf("Blocked on WaitGroup longer than the timeout")
	}
}
