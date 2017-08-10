package device

import (
	"bytes"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

const (
	stateOpen int32 = iota
	stateClosed
)

var (
	nullConvey = []byte("null")
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

	// Key returns the current unique key for this device.
	Key() Key

	// Convey returns the payload to convey with each web-bound request
	Convey() Convey

	// EncodedConvey returns the exact value of the convey header sent at the time
	// this device connected to the manager
	EncodedConvey() string

	// SetConveyHeader sets the appropriate header if this device has any associated convey data.
	// If this device has no convey data, this method does nothing.
	SetConveyHeader(http.Header)

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
}

// device is the internal Interface implementation.  This type holds the internal
// metadata exposed publicly, and provides some internal data structures for housekeeping.
type device struct {
	id  ID
	key atomic.Value

	convey        Convey
	encodedConvey string

	statistics Statistics

	state int32

	shutdown     chan struct{}
	messages     chan *envelope
	transactions *Transactions
}

// newDevice is an internal factory function for devices
func newDevice(id ID, initialKey Key, convey Convey, encodedConvey string, queueSize int) *device {
	d := &device{
		id:            id,
		convey:        convey,
		encodedConvey: encodedConvey,
		statistics:    NewStatistics(time.Now().UTC()),
		state:         stateOpen,
		shutdown:      make(chan struct{}),
		messages:      make(chan *envelope, queueSize),
		transactions:  NewTransactions(),
	}

	d.updateKey(initialKey)
	return d
}

// MarshalJSON exposes public metadata about this device as JSON.  This
// method will always return a nil error and produce valid JSON.
func (d *device) MarshalJSON() ([]byte, error) {
	conveyJSON := nullConvey
	if d.convey != nil {
		if decoded, conveyError := EncodeConvey(d.convey, nil); conveyError != nil {
			// just dump the error text into the convey property,
			// so at least it can be viewed
			conveyJSON = []byte(fmt.Sprintf(`"%s"`, decoded))
		}
	}

	output := new(bytes.Buffer)
	fmt.Fprintf(
		output,
		`{"id": "%s", "key": "%s", "closed": %t, "convey": %s}`,
		d.id,
		d.Key(),
		d.Closed(),
		conveyJSON,
	)

	return output.Bytes(), nil
}

// String returns the JSON representation of this device
func (d *device) String() string {
	data, _ := d.MarshalJSON()
	return string(data)
}

func (d *device) requestClose() {
	if atomic.CompareAndSwapInt32(&d.state, stateOpen, stateClosed) {
		close(d.shutdown)
	}
}

func (d *device) ID() ID {
	return d.id
}

func (d *device) Key() Key {
	return d.key.Load().(Key)
}

func (d *device) updateKey(newKey Key) {
	d.key.Store(newKey)
}

func (d *device) Convey() Convey {
	return d.convey
}

func (d *device) EncodedConvey() string {
	return d.encodedConvey
}

func (d *device) SetConveyHeader(header http.Header) {
	if len(d.encodedConvey) > 0 {
		header.Set(ConveyHeader, d.encodedConvey)
	}
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
		transactionKey = request.TransactionKey()
		result         <-chan *Response
	)

	if len(transactionKey) > 0 {
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
