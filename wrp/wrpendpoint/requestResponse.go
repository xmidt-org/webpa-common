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
	Encode(output io.Writer, format wrp.Format) error

	// EncodeBytes works like Encode, except that it returns a []byte.
	EncodeBytes(format wrp.Format) ([]byte, error)
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

func (n *note) Encode(output io.Writer, format wrp.Format) error {
	if n.format == format && len(n.contents) > 0 {
		_, err := output.Write(n.contents)
		return err
	}

	return wrp.NewEncoder(output, format).Encode(n.message)
}

func (n *note) EncodeBytes(format wrp.Format) ([]byte, error) {
	if n.format == format && len(n.contents) > 0 {
		copyOf := make([]byte, len(n.contents))
		copy(copyOf, n.contents)
		return copyOf, nil
	}

	var output []byte
	err := wrp.NewEncoderBytes(&output, format).Encode(n.message)
	return output, err
}

// Request is a WRP request.  In addition to implementing Note, this type also provides contextual logging.
// A Request is considered immutable once instantiated.  Methods that update a Request return a shallow copy
// with the modification.
type Request interface {
	Note

	// Logger returns the enriched logger associated with this Request.  Client code should take care
	// to preserve this logger when changing loggers via WithLogger.
	//
	// If the logger associated with this Request is nil, for example via WithLogger(nil), this method
	// returns logging.DefaultLogger().
	Logger() log.Logger

	// WithLogger produces a shallow copy of this Request with a new Logger.  Generally, when creating a new
	// request logger, start with Logger():
	//
	//    var request Request = ...
	//    request = request.WithLogger(log.With(request.Logger(), "more", "stuff"))
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

// withLogger enriches a given logger with information about the message
func withLogger(logger log.Logger, m *wrp.Message, keyvals ...interface{}) log.Logger {
	return log.WithPrefix(
		logger,
		append([]interface{}{
			"source", m.Source,
			"destination", m.Destination,
			"transactionUUID", m.TransactionUUID,
			"path", m.Path,
			"payloadLength", len(m.Payload),
		}, keyvals...,
		)...,
	)
}

// DecodeRequest extracts a WRP request from the given source.
func DecodeRequest(logger log.Logger, source io.Reader, format wrp.Format) (Request, error) {
	contents, err := ioutil.ReadAll(source)
	if err != nil {
		return nil, err
	}

	return DecodeRequestBytes(logger, contents, format)
}

// DecodeRequestBytes returns a Request taken from the contents.  The given pool is used to decode the WRP message.
//
// This function also enhances the given logger with contextual information about the returned WRP request.  The
// logger that is passed to this function should never be nil and should never have a Caller or DefaultCaller set.
func DecodeRequestBytes(logger log.Logger, contents []byte, format wrp.Format) (Request, error) {
	m := new(wrp.Message)
	if err := wrp.NewDecoderBytes(contents, format).Decode(m); err != nil {
		return nil, err
	}

	return &request{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
			contents:      contents,
			format:        format,
		},
		logger: withLogger(logger, m, "format", format),
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
		logger: withLogger(logger, m),
	}
}

// Response represents a WRP response to a Request.  Note that not all WRP requests will have responses, e.g. SimpleEvents.
// A Response instance is considered immutable once created.  Methods that modify a response will return a shallow copy with
// the modification.
type Response interface {
	Note
	tracing.Mergeable
}

// response is the internal Response implementation
type response struct {
	note
	spans []tracing.Span
}

func (r *response) Spans() []tracing.Span {
	return r.spans
}

func (r *response) WithSpans(spans ...tracing.Span) interface{} {
	if len(spans) > 0 {
		return &response{
			note:  r.note,
			spans: spans,
		}
	}

	return r
}

// DecodeResponse extracts a WRP response from the given source.
func DecodeResponse(source io.Reader, format wrp.Format) (Response, error) {
	contents, err := ioutil.ReadAll(source)
	if err != nil {
		return nil, err
	}

	return DecodeResponseBytes(contents, format)
}

// DecodeResponseBytes returns a Response taken from the contents.  The given pool is used to decode the WRP message.
func DecodeResponseBytes(contents []byte, format wrp.Format) (Response, error) {
	m := new(wrp.Message)
	if err := wrp.NewDecoderBytes(contents, format).Decode(m); err != nil {
		return nil, err
	}

	return &response{
		note: note{
			destination:   m.Destination,
			transactionID: m.TransactionUUID,
			message:       m,
			contents:      contents,
			format:        format,
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
