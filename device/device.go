package device

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/wrp"
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

// envelope is a tuple of an original WRP message together with that message's
// (optional) encoded representation.
type envelope struct {
	message wrp.Routable
	encoded []byte
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
type Interface interface {
	// ID returns the canonicalized identifer for this device.  Note that
	// this is NOT globally unique.  It is possible for multiple devices
	// with the same ID to be connected.  This typically occurs due to fraud,
	// but we don't want to turn away duped devices.
	ID() ID

	// Key returns the current unique key for this device.
	Key() Key

	// Convey returns the payload to convey with each web-bound request
	Convey() Convey

	// ConnectedAt returns the time at which this device connected to the system
	ConnectedAt() time.Time

	// Pending returns the count of pending messages for this device
	Pending() int

	// RequestClose posts a request for this device to be disconnected.  This method
	// is asynchronous and idempotent.  If this method is invoked when a shutdown
	// request has already been queued or when this device is already shut down, this
	// method returns an error.
	RequestClose()

	// Closed tests if this device is closed.  When this method returns true,
	// any attempt to send messages to this device will result in an error.
	//
	// Once closed, a device cannot be reopened.
	Closed() bool

	// Send dispatches a message to this device.  This method is useful outside
	// a Manager if multiple messages should be sent to the device.
	//
	// Similar to Manager.Route, the byte slice, if supplied, must be valid msgpack-encoded
	// WRP to send to the device.  If this byte slice is empty, the given message is encoded
	// using msgpack.
	//
	// This method will return an error if this device has been closed or
	// if the device is busy and cannot accept more messages.
	Send(wrp.Routable, []byte) error
}

// device is the internal Interface implementation.  This type holds the internal
// metadata exposed publicly, and provides some internal data structures for housekeeping.
type device struct {
	id  ID
	key atomic.Value

	convey      Convey
	connectedAt time.Time

	state int32

	shutdown chan struct{}
	messages chan *envelope
}

func newDevice(id ID, initialKey Key, convey Convey, queueSize int) *device {
	d := &device{
		id:          id,
		convey:      convey,
		connectedAt: time.Now(),
		state:       stateOpen,
		shutdown:    make(chan struct{}),
		messages:    make(chan *envelope, queueSize),
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
		`{"id": "%s", "key": "%s", "pending": %d, "connectedAt": "%s", "closed": %t, "convey": %s}`,
		d.id,
		d.Key(),
		d.Pending(),
		d.connectedAt.Format(time.RFC3339),
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

func (d *device) RequestClose() {
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

func (d *device) ConnectedAt() time.Time {
	return d.connectedAt
}

func (d *device) Pending() int {
	return len(d.messages)
}

func (d *device) Closed() bool {
	return atomic.LoadInt32(&d.state) != stateOpen
}

func (d *device) Send(message wrp.Routable, encoded []byte) (err error) {
	if d.Closed() {
		return NewClosedError(d.id, d.Key())
	}

	select {
	case d.messages <- &envelope{message, encoded}:
		return nil
	default:
		return NewBusyError(d.id, d.Key())
	}
}
