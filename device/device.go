package device

import (
	"bytes"
	"fmt"
	"io"
	"sync/atomic"
	"time"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
)

const (
	stateOpen int32 = iota
	stateClosed
)

// envelope is a tuple of a device Request and a send-only channel for errors.
// The write pump goroutine will use the complete channel to communicate the result
// of the write operation.
type envelope struct {
	request  *Request
	complete chan<- error
}

// Interface is the core type for this package.  It provides
// access to public device metadata and the ability to send messages
// directly the a device.
//
// Instances are mostly immutable, and have a strict lifecycle.  Devices are
// initially open, and when closed cannot be reused or reopened.  A new
// device instance is required if further communication is desired after
// the original device instance is closed.
//
// The only piece of metadata that is mutable is the Key.  A device Manager
// allows clients to change the routing Key of a device.  All other public
// metadata is immutable.
//
// Each device will have a pair of goroutines within the enclosing manager:
// a read and write, referred to as pumps.  The write pump services the queue
// of messages used by Send, while the read pump rarely needs to interact
// with devices directly.
//
// The String() method will always return a valid JSON object representation
// of this device.
type Interface interface {
	fmt.Stringer

	// ID returns the canonicalized identifer for this device.  Note that
	// this is NOT globally unique.  It is possible for multiple devices
	// with the same ID to be connected.  This typically occurs due to fraud,
	// but we don't want to turn away duped devices.
	ID() ID

	// Pending returns the count of pending messages for this device
	Pending() int

	// Closed tests if this device is closed.  When this method returns true,
	// any attempt to send messages to this device will result in an error.
	//
	// Once closed, a device cannot be reopened.
	Closed() bool

	// Send dispatches a message to this device.  This method is useful outside
	// a Manager if multiple messages should be sent to the device.  The Request.Message field
	// is not required if Request.Contents and Request.Format are set appropriately.  However,
	// a Request.Message is the only way to start a transaction.
	//
	// This method is synchronous.  If the request is of a type that should expect a response,
	// that response is returned.  An error is returned if this device has been closed or
	// if there were any I/O issues sending the request.
	//
	// Internally, the requests passed to this method are serviced by the write pump in
	// the enclosing Manager instance.  The read pump will handle sending the response.
	Send(*Request) (*Response, error)

	// Statistics returns the current, tracked Statistics instance for this device
	Statistics() Statistics

	// MarshalJSONTo writes a JSON representation to the given output io.Writer
	MarshalJSONTo(func(time.Time) time.Duration, io.Writer) (int, error)
}

// device is the internal Interface implementation.  This type holds the internal
// metadata exposed publicly, and provides some internal data structures for housekeeping.
type device struct {
	id ID

	errorLog log.Logger
	infoLog  log.Logger
	debugLog log.Logger

	statistics Statistics

	state int32

	shutdown     chan struct{}
	messages     chan *envelope
	transactions *Transactions
}

// newDevice is an internal factory function for devices
func newDevice(id ID, queueSize int, connectedAt time.Time, logger log.Logger) *device {
	return &device{
		id:           id,
		errorLog:     logging.Error(logger, "id", id),
		infoLog:      logging.Info(logger, "id", id),
		debugLog:     logging.Debug(logger, "id", id),
		statistics:   NewStatistics(connectedAt),
		state:        stateOpen,
		shutdown:     make(chan struct{}),
		messages:     make(chan *envelope, queueSize),
		transactions: NewTransactions(),
	}
}

// String returns the JSON representation of this device
func (d *device) String() string {
	return string(d.id)
}

func (d *device) MarshalJSONTo(since func(time.Time) time.Duration, output io.Writer) (int, error) {
	return fmt.Fprintf(
		output,
		`{"id": "%s", "closed": %t, "bytesReceived": %d, "bytesSent": %d, "messagesSent": %d, "connectedAt": "%s", "upTime": "%s"}`,
		d.id,
		d.Closed(),
		d.statistics.BytesReceived(),
		d.statistics.BytesSent(),
		d.statistics.MessagesSent(),
		d.statistics.ConnectedAt().Format(time.RFC3339),
		since(d.statistics.ConnectedAt()),
	)
}

func (d *device) MarshalJSON() ([]byte, error) {
	var output bytes.Buffer
	_, err := d.MarshalJSONTo(time.Since, &output)
	return output.Bytes(), err
}

func (d *device) requestClose() {
	if atomic.CompareAndSwapInt32(&d.state, stateOpen, stateClosed) {
		close(d.shutdown)
	}
}

func (d *device) ID() ID {
	return d.id
}

func (d *device) Pending() int {
	return len(d.messages)
}

func (d *device) Closed() bool {
	return atomic.LoadInt32(&d.state) != stateOpen
}

// sendRequest attempts to enqueue the given request for the write pump that is
// servicing this device.  This method honors the request context's cancellation semantics.
//
// This function returns when either (1) the write pump has attempted to send the message to
// the device, or (2) the request's context has been cancelled, which includes timing out.
func (d *device) sendRequest(request *Request) error {
	var (
		done     = request.Context().Done()
		complete = make(chan error, 1)
		envelope = &envelope{
			request,
			complete,
		}
	)

	// attempt to enqueue the message
	select {
	case <-done:
		return request.Context().Err()
	case <-d.shutdown:
		return ErrorDeviceClosed
	case d.messages <- envelope:
	}

	// once enqueued, wait until the context is cancelled
	// or there's a result
	select {
	case <-done:
		return request.Context().Err()
	case <-d.shutdown:
		return ErrorDeviceClosed
	case err := <-complete:
		return err
	}
}

// awaitResponse waits for the read pump to acquire a response that corresponds to the
// request's transaction key.  The result channel will receive the response from the
// read pump.
func (d *device) awaitResponse(request *Request, result <-chan *Response) (*Response, error) {
	select {
	case <-request.Context().Done():
		return nil, request.Context().Err()
	case <-d.shutdown:
		return nil, ErrorDeviceClosed
	case response := <-result:
		if response == nil {
			return nil, ErrorTransactionCancelled
		}

		return response, nil
	}
}

func (d *device) Send(request *Request) (*Response, error) {
	if d.Closed() {
		return nil, ErrorDeviceClosed
	}

	var (
		transactionKey, transactional = request.Transactional()
		result                        <-chan *Response
	)

	if transactional {
		var err error
		if result, err = d.transactions.Register(transactionKey); err != nil {
			// if a transaction key cannot be registered, we don't want to proceed.
			// this indicates some larger problem, most often a duplicate transaction key.
			return nil, err
		}

		// ensure that the transaction is cleared
		defer d.transactions.Cancel(transactionKey)
	}

	if err := d.sendRequest(request); err != nil {
		return nil, err
	}

	if result == nil {
		// if there is no pending transaction, we're done
		return nil, nil
	}

	return d.awaitResponse(request, result)
}

func (d *device) Statistics() Statistics {
	return d.statistics
}
