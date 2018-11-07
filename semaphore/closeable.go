package semaphore

import (
	"context"
	"errors"
	"io"
	"sync/atomic"
	"time"
)

var (
	// ErrClosed is returned when a closeable semaphore has been closed
	ErrClosed = errors.New("The semaphore has been closed")
)

const (
	stateOpen   int32 = 0
	stateClosed int32 = 1
)

// Closeable represents a semaphore than can be closed.  Once closed, a semaphore cannot be reopened.
//
// Any goroutines waiting for resources when a Closeable is closed will receive ErrClosed from the
// blocked acquire method.  Subsequent attempts to acquire resources will also result in ErrClosed.
//
// Both Close() and Release() are idempotent.  Once closed, both methods return ErrClosed without modifying
// the instance.
type Closeable interface {
	io.Closer
	Interface

	// Closed() returns a channel that is closed when this semaphore has been closed.
	// This channel has similar use cases to context.Done().
	Closed() <-chan struct{}
}

// NewCloseable returns a semaphore which honors close-once semantics.  The returned semaphore
// is not as efficient as that returned from New().
func NewCloseable(count int) Closeable {
	if count < 1 {
		panic("The count must be positive")
	}

	return &closeable{
		c:      make(chan struct{}, count),
		closed: make(chan struct{}),
	}
}

// CloseableMutex is syntactic sugar for NewCloseable(1)
func CloseableMutex() Closeable {
	return NewCloseable(1)
}

type closeable struct {
	c chan struct{}

	state  int32
	closed chan struct{}
}

func (cs *closeable) Close() error {
	if atomic.CompareAndSwapInt32(&cs.state, stateOpen, stateClosed) {
		close(cs.closed)
		return nil
	}

	return ErrClosed
}

func (cs *closeable) Closed() <-chan struct{} {
	return cs.closed
}

func (cs *closeable) checkClosed() bool {
	return atomic.LoadInt32(&cs.state) == stateClosed
}

func (cs *closeable) Acquire() error {
	if cs.checkClosed() {
		return ErrClosed
	}

	select {
	case cs.c <- struct{}{}:
		if cs.checkClosed() {
			return ErrClosed
		}

		return nil

	case <-cs.closed:
		return ErrClosed
	}
}

func (cs *closeable) AcquireWait(t <-chan time.Time) error {
	if cs.checkClosed() {
		return ErrClosed
	}

	select {
	case cs.c <- struct{}{}:
		if cs.checkClosed() {
			return ErrClosed
		}

		return nil

	case <-t:
		return ErrTimeout

	case <-cs.closed:
		return ErrClosed
	}
}

func (cs *closeable) AcquireCtx(ctx context.Context) error {
	if cs.checkClosed() {
		return ErrClosed
	}

	select {
	case cs.c <- struct{}{}:
		if cs.checkClosed() {
			return ErrClosed
		}

		return nil

	case <-ctx.Done():
		return ctx.Err()

	case <-cs.closed:
		return ErrClosed
	}
}

func (cs *closeable) TryAcquire() bool {
	if cs.checkClosed() {
		return false
	}

	select {
	case cs.c <- struct{}{}:
		if cs.checkClosed() {
			return false
		}

		return true

	default:
		return false
	}
}

func (cs *closeable) Release() error {
	if cs.checkClosed() {
		return ErrClosed
	}

	<-cs.c
	return nil
}
