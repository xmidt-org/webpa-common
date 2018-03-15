package service

import (
	"errors"
	"testing"

	"github.com/Comcast/webpa-common/service/servicemock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNopCloser(t *testing.T) {
	assert := assert.New(t)
	assert.NotPanics(func() {
		assert.NoError(NopCloser())
	})
}

func testNewEnvironmentNoOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		e = NewEnvironment()
	)

	require.NotNil(e)
	assert.NotPanics(e.Register)
	assert.NotPanics(e.Deregister)
	assert.Equal(0, e.Instancers().Len())
	assert.NotNil(e.AccessorFactory())

	select {
	case <-e.Closed():
		assert.Fail("The closed channel should still be open")
	default:
		// the passing case
	}

	assert.NoError(e.Close())

	select {
	case <-e.Closed():
		// the passing case
	default:
		assert.Fail("The closed channel should have been closed")
	}

	// idempotency
	assert.NoError(e.Close())

	select {
	case <-e.Closed():
		// the passing case
	default:
		assert.Fail("The closed channel should have been closed")
	}
}

func testNewEnvironmentWithOptions(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		registrar = new(servicemock.Registrar)
		instancer = new(servicemock.Instancer)

		accessorFactoryCalled = false
		accessorFactory       = func(i []string) Accessor {
			accessorFactoryCalled = true
			return EmptyAccessor()
		}

		expectedCloseError = errors.New("expected")
		closerCalled       = false
		closer             = func() error {
			closerCalled = true
			return expectedCloseError
		}

		e = NewEnvironment(
			WithRegistrar(registrar),
			WithInstancers(Instancers{"test": instancer}),
			WithAccessorFactory(accessorFactory),
			WithCloser(closer),
		)
	)

	require.NotNil(e)

	registrar.On("Register").Once()
	registrar.On("Deregister").Twice() // during the NotPanics assert and Close()
	instancer.On("Stop").Once()        // only during Close()

	assert.NotPanics(e.Register)
	assert.NotPanics(e.Deregister)
	assert.Equal(Instancers{"test": instancer}, e.Instancers())
	require.NotNil(e.AccessorFactory())
	assert.NotNil(e.AccessorFactory()([]string{}))
	assert.True(accessorFactoryCalled)

	select {
	case <-e.Closed():
		assert.Fail("The closed channel should still be open")
	default:
		// the passing case
	}

	assert.Equal(expectedCloseError, e.Close())
	assert.True(closerCalled)

	select {
	case <-e.Closed():
		// the passing case
	default:
		assert.Fail("The closed channel should have been closed")
	}

	// idempotency
	closerCalled = false
	assert.NoError(e.Close())
	assert.False(closerCalled)

	select {
	case <-e.Closed():
		// the passing case
	default:
		assert.Fail("The closed channel should have been closed")
	}

	registrar.AssertExpectations(t)
	instancer.AssertExpectations(t)
}

func testNewEnvironmentExplicitDefaultAccessorFactory(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		e = NewEnvironment(
			WithAccessorFactory(nil),
		)
	)

	require.NotNil(e)
	assert.NotNil(e.AccessorFactory())
}

func testNewEnvironmentExplicitNopCloser(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)

		e = NewEnvironment(
			WithCloser(nil),
		)
	)

	require.NotNil(e)
	assert.NotPanics(func() {
		assert.NoError(e.Close())
	})
}

func TestNewEnvironment(t *testing.T) {
	t.Run("NoOptions", testNewEnvironmentNoOptions)
	t.Run("WithOptions", testNewEnvironmentWithOptions)
	t.Run("ExplicitDefaultAccessorFactory", testNewEnvironmentExplicitDefaultAccessorFactory)
	t.Run("ExplicitNopCloser", testNewEnvironmentExplicitNopCloser)
}
