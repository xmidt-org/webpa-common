package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/convey"
	"github.com/xmidt-org/webpa-common/v2/convey/conveymetric"
	"go.uber.org/zap"
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
	json.Marshaler

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

	// Convey returns a read-only view of the device convey information
	Convey() convey.Interface

	// ConveyCompliance returns the result of attempting to parse the convey information
	// sent during device connection
	ConveyCompliance() convey.Compliance

	// Metadata returns a key value store object for information that's useful to guide interactions
	// with a device such as security credentials.
	Metadata() *Metadata

	// CloseReason returns the metadata explaining why a device was closed.  If this device
	// is not closed, this method's return is undefined.
	CloseReason() CloseReason
}

// device is the internal Interface implementation.  This type holds the internal
// metadata exposed publicly, and provides some internal data structures for housekeeping.
type device struct {
	id ID

	logger *zap.Logger

	statistics Statistics

	state int32

	shutdown     chan struct{}
	messages     chan *envelope
	transactions *Transactions

	c             convey.Interface
	compliance    convey.Compliance
	conveyClosure conveymetric.Closure

	metadata *Metadata

	closeReason atomic.Value
}

type deviceOptions struct {
	ID          ID
	C           convey.Interface
	Compliance  convey.Compliance
	QueueSize   int
	ConnectedAt time.Time
	Logger      *zap.Logger
	Metadata    *Metadata
}

// newDevice is an internal factory function for devices
func newDevice(o deviceOptions) *device {
	if o.ConnectedAt.IsZero() {
		o.ConnectedAt = time.Now()
	}

	if o.Logger == nil {
		o.Logger = sallust.Default()
	}

	if o.QueueSize < 1 {
		o.QueueSize = DefaultDeviceMessageQueueSize
	}

	return &device{
		id:           o.ID,
		logger:       o.Logger.With(zap.String("id", string(o.ID))),
		statistics:   NewStatistics(nil, o.ConnectedAt),
		c:            o.C,
		compliance:   o.Compliance,
		state:        stateOpen,
		shutdown:     make(chan struct{}),
		messages:     make(chan *envelope, o.QueueSize),
		transactions: NewTransactions(),
		metadata:     o.Metadata,
	}
}

// String returns the JSON representation of this device
func (d *device) String() string {
	return string(d.id)
}

func (d *device) MarshalJSON() ([]byte, error) {
	var output bytes.Buffer
	_, err := fmt.Fprintf(
		&output,
		`{"id": "%s", "pending": %d, "statistics": %s}`,
		d.id,
		len(d.messages),
		d.statistics,
	)

	return output.Bytes(), err
}

func (d *device) requestClose(reason CloseReason) error {
	if atomic.CompareAndSwapInt32(&d.state, stateOpen, stateClosed) {
		close(d.shutdown)
		d.transactions.Close()

		if len(reason.Text) == 0 {
			reason.Text = "unknown"
		}

		d.closeReason.Store(reason)
	}

	return nil
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

func (d *device) Convey() convey.Interface {
	return d.c
}

func (d *device) ConveyCompliance() convey.Compliance {
	return d.compliance
}

func (d *device) Metadata() *Metadata {
	return d.metadata
}

func (d *device) CloseReason() CloseReason {
	if v, ok := d.closeReason.Load().(CloseReason); ok {
		return v
	}

	return CloseReason{}
}
