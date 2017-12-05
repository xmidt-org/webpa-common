package fanouthttp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/tracing"
	gokithttp "github.com/go-kit/kit/transport/http"
)

// PassThrough holds the raw contents of an original fanout request.  This is useful
// when the fanout doesn't need to do any thing to the original request except pass it on.
type PassThrough struct {
	// StatusCode is the original status code from an http.Response.  This field doesn't apply to requests,
	// and is generally set to a negative value for requests.
	StatusCode int

	// ContentType is the original content type of the request or response entity
	ContentType string

	// CopyHeader is an optional set of HTTP headers which was copied from the source request or response.  Encoding
	// should generally transfer the contents of this header onto the resulting request or response.
	CopyHeader http.Header

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

func (pt *PassThrough) Headers() http.Header {
	return pt.CopyHeader
}

// ReadCloser returns a distinct io.ReadCloser which can read the Entity bytes
func (pt *PassThrough) ReadCloser() io.ReadCloser {
	return ioutil.NopCloser(bytes.NewReader(pt.Entity))
}

// GetBody is a convenient request.GetBody implementation
func (pt *PassThrough) GetBody() (io.ReadCloser, error) {
	return pt.ReadCloser(), nil
}

// EncodeRequest handles transferring the information from this PassThrough onto the given request.
func (pt *PassThrough) EncodeRequest(r *http.Request) {
	r.Body = pt.ReadCloser()
	r.GetBody = pt.GetBody

	for n, v := range pt.CopyHeader {
		r.Header[n] = append(r.Header[n], v...)
	}

	if len(pt.ContentType) > 0 {
		r.Header.Set("Content-Type", pt.ContentType)
	}
}

// EncodeResponse handles transferring the information from this PassThrough onto the given response writer.
func (pt *PassThrough) EncodeResponse(r http.ResponseWriter) error {
	header := r.Header()
	for n, v := range pt.CopyHeader {
		header[n] = append(header[n], v...)
	}

	if len(pt.ContentType) > 0 {
		header.Set("Content-Type", pt.ContentType)
	}

	if pt.StatusCode > 0 {
		r.WriteHeader(pt.StatusCode)
	}

	_, err := r.Write(pt.Entity)
	return err
}

// DecodePassThroughRequest returns a fanout entity decoder which returns a *PassThrough with the original request's contents.
// If supplied, the headers contains the set of original headers that are copied as is from the original HTTP request.
func DecodePassThroughRequest(hs HeaderSet) gokithttp.DecodeRequestFunc {
	return func(_ context.Context, original *http.Request) (interface{}, error) {
		entity, err := ioutil.ReadAll(original.Body)
		if err != nil {
			return nil, err
		}

		return &PassThrough{
			StatusCode:  -1,
			ContentType: original.Header.Get("Content-Type"),
			CopyHeader:  hs.Filter(nil, original.Header),
			Entity:      entity,
		}, nil
	}
}

// DecodePassThroughResponse returns a component response entity decoder that returns a *PassThrough containing the response
// information.
func DecodePassThroughResponse(hs HeaderSet) gokithttp.DecodeResponseFunc {
	return func(_ context.Context, component *http.Response) (interface{}, error) {
		entity, err := ioutil.ReadAll(component.Body)
		if err != nil {
			return nil, err
		}

		if component.StatusCode > 399 {
			return nil, &httperror.E{
				Code:   component.StatusCode,
				Header: hs.Filter(nil, component.Header),
				Text:   fmt.Sprintf("HTTP transaction failed with code: %d", component.StatusCode),
				Entity: entity,
			}
		}

		return &PassThrough{
			StatusCode:  component.StatusCode,
			ContentType: component.Header.Get("Content-Type"),
			CopyHeader:  hs.Filter(nil, component.Header),
			Entity:      entity,
		}, nil
	}
}

// EncodePassThroughRequest is a component entity encoder that assumes a *PassThrough is passed as the value
// and writes out the entity and content type to the component request.  This functional also sets GetBody
// so that redirects are handled appropriately.
func EncodePassThroughRequest(_ context.Context, component *http.Request, v interface{}) error {
	pt := v.(*PassThrough)
	pt.EncodeRequest(component)
	return nil
}

// EncodePassThroughResponse is a fanout entity encoder that handles taking a *PassThrough from a component response
// and writing it to the fanout's original response.
func EncodePassThroughResponse(_ context.Context, original http.ResponseWriter, v interface{}) error {
	pt := v.(*PassThrough)
	return pt.EncodeResponse(original)
}
