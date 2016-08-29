package key

import (
	"fmt"
	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/resource"
	"github.com/Comcast/webpa-common/types"
	"time"
)

const (
	// KeyIdParameterName is the template parameter that must be present in URI templates
	// if there are any parameters.  URI templates accepted by this package have either no parameters
	// or exactly one (1) parameter with this name.
	KeyIdParameterName = "keyId"
)

var (
	// ErrorInvalidTemplate is the error returned when a URI template is invalid for a key resource
	ErrorInvalidTemplate = fmt.Errorf(
		"Key resource template must support either no parameters are the %s parameter",
		KeyIdParameterName,
	)
)

// ResolverFactory provides a JSON representation of a collection of keys together
// with a factory interface for creating distinct Resolver instances.
//
// This factory uses resource.NewExpander() to create a resource template used in resolving keys.
// This template can have no parameters, in which case the same resource is used regardless
// of the key id.  If the template has any parameters, it must have exactly (1) parameter
// and that parameter's name must be equal to KeyIdParameterName.
type ResolverFactory struct {
	resource.Factory

	// All keys resolved by this factory will have this purpose, which affects
	// how keys are parsed.
	Purpose Purpose `json:"purpose"`

	// UpdateInterval specifies how often keys should be refreshed.
	// If negative or zero, keys are never refreshed and are cached forever.
	UpdateInterval types.Duration `json:"updateInterval"`
}

// NewResolver() creates a Resolver using this factory's configuration.  The
// returned Resolver always caches keys forever once they have been loaded.
func (factory *ResolverFactory) NewResolver() (Resolver, error) {
	expander, err := factory.NewExpander()
	if err != nil {
		return nil, err
	}

	names := expander.Names()
	nameCount := len(names)
	if nameCount == 0 {
		// the template had no parameters, so we can create a simpler object
		loader, err := factory.NewLoader()
		if err != nil {
			return nil, err
		}

		return &singleCache{
			basicCache{
				delegate: &singleResolver{
					loader: loader,
					parser: factory.Purpose,
				},
			},
		}, nil
	} else if nameCount == 1 && names[0] == KeyIdParameterName {
		return &multiCache{
			basicCache{
				delegate: &multiResolver{
					expander: expander,
					parser:   factory.Purpose,
				},
			},
		}, nil
	}

	return nil, ErrorInvalidTemplate
}

// NewUpdater uses this factory's configuration to conditionally create a Runnable updater
// for the given resolver.  This method delegates to the NewUpdater function, and may
// return a nil Runnable if no updates are necessary.
func (factory *ResolverFactory) NewUpdater(resolver Resolver) concurrent.Runnable {
	return NewUpdater(time.Duration(factory.UpdateInterval), resolver)
}
