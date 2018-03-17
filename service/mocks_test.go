package service

import (
	"errors"
	"testing"

	"github.com/go-kit/kit/sd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockRegistrar(t *testing.T) {
	r := new(MockRegistrar)
	r.On("Register").Once()
	r.On("Deregister").Once()

	r.Register()
	r.Deregister()
	r.AssertExpectations(t)
}

func TestMockInstancer(t *testing.T) {
	var (
		i      = new(MockInstancer)
		events = make(chan sd.Event, 5)
	)

	i.On("Register", (chan<- sd.Event)(events)).Once()
	i.On("Deregister", (chan<- sd.Event)(events)).Once()
	i.On("Stop").Once()

	i.Register(events)
	i.Deregister(events)
	i.Stop()
	i.AssertExpectations(t)
}

func TestMockEnvironment(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		e             = new(MockEnvironment)
		instance1     = "localhost:8080"
		instance2     = "localhost:8081"
		defaultScheme = "ftp"
		instancers    = Instancers{"test": new(MockInstancer)}

		closed     = make(chan struct{})
		closeError = errors.New("expected close error")

		accessorFactoryCalled = false
		accessorFactory       = AccessorFactory(func(i []string) Accessor {
			accessorFactoryCalled = true
			return EmptyAccessor()
		})
	)

	e.On("Register").Once()
	e.On("Deregister").Once()
	e.On("Close").Return(closeError).Once()
	e.On("IsRegistered", instance1).Return(true).Once()
	e.On("IsRegistered", instance2).Return(false).Once()
	e.On("DefaultScheme").Return(defaultScheme).Once()
	e.On("Instancers").Return(instancers).Once()
	e.On("AccessorFactory").Return(accessorFactory).Once()
	e.On("Closed").Return((<-chan struct{})(closed))

	e.Register()
	e.Deregister()
	assert.Equal(closeError, e.Close())
	assert.True(e.IsRegistered(instance1))
	assert.False(e.IsRegistered(instance2))
	assert.Equal(defaultScheme, e.DefaultScheme())
	assert.Equal(instancers, e.Instancers())

	af := e.AccessorFactory()
	require.NotNil(af)
	assert.Equal(EmptyAccessor(), af([]string{}))
	assert.True(accessorFactoryCalled)

	assert.Equal((<-chan struct{})(closed), e.Closed())

	e.AssertExpectations(t)
}
