package device

import (
	"bytes"
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"io"
	"net/http"
	"sync"
	"time"
)

var (
	// emptyMessage is a convenient, internal message used to reset wrp.Message instances for reuse
	emptyMessage wrp.Message
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

	// DisconnectOne disconnects the single device associated with the given Key.  This method
	// returns the count of devices disconnected, which will be zero (0) if no device existed
	// or one (1) if there was a device with that key.
	DisconnectOne(Key) int

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
	// Route dispatches the envelope to one or more devices
	Route(*wrp.Message, []byte, func(Interface, error)) (ID, int, error)
}

// Registry is the strategy interface for querying the set of connected devices.  Methods
// in this interface follow the Visitor pattern and are typically executed under a read lock.
type Registry interface {
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

	return &manager{
		logger: o.logger(),

		deviceNameHeader:             o.deviceNameHeader(),
		missingDeviceNameHeaderError: fmt.Errorf("Missing header: %s", o.deviceNameHeader()),

		conveyHeader: o.conveyHeader(),

		connectionFactory:      cf,
		keyFunc:                o.keyFunc(),
		registry:               newRegistry(o.initialCapacity()),
		deviceMessageQueueSize: o.deviceMessageQueueSize(),
		pingPeriod:             o.pingPeriod(),

		messageReceivedListener: o.messageReceivedListener(),
		connectListener:         o.connectListener(),
		disconnectListener:      o.disconnectListener(),
		pongListener:            o.pongListener(),
	}
}

// manager is the internal Manager implementation.
type manager struct {
	logger logging.Logger

	deviceNameHeader             string
	missingDeviceNameHeaderError error

	conveyHeader string

	connectionFactory ConnectionFactory
	keyFunc           KeyFunc

	lock     sync.RWMutex
	registry *registry

	deviceMessageQueueSize int
	pingPeriod             time.Duration

	messageReceivedListener MessageReceivedListener
	connectListener         ConnectListener
	disconnectListener      DisconnectListener
	pongListener            PongListener
}

func (m *manager) Connect(response http.ResponseWriter, request *http.Request, responseHeader http.Header) (Interface, error) {
	m.logger.Debug("Connect(%s, %v)", request.URL, request.Header)
	deviceName := request.Header.Get(m.deviceNameHeader)
	if len(deviceName) == 0 {
		http.Error(response, m.missingDeviceNameHeaderError.Error(), http.StatusBadRequest)
		return nil, m.missingDeviceNameHeaderError
	}

	id, err := ParseID(deviceName)
	if err != nil {
		badDeviceNameError := fmt.Errorf("Bad device name: %s", err)
		http.Error(response, badDeviceNameError.Error(), http.StatusBadRequest)
		return nil, badDeviceNameError
	}

	var convey Convey
	if rawConvey := request.Header.Get(m.conveyHeader); len(rawConvey) > 0 {
		convey, err = ParseConvey(rawConvey, nil)
		if err != nil {
			badConveyError := fmt.Errorf("Bad convey value [%s]: %s", rawConvey, err)
			http.Error(response, badConveyError.Error(), http.StatusBadRequest)
			return nil, badConveyError
		}
	}

	var initialKey Key
	if initialKey, err = m.keyFunc(id, convey, request); err != nil {
		keyError := fmt.Errorf("Unable to obtain key for device [%s]: %s", id, err)
		http.Error(response, keyError.Error(), http.StatusBadRequest)
		return nil, keyError
	}

	c, err := m.connectionFactory.NewConnection(response, request, responseHeader)
	if err != nil {
		return nil, err
	}

	d := newDevice(id, initialKey, convey, m.deviceMessageQueueSize)
	closeOnce := new(sync.Once)
	go m.readPump(d, c, closeOnce)
	go m.writePump(d, c, closeOnce)

	return d, nil
}

func (m *manager) whenWriteLocked(when func()) {
	m.lock.Lock()
	defer m.lock.Unlock()
	when()
}

func (m *manager) whenReadLocked(when func()) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	when()
}

// pumpCleanup handles the proper shutdown and logging of a device's pumps.
// This method should be executed within a sync.Once, so that it only executes
// once for a given device.
func (m *manager) pumpClose(d *device, c Connection, pumpError error) {
	m.logger.Debug("pumpClose(%s, %s)", d.id, pumpError)

	defer m.disconnectListener(d)

	// always request a close, to ensure that the write goroutine is
	// shutdown and to signal to other goroutines that the device is closed
	d.RequestClose()

	if pumpError != nil {
		m.logger.Error("Device [%s] pump encountered error: %s", d.id, pumpError)
	}

	if closeError := c.Close(); closeError != nil {
		m.logger.Error("Error closing connection for device [%s]: %s", d.id, closeError)
	}
}

// pongCallbackFor creates a callback that delegates to this Manager's Listeners
// for the given device.
func (m *manager) pongCallbackFor(d *device) func(string) {
	return func(data string) {
		m.pongListener(d, data)
	}
}

// readPump is the goroutine which handles the stream of WRP messages from a device.
// This goroutine exits when any error occurs on the connection.
func (m *manager) readPump(d *device, c Connection, closeOnce *sync.Once) {
	m.logger.Debug("readPump(%s)", d.id)

	var (
		frameRead   bool
		readError   error
		frameBuffer bytes.Buffer

		message wrp.Message
		decoder = wrp.NewDecoder(nil, wrp.Msgpack)
	)

	defer func() {
		closeOnce.Do(func() { m.pumpClose(d, c, readError) })
	}()

	c.SetPongCallback(m.pongCallbackFor(d))

	for {
		frameBuffer.Reset()
		frameRead, readError = c.Read(&frameBuffer)
		if readError != nil {
			return
		} else if !frameRead {
			m.logger.Warn("Skipping frame from device [%s]", d.id)
			continue
		}

		rawFrame := frameBuffer.Bytes()
		message = emptyMessage
		decoder.ResetBytes(rawFrame)
		if decodeError := decoder.Decode(&message); decodeError != nil {
			// malformed WRP messages are allowed: the read pump will keep on chugging
			m.logger.Error("Skipping malformed frame from device [%s]: %s", d.id, decodeError)
			continue
		}

		m.messageReceivedListener(d, &message, rawFrame)
	}
}

// writePump is the goroutine which services messages addressed to the device.
// this goroutine exits when either an explicit shutdown is requested or any
// error occurs on the connection.
func (m *manager) writePump(d *device, c Connection, closeOnce *sync.Once) {
	m.logger.Debug("writePump(%s)", d.id)

	// this makes this device addressable via the enclosing Manager:
	m.whenWriteLocked(func() {
		m.registry.add(d)
	})

	defer m.whenWriteLocked(func() {
		m.registry.removeOne(d)
	})

	// dispatch to the connect listener after the device is addressable
	m.connectListener(d)

	var (
		frame      io.WriteCloser
		writeError error
		encoder    = wrp.NewEncoder(nil, wrp.Msgpack)
	)

	defer func() {
		closeOnce.Do(func() { m.pumpClose(d, c, writeError) })
	}()

	pingMessage := []byte(fmt.Sprintf("ping[%s]", d.id))
	pingTicker := time.NewTicker(m.pingPeriod)
	defer pingTicker.Stop()

	for writeError == nil {
		select {
		case <-d.shutdown:
			writeError = c.SendClose()
			return

		case envelope := <-d.messages:
			frame, writeError = c.NextWriter()
			if writeError != nil {
				return
			}

			// if we have a pre-encoded byte slice, just write that
			if len(envelope.encoded) > 0 {
				_, writeError = frame.Write(envelope.encoded)
				if writeError != nil {
					return
				}
			} else {
				encoder.Reset(frame)
				if writeError = encoder.Encode(&envelope.message); writeError != nil {
					return
				}
			}

			writeError = frame.Close()

		case <-pingTicker.C:
			writeError = c.Ping(pingMessage)
		}
	}
}

// requestClose is a convenient, internal visitor
// that the various Disconnect methods use.
func (m *manager) requestClose(d *device) {
	d.RequestClose()
}

// wrapVisitor produces an internal visitor that wraps a delegate
// and preserves encapsulation
func (m *manager) wrapVisitor(delegate func(Interface)) func(*device) {
	return func(d *device) {
		delegate(d)
	}
}

func (m *manager) Disconnect(id ID) (count int) {
	m.logger.Debug("Disconnect(%s)", id)

	m.whenReadLocked(func() {
		count = m.registry.visitID(id, m.requestClose)
	})

	return
}

func (m *manager) DisconnectOne(key Key) (count int) {
	m.logger.Debug("DisconnectOne(%s)", key)

	m.whenReadLocked(func() {
		count = m.registry.visitKey(key, m.requestClose)
	})

	return
}

func (m *manager) DisconnectIf(filter func(ID) bool) (count int) {
	m.logger.Debug("DisconnectIf()")

	m.whenReadLocked(func() {
		count = m.registry.visitIf(filter, m.requestClose)
	})

	return count
}

func (m *manager) VisitIf(filter func(ID) bool, visitor func(Interface)) (count int) {
	m.logger.Debug("VisitIf")

	m.whenReadLocked(func() {
		count = m.registry.visitIf(filter, m.wrapVisitor(visitor))
	})

	return
}

func (m *manager) VisitAll(visitor func(Interface)) (count int) {
	m.logger.Debug("VisitAll")

	m.whenReadLocked(func() {
		count = m.registry.visitAll(m.wrapVisitor(visitor))
	})

	return
}

func (m *manager) Route(message *wrp.Message, encoded []byte, callback func(Interface, error)) (recipient ID, count int, err error) {
	recipient, err = ParseID(message.Destination)
	if err != nil {
		return
	}

	m.whenReadLocked(func() {
		count = m.registry.visitID(recipient, func(d *device) {
			if sendError := d.Send(message, encoded); sendError != nil && callback != nil {
				callback(d, sendError)
			}
		})
	})

	return
}
