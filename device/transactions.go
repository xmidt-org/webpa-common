package device

import (
	"context"
	"errors"
	"github.com/Comcast/webpa-common/wrp"
	"sync"
)

var (
	ErrorInvalidTransactionKey        = errors.New("Transaction keys must be non-empty strings")
	ErrorNoSuchTransactionKey         = errors.New("That transaction key is not registered")
	ErrorTransactionAlreadyRegistered = errors.New("That transaction is already registered")
)

// Request represents a single device Request, carrying routing information and message contents.
type Request struct {
	// ID is the device identifier to which this request is addressed
	ID ID

	// Routing is the original, decoded WRP message containing the routing information
	Routing wrp.Routable

	// Contents is the required Msgpack-encoded WRP message.  This is sent on-the-wire to the device.
	Contents []byte

	// ctx is the API context for this request, which can be nil.  Normally, it's best to
	// set this to context.Background() if no cancellation semantics are desired.
	ctx context.Context
}

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

// NewRequest creates a Request addressed to the Destination of the message, which
// must be a valid device ID.  The context of the returned request is context.Background(),
// which can be changed after this function returns.
//
// If the destination of the message could not be parsed into a device ID, this function
// returns a nil Request with the parse error.
func NewRequest(routing wrp.Routable, contents []byte, ctx context.Context) (*Request, error) {
	destination, err := ParseID(routing.To())
	if err != nil {
		return nil, err
	}

	return &Request{
		ID:       destination,
		Routing:  routing,
		Contents: contents,
		ctx:      ctx,
	}, nil
}

// Response represents the response to a device request.  Some requests have no response, in which case
// a Response without a Routing or Contents will be returned.
type Response struct {
	Device   Interface
	Routing  wrp.Routable
	Contents []byte
	Error    error
}

// Transactions represents a set of pending transactions.
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

func (t *Transactions) Complete(transactionKey string, response *Response) error {
	if len(transactionKey) == 0 {
		return ErrorInvalidTransactionKey
	}

	t.lock.Lock()
	result, ok := t.pending[transactionKey]
	delete(t.pending, transactionKey)
	t.lock.Unlock()

	if !ok {
		return ErrorNoSuchTransactionKey
	}

	if response != nil {
		result <- response
	}

	close(result)
	return nil
}

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
