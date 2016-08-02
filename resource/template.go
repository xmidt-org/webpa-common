package resource

import (
	"github.com/jtacoma/uritemplates"
	"net/http"
)

// Template is a factory type which allows URI template expansion
// to produce Loaders.  A Template instance acts as a sort of "factory of factories",
// delegating to a Factory instance to deal with expanded resource URIs.
type Template struct {
	URITemplate *uritemplates.UriTemplate
	Header      http.Header
	Method      string
	HTTPClient  httpClient
}

func (t *Template) String() string {
	return t.URITemplate.String()
}

// Expand uses the supplied value to expand the URITemplate.  Internally, a Factory
// instance wraps the expanded URI and is then used to produce the Loader.
//
// The value used to expand the URI template is passed to UriTemplate.Expand().  It
// can be a map or a struct.
func (t *Template) Expand(value interface{}) (Loader, error) {
	uri, err := t.URITemplate.Expand(value)
	if err != nil {
		return nil, err
	}

	return (&Factory{
		URI:        uri,
		Header:     t.Header,
		Method:     t.Method,
		HTTPClient: t.HTTPClient,
	}).NewLoader()
}
