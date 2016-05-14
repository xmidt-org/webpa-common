package server

import (
	"sync"
)

// Runnable represents any operation that can spawn zero or more goroutines.
type Runnable interface {
	// Run executes this operation, possibly returning an error if the operation
	// could not be started.  It is the responsibility of this method to invoke
	// WaitGroup.Add() for any goroutines it spawns.
	//
	// Generally speaking, Run() should be idempotent.
	Run(*sync.WaitGroup) error
}

// RunnableSet is a slice type that allows grouping of operations.
// This type implements Runnable as well.
type RunnableSet []Runnable

func (set RunnableSet) Run(waitGroup *sync.WaitGroup) error {
	for _, operation := range set {
		if err := operation.Run(waitGroup); err != nil {
			return err
		}
	}

	return nil
}
