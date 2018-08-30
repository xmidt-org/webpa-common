package wrphttp

import (
	"net/http"

	gokithttp "github.com/go-kit/kit/transport/http"
)

// Handler is a WRP handler for messages over HTTP.  This is the analog of http.Handler.
type Handler interface {
	ServeWRP(ResponseWriter, *Request)
}

type HandlerFunc func(ResponseWriter, *Request)

func (hf HandlerFunc) ServeWRP(w ResponseWriter, r *Request) {
	hf(w, r)
}

// Option is a configurable option for an HTTP handler that works with WRP
type Option func(*wrpHandler)

// WithErrorEncoder establishes a go-kit ErrorEncoder for the given handler.
// By default, DefaultErrorEncoder is used.  If the supplied ErrorEncoder is
// nil, it reverts to the default.
func WithErrorEncoder(ee gokithttp.ErrorEncoder) Option {
	return func(wh *wrpHandler) {
		if ee != nil {
			wh.errorEncoder = ee
		} else {
			wh.errorEncoder = gokithttp.DefaultErrorEncoder
		}
	}
}

// WithResponseWriter establishes a factory function for ResponseWriter objects.
// By default, DefaultResponseWriterFunc() is used.  If the supplied strategy function
// is nil, it reverts to the default.
func WithResponseWriter(rwf ResponseWriterFunc) Option {
	return func(wh *wrpHandler) {
		if rwf != nil {
			wh.newResponseWriter = rwf
		} else {
			wh.newResponseWriter = DefaultResponseWriterFunc()
		}
	}
}

// WithDecoder sets a go-kit DecodeRequestFunc strategy that turns an http.Request into a WRP request.
// By default, DefaultDecoder() is used.  If the supplied strategy is nil, it reverts to the default.
func WithDecoder(d Decoder) Option {
	return func(wh *wrpHandler) {
		if d != nil {
			wh.decoder = d
		} else {
			wh.decoder = DefaultDecoder()
		}
	}
}

func WithBefore(funcs ...MessageFunc) Option {
	return func(wh *wrpHandler) {
		wh.before = append(wh.before, funcs...)
	}
}

// NewHTTPHandler creates an http.Handler that forwards WRP requests to the supplied WRP handler.
func NewHTTPHandler(h Handler, options ...Option) http.Handler {
	if h == nil {
		panic("A WRP Handler is required")
	}

	wh := &wrpHandler{
		handler:           h,
		errorEncoder:      gokithttp.DefaultErrorEncoder,
		decoder:           DefaultDecoder(),
		newResponseWriter: DefaultResponseWriterFunc(),
	}

	for _, o := range options {
		o(wh)
	}

	return wh
}

type wrpHandler struct {
	handler           Handler
	errorEncoder      gokithttp.ErrorEncoder
	before            []MessageFunc
	decoder           Decoder
	newResponseWriter ResponseWriterFunc
}

func (wh *wrpHandler) ServeHTTP(httpResponse http.ResponseWriter, httpRequest *http.Request) {
	ctx := httpRequest.Context()
	entity, err := wh.decoder(ctx, httpRequest)
	if err != nil {
		wh.errorEncoder(ctx, err, httpResponse)
		return
	}

	for _, mf := range wh.before {
		ctx = mf(ctx, &entity.Message)
	}

	wrpRequest := &Request{
		Original: httpRequest,
		Entity:   entity,
		ctx:      ctx,
	}

	wrpResponse, err := wh.newResponseWriter(httpResponse, wrpRequest)
	if err != nil {
		wh.errorEncoder(wrpRequest.Context(), err, httpResponse)
		return
	}

	wh.handler.ServeWRP(wrpResponse, wrpRequest)
}
