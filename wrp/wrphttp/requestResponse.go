package wrphttp

import (
	"context"
	"net/http"

	"github.com/xmidt-org/webpa-common/wrp"
)

// DetermineFormat examines zero or more headers to determine which WRP format is to be used, either
// for decoding or encoding.  The headers are tried in order, and the first non-empty value that maps
// to a WRP format is returned.  Any non-empty header that is invalid results in an error.  If none of
// the headers are present, this function returns the defaultFormat.
//
// This function can be used with a single header, e.g. DetermineFormat(wrp.Msgpack, header, "Content-Type").
// It can also be used for simple content negotiation, e.g. DetermineFormat(wrp.Msgpack, header, "Accept", "Content-Type").
func DetermineFormat(defaultFormat wrp.Format, h http.Header, names ...string) (wrp.Format, error) {
	for _, n := range names {
		v := h.Get(n)
		if len(v) > 0 {
			return wrp.FormatFromContentType(v)
		}
	}

	return defaultFormat, nil
}

// Request wraps an original http.Request and contains WRP message information.  Context handling
// mimics http.Request.
type Request struct {
	// Original is the HTTP request which corresponds to this WRP request.  The request body will have
	// already been read to produce the entity.
	Original *http.Request

	// Entity is the decoded WRP message
	Entity *Entity

	// ctx is the constructed context, not necessarily the same as that in the original request
	ctx context.Context
}

// Context returns the context associated with this WRP Request, which is not necessarily the
// same as the context returned by r.Original.Context().  Use this method instead of the original
// request.
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}

	return context.Background()
}

// WithContext returns a shallow copy of this WRP Request using the supplied context.
// The semantics of this method are the same as http.Request.WithContext.  Note that the
// original request's context is not updated via this method.
func (r *Request) WithContext(ctx context.Context) *Request {
	if ctx == nil {
		panic("the context cannot be nil")
	}

	copy := new(Request)
	*copy = *r
	copy.ctx = ctx
	return copy
}

// ResponseWriter extends http.ResponseWriter with some WRP behavior.
type ResponseWriter interface {
	http.ResponseWriter

	// WriteWRP writes a WRP message to the underlying response.  The format used is determined
	// by the configuration of the underlying implementation.  This method is idempotent, and returns
	// an error if called multiple times for the same instance.
	WriteWRP(interface{}) (int, error)
}

type ResponseWriterFunc func(http.ResponseWriter, *Request) (ResponseWriter, error)

var defaultResponseWriterFunc ResponseWriterFunc = NewEntityResponseWriter(wrp.Msgpack)

func DefaultResponseWriterFunc() ResponseWriterFunc {
	return defaultResponseWriterFunc
}

// NewEntityResponseWriter creates a ResponseWriterFunc that returns an entity-based ResponseWriter.  The returned
// ResponseWriter writes WRP messages to the response body, using content negotation with a fallback to the supplied
// default format.
func NewEntityResponseWriter(defaultFormat wrp.Format) ResponseWriterFunc {
	return func(httpResponse http.ResponseWriter, wrpRequest *Request) (ResponseWriter, error) {
		format, err := DetermineFormat(defaultFormat, wrpRequest.Original.Header, "Accept")
		if err != nil {
			return nil, err
		}

		return &entityResponseWriter{
			ResponseWriter: httpResponse,
			f:              format,
		}, nil
	}
}

// entityResponseWriter provides ResponseWriter behavior that marshals WRP messages into the HTTP entity (body)
type entityResponseWriter struct {
	http.ResponseWriter
	f wrp.Format
}

func (erw *entityResponseWriter) WriteWRP(v interface{}) (int, error) {
	var (
		output  []byte
		encoder = wrp.NewEncoderBytes(&output, erw.f)
	)

	if err := encoder.Encode(v); err != nil {
		return 0, err
	}

	erw.ResponseWriter.Header().Set("Content-Type", erw.f.ContentType())
	return erw.ResponseWriter.Write(output)
}
