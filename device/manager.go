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
	Connect(http.ResponseWriter, *http.Request) (Interface, error)

	// Disconnect disconnects all devices (including duplicates) which connected
	// with the given ID
	Disconnect(ID)

	// Send sends a message to all devices registered with the given identifier
	// This method returns the number of devices to which the message was enqueued.
	Send(ID, *wrp.Message) int
}

// NewManager constructs a Manager using a set of Options.
func NewManager(o *Options) Manager {
	if o == nil {
		o = &defaultOptions
	}

	return &websocketManager{
		idParser:     NewIDParser(o.DeviceNameHeader),
		conveyParser: NewConveyParser(o.ConveyHeader, nil),
		upgrader: websocket.Upgrader{
			HandshakeTimeout: o.HandshakeTimeout,
			ReadBufferSize:   o.ReadBufferSize,
			WriteBufferSize:  o.WriteBufferSize,
			Subprotocols:     o.subprotocols(),
		},
		registry:               make(registry, o.initialRegistrySize()),
		logger:                 o.logger(),
		deviceMessageQueueSize: o.deviceMessageQueueSize(),
		listeners:              o.Listeners.Clone(),
	}
}

type websocketManager struct {
	logger       logging.Logger
	idParser     IDParser
	conveyParser ConveyParser
	upgrader     websocket.Upgrader

	registry     registry
	registryLock sync.RWMutex

	deviceMessageQueueSize int

	pingPeriod   time.Duration
	idleInterval time.Duration
	writeTimeout time.Duration

	listeners *Listeners
}

func (wm *websocketManager) Connect(response http.ResponseWriter, request *http.Request) (Interface, error) {
	wm.logger.Debug("Connect(%s, %v)", request.URL, request.Header)
	id, err := wm.idParser.FromRequest(request)
	if err != nil {
		httperror.Write(response, err)
		return nil, err
	}

	convey, err := wm.conveyParser.FromRequest(request)
	if err != nil {
		httperror.Write(response, err)
		return nil, err
	}

	connection, err := wm.upgrader.Upgrade(response, request, nil)
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
		idleInterval:       wm.idleInterval,
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
func (wm *websocketManager) removeOne(device *device) {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	wm.registry.removeOne(device)
}

func (wm *websocketManager) Disconnect(id ID) {
	wm.logger.Debug("Disconnect(%s)", id)
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	removedDevices := wm.registry.removeAll(id)
	for _, device := range removedDevices {
		// pass nil for the preClose, as we've already removed the device(s)
		device.close(nil, nil)
	}
}

func (wm *websocketManager) Send(id ID, message *wrp.Message) int {
	defer wm.registryLock.RUnlock()
	wm.registryLock.RLock()
	if devices, ok := wm.registry[id]; ok {
		for _, device := range devices {
			device.sendMessage(message)
		}

		return len(devices)
	}

	return 0
}
