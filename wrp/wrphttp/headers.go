package wrphttp

import (
	"fmt"
	"io"
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

func readPayload(h http.Header, p io.Reader) ([]byte, string) {
	if p == nil {
		return nil, ""
	}

	payload, err := ioutil.ReadAll(p)
	if err != nil {
		panic(err)
	}

	if len(payload) == 0 {
		return nil, ""
	}

	contentType := h.Get("Content-Type")
	if len(contentType) == 0 && len(payload) > 0 {
		contentType = "application/octet-stream"
	}

	return payload, contentType
}

func NewMessageFromHeaders(h http.Header, p io.Reader) (message *wrp.Message, err error) {
	defer func() {
		if r := recover(); r != nil {
			message = nil
			switch v := r.(type) {
			case error:
				err = v
			default:
				err = fmt.Errorf("Unable to create WRP message: %s", v)
			}
		}
	}()

	payload, contentType := readPayload(h, p)

	message = &wrp.Message{
		Type:                    getMessageType(h),
		Source:                  h.Get(SourceHeader),
		TransactionUUID:         h.Get(TransactionUuidHeader),
		Status:                  getIntHeader(h, StatusHeader),
		RequestDeliveryResponse: getIntHeader(h, RequestDeliveryResponseHeader),
		IncludeSpans:            getBoolHeader(h, IncludeSpansHeader),
		Spans:                   getSpans(h),
		Payload:                 payload,
		ContentType:             contentType,
		Accept:                  h.Get(AcceptHeader),
		Path:                    h.Get(PathHeader),
	}

	return
}

// AddMessageHeaders adds the HTTP header representation of a given WRP message.
// This function does not handle the payload, to allow further headers to be written by
// calling code.
func AddMessageHeaders(h http.Header, m *wrp.Message) {
	h.Set(MessageTypeHeader, m.Type.FriendlyName())

	if len(m.Source) > 0 {
		h.Set(SourceHeader, m.Source)
	}

	if len(m.TransactionUUID) > 0 {
		h.Set(TransactionUuidHeader, m.TransactionUUID)
	}

	if m.Status != nil {
		h.Set(StatusHeader, strconv.FormatInt(*m.Status, 10))
	}

	if m.RequestDeliveryResponse != nil {
		h.Set(RequestDeliveryResponseHeader, strconv.FormatInt(*m.RequestDeliveryResponse, 10))
	}

	if m.IncludeSpans != nil {
		h.Set(IncludeSpansHeader, strconv.FormatBool(*m.IncludeSpans))
	}

	for _, s := range m.Spans {
		h.Add(SpanHeader, strings.Join(s, ","))
	}

	if len(m.Accept) > 0 {
		h.Set(AcceptHeader, m.Accept)
	}

	if len(m.Path) > 0 {
		h.Set(PathHeader, m.Path)
	}
}

// WriteMessagePayload writes the WRP payload to the given io.Writer.  If the message has no
// payload, this function does nothing.
//
// The http.Header is optional.  If supplied, the header's Content-Type and Content-Length
// will be set appropriately.
func WriteMessagePayload(h http.Header, p io.Writer, m *wrp.Message) error {
	if len(m.Payload) == 0 {
		return nil
	}

	if h != nil {
		if len(m.ContentType) > 0 {
			h.Set("Content-Type", m.ContentType)
		} else {
			h.Set("Content-Type", "application/octet-stream")
		}

		h.Set("Content-Length", strconv.Itoa(len(m.Payload)))
	}

	_, err := p.Write(m.Payload)
	return err
}
