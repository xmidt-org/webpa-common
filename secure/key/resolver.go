package key

import (
	"fmt"
	"github.com/Comcast/webpa-common/resource"
)

// Resolver loads and parses keys associated with key identifiers.
type Resolver interface {
	// ResolveKey returns a key with the given identifier.  The exact mechanics of resolving
	// a keyId into the raw key data are implementation-specific.  Implementations are free
	// to ignore the keyId parameter altogether.
	ResolveKey(keyId string) (interface{}, error)
}

// singleResolver is a Resolver which expects only (1) key for all key ids.
type singleResolver struct {
	loader resource.Loader
	parser Parser
}

func (r *singleResolver) String() string {
	return fmt.Sprintf("%s: %s", r.parser, r.loader)
}

func (r *singleResolver) ResolveKey(keyId string) (interface{}, error) {
	data, err := resource.ReadAll(r.loader)
	if err != nil {
		return nil, err
	}

	return r.parser.ParseKey(data)
}

// multiResolver is a Resolver which uses the key id and will most likely return
// different keys for each key id value.
type multiResolver struct {
	expander resource.Expander
	parser   Parser
}

func (r *multiResolver) String() string {
	return fmt.Sprintf("%s: %s", r.parser, r.expander)
}

func (r *multiResolver) ResolveKey(keyId string) (interface{}, error) {
	values := map[string]interface{}{
		KeyIdParameterName: keyId,
	}

	loader, err := r.expander.Expand(values)
	if err != nil {
		return nil, err
	}

	data, err := resource.ReadAll(loader)
	if err != nil {
		return nil, err
	}

	return r.parser.ParseKey(data)
}
