package conveyhttp

import (
	"errors"
	"net/http"

	"github.com/Comcast/webpa-common/convey"
)

// DefaultHeaderName is the HTTP header assumed to contain Convey data when no header is supplied
const DefaultHeaderName = "X-Webpa-Convey"

// ErrMissingHeader indicates that no HTTP header exists which contains convey information
var ErrMissingHeader = errors.New("No convey header present")

// HeaderTranslator is an analog to convey.Translator, except that this type works with http.Header.
type HeaderTranslator interface {
	FromHeader(http.Header) (convey.C, error)
	ToHeader(http.Header, convey.C) error
}

type headerTranslator struct {
	headerName string
	translator convey.Translator
}

// NewHeaderTranslator creates a HeaderTranslator that uses a convey.Translator to produce
// convey maps.
func NewHeaderTranslator(headerName string, translator convey.Translator) HeaderTranslator {
	if len(headerName) == 0 {
		headerName = DefaultHeaderName
	}

	if translator == nil {
		translator = convey.NewTranslator(nil)
	}

	return &headerTranslator{
		headerName: headerName,
		translator: translator,
	}
}

func (ht *headerTranslator) FromHeader(h http.Header) (convey.C, error) {
	v := h.Get(ht.headerName)
	if len(v) == 0 {
		return nil, ErrMissingHeader
	}

	return convey.ReadString(ht.translator, v)
}

func (ht *headerTranslator) ToHeader(h http.Header, c convey.C) error {
	v, err := convey.WriteString(ht.translator, c)
	if err == nil {
		h.Set(ht.headerName, v)
	}

	return err
}
