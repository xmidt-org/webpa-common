package resource

import (
	"errors"
	"fmt"
	"github.com/jtacoma/uritemplates"
	"net/http"
	"net/url"
	"sort"
)

const (
	// NoScheme indicates the value of a URI without a scheme prefix, e.g. "/etc/appname/config.json"
	NoScheme = ""

	// FileScheme indicates a file URI according to https://en.wikipedia.org/wiki/File_URI_scheme.
	// When a URL is parsed that has no scheme, url.URL.Scheme is set to this value.
	FileScheme = "file"

	// HttpScheme is plain old HTTP
	HttpScheme = "http"

	// HttpsScheme is secure HTTP
	HttpsScheme = "https"
)

var (
	// supportedSchemes provides a quick, map-based way to test for a valid scheme
	supportedSchemes = map[string]bool{
		FileScheme:  true,
		HttpScheme:  true,
		HttpsScheme: true,
	}

	ErrorAmbiguousResource = errors.New("Either URI or Data must be supplied, but not both")
	ErrorNoResource        = errors.New("URI or Data are required")
	ErrorURIRequired       = errors.New("A URI is required")
)

// Factory provides a common way to configure all types of resources
// supported by this package.  This type allows client code to use JSON configuration
// to specify resources in an abstract way.
//
// The primary purpose for this type is to allow external configuration of application
// resources in a file or other source of JSON.  For code which does not require this
// level of abstraction, the other resources types in this package (e.g. HTTP, Data, Template, etc)
// can be used directly.
type Factory struct {
	// URI specifies the external resource's location.  This can be a filesystem
	// path, which is a valid URI.  file:// resources are also supported.
	URI string `json:"uri"`

	// Data specfies the actual data of the resource.  Either this or URI
	// must be set, but not both.
	Data string `json:"data"`

	// Header supplies any HTTP headers to use when obtaining the resource.
	// Ignored if URI is not an HTTP or HTTPS URI.
	Header http.Header `json:"header"`

	// Method is the HTTP method to use when obtaining the resource.
	// Ignored if URI is not an HTTP or HTTPS URI.
	Method string `json:"method"`

	// HTTPClient is any object that supplies a method with the signature func(*http.Request) (*http.Response, error).
	// It is omitted from all JSON operations, so it must be supplied after a Factory is unmarshalled.
	// If not supplied, http.DefaultClient is used.  Any *http.Client value can be used here so that
	// all resources share a common Client configuration.
	//
	// Ignored if URI is not an HTTP or HTTPS URI.
	HTTPClient httpClient `json:"-"`
}

// URL returns the url.URL that should be used to obtain the resource.  If this factory
// represents an in-memory resource, a nil url.URL pointer is returned.
//
// This method also does basic validation on the state of the factory.  If the returned
// error is non-nil, the url will always be nil.
func (f *Factory) URL() (*url.URL, error) {
	if len(f.URI) > 0 {
		if len(f.Data) > 0 {
			return nil, ErrorAmbiguousResource
		}

		resourceURL, err := url.Parse(f.URI)
		if err != nil {
			return nil, err
		} else if len(resourceURL.Scheme) == 0 {
			// supports URIs like "/etc/foobar.txt" as files
			resourceURL.Scheme = FileScheme
		} else if !supportedSchemes[resourceURL.Scheme] {
			return nil, fmt.Errorf("Unsupported scheme: %s", resourceURL.Scheme)
		}

		return resourceURL, nil
	} else if len(f.Data) == 0 {
		return nil, ErrorNoResource
	}

	return nil, nil
}

// NewLoader creates a Loader which loads from the literal URI.
// The URI must be a valid URL with the file, http, or https schemes.
func (f *Factory) NewLoader() (Loader, error) {
	resourceURL, err := f.URL()
	if err != nil {
		return nil, err
	}

	switch {
	case resourceURL == nil:
		return &Data{Source: []byte(f.Data)}, nil

	case resourceURL.Scheme == FileScheme:
		return &File{Path: resourceURL.Path}, nil

	default:
		return &HTTP{
			URL:        resourceURL.String(),
			Header:     f.Header,
			Method:     f.Method,
			HTTPClient: f.HTTPClient,
		}, nil
	}
}

// NewExpander treats URI as a URI template and produces an Expander object
// which can be used to expand the URI template into Loader instances.
//
// If any requiredNames are supplied, an error will be returned if the URI template
// does not contain only those names.
func (f *Factory) NewExpander(requiredNames ...string) (Expander, error) {
	if len(f.URI) == 0 {
		return nil, ErrorURIRequired
	} else if len(f.Data) > 0 {
		return nil, ErrorAmbiguousResource
	}

	uriTemplate, err := uritemplates.Parse(f.URI)
	if err != nil {
		return nil, err
	}

	if len(requiredNames) > 0 {
		missingNames := make([]string, 0, len(requiredNames))
		actualNames := sort.StringSlice(uriTemplate.Names())
		actualNames.Sort()

		for _, requiredName := range requiredNames {
			if position := actualNames.Search(requiredName); position >= actualNames.Len() || actualNames[position] != requiredName {
				missingNames = append(missingNames, requiredName)
			}
		}

		if len(missingNames) > 0 {
			return nil, fmt.Errorf("URI template %s does not contain names %s", uriTemplate, missingNames)
		}
	}

	return &Template{
		URITemplate: uriTemplate,
		Header:      f.Header,
		Method:      f.Method,
		HTTPClient:  f.HTTPClient,
	}, nil
}
