package key

import (
	"github.com/Comcast/webpa-common/resource"
	"github.com/Comcast/webpa-common/store"
)

// source is the internal store.Value implementation that loads the actual key
type source struct {
	purpose Purpose
	raw     resource.Loader
}

func (s *source) Load() (interface{}, error) {
	data, err := resource.ReadAll(s.raw)
	if err != nil {
		return nil, err
	}

	return s.purpose.ParseKey(data)
}

// Factory creates store.Value instances that load keys.
// This type also exposes a JSON representation for configuration.
type Factory struct {
	Name        string                 `json:"name"`
	Purpose     Purpose                `json:"purpose"`
	Resource    resource.LoaderFactory `json:"resource"`
	CachePeriod store.CachePeriod      `json:"cachePeriod"`
}

func (f *Factory) NewKey() (store.Value, error) {
	source := &source{
		purpose: f.Purpose,
		raw:     f.Resource.NewLoader(),
	}

	return store.NewValue(source, f.CachePeriod)
}
