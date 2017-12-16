package fanouthttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/xhttp"
)

// PassThrough holds the raw contents of an original fanout request.  This is useful
// when the fanout doesn't need to do any thing to the original request except pass it on.
type PassThrough struct {
	// StatusCode is the original status code from an http.Response.  This field doesn't apply to requests,
	// and is generally set to a negative value for requests.
	StatusCode int

	// ContentType is the original content type of the request or response entity
	ContentType string

	// Entity is the optional original entity of the request or response.
	Entity []byte

	spans []tracing.Span
}

func (pt *PassThrough) Spans() []tracing.Span {
	return pt.spans
}

func (pt *PassThrough) WithSpans(s ...tracing.Span) interface{} {
	copyOf := *pt
	copyOf.spans = s
	return &copyOf
}

// ReadCloser returns a distinct io.ReadCloser which can read the Entity bytes
func (pt *PassThrough) ReadCloser() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader(pt.Entity))
}

// GetBody is a convenient request.GetBody implementation
func (pt *PassThrough) GetBody() (io.ReadCloser, error) {
	return pt.ReadCloser(), nil
}

// DecodePassThroughRequest is a fanout entity decoder which returns a *PassThrough with the original request's contents.
// If supplied, the headers contains the set of original headers that are copied as is from the original HTTP request.
func DecodePassThroughRequest(_ context.Context, original *http.Request) (interface{}, error) {
	entity, err := ioutil.ReadAll(original.Body)
	if err != nil {
		return nil, err
	}

	return &PassThrough{
		StatusCode:  -1,
		ContentType: original.Header.Get("Content-Type"),
		Entity:      entity,
	}, nil
}

// DecodePassThroughResponse is a component response entity decoder that returns a *PassThrough containing the response
// information.
func DecodePassThroughResponse(_ context.Context, component *http.Response) (interface{}, error) {
	entity, err := ioutil.ReadAll(component.Body)
	if err != nil {
		return nil, err
	}

	if component.StatusCode > 399 {
		return nil, &xhttp.Error{
			Code:   component.StatusCode,
			Text:   fmt.Sprintf("HTTP transaction failed with code: %d", component.StatusCode),
			Entity: entity,
		}
	}

	return &PassThrough{
		StatusCode:  component.StatusCode,
		ContentType: component.Header.Get("Content-Type"),
		Entity:      entity,
	}, nil
}

// EncodePassThroughRequest is a component entity encoder that assumes a *PassThrough is passed as the value
// and writes out the entity and content type to the component request.  This functional also sets GetBody
// so that redirects are handled appropriately.
func EncodePassThroughRequest(_ context.Context, component *http.Request, v interface{}) error {
	pt := v.(*PassThrough)
	component.Body = pt.ReadCloser()
	component.GetBody = pt.GetBody

	if len(pt.ContentType) > 0 {
		component.Header.Set("Content-Type", pt.ContentType)
	}

	return nil
}

// EncodePassThroughResponse is a fanout entity encoder that handles taking a *PassThrough from a component response
// and writing it to the fanout's original response.
func EncodePassThroughResponse(_ context.Context, original http.ResponseWriter, v interface{}) error {
	pt := v.(*PassThrough)
	if len(pt.ContentType) > 0 {
		original.Header().Set("Content-Type", pt.ContentType)
	}

	if pt.StatusCode > 0 {
		original.WriteHeader(pt.StatusCode)
	}

	_, err := original.Write(pt.Entity)
	return err
}
