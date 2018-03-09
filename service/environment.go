package service

import (
	"io"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
)

func nopCloser() error { return nil }

// Environment represents everything known about a service discovery backend.  It also
// provide a central handle for tasks related to service discovery, such as Accessor hashing.
type Environment interface {
	sd.Registrar
	io.Closer

	Registrars() Registrars
	Instancers() Instancers
	Closed() <-chan struct{}
}

type Option func(*environment)

func WithRegistrars(r Registrars) Option {
	return func(e *environment) {
		e.registrars = r
	}
}

func WithInstancers(i Instancers) Option {
	return func(e *environment) {
		e.instancers = i
	}
}

func WithAccessorFactory(af AccessorFactory) Option {
	return func(e *environment) {
		if af == nil {
			e.accessorFactory = DefaultAccessorFactory
		} else {
			e.accessorFactory = af
		}
	}
}

func WithCloser(f func() error) Option {
	return func(e *environment) {
		if f == nil {
			e.closer = nopCloser
		} else {
			e.closer = f
		}
	}
}

func NewEnvironment(options ...Option) Environment {
	e := &environment{
		accessorFactory: DefaultAccessorFactory,
		closer:          nopCloser,
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

func (e *environment) Registrars() Registrars {
	return e.registrars
}

func (e *environment) Instancers() Instancers {
	return e.instancers
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

func (e *environment) Close() (err error) {
	e.closeOnce.Do(func() {
		e.Deregister()

		e.instancers.Each(func(_ string, _ log.Logger, i sd.Instancer) {
			i.Stop()
		})

		err = e.closer()
	})

	return
}
