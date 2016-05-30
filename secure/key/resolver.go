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
type ResolverFactory []ValueFactory

// NewResolver creates a distinct Resolver using this factory's configuration.
func (rf ResolverFactory) NewResolver() (*Resolver, error) {
	resolver := &Resolver{
		values: make(map[keyId]store.Value, 10),
	}

	for _, valueFactory := range rf {
		value, err := valueFactory.NewValue()
		if err != nil {
			return nil, err
		}

		valueId := keyId{
			name:    valueFactory.Name,
			purpose: valueFactory.Purpose,
		}

		if _, ok := resolver.values[valueId]; !ok {
			resolver.values[valueId] = value
		} else {
			return nil,
				fmt.Errorf("Duplicate key: %s, %s", valueId.name, valueId.purpose)
		}
	}

	return resolver, nil
}

// Resolver maintains an immutable registry of keys
type Resolver struct {
	values map[keyId]store.Value
}

// ResolveKey returns the parsed key.
func (r *Resolver) ResolveKey(name string, purpose Purpose) (interface{}, error) {
	valueId := keyId{
		name:    name,
		purpose: purpose,
	}

	if value, ok := r.values[valueId]; ok {
		return value.Load()
	}

	return nil, NoSuchKey
}
