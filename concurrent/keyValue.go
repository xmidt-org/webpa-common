package concurrent

import (
	"sync"
)

// KeyValueStorage is the map type used by KeyValue.  It is the type that is
// directly modifiable by operations.
type KeyValueStorage map[interface{}]interface{}

// KeyValueOperation represents an atomic operation that is allowed to mutate the
// storage of a KeyValue.  Operations are always executed within a critical
// section bounded by a write lock.
type KeyValueOperation interface {
	Execute(KeyValueStorage)
}

// KeyValueOperationFunc is a function type that implements KeyValueOperation.
type KeyValueOperationFunc func(KeyValueStorage)

func (f KeyValueOperationFunc) Execute(storage KeyValueStorage) {
	f(storage)
}

// KeyValueTransformer is a binary operation that produces a result from a key/value pair.
// Transformers cannot mutate the storage of a KeyValue.  Transformers are always
// executed within the context of a read lock.  Multiple transformers can execute
// simultaneously.
type KeyValueTransformer interface {
	Execute(key, value interface{}) interface{}
}

// KeyValueTransformerFunc is a function type that implements KeyValueTransformer.
type KeyValueTransformerFunc func(key, value interface{}) interface{}

func (f KeyValueTransformerFunc) Execute(key, value interface{}) interface{} {
	return f(key, value)
}

// KeyValue is a concurrent mapping of arbitrary types with a completely asynchronous API.
// Instances of this type must be created via NewKeyValue.
type KeyValue struct {
	storage KeyValueStorage
	lock    sync.RWMutex
}

// NewKeyValue initializes and returns a distinct KeyValue instance.
func NewKeyValue() *KeyValue {
	return &KeyValue{
		storage: make(KeyValueStorage),
	}
}

// Apply uses the given transformer to produce a result for each key/value pair in the storage.
// A channel of channels is returned: The channel has a buffer size of 1 and will receive another
// channel containing the results of applying the transformer.
func (kv *KeyValue) Apply(transformer KeyValueTransformer) <-chan chan interface{} {
	output := make(chan chan interface{}, 1)
	go func() {
		kv.lock.RLock()
		defer kv.lock.RUnlock()
		defer close(output)

		results := make(chan interface{}, len(kv.storage))
		defer close(results)
		output <- results

		for key, value := range kv.storage {
			results <- transformer.Execute(key, value)
		}
	}()

	return output
}

// Keys is a special usage of Apply:  It returns a channel which in turn receives a channel
// containing the keys in the internal storage.
func (kv *KeyValue) Keys() <-chan chan interface{} {
	return kv.Apply(
		KeyValueTransformerFunc(
			func(key, value interface{}) interface{} { return key },
		),
	)
}

// Values is a special usage of Apply:  It returns a channel which in turn receives a channel
// containing the values in the internal storage.
func (kv *KeyValue) Values() <-chan chan interface{} {
	return kv.Apply(
		KeyValueTransformerFunc(
			func(key, value interface{}) interface{} { return value },
		),
	)
}

// Do asynchronously executes a bulk operation against the internal storage.
// This method contends on the internal write lock.
func (kv *KeyValue) Do(operation KeyValueOperation) {
	go func() {
		kv.lock.Lock()
		defer kv.lock.Unlock()
		operation.Execute(kv.storage)
	}()
}

// Get asynchronously obtains the value associated with the given key.  The returned
// channel always receives exactly one (1) value.  It will receive nil if the given
// key was not present in the storage.
func (kv *KeyValue) Get(key interface{}) <-chan interface{} {
	output := make(chan interface{}, 1)
	go func() {
		kv.lock.RLock()
		defer kv.lock.RUnlock()
		defer close(output)
		output <- kv.storage[key]
	}()

	return output
}

// Add asynchronously adds (or, replaces) a key/value pair.
func (kv *KeyValue) Add(key, value interface{}) {
	go func() {
		kv.lock.Lock()
		defer kv.lock.Unlock()
		kv.storage[key] = value
	}()
}

// Delete asynchronously removes zero or more keys from the internal storage.
func (kv *KeyValue) Delete(keys ...interface{}) {
	if len(keys) > 0 {
		go func() {
			kv.lock.Lock()
			defer kv.lock.Unlock()
			for _, key := range keys {
				delete(kv.storage, key)
			}
		}()
	}
}
