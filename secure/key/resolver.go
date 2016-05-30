package key

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/store"
)

var (
	NoSuchKey = errors.New("Key not found")
)

// keyId is an internal type used as a composite map key
type keyId struct {
	name    string
	purpose Purpose
}

// ResolverFactory provides a JSON representation of a collection of keys together
// with a factory interface for creating distinct Resolver instances.
type ResolverFactory []Factory

// NewResolver creates a distinct Resolver using this factory's configuration.
func (rf ResolverFactory) NewResolver() (*Resolver, error) {
	resolver := &Resolver{
		keys: make(map[keyId]store.Value, 10),
	}

	for _, factory := range rf {
		key, err := factory.NewKey()
		if err != nil {
			return nil, err
		}

		keyId := keyId{
			name:    factory.Name,
			purpose: factory.Purpose,
		}

		if _, ok := resolver.keys[keyId]; !ok {
			resolver.keys[keyId] = key
		} else {
			return nil,
				fmt.Errorf("Duplicate key: %s, %s", keyId.name, keyId.purpose)
		}
	}

	return resolver, nil
}

// Resolver maintains an immutable registry of keys
type Resolver struct {
	keys map[keyId]store.Value
}

// ResolveKey returns the store.Value containing the actual key.
func (r *Resolver) ResolveKey(name string, purpose Purpose) (store.Value, error) {
	keyId := keyId{
		name:    name,
		purpose: purpose,
	}

	if key, ok := r.keys[keyId]; ok {
		return key, nil
	}

	return nil, NoSuchKey
}
