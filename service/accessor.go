package service

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/billhathaway/consistentHash"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

const (
	DefaultVNodeCount = 211
)

var (
	ErrAccessorUninitialized     = errors.New("Accessor has not been initialized")
	ErrSubscriptionAlreadyClosed = errors.New("Subscription has already been closed")
)

// InstancesFilter represents a function which can preprocess slices of instances from the
// service discovery subsystem.
type InstancesFilter func([]string) []string

// DefaultInstancesFilter removes blank nodes and sorts the remaining nodes so that
// there is a consistent ordering.
func DefaultInstancesFilter(original []string) []string {
	filtered := make([]string, 0, len(original))

	for _, o := range original {
		f := strings.TrimSpace(o)
		if len(f) > 0 {
			filtered = append(filtered, f)
		}
	}

	sort.Strings(filtered)
	return filtered
}

// AccessorFactory defines the behavior of functions which can take a set
// of nodes and turn them into an Accessor.
//
// A Subscription will use an InstancesFilter prior to invoking this factory.
type AccessorFactory func([]string) Accessor

// ConsistentAccessorFactory produces a factory which uses consistent hashing
// of server nodes.
func ConsistentAccessorFactory(vnodeCount int) AccessorFactory {
	if vnodeCount < 1 {
		vnodeCount = DefaultVNodeCount
	}

	return func(instances []string) Accessor {
		hasher := consistentHash.New()
		hasher.SetVnodeCount(vnodeCount)
		for _, i := range instances {
			hasher.Add(i)
		}

		return hasher
	}
}

// Accessor holds a hash of server nodes.
type Accessor interface {
	// Get fetches the server node associated with a particular key.
	Get(key []byte) (string, error)
}

// Subscription represents an Accessor which is listening for updates from some the service discovery
// subsystem.  Once closed, a Subscription should be abandoned.
type Subscription interface {
	Accessor
	Update([]string)
	Close() error
}

// subscription is the internal Subscription implementation
type subscription struct {
	errorLog log.Logger
	infoLog  log.Logger

	state    uint32
	shutdown chan struct{}

	lock    sync.RWMutex
	current Accessor

	updateDelay time.Duration
	after       func(time.Duration) <-chan time.Time
	instancer   sd.Instancer
	filter      InstancesFilter
	factory     AccessorFactory
}

func (s *subscription) Get(key []byte) (string, error) {
	defer s.lock.RUnlock()
	s.lock.RLock()
	if s.current == nil {
		return "", ErrAccessorUninitialized
	}

	return s.current.Get(key)
}

func (s *subscription) Update(instances []string) {
	filtered := s.filter(instances)
	s.infoLog.Log(logging.MessageKey(), "updating instances", "filtered", true, "instances", filtered)

	defer s.lock.Unlock()
	s.lock.Lock()
	s.current = s.factory(filtered)
}

func (s *subscription) Close() error {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		close(s.shutdown)
		return nil
	}

	return ErrSubscriptionAlreadyClosed
}

func (s *subscription) run() {
	s.infoLog.Log(logging.MessageKey(), "monitor starting")
	var (
		events           = make(chan sd.Event, 5)
		delayedInstances []string
		delay            <-chan time.Time
	)

	defer func() {
		s.infoLog.Log(logging.MessageKey(), "monitor shutting down")
		s.instancer.Deregister(events)
	}()

	s.instancer.Register(events)

	for {
		select {
		case e := <-events:
			if e.Err != nil {
				s.errorLog.Log(logging.MessageKey(), "service discovery error", logging.ErrorKey(), e.Err)
			} else {
				if s.updateDelay > 0 {
					if delay == nil {
						delay = s.after(s.updateDelay)
					}

					s.infoLog.Log(logging.MessageKey(), "waiting to dispatch new instances", "delayed", true, "filtered", false, "instances", e.Instances)
					delayedInstances = make([]string, len(e.Instances))
					copy(delayedInstances, e.Instances)
				} else {
					// dispatch immediately
					s.Update(e.Instances)
					continue
				}
			}

		case <-delay:
			s.Update(delayedInstances)
			delay = nil
			delayedInstances = nil

		case <-s.shutdown:
			return
		}
	}
}

// NewSubscription creates a Subscription which monitors the given instancer.
func NewSubscription(o *Options, i sd.Instancer) Subscription {
	var (
		logger      = o.logger()
		vnodeCount  = o.vnodeCount()
		updateDelay = o.updateDelay()

		s = &subscription{
			errorLog:    logging.Error(logger, "serviceName", o.serviceName(), "path", o.path(), "vnodeCount", vnodeCount, "updateDelay", updateDelay),
			infoLog:     logging.Info(logger, "serviceName", o.serviceName(), "path", o.path(), "vnodeCount", vnodeCount, "updateDelay", updateDelay),
			shutdown:    make(chan struct{}),
			updateDelay: updateDelay,
			after:       time.After,
			instancer:   i,
			filter:      DefaultInstancesFilter,
			factory:     ConsistentAccessorFactory(vnodeCount),
		}
	)

	go s.run()
	return s
}
