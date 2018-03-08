package service

import (
	"errors"
	"sync"

	"github.com/billhathaway/consistentHash"
)

const (
	DefaultVnodeCount = 211
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

// AccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an Accessor.
type AccessorFactory func([]string) Accessor

// ConsistentAccessorFactory produces a factory which uses consistent hashing
// of server nodes.  The returned factory does not modify instances passed to it.
// Instances are hashed as is.
func ConsistentAccessorFactory(vnodeCount int) AccessorFactory {
	if vnodeCount < 1 {
		vnodeCount = DefaultVnodeCount
	}

	return func(instances []string) Accessor {
		if len(instances) == 0 {
			return emptyAccessor{}
		}

		hasher := consistentHash.New()
		hasher.SetVnodeCount(vnodeCount)
		for _, i := range instances {
			hasher.Add(i)
		}

		return hasher
	}
}

var defaultAccessorFactory AccessorFactory = ConsistentAccessorFactory(DefaultVnodeCount)

// DefaultAccessorFactory returns a global default AccessorFactory
func DefaultAccessorFactory() AccessorFactory {
	return defaultAccessorFactory
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
		return "", errNoInstances
	}

	ua.lock.RUnlock()
	return
}

func (ua *UpdatableAccessor) SetError(err error) {
	ua.lock.Lock()
	ua.err = err
	ua.current = nil
	ua.lock.Unlock()
}

func (ua *UpdatableAccessor) Update(a Accessor) {
	ua.lock.Lock()
	ua.err = nil
	ua.current = a
	ua.lock.Unlock()
}

// NewUpdatableListener creates a Listener for service discovery events that updates a given
// UpdatableAccessor.  The supplied factory is used to create Accessor instances.
func NewUpdatableListener(f AccessorFactory, ua *UpdatableAccessor) Listener {
	return func(e Event) {
		if e.Err != nil {
			ua.SetError(e.Err)
		} else {
			ua.Update(f(e.Instances))
		}
	}
}
