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

const (
	DefaultInitialRegistrySize    = 10000
	DefaultDeviceMessageQueueSize = 100

	DefaultPingPeriod   time.Duration = 45 * time.Second
	DefaultIdleInterval time.Duration = 135 * time.Second
	DefaultWriteTimeout time.Duration = 60 * time.Second
)

// Options represent the available configuration options for device Managers
type Options struct {
	// DeviceNameHeader is the name of the HTTP request header which contains the
	// device name.  If not specified, DefaultDeviceNameHeader is used.
	DeviceNameHeader string

	// ConveyHeader is the name of the HTTP request header which contains the
	// base64-encoded JSON payload to forward with each outbound device request.
	// If not specified, DefaultConveyHeader is used.
	ConveyHeader string

	// HandshakeTimeout is the optional websocket handshake timeout.  If not supplied,
	// the internal gorilla default is used.
	HandshakeTimeout time.Duration

	// ReadBufferSize is the optional size of websocket read buffers.  If not supplied,
	// the internal gorilla default is used.
	ReadBufferSize int

	// WriteBufferSize is the optional size of websocket write buffers.  If not supplied,
	// the internal gorilla default is used.
	WriteBufferSize int

	// Subprotocols is the optional slice of websocket subprotocols to use.
	Subprotocols []string

	// InitialRegistrySize is the initial capacity of the internal map of devices.
	// If not supplied, DefaultInitialRegistrySize is used.
	InitialRegistrySize int

	// DeviceMessageQueueSize is the capacity of the channel which stores messages waiting
	// to be transmitted to a device.  If not supplied, DefaultDeviceMessageQueueSize is used.
	DeviceMessageQueueSize int

	// PingPeriod is the time between pings sent to each device
	PingPeriod time.Duration

	// IdleInterval is the length of time a device connection is allowed to be idle,
	// with no traffic coming from the device.  If not supplied, DefaultIdleInterval is used.
	IdleInterval time.Duration

	// WriteTimeout is the write timeout for each device's websocket.  If not supplied,
	// DefaultWriteTimeout is used.
	WriteTimeout time.Duration

	// MessageCallback is the callback used to receive messages from devices.  If nil,
	// an internal default function that simply logs messages is used.
	MessageCallback func(Interface, *wrp.Message)

	// Logger is the output sink for log messages.  If not supplied, log output
	// is sent to logging.DefaultLogger().
	Logger logging.Logger
}

func (o *Options) initialRegistrySize() int {
	if o.InitialRegistrySize > 0 {
		return o.InitialRegistrySize
	}

	return DefaultInitialRegistrySize
}

func (o *Options) pingPeriod() time.Duration {
	if o.PingPeriod > 0 {
		return o.PingPeriod
	}

	return DefaultPingPeriod
}

func (o *Options) subprotocols() []string {
	if len(o.Subprotocols) == 0 {
		return nil
	}

	subprotocols := make([]string, len(o.Subprotocols))
	copy(subprotocols, o.Subprotocols)
	return subprotocols
}

func (o *Options) deviceMessageQueueSize() int {
	if o.DeviceMessageQueueSize > 0 {
		return o.DeviceMessageQueueSize
	}

	return DefaultDeviceMessageQueueSize
}

func (o *Options) logger() logging.Logger {
	if o.Logger != nil {
		return o.Logger
	}

	return logging.DefaultLogger()
}

// Manager supplies a hub for connecting and disconnecting devices as well as
// an access point for obtaining device metadata.
type Manager interface {
	Connect(http.ResponseWriter, *http.Request) (Interface, error)
	Disconnect(ID)
}

func NewManager(o *Options) Manager {
	manager := &websocketManager{
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
	}

	if o.MessageCallback != nil {
		manager.messageCallback = o.MessageCallback
	} else {
		manager.messageCallback = manager.defaultMessageCallback
	}

	return manager
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

	messageCallback func(Interface, *wrp.Message)
}

func (wm *websocketManager) Connect(response http.ResponseWriter, request *http.Request) (Interface, error) {
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
	go device.readPump(wm.messageCallback)

	go func() {
		// both pumps will exit nearly at the same time.  remove the device
		// here, to avoid doing double work when the read pump exits.
		defer wm.removeOne(device)
		device.writePump(wm.pingPeriod)
	}()

	wm.add(device)
	return device, nil
}

// newDevice is an internal Factory Method for devices.  This method only
// handles the instantiation of a device.
func (wm *websocketManager) newDevice(id ID, convey *Convey, connection connection) *device {
	wm.logger.Debug("newDevice(%s, %v, %v)", id, convey, connection)

	return &device{
		id:           id,
		convey:       convey,
		connectedAt:  time.Now(),
		logger:       wm.logger,
		connection:   connection,
		messages:     make(chan *wrp.Message, wm.deviceMessageQueueSize),
		shutdown:     make(chan struct{}),
		idleInterval: wm.idleInterval,
		writeTimeout: wm.writeTimeout,
	}
}

// defaultMessageCallback is the default callback function used for messages
func (wm *websocketManager) defaultMessageCallback(device Interface, message *wrp.Message) {
	wm.logger.Debug("[%s]: %v", device.ID(), message)
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

func (wm *websocketManager) removeAll(key ID) []*device {
	defer wm.registryLock.Unlock()
	wm.registryLock.Lock()
	return wm.registry.removeAll(key)
}

func (wm *websocketManager) Disconnect(id ID) {
	wm.logger.Debug("Disconnect(%s)", id)
	for _, device := range wm.removeAll(id) {
		device.close(nil)
	}
}
