package wrphttp

import (
	"context"
	"net/http"

	"github.com/Comcast/webpa-common/wrp"
)

const (
	MessageTypeHeader             = "X-XMiDT-Message-Type"
	TransactionUUIDHeader         = "X-XMiDT-Transaction-UUID"
	Status                        = "X-XMiDT-Status"
	RequestDeliveryResponseHeader = "X-XMiDT-Request-Delivery-Response"
	IncludeSpansHeader            = "X-XMiDT-Include-Spans"
	SpansHeader                   = "X-XMiDT-Spans"
	PathHeader                    = "X-XMiDT-Path"
	SourceHeader                  = "X-XMiDT-Source"
)

// Body is a go-kit transport/http.DecodeRequestFunc function that parses the body of an HTTP
// request as a WRP message in the format used by the given pool.  The supplied pool should match the
// Content-Type of the request, or an error is returned.
func Body(pool *wrp.DecoderPool) func(context.Context, *http.Request) (interface{}, error) {
	return func(ctx context.Context, request *http.Request) (message interface{}, err error) {
		message = new(wrp.Message)
		err = pool.Decode(message, request.Body)
		return
	}
}

// NewMessage constructs a WRP Message from a set of headers
func NewMessage(h http.Header) (*wrp.Message, error) {
	message := new(wrp.Message)

	messageType := h.Get(MessageTypeHeader)
	if len(messageType) == 0 {
		return nil, 
	return message, nil
}

// Headers uses headers to supply the WRP message fields.
func Headers(ctx context.Context, request *http.Request) (interface{}, error) {
	return NewMessage(request.Header)
}
