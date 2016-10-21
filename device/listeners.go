package device

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
)

// MessageListener represents a sink for device messages
type MessageListener interface {
	OnMessage(Interface, *wrp.Message)
}

// ConnectListener instances are notified whenever a device successfully
// connects to a manager
type ConnectListener interface {
	OnConnect(Interface)
}

// DisconnectListener instances are notified whenever a device disconnects
// from a manager
type DisconnectListener interface {
	OnDisconnect(Interface)
}

// PongListener instances are notified each time a pong to a device is received
type PongListener interface {
	OnPong(Interface, string)
}

// LoggingListener implements all the device listeners in this package and
// simply logs all events.  Connect/Disconnect are logged to INFO, while
// messages and pongs are logged to DEBUG.
type LoggingListener struct {
	logging.Logger
}

func (l LoggingListener) OnMessage(device Interface, message *wrp.Message) {
	l.Debug("[%s]: message: %v", device.ID(), message)
}

func (l LoggingListener) OnConnect(device Interface) {
	l.Info("[%s]: connect", device.ID())
}

func (l LoggingListener) OnDisconnect(device Interface) {
	l.Info("[%s]: disconnect", device.ID())
}

func (l LoggingListener) OnPong(device Interface, data string) {
	l.Debug("[%s]: pong: %s", device.ID(), data)
}

// Listeners is an aggregate data structure for all device-related listeners.
// This type allows arbitrary numbers of listeners to be notified of events,
// and it implements those listener interfaces directly.
type Listeners struct {
	messageListeners    []MessageListener
	connectListeners    []ConnectListener
	disconnectListeners []DisconnectListener
	pongListeners       []PongListener
}

func (l *Listeners) AddMessageListeners(listeners ...MessageListener) *Listeners {
	l.messageListeners = append(l.messageListeners, listeners...)
	return l
}

func (l *Listeners) AddConnectListeners(listeners ...ConnectListener) *Listeners {
	l.connectListeners = append(l.connectListeners, listeners...)
	return l
}

func (l *Listeners) AddDisconnectListeners(listeners ...DisconnectListener) *Listeners {
	l.disconnectListeners = append(l.disconnectListeners, listeners...)
	return l
}

func (l *Listeners) AddPongListeners(listeners ...PongListener) *Listeners {
	l.pongListeners = append(l.pongListeners, listeners...)
	return l
}

func (l *Listeners) Add(listeners ...interface{}) *Listeners {
	for _, listener := range listeners {
		if messageListener, ok := listener.(MessageListener); ok {
			l.AddMessageListeners(messageListener)
		}

		if connectListener, ok := listener.(ConnectListener); ok {
			l.AddConnectListeners(connectListener)
		}

		if disconnectListener, ok := listener.(DisconnectListener); ok {
			l.AddDisconnectListeners(disconnectListener)
		}

		if pongListener, ok := listener.(PongListener); ok {
			l.AddPongListeners(pongListener)
		}
	}

	return l
}

func (l *Listeners) OnMessage(device Interface, message *wrp.Message) {
	for _, listener := range l.messageListeners {
		listener.OnMessage(device, message)
	}
}

func (l *Listeners) OnConnect(device Interface) {
	for _, listener := range l.connectListeners {
		listener.OnConnect(device)
	}
}

func (l *Listeners) OnDisconnect(device Interface) {
	for _, listener := range l.disconnectListeners {
		listener.OnDisconnect(device)
	}
}

func (l *Listeners) OnPong(device Interface, data string) {
	for _, listener := range l.pongListeners {
		listener.OnPong(device, data)
	}
}

// Clone returns a distinct copy of this instance.  Internally, the device Manager
// clones the Listeners so that it has an instance safe for concurrent access.
func (l *Listeners) Clone() *Listeners {
	return &Listeners{
		messageListeners:    append([]MessageListener{}, l.messageListeners...),
		connectListeners:    append([]ConnectListener{}, l.connectListeners...),
		disconnectListeners: append([]DisconnectListener{}, l.disconnectListeners...),
		pongListeners:       append([]PongListener{}, l.pongListeners...),
	}
}
