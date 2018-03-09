package service

import (
	"errors"
	"fmt"
	"sync"
)

var (
	errNoInstances = errors.New("There are no instances available")
)

// Accessor holds a hash of server nodes.
type Accessor interface {
	// Get fetches the server node associated with a particular key.
	Get(key []byte) (string, error)
}

type emptyAccessor struct{}

func (ea emptyAccessor) Get([]byte) (string, error) {
	return "", errNoInstances
}

// EmptyAccessor returns an Accessor that always returns an error from Get.
func EmptyAccessor() Accessor {
	return emptyAccessor{}
}

// MapAccessor is a static Accessor that honors a set of known keys.  Any other key
// will result in an error.  Mostly useful for testing.
type MapAccessor map[string]string

func (ma MapAccessor) Get(key []byte) (string, error) {
	if v, ok := ma[string(key)]; ok {
		return v, nil
	} else {
		return "", fmt.Errorf("No such key: %s", string(key))
	}
}

// UpdatableAccessor is an Accessor whose contents can be mutated safely under concurrency.
// The zero value of this struct is a valid Accessor initialized with no instances.  Get will
// return an error until there is an update with at least (1) instance.
type UpdatableAccessor struct {
	lock sync.RWMutex

	err     error
	current Accessor
}

// Get hashes the key against the current set of instances to select an instance consistently.
// This method will return an error if this instance isn't updated yet or has been updated with
// no instances.
func (ua *UpdatableAccessor) Get(key []byte) (instance string, err error) {
	ua.lock.RLock()

	switch {
	case ua.err != nil:
		err = ua.err

	case ua.current != nil:
		instance, err = ua.current.Get(key)

	default:
		err = errNoInstances
	}

	ua.lock.RUnlock()
	return
}

// SetError clears the instances being used by this instance and sets the error to be returned
// by Get with every call.  This error will be returned by Get until an update with one or more instances
// occurs.
func (ua *UpdatableAccessor) SetError(err error) {
	ua.lock.Lock()
	ua.err = err
	ua.current = nil
	ua.lock.Unlock()
}

// SetInstances changes the instances used by this UpdateAccessor, clearing any error.  Note that Get will
// still return an error if a is nil or empty.
func (ua *UpdatableAccessor) SetInstances(a Accessor) {
	ua.lock.Lock()
	ua.err = nil
	ua.current = a
	ua.lock.Unlock()
}

// Update sets both the instances and the Get error in a single, atomic call.
func (ua *UpdatableAccessor) Update(a Accessor, err error) {
	ua.lock.Lock()
	ua.err = err
	ua.current = a
	ua.lock.Unlock()
}
