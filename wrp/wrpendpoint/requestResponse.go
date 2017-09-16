package wrpendpoint

import (
	"io"
	"io/ioutil"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/tracing"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/go-kit/kit/log"
)

// Note is the core type implemented by any entity which carries a WRP message.
type Note interface {
	// Destination returns the WRP destination string from the decoded message
	Destination() string

	// TransactionID returns the transaction identifier, if any
	TransactionID() string

	// Message returns the underlying decoded WRP message.  This can be nil in the case of
	// raw messages.  Callers should never modify the returned Message instance.
	Message() *wrp.Message

	// Encode writes out the WRP message fronted by this Note in the format supported by the pool.
	Encode(output io.Writer, pool *wrp.EncoderPool) error

	// EncodeBytes works like Encode, except that it returns a []byte.
	EncodeBytes(pool *wrp.EncoderPool) ([]byte, error)
}

type note struct {
	destination   string
	transactionID string
	message       *wrp.Message
	contents      []byte
	format        wrp.Format
}

func (n *note) Destination() string {
	return n.destination
}

func (n *note) TransactionID() string {
	return n.transactionID
}

func (n *note) Message() *wrp.Message {
	return n.message
}

func (n *note) Encode(output io.Writer, pool *wrp.EncoderPool) error {
	if n.format == pool.Format() && len(n.contents) > 0 {
		_, err := output.Write(n.contents)
		return err
	}

	return pool.Encode(output, n.message)
}

func (n *note) EncodeBytes(pool *wrp.EncoderPool) ([]byte, error) {
	if n.format == pool.Format() && len(n.contents) > 0 {
		copyOf := make([]byte, len(n.contents))
		copy(copyOf, n.contents)
		return copyOf, nil
	}

	var output []byte
	err := pool.EncodeBytes(&output, n.message)
	return output, err
}

// Request is a WRP request.  In addition to implementing Note, this type also provides contextual logging.
// A Request is considered immutable once instantiated.  Methods that update a Request return a shallow copy
// with the modification.
type Request interface {
	Note
	Logger() log.Logger
	WithLogger(log.Logger) Request
}

// request is the internal Request implementation
type request struct {
	note
	logger log.Logger
}

func (r *request) Logger() log.Logger {
	if r.logger != nil {
		return r.logger
	}

	return logging.DefaultLogger()
}

func (r *request) WithLogger(logger log.Logger) Request {
	copyOf := new(request)
	*copyOf = *r
	copyOf.logger = logger
	return copyOf
}

// DecodeRequest extracts a WRP request from the given source.
func DecodeRequest(logger log.Logger, source io.Reader, pool *wrp.DecoderPool) (Request, error) {
	contents, err := ioutil.ReadAll(source)
	if err != nil {
		return nil, err
	}

	return DecodeRequestBytes(logger, contents, pool)
}

// DecodeRequestBytes returns a Request taken from the contents.  The given pool is used to decode the WRP message.
func DecodeRequestBytes(logger log.Logger, contents []byte, pool *wrp.DecoderPool) (Request, error) {
	d := pool.Get()
	defer pool.Put(d)

	d.ResetBytes(contents)
	m := new(wrp.Message)
	if err := d.Decode(m); err != nil {
		return nil, err
	}

	return &request{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
			contents:      contents,
			format:        pool.Format(),
		},
		logger: logger,
	}, nil
}

// WrapAsRequest takes an existing WRP message and produces a Request for that message.
func WrapAsRequest(logger log.Logger, m *wrp.Message) Request {
	return &request{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
		},
		logger: logger,
	}
}

// Response represents a WRP response to a Request.  Note that not all WRP requests will have responses, e.g. SimpleEvents.
// A Response instance is considered immutable once created.  Methods that modify a response will return a shallow copy with
// the modification.
type Response interface {
	Note

	// Spans returns the spans associated with this response.  This implements tracing.Spanned.
	Spans() []tracing.Span

	// AddSpans returns a shallow copy of this response with the given spans appended.  If no spans are passed,
	// this method returns the original Response unmodified.
	AddSpans(...tracing.Span) Response
}

// response is the internal Response implementation
type response struct {
	note
	spans []tracing.Span
}

func (r *response) Spans() []tracing.Span {
	return r.spans
}

func (r *response) AddSpans(spans ...tracing.Span) Response {
	if len(spans) == 0 {
		return r
	}

	copyOf := new(response)
	*copyOf = *r
	copyOf.spans = make([]tracing.Span, len(r.spans)+len(spans))
	copy(copyOf.spans, r.spans)
	copy(copyOf.spans[len(r.spans):], spans)

	return copyOf
}

// DecodeResponse extracts a WRP response from the given source.
func DecodeResponse(source io.Reader, pool *wrp.DecoderPool) (Response, error) {
	contents, err := ioutil.ReadAll(source)
	if err != nil {
		return nil, err
	}

	return DecodeResponseBytes(contents, pool)
}

// DecodeResponseBytes returns a Response taken from the contents.  The given pool is used to decode the WRP message.
func DecodeResponseBytes(contents []byte, pool *wrp.DecoderPool) (Response, error) {
	d := pool.Get()
	defer pool.Put(d)

	d.ResetBytes(contents)
	m := new(wrp.Message)
	if err := d.Decode(m); err != nil {
		return nil, err
	}

	return &response{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
			contents:      contents,
			format:        pool.Format(),
		},
	}, nil
}

// WrapAsResponse takes an existing WRP message and produces a Response for that message.
func WrapAsResponse(m *wrp.Message) Response {
	return &response{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
		},
	}
}
