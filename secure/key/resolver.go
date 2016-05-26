package key

import (
	"errors"
	"fmt"
)

// Resolver provides the behavior for obtaining keys for a particular purpose.
// A Resolver instance is considered immutable after construction.
type Resolver interface {
	ResolveKey(name string, purpose Purpose) (interface{}, error)
}

// loaderKey is used as the composite key for MapResolver
type loaderKey struct {
	name    string
	purpose Purpose
}

// mapResolver holds an in-memory map of Loader instances.  Internally, the tuple
// of (keyName, keyPurpose) is used as the key for each Loader.
type mapResolver struct {
	loaders map[loaderKey]Loader
}

// ResolveKey implementation for mapResolver.  This method looks up the key in an in-memory
// data structure, returning a nil key if no such key is found.
func (resolver *mapResolver) ResolveKey(name string, purpose Purpose) (interface{}, error) {
	if resolver.loaders != nil {
		if loader := resolver.loaders[loaderKey{name, purpose}]; loader != nil {
			return loader.LoadKey()
		}
	}

	return nil, nil
}

// ResolverBuilder implements both a builder for Resolver instances and the
// external JSON representation of a Resolver.
type ResolverBuilder []LoaderBuilder

// NewResolver creates a new, immutable Resolver from this builder's configuration
func (builder *ResolverBuilder) NewResolver() (Resolver, error) {
	loaders := make(map[loaderKey]Loader, len(*builder))

	for _, loaderBuilder := range *builder {
		loader, err := loaderBuilder.NewLoader()
		if err != nil {
			return nil, err
		}

		mapKey := loaderKey{loader.Name(), loader.Purpose()}
		if _, duplicate := loaders[mapKey]; duplicate {
			return nil, errors.New(fmt.Sprintf("Duplicate key name: %s", loader.Name()))
		}

		loaders[mapKey] = loader
	}

	return &mapResolver{loaders: loaders}, nil
}
