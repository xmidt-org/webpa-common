package service

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
)

var (
	errNoInstances     = errors.New("There are no instances available")
	errNoFailOvers     = errors.New("no failover instances available")
	errNoRouter        = errors.New("traffic router interface not set")
	errFailOversFailed = errors.New("failovers could not find an instance")
)

// Accessor holds a hash of server nodes.
type Accessor interface {
	// Get fetches the server node associated with a particular key.
	Get(key []byte) (string, error)
}

type emptyAccessor struct{}

func (ea emptyAccessor) Get([]byte) (string, error) {
	return "", errNoInstances
}

// EmptyAccessor returns an Accessor that always returns an error from Get.
func EmptyAccessor() Accessor {
	return emptyAccessor{}
}

// MapAccessor is a static Accessor that honors a set of known keys.  Any other key
// will result in an error.  Mostly useful for testing.
type MapAccessor map[string]string

func (ma MapAccessor) Get(key []byte) (string, error) {
	if v, ok := ma[string(key)]; ok {
		return v, nil
	} else {
		return "", fmt.Errorf("No such key: %s", string(key))
	}
}

// UpdatableAccessor is an Accessor whose contents can be mutated safely under concurrency.
// The zero value of this struct is a valid Accessor initialized with no instances.  Get will
// return an error until there is an update with at least (1) instance.
type UpdatableAccessor struct {
	lock sync.RWMutex

	err     error
	current Accessor
}

// Get hashes the key against the current set of instances to select an instance consistently.
// This method will return an error if this instance isn't updated yet or has been updated with
// no instances.
func (ua *UpdatableAccessor) Get(key []byte) (instance string, err error) {
	ua.lock.RLock()

	switch {
	case ua.err != nil:
		err = ua.err

	case ua.current != nil:
		instance, err = ua.current.Get(key)

	default:
		err = errNoInstances
	}

	ua.lock.RUnlock()
	return
}

// SetError clears the instances being used by this instance and sets the error to be returned
// by Get with every call.  This error will be returned by Get until an update with one or more instances
// occurs.
func (ua *UpdatableAccessor) SetError(err error) {
	ua.lock.Lock()
	ua.err = err
	ua.current = nil
	ua.lock.Unlock()
}

// SetInstances changes the instances used by this UpdateAccessor, clearing any error.  Note that Get will
// still return an error if a is nil or empty.
func (ua *UpdatableAccessor) SetInstances(a Accessor) {
	ua.lock.Lock()
	ua.err = nil
	ua.current = a
	ua.lock.Unlock()
}

// Update sets both the instances and the Get error in a single, atomic call.
func (ua *UpdatableAccessor) Update(a Accessor, err error) {
	ua.lock.Lock()
	ua.err = err
	ua.current = a
	ua.lock.Unlock()
}

type RouteTraffic interface {
	Route(instance string) error
}

type emptyRouter struct {
}

func (r emptyRouter) Route(instance string) error {
	return nil
}

func DefaultTrafficRouter() RouteTraffic {
	return emptyRouter{}
}

type OrderAccessor interface {
	Choose([]string) []string
}

type randomChooser struct {
	r *rand.Rand
}

func DefaultOrder() OrderAccessor {
	return randomChooser{}
}

func (r randomChooser) Choose(keys []string) []string {
	return keys
}

type AccessorValue struct {
	Accessor Accessor
	Err      error
}

type LayeredAccessor struct {
	lock sync.RWMutex

	router  RouteTraffic
	chooser OrderAccessor

	err     error
	primary Accessor

	failover map[string]AccessorValue
}

// SetRouter will update teh router, which will determine if the accessor should return the instance or fail
func (la *LayeredAccessor) SetRouter(router RouteTraffic) {
	la.lock.Lock()
	la.router = router
	la.lock.Unlock()
}

// SetRouter will update teh router, which will determine if the accessor should return the instance or fail
func (la *LayeredAccessor) SetChoooser(chooser OrderAccessor) {
	la.lock.Lock()
	la.chooser = chooser
	la.lock.Unlock()
}

// SetError clears the instances being used by this instance and sets the error to be returned
// by Get with every call.  This error will be returned by Get until an update with one or more instances
// occurs.
func (la *LayeredAccessor) SetError(err error) {
	la.lock.Lock()
	la.err = err
	la.primary = nil
	la.lock.Unlock()
}

// SetPrimary changes the instances used by this UpdateAccessor, clearing any error.  Note that Get will
// still return an error if a is nil or empty.
func (la *LayeredAccessor) SetPrimary(a Accessor) {
	la.lock.Lock()
	la.err = nil
	la.primary = a
	la.lock.Unlock()
}

// SetPrimary changes the instances used by this UpdateAccessor, clearing any error.  Note that Get will
// still return an error if a is nil or empty.
func (la *LayeredAccessor) SetFailOver(failover map[string]AccessorValue) {
	la.lock.Lock()
	la.failover = failover
	la.lock.Unlock()
}

// Update sets both the instances and the Get error in a single, atomic call.
func (la *LayeredAccessor) UpdatePrimary(a Accessor, err error) {
	la.lock.Lock()
	la.err = err
	la.primary = a
	la.lock.Unlock()
}

// Update sets the instances, failovers and the Get error in a single, atomic call.
func (la *LayeredAccessor) UpdateFailOver(key string, a Accessor, err error) {
	la.lock.Lock()
	if la.failover == nil {
		la.failover = make(map[string]AccessorValue)
	}

	la.failover[key] = AccessorValue{a, err}
	la.lock.Unlock()
}

// Get hashes the key against the current set of instances to select an instance consistently.
// This method will return an error if this instance isn't updated yet or has been updated with
// no instances.
func (la *LayeredAccessor) Get(key []byte) (string, error) {
	var instance string
	var err error
	la.lock.RLock()

	routeErr := RouteError{}

	switch {
	case la.err != nil:
		routeErr.addError(la.err)
		instance, err = la.getFailOverInstance(key)
		routeErr.addError(err)

	case la.primary != nil:
		instance, err = la.primary.Get(key)

		if err != nil {
			routeErr.addError(err)
			instance, err = la.getFailOverInstance(key)
			routeErr.addError(err)
		} else if la.router == nil {
			routeErr.addError(errNoRouter)
			break
		}

		if err := la.router.Route(instance); err != nil {
			routeErr.addError(err)
			tempInstance, err := la.getFailOverInstance(key)
			if err != nil {
				routeErr.addError(err)
			} else {
				instance = tempInstance
			}
		}
	case la.failover != nil && len(la.failover) > 0:
		instance, err = la.getFailOverInstance(key)
		routeErr.addError(err)
	default:
		routeErr.addError(errNoInstances)
	}

	la.lock.RUnlock()
	if routeErr.ErrChain == nil {
		return instance, nil
	}
	routeErr.Instance = instance
	return instance, routeErr
}

func (la *LayeredAccessor) getFailOverInstance(key []byte) (instance string, err error) {
	if la.failover == nil || len(la.failover) == 0 {
		return "", errNoFailOvers
	}

	var order []string
	dcs := make([]string, len(la.failover))
	index := 0
	for dc := range la.failover {
		dcs[index] = dc
		index++
	}
	if la.chooser != nil {
		order = la.chooser.Choose(dcs)
	} else {
		order = dcs
	}

	for _, dc := range order {
		instance, err = la.failover[dc].Accessor.Get(key)
		if la.router == nil && err == nil {
			return
		} else if la.router != nil {
			if tempErr := la.router.Route(instance); tempErr == nil {
				return
			}
		}

	}
	return "", errFailOversFailed
}

type ErrorChain struct {
	SubError error
	Err      error
}

func (err ErrorChain) Error() string {
	if err.SubError == nil {
		return err.Err.Error()
	}
	if err.Err == nil {
		panic("main Err can't be nil")
	}
	return fmt.Sprintf("%s(%s)", err.Err, err.SubError)
}

type RouteError struct {
	ErrChain *ErrorChain
	Instance string
}

func (err *RouteError) addError(e error) {
	if e != nil {
		if err.ErrChain == nil {
			err.ErrChain = &ErrorChain{Err: e}
		} else {
			err.ErrChain = &ErrorChain{Err: e, SubError: *err.ErrChain}
		}
	}
}

func (err RouteError) Error() string {
	return fmt.Sprintf("failed to route `%s`. reason: %s", err.Instance, err.ErrChain.Error())
}
