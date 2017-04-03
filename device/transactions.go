package device

import (
	"context"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

// Request represents a single device Request, carrying routing information and message contents.
type Request struct {
	// Message is the original, decoded WRP message containing the routing information.  This is the
	// only absolutely required field of a device Request.
	Message wrp.Routable

	// Format is the WRP format of the Contents member.  If Format is not JSON, then Routing
	// will be encoded prior to sending to devices.
	Format wrp.Format

	// Contents is the encoded form of Routing in Format format.  If this member is of 0 length,
	// then Routing will be encoded prior to sending to devices.
	Contents []byte

	// ctx is the API context for this request, which can be nil.  Normally, it's best to
	// set this to context.Background() if no cancellation semantics are desired.
	ctx context.Context
}

// Context returns the context.Context object associated with this Request.
// This method never returns nil.  If no context is associated with this Request,
// this method returns context.Background().
func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}

	return context.Background()
}

// WithContext is similar to net/http.Request.WithContext.  This method does not, however,
// create a copy of the original device Request.  Rather, it returns the request modified
// with the next context.
func (r *Request) WithContext(ctx context.Context) *Request {
	// mimic the behavior of net/http.Request
	if ctx == nil {
		panic("nil context")
	}

	r.ctx = ctx
	return r
}

// ID parses the Routing.To() value into a device identifier.
func (r *Request) ID() (ID, error) {
	return ParseID(r.Message.To())
}

// DecodeRequest decodes a WRP source into a device Request.  Typically, this is used
// to produce a device Request from an http.Request.
//
// The returned request will not be associated with any context.
func DecodeRequest(source io.Reader, pool *wrp.DecoderPool) (*Request, error) {
	contents, err := ioutil.ReadAll(source)
	if err != nil {
		return nil, err
	}

	message := new(wrp.Message)
	if err := pool.DecodeBytes(message, contents); err != nil {
		return nil, err
	}

	return &Request{
		Message:  message,
		Format:   pool.Format(),
		Contents: contents,
	}, nil
}

// Response represents the response to a device request.  Some requests have no response, in which case
// a Response without a Routing or Contents will be returned.
type Response struct {
	// Device is the sink to which the corresponding Request was sent
	Device Interface

	// Message is the decoded WRP message received from the device
	Message *wrp.Message

	// Format is the encoding Format of the Contents field.  Almost always, this will be Msgpack.
	Format wrp.Format

	// Contents is the encoded form of Routing, formatted in Format
	Contents []byte

	// Error is set if there was any error while processing the transaction.  If this field
	// is set, Format and Contents should be ignored.
	Error error
}

// EncodeResponse writes out a device transaction Response to an http Response.
//
// If response.Error is set, a JSON-formatted error with status http.StatusInternalServerError is
// written to the HTTP response.
//
// If the encoder pool is nil, or if the pool is supplied but it's format is the same as the response,
// this function assumes that the format of the HTTP response is the same as response.Contents.
// It is an error if response.Contents is empty in this case.  The response.Format field dictates
// the Content-Type of the HTTP response.
//
// If none of the above applies, the encoder pool is used to encode response.Routing to the HTTP
// response.  The content type is set to pool.Format().
func EncodeResponse(output http.ResponseWriter, response *Response, pool *wrp.EncoderPool) (err error) {
	if response.Error != nil {
		_, err = httperror.Formatf(
			output,
			http.StatusInternalServerError,
			"Transaction error: %s",
			response.Error,
		)

		return
	}

	if pool == nil || pool.Format() == response.Format {
		if len(response.Contents) == 0 {
			_, err = httperror.Format(
				output,
				http.StatusInternalServerError,
				"Transaction response had no content",
			)

			return
		}

		output.Header().Set("Content-Type", response.Format.ContentType())
		_, err = output.Write(response.Contents)
		return
	}

	output.Header().Set("Content-Type", pool.Format().ContentType())
	err = pool.Encode(output, response.Message)
	return
}

// Transactions represents a set of pending transactions.  Instances are safe for
// concurrent access.
type Transactions struct {
	lock    sync.RWMutex
	pending map[string]chan *Response
}

func NewTransactions() *Transactions {
	return &Transactions{
		pending: make(map[string]chan *Response, 1000),
	}
}

// Len returns the count of pending transactions
func (t *Transactions) Len() int {
	t.lock.RLock()
	defer t.lock.RUnlock()
	return len(t.pending)
}

// Keys returns a slice containing the transaction keys that are pending
func (t *Transactions) Keys() []string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	var (
		keys     = make([]string, len(t.pending))
		position int
	)

	for key, _ := range t.pending {
		keys[position] = key
		position++
	}

	return keys
}

// Complete dispatches the given response to the appropriate channel returned from Register
// and removes the transaction from the internal pending set.  This method is intended for
// goroutines that are servicing queues of messages, e.g. the read pump of a Manager.  Such goroutines
// use this method to indicate that a transaction is complete.
//
// If this method is passed a nil response, it panics.
func (t *Transactions) Complete(transactionKey string, response *Response) error {
	if len(transactionKey) == 0 {
		return ErrorInvalidTransactionKey
	} else if response == nil {
		panic("nil response")
	}

	t.lock.Lock()
	result, ok := t.pending[transactionKey]
	delete(t.pending, transactionKey)
	t.lock.Unlock()

	if !ok {
		return ErrorNoSuchTransactionKey
	}

	result <- response
	close(result)
	return nil
}

// Cancel simply cancels a transaction.  The transaction key is removed from the pending set.  If that
// transaction key is not registered, this method does nothing.  The channel returned from Register
// is closed, which will cause any code waiting for a response to get a nil Response.
//
// This method is normally called by the same goroutine that calls Register to ensure that transactions
// are cleaned up.
func (t *Transactions) Cancel(transactionKey string) {
	t.lock.Lock()
	result, ok := t.pending[transactionKey]
	delete(t.pending, transactionKey)
	t.lock.Unlock()

	if ok {
		close(result)
	}
}

// Register inserts a transaction key into the pending set and returns a channel that a Response
// will be repoted on.  This method is intended to be called by goroutines which want to wait for
// a transaction to complete.
//
// This method returns an error if either transactionKey is the empty string or if a transaction
// with this key has already been registered.  The latter is a more serious problem, since it indicates
// that higher-level code has generated duplicate transaction identifiers.  For safety, a Transactions
// instance expressly does not allow that case.
//
// The returned channel will either receive a non-nil response from some code calling Complete, or will
// see a channel closure (nil Response) from some code calling Cancel.
func (t *Transactions) Register(transactionKey string) (<-chan *Response, error) {
	if len(transactionKey) == 0 {
		return nil, ErrorInvalidTransactionKey
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	if _, ok := t.pending[transactionKey]; ok {
		return nil, ErrorTransactionAlreadyRegistered
	}

	result := make(chan *Response, 1)
	t.pending[transactionKey] = result
	return result, nil
}
