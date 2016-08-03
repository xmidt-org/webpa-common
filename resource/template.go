package resource

import (
	"github.com/jtacoma/uritemplates"
	"net/http"
)

// Expander is a strategy for expanding URI templates into resource Loaders.
type Expander interface {
	// Expand uses the supplied object as a source for name/value pairs to use
	// when expanding the URI template.  Typically, this method is called with
	// a map[string]interface{} or a struct whose exported members supply the name/value
	// pairs.
	Expand(interface{}) (Loader, error)
}

// Template is an Expander implementation which uses a uritemplates.UriTemplate
// to generate URIs.  The URIs are then supplied to a Factory which is used to
// produce the Loaders.
//
// Typically, a Factory will be used to create instances of this type, which are
// used through the Expander interface.  However, this type is exported for simple
// use cases which do not require the full configuration logic of a Factory.
type Template struct {
	URITemplate *uritemplates.UriTemplate
	Header      http.Header
	Method      string
	HTTPClient  httpClient
}

func (t *Template) String() string {
	return t.URITemplate.String()
}

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
