// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package concurrent

import (
	"os"
	"sync"
)

// Runnable represents any operation that can spawn zero or more goroutines.
type Runnable interface {
	// Run executes this operation, possibly returning an error if the operation
	// could not be started.  This method is responsible for spawning any necessary
	// goroutines and to ensure WaitGroup.Add() and WaitGroup.Done() are called appropriately.
	// Generally speaking, Run() should be idempotent.
	//
	// The supplied shutdown channel is used to signal any goroutines spawned by this
	// method that they should gracefully exit.  Callers can then use the waitGroup to
	// wait until things have been cleaned up properly.
	Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error
}

// RunnableFunc is a function type that implements Runnable
type RunnableFunc func(*sync.WaitGroup, <-chan struct{}) error

func (r RunnableFunc) Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
	return r(waitGroup, shutdown)
}

// RunnableSet is a slice type that allows grouping of operations.
// This type implements Runnable as well.
type RunnableSet []Runnable

func (set RunnableSet) Run(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
	for _, operation := range set {
		if err := operation.Run(waitGroup, shutdown); err != nil {
			return err
		}
	}

	return nil
}

// Execute is a convenience function that creates the necessary synchronization objects
// and then invokes Run().
func Execute(runnable Runnable) (waitGroup *sync.WaitGroup, shutdown chan struct{}, err error) {
	waitGroup = &sync.WaitGroup{}
	shutdown = make(chan struct{})
	err = runnable.Run(waitGroup, shutdown)
	return
}

// Await uses Execute() to invoke a runnable, then waits for any traffic
// on a signal channel before shutting down gracefully.
func Await(runnable Runnable, signals <-chan os.Signal) error {
	waitGroup, shutdown, err := Execute(runnable)
	if err != nil {
		return err
	}

	<-signals

	close(shutdown)
	waitGroup.Wait()
	return nil
}
