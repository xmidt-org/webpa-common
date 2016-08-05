package key

import (
	"github.com/Comcast/webpa-common/concurrent"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// dummyKeyId is used when no actual keyId is necessary.
	dummyKeyId = ""
)

// KeyCache is a Resolver type which provides caching for keys based on keyId.
//
// All implementations will block the first time a particular key is accessed
// and will initialize the value for that key.  Thereafter, all updates happen
// in a separate goroutine.  This allows HTTP transactions to avoid paying
// the cost of loading a key after the initial fetch.
type KeyCache interface {
	Resolver

	// UpdateKeys updates all keys known to this cache.  This method makes
	// a best-effort to avoid blocking other goroutines which use ResolveKey,
	// which may mean copy-on-write semantics.
	//
	// The first return value is the count of keys for which attempts were
	// made to update.
	//
	// UpdateKeys may run multiple I/O operations.  The second return value is a slice of
	// errors that occurred while it attempted to update each key.  Exactly one (1)
	// attempt will be made to update each key present in the cache, regardless
	// of any update errors for each individual key.  This slice may be nil if no
	// errors occurred.
	UpdateKeys() (int, []error)
}

// basicCache contains the internal members common to all cache implementations
type basicCache struct {
	delegate   Resolver
	value      atomic.Value
	updateLock sync.Mutex
}

func (b *basicCache) load() interface{} {
	return b.value.Load()
}

func (b *basicCache) store(newValue interface{}) {
	b.value.Store(newValue)
}

// update provides a critical section for an update operation
func (b *basicCache) update(operation func()) {
	b.updateLock.Lock()
	defer b.updateLock.Unlock()
	operation()
}

func (b *basicCache) UsesKeyId() bool {
	return b.delegate.UsesKeyId()
}

// singleKeyCache assumes that the delegate Resolver
// only returns (1) key.
type singleKeyCache struct {
	basicCache
}

func (cache *singleKeyCache) ResolveKey(keyId string) (key interface{}, err error) {
	key = cache.load()
	if key == nil {
		cache.update(func() {
			key = cache.load()
			if key == nil {
				key, err = cache.delegate.ResolveKey(keyId)
				if err == nil {
					cache.store(key)
				}
			}
		})
	}

	return
}

func (cache *singleKeyCache) UpdateKeys() (count int, errors []error) {
	count = 1
	cache.update(func() {
		// this type of cache is specifically for resolvers which don't use the keyId,
		// so just pass an empty string in
		if key, err := cache.delegate.ResolveKey(dummyKeyId); err == nil {
			cache.store(key)
		} else {
			errors = []error{err}
		}
	})

	return
}

// multiKeyCache uses an atomic map reference to store keys.
// Once created, each internal map instance will never be written
// to again, thus removing the need to lock for reads.  This approach
// does consume more memory, however.  The updateLock ensures that only
// (1) goroutine will ever be updating the map at anytime.
type multiKeyCache struct {
	basicCache
}

// fetchKey uses the atomic reference to the keys map and attempts
// to fetch the key from the cache.
func (cache *multiKeyCache) fetchKey(keyId string) (key interface{}, ok bool) {
	if keys, hasKeys := cache.load().(map[string]interface{}); hasKeys {
		key, ok = keys[keyId]
	}

	return
}

// copyKeys creates a copy of the current key cache.  If no keys are present
// yet, this method returns a non-nil empty map.
func (cache *multiKeyCache) copyKeys() map[string]interface{} {
	keys, _ := cache.load().(map[string]interface{})

	// make the capacity 1 larger, since this method is almost always
	// going to be invoked prior to doing a copy-on-write update.
	newKeys := make(map[string]interface{}, len(keys)+1)

	for currentId, currentKey := range keys {
		newKeys[currentId] = currentKey
	}

	return newKeys
}

func (cache *multiKeyCache) ResolveKey(keyId string) (key interface{}, err error) {
	key, ok := cache.fetchKey(keyId)
	if !ok {
		cache.update(func() {
			key, ok = cache.fetchKey(keyId)
			if !ok {
				key, err = cache.delegate.ResolveKey(keyId)
				if err == nil {
					newKeys := cache.copyKeys()
					newKeys[keyId] = key
					cache.store(newKeys)
				}
			}
		})
	}

	return
}

func (cache *multiKeyCache) UpdateKeys() (count int, errors []error) {
	if existingKeys, ok := cache.load().(map[string]interface{}); ok {
		count = len(existingKeys)
		cache.update(func() {
			newCount := 0
			newKeys := make(map[string]interface{}, len(existingKeys))
			for keyId, oldKey := range existingKeys {
				if newKey, err := cache.delegate.ResolveKey(keyId); err == nil {
					newCount++
					newKeys[keyId] = newKey
				} else {
					// keep the old key in the event of an error
					newKeys[keyId] = oldKey
					errors = append(errors, err)
				}
			}

			// small optimization: don't bother doing the atomic swap
			// if every key operation failed
			if newCount > 0 {
				cache.store(newKeys)
			}
		})
	}

	return
}

// NewUpdater conditionally creates a Runnable which will update the keys in
// the given resolver on the configured updateInterval.  If both (1) the
// updateInterval is positive, and (2) resolver implements KeyCache, then this
// method returns a non-nil function that will spawn a goroutine to update
// the cache in the background.  Otherwise, this method returns nil.
func NewUpdater(updateInterval time.Duration, resolver Resolver) (concurrent.Runnable, bool) {
	if updateInterval < 1 {
		return nil, false
	}

	if keyCache, ok := resolver.(KeyCache); ok {
		return concurrent.RunnableFunc(func(waitGroup *sync.WaitGroup, shutdown <-chan struct{}) error {
			waitGroup.Add(1)

			go func() {
				defer waitGroup.Done()

				ticker := time.NewTicker(updateInterval)
				defer ticker.Stop()

				for {
					select {
					case <-shutdown:
						return
					case <-ticker.C:
						keyCache.UpdateKeys()
					}
				}
			}()

			return nil
		}), true
	}

	return nil, false
}
