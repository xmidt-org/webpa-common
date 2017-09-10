package wrphttp

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/Comcast/webpa-common/wrp"
)

const (
	MessageTypeHeader             = "X-Xmidt-Message-Type"
	TransactionUuidHeader         = "X-Xmidt-Transaction-Uuid"
	StatusHeader                  = "X-Xmidt-Status"
	RequestDeliveryResponseHeader = "X-Xmidt-Request-Delivery-Response"
	IncludeSpansHeader            = "X-Xmidt-Include-Spans"
	SpanHeader                    = "X-Xmidt-Span"
	PathHeader                    = "X-Xmidt-Path"
	SourceHeader                  = "X-Xmidt-Source"
	AcceptHeader                  = "X-Xmidt-Accept"
)

var (
	ErrMissingMessageTypeHeader = fmt.Errorf("Missing %s header", MessageTypeHeader)
)

// WrpRequest represents an HTTP request containing a WRP message.
type WrpRequest struct {
	// Message is the decoded WRP message
	Message wrp.Message

	// Accept is the value of the Accept header in the originating HTTP request
	Accept string

	// ctx is the originating Context
	ctx context.Context
}

func (w *WrpRequest) Context() context.Context {
	if w.ctx == nil {
		return context.Background()
	}

	return w.ctx
}

// newWrpRequest creates a WrpRequest from an HTTP request with everything
// but the Message populated.
func newWrpRequest(ctx context.Context, r *http.Request) *WrpRequest {
	return &WrpRequest{
		Accept: r.Header.Get("Accept"),
		ctx:    ctx,
	}
}

// DecodeBody is a go-kit transport/http.DecodeRequestFunc function that parses the body of an HTTP
// request as a WRP message in the format used by the given pool.  The supplied pool should match the
// Content-Type of the request, or an error is returned.
//
// This decoder function is appropriate when the HTTP request body contains a full WRP message.  For situations
// where the HTTP body is only the payload, use the Headers decoder.
func DecodeBody(pool *wrp.DecoderPool) func(context.Context, *http.Request) (interface{}, error) {
	return func(ctx context.Context, httpRequest *http.Request) (interface{}, error) {
		var (
			wrpRequest = newWrpRequest(ctx, httpRequest)
			err        = pool.Decode(&wrpRequest.Message, httpRequest.Body)
		)

		return wrpRequest, err
	}
}

// getMessageType extracts the wrp.MessageType from header.  This is a required field.
//
// This function panics if the message type header is missing or invalid.
func getMessageType(h http.Header) wrp.MessageType {
	value := h.Get(MessageTypeHeader)
	if len(value) == 0 {
		panic(ErrMissingMessageTypeHeader)
	}

	messageType, err := wrp.StringToMessageType(value)
	if err != nil {
		panic(err)
	}

	return messageType
}

// getIntHeader returns the header as a int64, or returns nil if the header is absent.
// This function panics if the header is present but not a valid integer.
func getIntHeader(h http.Header, n string) *int64 {
	value := h.Get(n)
	if len(value) == 0 {
		return nil
	}

	i, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		panic(err)
	}

	return &i
}

func getBoolHeader(h http.Header, n string) *bool {
	value := h.Get(n)
	if len(value) == 0 {
		return nil
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		panic(err)
	}

	return &b
}

func getSpans(h http.Header) [][]string {
	var spans [][]string

	for _, value := range h[SpanHeader] {
		fields := strings.Split(value, ",")
		if len(fields) != 3 {
			panic(fmt.Errorf("Invalid %s header: %s", SpanHeader, value))
		}

		spans = append(spans, fields)
	}

	return spans
}

// populateMessage populates a WRP message from a set of headers
func populateMessage(h http.Header, m *wrp.Message) {
	m.Type = getMessageType(h)
	m.Source = h.Get(SourceHeader)
	m.TransactionUUID = h.Get(TransactionUuidHeader)
	m.Status = getIntHeader(h, StatusHeader)
	m.RequestDeliveryResponse = getIntHeader(h, RequestDeliveryResponseHeader)
	m.IncludeSpans = getBoolHeader(h, IncludeSpansHeader)
	m.Spans = getSpans(h)
	m.ContentType = h.Get("Content-Type")
	m.Accept = h.Get(AcceptHeader)
	m.Path = h.Get(PathHeader)
}

// setPayload transfers the payload in the HTTP request to the given WRP message.  If the
// ContentType of the message hasn't been set yet, it is set to application/octet-stream.
func setPayload(r *http.Request, m *wrp.Message) {
	payload, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}

	if len(payload) > 0 {
		if len(m.ContentType) == 0 {
			m.ContentType = "application/octet-stream"
		}

		m.Payload = payload
	}
}

// DecodeHeaders uses headers to supply the WRP message fields.  The HTTP request body, if supplied, is assumed
// to be the payload of the WRP message and is read in unmodified.
func DecodeHeaders(ctx context.Context, httpRequest *http.Request) (value interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			value = nil

			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("Unable to create WRP message: %s", v)
			}
		}
	}()

	wrpRequest := newWrpRequest(ctx, httpRequest)
	populateMessage(httpRequest.Header, &wrpRequest.Message)
	setPayload(httpRequest, &wrpRequest.Message)

	value = wrpRequest
	return
}
