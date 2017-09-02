package service

import (
	"errors"
	"sync"
)

var (
	ErrAccessorUninitialized = errors.New("Accessor has not been initialized")
)

// UpdatableAccessor represents an Accessor whose state can be updated.
// Another Accessor is delegated to for Get calls, and this Accessor can
// be changed via Update.
type UpdatableAccessor struct {
	lock    sync.RWMutex
	current Accessor
}

// Get uses the current Accessor delegate to hash the key.  This method
// returns ErrAccessorUninitialized if there is no current Accessor (yet).
//
// It is safe to invoke this method concurrently with itself or Update.
func (u *UpdatableAccessor) Get(key []byte) (string, error) {
	defer u.lock.RUnlock()
	u.lock.RLock()
	if u.current == nil {
		return "", ErrAccessorUninitialized
	}

	return u.current.Get(key)
}

// Update changes the current Accessor delegate.  It is legal to call Update(nil),
// in which case Get will return ErrAccessorUninitialized.
//
// It is safe to invoke this method concurrently with itself or Get.
func (u *UpdatableAccessor) Update(a Accessor) {
	defer u.lock.Unlock()
	u.lock.Lock()
	u.current = a
}

// Consume spawns a goroutine that updates the current Accessor delegate in response
// to subscription events.  When the subscription is stopped, the goroutine spawned
// by this method exits.
//
// Consume may be called multiple times with different subscriptions.  Although calling
// this method repeatedly with the same subscription is allowed, it's not useful.
func (u *UpdatableAccessor) Consume(s *Subscription) {
	go func() {
		for {
			select {
			case a := <-s.Updates():
				u.Update(a)

			case <-s.Stopped():
				return
			}
		}
	}()
}
