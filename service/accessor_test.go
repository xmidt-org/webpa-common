package service

import (
	"errors"
	"sort"
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

type testRouter struct {
	allow map[string]struct{}
}

func (r *testRouter) addRoute(instance string) {
	r.allow[instance] = struct{}{}
}

func (r *testRouter) removeRoute(instance string) {
	delete(r.allow, instance)
}

var testErrorNoRoute = errors.New("instance does not match allowed instance")

func (r testRouter) Route(instance string) error {
	if _, ok := r.allow[instance]; ok {
		return nil
	}
	return testErrorNoRoute
}

type testOrder struct {}

func (r testOrder) Order(keys []string) []string {
	sort.Strings(keys)
	return keys
}

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
	assert.Equal(RouteError{ErrChain: &ErrorChain{Err: errNoRouter}, Instance: i}, err)

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
	assert.Equal(RouteError{ErrChain: &ErrorChain{Err: errNoFailOvers, SubError: expectedError}}.Error(), err.Error())
	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Equal(RouteError{ErrChain: &ErrorChain{Err: errNoFailOvers, SubError: expectedError}}.Error(), err.Error())

	primaryInstance := "a valid instance"
	la.UpdatePrimary(MapAccessor{"test": primaryInstance}, nil)
	i, err = la.Get([]byte("test"))
	assert.Equal(primaryInstance, i)
	assert.NoError(err)
	i, err = la.Get([]byte("nosuch"))
	assert.Empty(i)
	assert.Error(err)

	r := testRouter{make(map[string]struct{})}
	la.SetRouter(r)
	la.SetAccessorQueue(DefaultOrder())

	i, err = la.Get([]byte("test"))
	assert.Equal("a valid instance", i)
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: errNoFailOvers, SubError: testErrorNoRoute}}.Error(), err.Error())

	dc2Instance := "a valid instance in dc2"
	r.addRoute(dc2Instance)
	la.UpdateFailOver("dc2", MapAccessor{"test": dc2Instance}, nil)
	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: testErrorNoRoute}}.Error(), err.Error())
	assert.Equal(dc2Instance, i)

	dc1Instance := "a valid instance in dc1"
	r.addRoute(dc1Instance)
	la.UpdateFailOver("dc1", MapAccessor{"test": dc1Instance}, nil)

	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: testErrorNoRoute}}.Error(), err.Error())
	assert.Equal(dc2Instance, i)

	la.SetAccessorQueue(testOrder{})
	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: testErrorNoRoute}}.Error(), err.Error())
	assert.Equal(dc1Instance, i)

	r.removeRoute(dc1Instance)
	i, err = la.Get([]byte("test"))
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: testErrorNoRoute}}.Error(), err.Error())
	assert.Equal(dc2Instance, i)

	expectedError = errors.New("data center went down")
	la.UpdatePrimary(EmptyAccessor(), expectedError)
	i, err = la.Get([]byte("test"))
	assert.Equal(dc2Instance, i)
	assert.Equal(RouteError{Instance: i, ErrChain: &ErrorChain{Err: expectedError}}.Error(), err.Error())
}
