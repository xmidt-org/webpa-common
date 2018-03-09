package service

import (
	"io"
	"sync"

	"github.com/go-kit/kit/sd"
)

func nopCloser() error { return nil }

type Environment interface {
	sd.Registrar
	io.Closer

	AccessorFactory() AccessorFactory
	StartMonitor(...MonitorOption) (Monitor, error)
}

type EnvironmentOption func(*environment)

func WithRegistrars(r Registrars) EnvironmentOption {
	return func(e *environment) {
		e.r = r
	}
}

func WithInstancers(i Instancers) EnvironmentOption {
	return func(e *environment) {
		e.i = i
	}
}

func WithCloser(c func() error) EnvironmentOption {
	return func(e *environment) {
		if c == nil {
			e.c = nopCloser
		} else {
			e.c = c
		}
	}
}

func WithVnodeCount(v int) EnvironmentOption {
	return func(e *environment) {
		e.af = NewConsistentAccessorFactory(v)
	}
}

func WithAccessorFactory(af AccessorFactory) EnvironmentOption {
	return func(e *environment) {
		if af == nil {
			e.af = DefaultAccessorFactory
		} else {
			e.af = af
		}
	}
}

func NewEnvironment(options ...EnvironmentOption) Environment {
	e := &environment{
		c:  nopCloser,
		af: DefaultAccessorFactory,
	}

	for _, o := range options {
		o(e)
	}

	return e
}

type environment struct {
	r  Registrars
	i  Instancers
	af AccessorFactory
	c  func() error

	closeOnce sync.Once
}

func (e *environment) Register() {
	e.r.Register()
}

func (e *environment) Deregister() {
	e.r.Deregister()
}

func (e *environment) Close() (err error) {
	e.closeOnce.Do(func() {
		err = e.c()
	})

	return
}

func (e *environment) StartMonitor(options ...MonitorOption) (Monitor, error) {
	return StartMonitor(e.i, options...)
}

func (e *environment) AccessorFactory() AccessorFactory {
	return e.af
}
