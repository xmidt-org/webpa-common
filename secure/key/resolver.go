package key

import (
	"errors"
	"fmt"
	"github.com/Comcast/webpa-common/store"
)

var (
	NoSuchKey = errors.New("Key not found")
)

type keyId struct {
	name    string
	purpose Purpose
}

type ResolverFactory []ValueFactory

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

type Resolver struct {
	values map[keyId]store.Value
}

func (r *Resolver) ResolveKeyValue(name string, purpose Purpose) (store.Value, error) {
	valueId := keyId{
		name:    name,
		purpose: purpose,
	}

	if value, ok := r.values[valueId]; ok {
		return value, nil
	}

	return nil, NoSuchKey
}
