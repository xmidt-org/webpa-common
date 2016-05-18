package server

import (
	"sync"
)

// Runnable represents any operation that can spawn zero or more goroutines.
type Runnable interface {
	// Run executes this operation, possibly returning an error if the operation
	// could not be started.  It is the responsibility of this method to invoke
	// WaitGroup.Add() for any goroutines it spawns.  Generally speaking, Run() should be idempotent.
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
