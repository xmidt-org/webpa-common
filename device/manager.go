package device

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/go-kit/kit/log"
)

var (
	authStatus = &wrp.AuthorizationStatus{Status: wrp.AuthStatusAuthorized}

	// authStatusRequest is the device Request sent for a successful authorization.
	authStatusRequest = Request{
		Message: authStatus,
		Contents: wrp.MustEncode(
			authStatus,
			wrp.Msgpack,
		),
		Format: wrp.Msgpack,
	}
)

// Connector is a strategy interface for managing device connections to a server.
// Implementations are responsible for upgrading websocket connections and providing
// for explicit disconnection.
type Connector interface {
	// Connect upgrade an HTTP connection to a websocket and begins concurrent
	// managment of the device.
	Connect(http.ResponseWriter, *http.Request, http.Header) (Interface, error)

	// Disconnect disconnects all devices (including duplicates) which connected
	// with the given ID.  This method returns the number of devices disconnected,
	// which can be zero or a positive integer.  Multiple devices are permitted with
	// the same ID, and this method disconnects all duplicate devices associated with that ID.
	Disconnect(ID) int

	// DisconnectIf iterates over all devices known to this manager, applying the
	// given predicate.  For any devices that result in true, this method disconnects them.
	// Note that this method may pause connections and disconnections while it is executing.
	// This method returns the number of devices that were disconnected.
	//
	// Only disconnection by ID is supported, which means that any identifier matching
	// the predicate will result in *all* duplicate devices under that ID being removed.
	//
	// No methods on this Manager should be called from within the predicate function, or
	// a deadlock will likely occur.
	DisconnectIf(func(ID) bool) int
}

// Router handles dispatching messages to devices.
type Router interface {
	// Route dispatches a WRP request to exactly one device, identified by the ID
	// field of the request.  Route is synchronous, and honors the cancellation semantics
	// of the Request's context.
	Route(*Request) (*Response, error)
}

// Registry is the strategy interface for querying the set of connected devices.  Methods
// in this interface follow the Visitor pattern and are typically executed under a read lock.
type Registry interface {
	// Statistics returns the tracked statistics for a given device.
	Statistics(ID) (Statistics, error)

	// VisitIf applies a visitor to any device matching the ID predicate.
	//
	// No methods on this Manager should be called from within either the predicate
	// or the visitor, or a deadlock will most definitely occur.
	VisitIf(func(ID) bool, func(Interface)) int

	// VisitAll applies the given visitor function to each device known to this manager.
	//
	// No methods on this Manager should be called from within the visitor function, or
	// a deadlock will likely occur.
	VisitAll(func(Interface)) int
}

// Manager supplies a hub for connecting and disconnecting devices as well as
// an access point for obtaining device metadata.
type Manager interface {
	Connector
	Router
	Registry
}

// NewManager constructs a Manager from a set of options.  A ConnectionFactory will be
// created from the options if one is not supplied.
func NewManager(o *Options, cf ConnectionFactory) Manager {
	if cf == nil {
		cf = NewConnectionFactory(o)
	}

	logger := o.logger()
	m := &manager{
		logger:   logger,
		errorLog: logging.Error(logger),
		debugLog: logging.Debug(logger),

		connectionFactory:      cf,
		registry:               newRegistry(o.initialCapacity()),
		deviceMessageQueueSize: o.deviceMessageQueueSize(),
		pingPeriod:             o.pingPeriod(),
		authDelay:              o.authDelay(),

		listeners: o.listeners(),
	}

	return m
}

// manager is the internal Manager implementation.
type manager struct {
	logger   log.Logger
	errorLog log.Logger
	debugLog log.Logger

	connectionFactory ConnectionFactory

	registry *registry

	deviceMessageQueueSize int
	pingPeriod             time.Duration
	authDelay              time.Duration

	listeners []Listener
}

func (m *manager) Connect(response http.ResponseWriter, request *http.Request, responseHeader http.Header) (Interface, error) {
	m.debugLog.Log(logging.MessageKey(), "device connect", "url", request.URL)
	id, ok := GetID(request.Context())
	if !ok {
		httperror.Format(
			response,
			http.StatusInternalServerError,
			ErrorMissingDeviceNameContext,
		)

		return nil, ErrorMissingDeviceNameContext
	}

	c, err := m.connectionFactory.NewConnection(response, request, responseHeader)
	if err != nil {
		return nil, err
	}

	d := newDevice(id, m.deviceMessageQueueSize, m.logger)
	closeOnce := new(sync.Once)
	go m.readPump(d, c, closeOnce)
	go m.writePump(d, c, closeOnce)
	m.registry.add(d)

	return d, nil
}

func (m *manager) dispatch(e *Event) {
	for _, listener := range m.listeners {
		listener(e)
	}
}

// pumpClose handles the proper shutdown and logging of a device's pumps.
// This method should be executed within a sync.Once, so that it only executes
// once for a given device.
//
// Note that the write pump does additional cleanup.  In particular, the write pump
// dispatches message failed events for any messages that were waiting to be delivered
// at the time of pump closure.
func (m *manager) pumpClose(d *device, c Connection, pumpError error) {
	if pumpError != nil {
		d.errorLog.Log(logging.MessageKey(), "pump close", logging.ErrorKey(), pumpError)
	} else {
		d.debugLog.Log(logging.MessageKey(), "pump close")
	}

	// always request a close, to ensure that the write goroutine is
	// shutdown and to signal to other goroutines that the device is closed
	d.requestClose()

	if closeError := c.Close(); closeError != nil {
		d.debugLog.Log(logging.MessageKey(), "Error closing device connection", logging.ErrorKey(), closeError)
	}

	m.dispatch(
		&Event{
			Type:   Disconnect,
			Device: d,
		},
	)
}

// pongCallbackFor creates a callback that delegates to this Manager's Listeners
// for the given device.
func (m *manager) pongCallbackFor(d *device) func(string) {
	// reuse the same event instance to ease gc pressure
	event := new(Event)

	return func(data string) {
		event.SetPong(d, data)
		m.dispatch(event)
	}
}

// readPump is the goroutine which handles the stream of WRP messages from a device.
// This goroutine exits when any error occurs on the connection.
func (m *manager) readPump(d *device, c Connection, closeOnce *sync.Once) {
	d.debugLog.Log(logging.MessageKey(), "readPump starting")

	var (
		frameRead bool
		readError error
		event     Event // reuse the same event as a carrier of data to listeners
		decoder   = wrp.NewDecoder(nil, wrp.Msgpack)
	)

	// all the read pump has to do is ensure the device and the connection are closed
	// it is the write pump's responsibility to do further cleanup
	defer closeOnce.Do(func() { m.pumpClose(d, c, readError) })
	c.SetPongCallback(m.pongCallbackFor(d))

	for {
		var frameBuffer bytes.Buffer
		frameRead, readError = c.Read(&frameBuffer)
		if readError != nil {
			return
		} else if !frameRead {
			d.errorLog.Log(logging.MessageKey(), "skipping unsupported frame")
			continue
		}

		var (
			message  = new(wrp.Message)
			rawFrame = frameBuffer.Bytes()
		)

		d.statistics.AddBytesReceived(uint32(len(rawFrame)))
		decoder.ResetBytes(rawFrame)
		if decodeError := decoder.Decode(message); decodeError != nil {
			// malformed WRP messages are allowed: the read pump will keep on chugging
			d.errorLog.Log(logging.MessageKey(), "skipping malformed frame", logging.ErrorKey(), decodeError)
			continue
		}

		d.statistics.AddMessagesReceived(1)
		event.SetMessageReceived(d, message, wrp.Msgpack, rawFrame)

		// update any waiting transaction
		if message.IsTransactionPart() {
			err := d.transactions.Complete(
				message.TransactionKey(),
				&Response{
					Device:   d,
					Message:  message,
					Format:   wrp.Msgpack,
					Contents: rawFrame,
				},
			)

			if err != nil {
				d.errorLog.Log(logging.MessageKey(), "Error while completing transaction", logging.ErrorKey(), err)
				event.Type = TransactionBroken
				event.Error = err
			} else {
				event.Type = TransactionComplete
			}
		}

		m.dispatch(&event)
	}
}

// writePump is the goroutine which services messages addressed to the device.
// this goroutine exits when either an explicit shutdown is requested or any
// error occurs on the connection.
func (m *manager) writePump(d *device, c Connection, closeOnce *sync.Once) {
	d.debugLog.Log(logging.MessageKey(), "writePump starting")

	var (
		// we'll reuse this event instance
		event = Event{Type: Connect, Device: d}

		envelope   *envelope
		frame      io.WriteCloser
		encoder    = wrp.NewEncoder(nil, wrp.Msgpack)
		writeError error

		pingData    = fmt.Sprintf("ping[%s]", d.id)
		pingMessage = []byte(pingData)
		pingTicker  = time.NewTicker(m.pingPeriod)
	)

	m.dispatch(&event)

	// cleanup: we not only ensure that the device and connection are closed but also
	// ensure that any messages that were waiting and/or failed are dispatched to
	// the configured listener
	defer func() {
		pingTicker.Stop()
		closeOnce.Do(func() { m.pumpClose(d, c, writeError) })

		// notify listener of any message that just now failed
		// any writeError is passed via this event
		if envelope != nil {
			event.SetRequestFailed(d, envelope.request, writeError)
			m.dispatch(&event)
		}

		// drain the messages, dispatching them as message failed events.  we never close
		// the message channel, so just drain until a receive would block.
		//
		// Nil is passed explicitly as the error to indicate that these messages failed due
		// to the device disconnecting, not due to an actual I/O error.
		for {
			select {
			case undeliverable := <-d.messages:
				d.errorLog.Log(logging.MessageKey(), "undeliverable message", "deviceMessage", undeliverable)
				event.SetRequestFailed(d, undeliverable.request, writeError)
				m.dispatch(&event)
			default:
				return
			}
		}
	}()

	// wait for the delay, then send an auth status request to the device
	time.AfterFunc(m.authDelay, func() {
		// TODO: This will keep the device from being garbage collected until the timer
		// triggers.  This is only a problem if a device connects then disconnects faster
		// than the authDelay setting.
		d.Send(&authStatusRequest)
	})

	for writeError == nil {
		envelope = nil

		select {
		case <-d.shutdown:
			writeError = c.SendClose()
			return

		case envelope = <-d.messages:
			if frame, writeError = c.NextWriter(); writeError == nil {
				var frameContents []byte
				if envelope.request.Format == wrp.Msgpack && len(envelope.request.Contents) > 0 {
					frameContents = envelope.request.Contents
				} else {
					// if the request was in a format other than Msgpack, or if the caller did not pass
					// Contents, then do the encoding here.
					encoder.ResetBytes(&frameContents)
					writeError = encoder.Encode(envelope.request.Message)
				}

				if writeError == nil {
					var bytesSent int
					if bytesSent, writeError = frame.Write(frameContents); writeError == nil {
						d.statistics.AddBytesSent(uint32(bytesSent))
						d.statistics.AddMessagesSent(1)
						writeError = frame.Close()
					} else {
						// don't mask the original error, but ensure the frame is closed
						frame.Close()
					}
				}
			}

			if writeError != nil {
				envelope.complete <- writeError
				event.SetRequestFailed(d, envelope.request, writeError)
			} else {
				event.SetRequestSuccess(d, envelope.request)
			}

			close(envelope.complete)
			m.dispatch(&event)

		case <-pingTicker.C:
			writeError = c.Ping(pingMessage)
			event.SetPing(d, pingData, writeError)
			m.dispatch(&event)
		}
	}
}

// wrapVisitor produces an internal visitor that wraps a delegate
// and preserves encapsulation
func (m *manager) wrapVisitor(delegate func(Interface)) func(*device) {
	return func(d *device) {
		delegate(d)
	}
}

func (m *manager) Disconnect(id ID) int {
	removedDevices := m.registry.removeAll(id)
	for _, d := range removedDevices {
		d.requestClose()
	}

	return len(removedDevices)
}

func (m *manager) DisconnectIf(filter func(ID) bool) int {
	return m.registry.removeIf(filter, func(d *device) {
		d.requestClose()
	})
}

func (m *manager) Statistics(id ID) (result Statistics, err error) {
	count := m.registry.visitID(id, func(d *device) {
		result = d.Statistics()
	})

	if count > 1 {
		err = ErrorNonUniqueID
	} else if count < 1 {
		err = ErrorDeviceNotFound
	}

	return
}

func (m *manager) VisitIf(filter func(ID) bool, visitor func(Interface)) int {
	return m.registry.visitIf(filter, m.wrapVisitor(visitor))
}

func (m *manager) VisitAll(visitor func(Interface)) int {
	return m.registry.visitAll(m.wrapVisitor(visitor))
}

func (m *manager) Route(request *Request) (*Response, error) {
	if destination, err := request.ID(); err != nil {
		return nil, err
	} else if d, err := m.registry.getOne(destination); err != nil {
		return nil, err
	} else {
		return d.Send(request)
	}
}
