package device

import (
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
	"time"
)

// Manager supplies a hub for connecting and disconnecting devices as well as
// an access point for obtaining device metadata.
type Manager interface {
	// Connect upgrade an HTTP connection to a websocket and begins concurrent
	// managment of the device.
	Connect(http.ResponseWriter, *http.Request, http.Header) (Interface, error)

	// Disconnect disconnects all devices (including duplicates) which connected
	// with the given ID
	Disconnect(ID)

	// DisconnectIf iterates over all devices known to this manager, applying the
	// given predicate.  For any devices that result in true, this method disconnects them.
	// Note that this method may pause connections and disconnections while it is executing.
	// This method returns the number of devices that were disconnected.
	//
	// Only disconnection by ID is supported, which means that any identifier matching
	// the predicate will result in *all* duplicate devices under that ID being removed.
	DisconnectIf(func(ID) bool) int

	// VisitAll applies the given visitor function to each device known to this manager.
	// VisitAll will typically lock the internal data structures for reading, which will
	// pause connections and disconnections while the visitor is applied.
	VisitAll(func(Interface))

	// Send sends a message to all devices registered with the given identifier
	// This method returns the number of devices to which the message was enqueued.
	// If no devices were connected with the given ID, this method returns zero (0).
	Send(ID, *wrp.Message) int
}

// NewManager constructs a Manager using a set of Options.
func NewManager(o *Options) Manager {
	if o == nil {
		o = &defaultOptions
	}

	return &websocketManager{
		logger:        o.logger(),
		idHandler:     NewIDHandler(o.DeviceNameHeader),
		conveyHandler: NewConveyHandler(o.ConveyHeader, nil),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: o.HandshakeTimeout,
			ReadBufferSize:   o.ReadBufferSize,
			WriteBufferSize:  o.WriteBufferSize,
			Subprotocols:     o.subprotocols(),
		},

		registry: newRegistry(o.initialRegistrySize()),

		deviceMessageQueueSize: o.deviceMessageQueueSize(),
		pingPeriod:             o.pingPeriod(),
		idlePeriod:             o.idlePeriod(),
		writeTimeout:           o.writeTimeout(),
		listeners:              o.Listeners.Clone(),
	}
}

type websocketManager struct {
	logger        logging.Logger
	idHandler     IDHandler
	conveyHandler ConveyHandler
	upgrader      websocket.Upgrader

	registry     *registry
	registryLock sync.RWMutex

	deviceMessageQueueSize int
	pingPeriod             time.Duration
	idlePeriod             time.Duration
	writeTimeout           time.Duration
	listeners              *Listeners
}

func (wm *websocketManager) Connect(response http.ResponseWriter, request *http.Request, responseHeader http.Header) (Interface, error) {
	wm.logger.Debug("Connect(%s, %v)", request.URL, request.Header)
	id, err := wm.idHandler.FromRequest(request)
	if err != nil {
		httperror.Write(response, err)
		return nil, err
	}

	convey, err := wm.conveyHandler.FromRequest(request)
	if err != nil {
		httperror.Write(response, err)
		return nil, err
	}

	connection, err := wm.upgrader.Upgrade(response, request, responseHeader)
	if err != nil {
		// Upgrade already writes to the response
		return nil, err
	}

	device := wm.newDevice(id, convey, connection)
	go device.readPump(wm.listeners, wm.removeOne)
	go device.writePump(wm.pingPeriod, wm.listeners, wm.removeOne)

	wm.add(device)
	wm.listeners.OnConnect(device)
	return device, nil
}

// newDevice is an internal Factory Method for devices.  This method only
// handles the instantiation of a device.
func (wm *websocketManager) newDevice(id ID, convey *Convey, c connection) *device {
	wm.logger.Debug("newDevice(%s, %v, %v)", id, convey, c)

	return &device{
		id:                 id,
		convey:             convey,
		connectedAt:        time.Now(),
		logger:             wm.logger,
		connection:         c,
		messages:           make(chan *wrp.Message, wm.deviceMessageQueueSize),
		shutdown:           make(chan struct{}),
		idlePeriod:         wm.idlePeriod,
		writeTimeout:       wm.writeTimeout,
		disconnectListener: wm.listeners,
	}
}

// add handles the addition of a new device, which might possibly be a duplicate
func (wm *websocketManager) add(device *device) {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	wm.registry.add(device)
}

// removeOne deletes a single device from the registry, leaving any other
// duplicates intact.
func (wm *websocketManager) removeOne(d *device) {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	wm.registry.removeOne(d.id, d.key)
}

func (wm *websocketManager) removeAll(id ID) keyMap {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	return wm.registry.removeAll(id)
}

// removeIf removes devices matching the given predicate
func (wm *websocketManager) removeIf(filter func(ID) bool) []*device {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	return wm.removeIf(filter)
}

func (wm *websocketManager) Disconnect(id ID) {
	wm.logger.Debug("Disconnect(%s)", id)
	removedDevices := wm.removeAll(id)

	// perform disconnection outside the mutex
	for _, device := range removedDevices {
		// pass nil for the preClose, as we've already removed the device(s)
		device.close(nil, nil)
	}
}

func (wm *websocketManager) DisconnectIf(filter func(ID) bool) int {
	wm.logger.Debug("DisconnectIf()")
	removedDevices := wm.removeIf(filter)

	// actual disconnection is done outside the mutex
	for _, device := range removedDevices {
		device.close(nil, nil)
	}

	return len(removedDevices)
}

func (wm *websocketManager) VisitAll(visitor func(Interface)) {
	wm.logger.Debug("VisitAll")
	defer wm.registryLock.RUnlock()
	wm.registryLock.RLock()
	wm.registry.visitAll(visitor)
}

func (wm *websocketManager) Send(id ID, message *wrp.Message) int {
	wm.logger.Debug("Send(%s, %v)", id, message)
	defer wm.registryLock.RUnlock()
	wm.registryLock.RLock()

	return wm.registry.visitID(id, func(d Interface) {
		d.Send(message)
	})
}
