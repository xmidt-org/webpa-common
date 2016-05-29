package store

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	NegativeCachePeriod = errors.New("The cache period must be positive")
)

// cacheEntry is an internal type that holds a value and a timestamp.
type cacheEntry struct {
	value  interface{}
	expiry time.Time
}

// Cache is an on-demand cache for an arbitrary value.  This type implements
// the Value interface.
//
// Cache management is done inline with the Load() method.  No other goroutine
// is necessary to monitor instances of this type.
type Cache struct {
	updateLock sync.Mutex
	cache      atomic.Value
	source     Value
	period     CachePeriod
}

// NewCache constructs a new Cache that uses the given source.  The period
// parameter must be positive, so special values such as CachePeriodForever are
// not supported by this function.
//
// To dynamically create a Value based on a CachePeriod, use NewValue() instead.
func NewCache(source Value, period CachePeriod) (*Cache, error) {
	if period <= 0 {
		return nil, NegativeCachePeriod
	}

	return &Cache{
		source: source,
		period: period,
	}, nil
}

// Refresh forces this cache to update itself.  Unlike Load(), this method will
// return errors that occur when consulting the source for a new value.  It will
// not update its cache with the old value and a new expiry when the source value
// returns an error.
func (c *Cache) Refresh() (interface{}, error) {
	now := time.Now()
	c.updateLock.Lock()
	defer c.updateLock.Unlock()

	if newValue, err := c.source.Load(); err != nil {
		// Unlike Load(), we don't want to update if there's an error.
		// Just let Load() keep returning the old value.
		return nil, err
	} else {
		entry := &cacheEntry{
			value:  newValue,
			expiry: c.period.Next(now),
		}

		c.cache.Store(entry)
		return entry.value, nil
	}
}

// Load will return the currently cached value if there is one and it hasn't expired.
// If the cache has no value yet or if the existing value has expired, this method
// consults the source Value to update its cache and returns the new value.
//
// In the event that a cache has an expired value and cannot get a fresh one from the source,
// this method uses the old cache value and sets a new expiry time.  This allows the application
// to continue using the old value in the event of intermediate I/O problems.  No error is returned
// in that case.
func (c *Cache) Load() (interface{}, error) {
	now := time.Now()
	entry, ok := c.cache.Load().(*cacheEntry)
	if !ok || entry.expiry.Before(now) {
		c.updateLock.Lock()
		defer c.updateLock.Unlock()

		entry, ok = c.cache.Load().(*cacheEntry)
		if !ok || entry.expiry.Before(now) {
			if newValue, err := c.source.Load(); err != nil {
				if !ok {
					// we couldn't even get the initial value for the cache ....
					return nil, err
				}

				// hang on to the old cached value in the event of an error
				// this prevents this code from hammering external resources
				// if there are I/O errors
				// TODO: record the error somehow
				entry = &cacheEntry{
					value:  entry.value,
					expiry: c.period.Next(now),
				}

				c.cache.Store(entry)
			} else {
				entry = &cacheEntry{
					value:  newValue,
					expiry: c.period.Next(now),
				}

				c.cache.Store(entry)
			}
		}
	}

	return entry.value, nil
}
