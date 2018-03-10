package service

import (
	"io"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

func NopCloser() error { return nil }

// Environment represents everything known about a service discovery backend.  It also
// provide a central handle for tasks related to service discovery, such as Accessor hashing.
type Environment interface {
	sd.Registrar
	io.Closer

	// Instancers returns the set of sd.Instancer objects associated with this environment.
	// This method can return nil or empty.
	Instancers() Instancers

	// AccessorFactory returns the creation strategy for Accessors used in this environment.
	// Typically, this factory is set via configuration by some external source.
	AccessorFactory() AccessorFactory

	// Closed returns a channel that is closed when this Environment in closed.
	Closed() <-chan struct{}
}

// Option represents a service discovery option for configuring an Environment
type Option func(*environment)

// WithRegistrars configures the set of sd.Registrar objects for use in the environment.
// The Regisrars may be nil or empty for applications which have no need of registering themselves
// with the service discovery backend.
func WithRegistrars(r Registrars) Option {
	return func(e *environment) {
		e.registrars = r
	}
}

// WithInstancers configures the set of sd.Instancer objects for use in the environment.
// The Instancers may be nil or empty for applications which have no need of monitoring
// discovered services.
func WithInstancers(i Instancers) Option {
	return func(e *environment) {
		e.instancers = i
	}
}

// WithAccessorFactory configures the creation strategy for Accessor objects.  By default,
// DefaultAccessorFactory is used.  Passing nil via this option sets (or resets) the environment
// back to using the DefaultAccessorFactory.
func WithAccessorFactory(af AccessorFactory) Option {
	return func(e *environment) {
		if af == nil {
			e.accessorFactory = DefaultAccessorFactory
		} else {
			e.accessorFactory = af
		}
	}
}

// WithCloser configures the function used to completely shut down the service discover backend.
// By default, NopCloser is used.  Passing a nil function for this option sets (or resets)
// the closer back to the NopCloser.
func WithCloser(f func() error) Option {
	return func(e *environment) {
		if f == nil {
			e.closer = NopCloser
		} else {
			e.closer = f
		}
	}
}

// NewEnvironment constructs a new service discovery client environment.  It is possible to construct
// an environment without any Registrars or Instancers, which essentially makes a no-op environment.
func NewEnvironment(options ...Option) Environment {
	e := &environment{
		accessorFactory: DefaultAccessorFactory,
		closer:          NopCloser,
		closed:          make(chan struct{}),
	}

	for _, o := range options {
		o(e)
	}

	return e
}

type environment struct {
	registrars      Registrars
	instancers      Instancers
	accessorFactory AccessorFactory

	closeOnce sync.Once
	closer    func() error
	closed    chan struct{}
}

func (e *environment) Instancers() Instancers {
	return e.instancers.Copy()
}

func (e *environment) AccessorFactory() AccessorFactory {
	return e.accessorFactory
}

func (e *environment) Register() {
	e.registrars.Register()
}

func (e *environment) Deregister() {
	e.registrars.Deregister()
}

func (e *environment) Closed() <-chan struct{} {
	return e.closed
}

// Close completely shuts down this environment.  Any registrars are deregistered, all instancers are stopped,
// the Closed() channel is closed, and any configured closer function is invoked.  This method is idempotent
// and safe for concurrent execution.
func (e *environment) Close() (err error) {
	e.closeOnce.Do(func() {
		e.Deregister()

		e.instancers.Each(func(_ string, _ log.Logger, i sd.Instancer) {
			i.Stop()
		})

		close(e.closed)
		err = e.closer()
	})

	return
}
