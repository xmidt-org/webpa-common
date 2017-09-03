package service

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

// Subscription represents a subscription to a specific Instancer.  A Subscription
// is initially active when created, and can be stopped via Stop.  Once stopped,
// a subscription cannot be restarted and will send no further updates.
type Subscription interface {
	// Stopped returns a channel that will be closed when this subscription has been stopped.
	// This method is similar to context.Context.Done().
	Stopped() <-chan struct{}

	// Stop halts all updates and deregisters this subscription with the Instancer.  This method
	// is idempotent.  Once this method is called, no further updates will be send on the updates
	// channel, and the Stopped channel will be closed.
	Stop()

	// Updates returns the channel that receives updates from the underlying Instancer.
	// This channel is never closed.  Use Stopped to react to this subscription being stopped.
	//
	// The returned channel is buffered, and the initial Accessor with the first set of instances
	// will be placed into the channel immediately when Subscribe is called.
	Updates() <-chan Accessor
}

// subscription is the internal Subscription implementation
type subscription struct {
	errorLog log.Logger
	infoLog  log.Logger
	debugLog log.Logger

	state   uint32
	stopped chan struct{}
	updates chan Accessor

	serviceName     string
	path            string
	updateDelay     time.Duration
	after           func(time.Duration) <-chan time.Time
	instancesFilter InstancesFilter
	accessorFactory AccessorFactory
}

// String returns a string representation of this Subscription, useful
// for logging and debugging.
func (s *subscription) String() string {
	return fmt.Sprintf(
		"serviceName: %s, path: %s, updateDelay: %s",
		s.serviceName,
		s.path,
		s.updateDelay,
	)
}

func (s *subscription) Stopped() <-chan struct{} {
	return s.stopped
}

func (s *subscription) Updates() <-chan Accessor {
	return s.updates
}

func (s *subscription) Stop() {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		close(s.stopped)
	}
}

// dispatch translates the given instances into an Accessor and sends that Accessor
// over the Updates channel
func (s *subscription) dispatch(instances []string) {
	filtered := s.instancesFilter(instances)
	s.infoLog.Log(logging.MessageKey(), "dispatching updated instances", "instances", filtered)
	s.updates <- s.accessorFactory(filtered)
}

// monitor is the goroutine that dispatches updated Accessor objects in response to
// Instancer events.
func (s *subscription) monitor(i sd.Instancer) {
	s.infoLog.Log(logging.MessageKey(), "subscription monitor starting")

	var (
		first            = true
		events           = make(chan sd.Event, 10)
		delayedInstances []string
		delay            <-chan time.Time
	)

	defer func() {
		if r := recover(); r != nil {
			s.errorLog.Log(logging.MessageKey(), "subscription monitor exiting", logging.ErrorKey(), r)
		} else {
			s.infoLog.Log(logging.MessageKey(), "subscription monitor exiting")
		}

		i.Deregister(events)

		// Always ensure that Stop is called to correctly reflect our state, esp. in the case of a panic
		// Stop is idempotent, so this will be safe.
		s.Stop()
	}()

	i.Register(events)

	for {
		select {
		case e := <-events:
			s.debugLog.Log(logging.MessageKey(), "service discovery event", "instances", e.Instances, logging.ErrorKey(), e.Err)

			switch {
			case e.Err != nil:
				s.errorLog.Log(logging.MessageKey(), "service discovery error", logging.ErrorKey(), e.Err)

			case first:
				// for the very first event, we want to dispatch immediately no matter what
				s.debugLog.Log(logging.MessageKey(), "dispatching first event immediately")
				first = false
				s.dispatch(e.Instances)

			case s.updateDelay > 0:
				if delay == nil {
					delay = s.after(s.updateDelay)
				}

				delayedInstances = make([]string, len(e.Instances))
				copy(delayedInstances, e.Instances)
				s.infoLog.Log(logging.MessageKey(), "waiting to dispatch updated instances", "instances", delayedInstances)

			default:
				s.dispatch(e.Instances)
			}

		case <-delay:
			s.debugLog.Log(logging.MessageKey(), "dispatching instances after delay")
			s.dispatch(delayedInstances)
			delay = nil
			delayedInstances = nil

		case <-s.stopped:
			s.infoLog.Log(logging.MessageKey(), "subscription stopped")
			return
		}
	}
}

// Subscribe starts monitoering an Instancer for updates.  The returned subscription will produce
// a stream of Accessor objects on the given updates channel.  The updates channel will also receive
// the initial set of instances, similar to Instancer.Register.  Slow consumers of updates will block
// subsequence update events.
func Subscribe(o *Options, i sd.Instancer) Subscription {
	var (
		logger      = o.logger()
		serviceName = o.serviceName()
		path        = o.path()
		updateDelay = o.updateDelay()

		s = &subscription{
			errorLog:        logging.Error(logger, "serviceName", serviceName, "path", path, "updateDelay", updateDelay),
			infoLog:         logging.Info(logger, "serviceName", serviceName, "path", path, "updateDelay", updateDelay),
			debugLog:        logging.Debug(logger, "serviceName", serviceName, "path", path, "updateDelay", updateDelay),
			stopped:         make(chan struct{}),
			updates:         make(chan Accessor, 10),
			serviceName:     serviceName,
			path:            path,
			updateDelay:     updateDelay,
			after:           o.after(),
			instancesFilter: o.instancesFilter(),
			accessorFactory: o.accessorFactory(),
		}
	)

	go s.monitor(i)
	return s
}
