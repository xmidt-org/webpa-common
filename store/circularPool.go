package store

import (
	"errors"
	"time"
)

var (
	ErrorInvalidMaxSize     = errors.New("The maxSize of a pool must be positive")
	ErrorInvalidInitialSize = errors.New("The initialSize of a pool cannot be larger than maxSize")
	ErrorNewRequired        = errors.New("A new function is required")
	ErrorPoolClosed         = errors.New("The pool is already closed")
)

// Pool represents an object pool.  sync.Pool satisfies this interface,
// as does CircularPool from this package.
type Pool interface {
	// Get fetches an object from the pool.  Implementations will vary on
	// the behavior of an empty pool.
	Get() interface{}

	// Put inserts and object into the pool.  Get and Put are typically used
	// with the same type, though that is not necessarily enforced.
	Put(value interface{})
}

// CircularPool represents a fixed-size pool whose objects rotate in
// and out.  Like sync.Pool, clients should not expect any relationship
// between objects obtained from a CircularPool and objects put into a
// CircularPool.
//
// Get will block until an object is returned to the pool. GetOrNew, however,
// will create a new object if the pool is exhausted.  This allows a client a
// choice between being limited by the pool or treating the pool as the minimum
// amount of objects to keep around.
//
// Put will drop objects if invoked when the pool is full.  This can happen
// if a client uses GetOrNew.
//
// The main difference between CircularPool and sync.Pool is that objects within
// a CircularPool will not get deallocated.  This makes a CircularPool appropriate
// for times when a statically-sized pool of permanent objects is desired.
//
// A CircularPool is a very flexible concurrent data structure.  It can be used as a simple pool
// of objects for canonicalization.  It can also be used for rate limiting or any situation where
// a lease is required.
type CircularPool interface {
	Pool

	// TryGet will never block.  It will attempt to grab an object from the pool,
	// returning nil and false if it cannot do so.
	TryGet() (interface{}, bool)

	// GetCancel will wait for an available object with cancellation semantics
	// similar to golang's Context.  Any activity, such as close, on the supplied
	// channel will interrupt the wait and this method will return nil and false.
	GetCancel(<-chan struct{}) (interface{}, bool)

	// GetTimeout will wait for an available object for the specified timeout.
	// If the timeout is not positive, it behaves exactly as TryGet.  Otherwise, this
	// method waits the given duration for an available object, returning nil
	// and false if the timeout elapses without an available object.
	GetTimeout(time.Duration) (interface{}, bool)

	// GetOrNew will never block.  If the pool is exhausted, the new function
	// will be used to create a new object.  This can be used to allow the number
	// objects to grow beyond the size of the pool.  However, when Put is called,
	// and this pool is full, the object is dropped.
	GetOrNew() interface{}
}

// NewCircularPool creates a fixed-size rotating object pool.  maxSize is the maximum
// number of objects in the pool, while initialSize controls how many objects are preallocated.
// Setting initialSize to 0 yields an initially empty pool.  If maxSize < 1 or initialSize > maxSize,
// and error is returned.
//
// The new function is used to create objects for the pool.  This function cannot be nil, or an
// error is returned, and must be safe for concurrent execution.  If initialSize > 0, then
// the new function is used to preallocate objects for the pool.
//
// For an initially empty pool, be aware that Get will block forever.  In that case, another
// goroutine must call Put in order to release the goroutine waiting on Get.  Initially empty
// pools are appropriate as concurrent barriers, for example.
func NewCircularPool(initialSize, maxSize int, new func() interface{}) (CircularPool, error) {
	if maxSize < 1 {
		return nil, ErrorInvalidMaxSize
	} else if initialSize > maxSize {
		return nil, ErrorInvalidInitialSize
	} else if new == nil {
		return nil, ErrorNewRequired
	}

	pool := &circularPool{
		new:     new,
		objects: make(chan interface{}, maxSize),
	}

	for preallocate := 0; preallocate < initialSize; preallocate++ {
		pool.objects <- pool.new()
	}

	return pool, nil
}

// circularPool is the internal implementation of CircularPool.  It's based around a channel
// that stores the objects.
type circularPool struct {
	new     func() interface{}
	objects chan interface{}
}

func (pool *circularPool) Get() interface{} {
	return <-pool.objects
}

func (pool *circularPool) TryGet() (interface{}, bool) {
	select {
	case value := <-pool.objects:
		return value, true
	default:
		return nil, false
	}
}

func (pool *circularPool) GetCancel(done <-chan struct{}) (interface{}, bool) {
	select {
	case value := <-pool.objects:
		return value, true
	case <-done:
		return nil, false
	}
}

func (pool *circularPool) GetTimeout(timeout time.Duration) (interface{}, bool) {
	if timeout < 1 {
		return pool.TryGet()
	}

	timer := time.NewTimer(timeout)

	// Stop() performs a tiny cleanup in the event
	// that a value was obtained before the timer fired
	defer timer.Stop()

	select {
	case value := <-pool.objects:
		return value, true
	case <-timer.C:
		return nil, false
	}
}

func (pool *circularPool) Put(value interface{}) {
	select {
	case pool.objects <- value:
	default:
	}
}

func (pool *circularPool) GetOrNew() interface{} {
	select {
	case value := <-pool.objects:
		return value
	default:
		return pool.new()
	}
}
