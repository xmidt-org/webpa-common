package service

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/billhathaway/consistentHash"
)

// Accessor provides access to services based around []byte keys.
// *consistentHash.ConsistentHash implements this interface.
type Accessor interface {
	Get([]byte) (string, error)
}

// AccessorFactory is a Factory Interface for creating service Accessors.
type AccessorFactory interface {
	// New creates an Accessor using a slice of endpoints
	New([]string) Accessor
}

// NewAccessoryFactory uses a set of Options to produce an AccessorFactory
func NewAccessorFactory(o *Options) AccessorFactory {
	return &consistentHashFactory{
		logger:     o.logger(),
		vnodeCount: o.vnodeCount(),
	}
}

// consistentHashFactory creates consistentHash instances, which implement Accessor.
// This is the standard implementation of AccessoryFactory.
type consistentHashFactory struct {
	logger     logging.Logger
	vnodeCount int
}

func (f *consistentHashFactory) New(endpoints []string) Accessor {
	hash := consistentHash.New()
	hash.SetVnodeCount(f.vnodeCount)
	for _, hostAndPort := range endpoints {
		f.logger.Debug("adding %s", hostAndPort)
		hash.Add(hostAndPort)
	}

	return hash
}

// Subscribe returns a channel which receives new Accessors when watch events occur.
// The pipeline of Accessors can be used for rehashing.
//
// The returned channel will have a buffer size of one (1).  The goroutine which processes
// watch events will block waiting for this channel to become writable.
func Subscribe(o *Options, factory AccessorFactory, watcher Watcher) (<-chan Accessor, error) {
	logger := o.logger()
	watch, err := watcher.Watch()
	if err != nil {
		return nil, err
	}

	if factory == nil {
		factory = NewAccessorFactory(o)
	}

	accessors := make(chan Accessor, 1)
	go func() {
		defer close(accessors)

		endpoints := watch.Endpoints()
		logger.Info("Initial discovered endpoints: %s", endpoints)
		accessors <- factory.New(endpoints)

		for {
			select {
			case <-watch.Event():
				if watch.IsClosed() {
					return
				}

				endpoints = watch.Endpoints()
				logger.Info("Updated endpoints: %s", endpoints)
				accessors <- factory.New(endpoints)
			}
		}
	}()

	return accessors, nil
}
