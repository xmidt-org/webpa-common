// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package semaphore

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrTimeout is returned when a timeout occurs while waiting to acquire a semaphore resource.
	// This error does not apply when using a context.  ctx.Err() is returned in that case.
	ErrTimeout = errors.New("The semaphore could not be acquired within the timeout")
)

// Interface represents a semaphore, either binary or counting.  When any acquire method is successful,
// Release *must* be called to return the resource to the semaphore.
type Interface interface {
	// Acquire acquires a resource.  Typically, this method will block forever.  Some semaphore implementations,
	// e.g. closeable semaphores, can immediately return an error from this method.
	Acquire() error

	// AcquireWait attempts to acquire a resource before the given time channel becomes signaled.
	// If the resource was acquired, this method returns nil.  If the time channel gets signaled
	// before a resource is available, ErrTimeout is returned.
	AcquireWait(<-chan time.Time) error

	// AcquireCtx attempts to acquire a resource before the given context is canceled.  If the resource
	// was acquired, this method returns nil.  Otherwise, this method returns ctx.Err().
	AcquireCtx(context.Context) error

	// TryAcquire attempts to acquire a release, returning false immediately if a resource was unavailable.
	// This method returns true if the resource was acquired.
	TryAcquire() bool

	// Release relinquishes control of a resource.  If called before a corresponding acquire method,
	// this method will likely result in a deadlock.  This method must be invoked after a successful
	// acquire in order to allow other goroutines to use the resource(s).
	//
	// Typically, this method returns a nil error.  It can return a non-nil error, as with a closeable semaphore
	// that has been closed.
	Release() error
}

// New constructs a semaphore with the given count.  A nonpositive count will result in a panic.
// A count of 1 is essentially a mutex, albeit with the ability to timeout or cancel the acquisition
// of the lock.
func New(count int) Interface {
	if count < 1 {
		panic("The count must be positive")
	}

	return &semaphore{
		c: make(chan struct{}, count),
	}
}

// Mutex is just syntactic sugar for New(1).  The returned object is a binary semaphore.
func Mutex() Interface {
	return New(1)
}

// semaphore is the internal Interface implementation
type semaphore struct {
	c chan struct{}
}

func (s *semaphore) Acquire() error {
	s.c <- struct{}{}
	return nil
}

func (s *semaphore) AcquireWait(t <-chan time.Time) error {
	select {
	case s.c <- struct{}{}:
		return nil
	case <-t:
		return ErrTimeout
	}
}

func (s *semaphore) AcquireCtx(ctx context.Context) error {
	select {
	case s.c <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *semaphore) TryAcquire() bool {
	select {
	case s.c <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s *semaphore) Release() error {
	<-s.c
	return nil
}
