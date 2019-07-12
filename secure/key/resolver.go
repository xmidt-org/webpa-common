package key

import (
	"fmt"

	"github.com/xmidt-org/webpa-common/resource"
)

// Resolver loads and parses keys associated with key identifiers.
type Resolver interface {
	// ResolveKey returns a key Pair associated with the given identifier.  The exact mechanics of resolving
	// a keyId into a Pair are implementation-specific.  Implementations are free
	// to ignore the keyId parameter altogether.
	ResolveKey(keyId string) (Pair, error)
}

// basicResolver contains common items for all resolvers.
type basicResolver struct {
	parser  Parser
	purpose Purpose
}

func (b *basicResolver) parseKey(data []byte) (Pair, error) {
	return b.parser.ParseKey(b.purpose, data)
}

// singleResolver is a Resolver which expects only (1) key for all key ids.
type singleResolver struct {
	basicResolver
	loader resource.Loader
}

func (r *singleResolver) String() string {
	return fmt.Sprintf(
		"singleResolver{parser: %v, purpose: %v, loader: %v}",
		r.parser,
		r.purpose,
		r.loader,
	)
}

func (r *singleResolver) ResolveKey(keyId string) (Pair, error) {
	data, err := resource.ReadAll(r.loader)
	if err != nil {
		return nil, err
	}

	return r.parseKey(data)
}

// multiResolver is a Resolver which uses the key id and will most likely return
// different keys for each key id value.
type multiResolver struct {
	basicResolver
	expander resource.Expander
}

func (r *multiResolver) String() string {
	return fmt.Sprintf(
		"multiResolver{parser: %v, purpose: %v, expander: %v}",
		r.parser,
		r.purpose,
		r.expander,
	)
}

func (r *multiResolver) ResolveKey(keyId string) (Pair, error) {
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

	return r.parseKey(data)
}
