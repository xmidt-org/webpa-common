package key

import (
	"fmt"
	"github.com/Comcast/webpa-common/concurrent"
	"github.com/Comcast/webpa-common/resource"
	"github.com/Comcast/webpa-common/types"
	"time"
)

const (
	KeyIdParameterName = "keyId"
	DefaultContentType = "text/plain"
)

var (
	ErrorInvalidTemplate = fmt.Errorf(
		"Key resource template must support either no parameters are the %s parameter",
		KeyIdParameterName,
	)
)

// ResolverFactory provides a JSON representation of a collection of keys together
// with a factory interface for creating distinct Resolver instances.
//
// This factory uses NewTemplate() to create a resource template used in resolving keys.
// This template can have no parameters, in which case the same resource is used regardless
// of the key id.  If the template has any parameters, it must have exactly (1) parameter
// and that parameter must be called "keyId".
type ResolverFactory struct {
	resource.Factory

	// All keys resolved by this factory will have this purpose, which affects
	// how keys are parsed.
	Purpose Purpose `json:"purpose"`

	// UpdateInterval specifies how often keys should be refreshed.
	// If negative or zero, keys are never refreshed and are cached forever.
	UpdateInterval types.Duration `json:"updateInterval"`
}

// NewResolver creates a Resolver using this factory's configuration.  The
// returned Resolver always caches keys forever once they have been loaded.
// Use NewUpdater to create an updater to freshen keys using this factory's configuration.
func (rf *ResolverFactory) NewResolver() (Resolver, error) {
	expander, err := rf.NewExpander()
	if err != nil {
		return nil, err
	}

	names := expander.Names()
	switch len(names) {
	case 0:
		loader, err := rf.Factory.NewLoader()
		if err != nil {
			return nil, err
		}

		return &singleKeyCache{
			basicCache{
				delegate: &singleResolver{
					loader: loader,
					parser: rf.Purpose,
				},
			},
		}, nil

	case 1:
		if names[0] == KeyIdParameterName {
			return &multiKeyCache{
				basicCache{
					delegate: &multiResolver{
						expander: expander,
						parser:   rf.Purpose,
					},
				},
			}, nil
		}

		fallthrough

	default:
		return nil, ErrorInvalidTemplate
	}
}

// NewUpdater uses this factory's configured UpdatedInterval and invokes the
// standalone NewUpdater function.
func (rf *ResolverFactory) NewUpdater(resolver Resolver) (concurrent.Runnable, bool) {
	return NewUpdater(time.Duration(rf.UpdateInterval), resolver)
}
