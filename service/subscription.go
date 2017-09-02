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
type Subscription struct {
	errorLog log.Logger
	infoLog  log.Logger

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
func (s *Subscription) String() string {
	return fmt.Sprintf(
		"serviceName: %s, path: %s, updateDelay: %s",
		s.serviceName,
		s.path,
		s.updateDelay,
	)
}

// Stopped returns a channel that will be closed when this subscription has been stopped.
// This method is similar to context.Context.Done().
func (s *Subscription) Stopped() <-chan struct{} {
	return s.stopped
}

// Updates returns the channel on which updated Accessor instances are sent.  This channel
// is never closed, to avoid signalling clients spuriously.  To react to a subscription being
// stopped, use the Stopped method.
//
// The channel returned by this method receives the initial set of instances known to the
// Instancer prior to any updates.  Thus, clients can rely on being initialized properly
// via this channel.
func (s *Subscription) Updates() <-chan Accessor {
	return s.updates
}

// Stop halts all updates and deregisters this subscription with the Instancer.  This method
// is idempotent.  Once this method is called, no further updates will be send on the Updates
// channel, and the Stopped channel will be closed.
func (s *Subscription) Stop() {
	if atomic.CompareAndSwapUint32(&s.state, 0, 1) {
		close(s.stopped)
	}
}

// dispatch translates the given instances into an Accessor and sends that Accessor
// over the Updates channel
func (s *Subscription) dispatch(instances []string) {
	filtered := s.instancesFilter(instances)
	s.infoLog.Log(logging.MessageKey(), "dispatching updated instances", "instances", filtered)
	s.updates <- s.accessorFactory(filtered)
}

// monitor is the goroutine that dispatches updated Accessor objects in response to
// Instancer events.
func (s *Subscription) monitor(i sd.Instancer) {
	s.infoLog.Log(logging.MessageKey(), "subscription monitor starting")

	var (
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
	}()

	i.Register(events)

	for {
		select {
		case e := <-events:
			if e.Err != nil {
				s.errorLog.Log(logging.MessageKey(), "service discovery error", logging.ErrorKey(), e.Err)
			} else if s.updateDelay > 0 {
				if delay == nil {
					delay = s.after(s.updateDelay)
				}

				delayedInstances = make([]string, len(e.Instances))
				copy(delayedInstances, e.Instances)
				s.infoLog.Log(logging.MessageKey(), "waiting to dispatch updated instances", "instances", delayedInstances)
			} else {
				s.dispatch(e.Instances)
			}

		case <-delay:
			s.dispatch(delayedInstances)
			delay = nil
			delayedInstances = nil

		case <-s.stopped:
			return
		}
	}
}

// Subscribe starts monitoering an Instancer for updates.  The returned subscription will produce
// a stream of Accessor objects on its Updates channel.  The Updates channel will also receive
// the initial set of instances, similar to Instancer.Register.
func Subscribe(o *Options, i sd.Instancer) *Subscription {
	var (
		logger      = o.logger()
		serviceName = o.serviceName()
		path        = o.path()
		updateDelay = o.updateDelay()

		s = &Subscription{
			errorLog:        logging.Error(logger, "serviceName", serviceName, "path", path, "updateDelay", updateDelay),
			infoLog:         logging.Info(logger, "serviceName", serviceName, "path", path, "updateDelay", updateDelay),
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
