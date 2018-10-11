package service

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyAccessor(t *testing.T) {
	var (
		assert  = assert.New(t)
		require = require.New(t)
		ea      = EmptyAccessor()
	)

	require.NotNil(ea)
	i, err := ea.Get([]byte("does not matter"))
	assert.Empty(i)
	assert.Error(err)
}

func TestMapAccessor(t *testing.T) {
	var (
		assert = assert.New(t)
		ma     = MapAccessor{"test": "a valid instance"}
	)

	i, err := ma.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)

	i, err = ma.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)
}

func TestUpdatableAccessor(t *testing.T) {
	var (
		assert = assert.New(t)
		ua     = new(UpdatableAccessor)
	)

	i, err := ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)

	ua.SetInstances(MapAccessor{"test": "a valid instance"})
	i, err = ua.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	ua.SetInstances(EmptyAccessor())
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	expectedError := errors.New("expected 1")
	ua.SetError(expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)

	ua.Update(MapAccessor{"test": "a valid instance"}, nil)
	i, err = ua.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.NoError(err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	expectedError = errors.New("expected 2")
	ua.Update(MapAccessor{"test": "a valid instance"}, expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)

	expectedError = errors.New("expected 3")
	ua.Update(nil, expectedError)
	i, err = ua.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
	i, err = ua.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(expectedError, err)
}

var testErrorNoRoute = errors.New("instance does not match allowed instance")

/****************** BEGIN MOCK DECLARATIONS ***********************/

type mockRouter struct {
	mock.Mock
}

func (r mockRouter) Route(instance string) error {
	args := r.Called(instance)
	return args.Error(0)
}

type mockOrder struct {
	mock.Mock
}

func (r mockOrder) Order(keys []string) []string {
	args := r.Called(keys)
	return args.Get(0).([]string)
}

/******************* END MOCK DECLARATIONS ************************/

func TestLayeredAccessor(t *testing.T) {
	var (
		assert = assert.New(t)
		la     = new(LayeredAccessor)
	)

	i, err := la.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)

	la.SetPrimary(MapAccessor{"test": "a valid instance"})
	i, err = la.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.Equal(RouteError{ErrChain: ErrorChain{Err: errNoRouter}, Instance: i}, err)

	la.SetRouter(DefaultTrafficRouter())

	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	la.SetPrimary(EmptyAccessor())
	i, err = la.Get([]byte("test"))
	assert.Empty(i)
	assert.Error(err)
	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	expectedError := errors.New("expected 1")
	la.SetError(expectedError)
	i, err = la.Get([]byte("test"))
	assert.Empty(i)
	assert.Equal(RouteError{ErrChain: ErrorChain{Err: errNoFailOvers, SubError: ErrorChain{Err: expectedError}}}, err)
	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(RouteError{ErrChain: ErrorChain{Err: errNoFailOvers, SubError: ErrorChain{Err: expectedError}}}, err)

	primaryInstance := "a valid instance"
	la.UpdatePrimary(MapAccessor{"test": primaryInstance}, nil)
	i, err = la.Get([]byte("test"))
	assert.Equal(primaryInstance, i)
	assert.NoError(err)
	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	fakeRouter := new(mockRouter)

	la.SetRouter(fakeRouter)
	la.SetAccessorQueue(DefaultOrder())

	expectedInstance := "a valid instance"
	fakeRouter.On("Route", expectedInstance).Return(testErrorNoRoute)
	i, err = la.Get([]byte("test"))
	assert.Equal(expectedInstance, i)
	assert.Equal(RouteError{Instance: i, ErrChain: ErrorChain{Err: errNoFailOvers, SubError: ErrorChain{Err: testErrorNoRoute}}}, err)

	dc2Instance := "a valid instance in dc2"
	fakeRouter.On("Route", dc2Instance).Return(nil).Twice()

	la.UpdateFailOver("dc2", MapAccessor{"test": dc2Instance}, nil)
	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: ErrorChain{Err: testErrorNoRoute}}, err)
	assert.Equal(dc2Instance, i)

	fakeOrder := new(mockOrder)
	la.SetAccessorQueue(fakeOrder)
	fakeOrder.On("Order", []string{"dc2", "dc1"}).Return([]string{"dc2", "dc1"}).Once()
	fakeOrder.On("Order", []string{"dc1", "dc2"}).Return([]string{"dc2", "dc1"}).Once()

	dc1Instance := "a valid instance in dc1"
	la.UpdateFailOver("dc1", MapAccessor{"test": dc1Instance}, nil)
	fakeRouter.On("Route", dc1Instance).Return(nil).Once()

	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: ErrorChain{Err: testErrorNoRoute}}, err)
	assert.Equal(dc2Instance, i)

	fakeOrder.On("Order", []string{"dc2", "dc1"}).Return([]string{"dc1", "dc2"})
	fakeOrder.On("Order", []string{"dc1", "dc2"}).Return([]string{"dc1", "dc2"})

	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: ErrorChain{Err: testErrorNoRoute}}, err)
	assert.Equal(dc1Instance, i)

	expectedError = errors.New("data center went down")
	la.UpdatePrimary(EmptyAccessor(), expectedError)
	la.UpdateFailOver("dc1", MapAccessor{"test": dc1Instance}, errors.New("region is closed"))

	fakeRouter.On("Route", dc2Instance).Return(nil).Once()
	i, err = la.Get([]byte("test"))
	assert.Equal(dc2Instance, i)
	expectedRouteErr := RouteError{Instance: i, ErrChain: ErrorChain{Err: expectedError}}
	assert.Equal(expectedRouteErr, err)
	assert.NotEmpty(expectedRouteErr.Error())

	fakeRouter.AssertExpectations(t)
}
