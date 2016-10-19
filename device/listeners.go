package device

import (
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
	OnPong(Interface)
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

func (l *Listeners) AddMessageListeners(listeners ...MessageListener) {
	l.messageListeners = append(l.messageListeners, listeners...)
}

func (l *Listeners) AddConnectListeners(listeners ...ConnectListener) {
	l.connectListeners = append(l.connectListeners, listeners...)
}

func (l *Listeners) AddDisconnectListeners(listeners ...DisconnectListener) {
	l.disconnectListeners = append(l.disconnectListeners, listeners...)
}

func (l *Listeners) AddPongListeners(listeners ...PongListener) {
	l.pongListeners = append(l.pongListeners, listeners...)
}

func (l *Listeners) Add(listeners ...interface{}) {
	for _, listener := range listeners {
		if messageListener, ok := listener.(MessageListener); ok {
			l.messageListeners = append(l.messageListeners, messageListener)
		}

		if connectListener, ok := listener.(ConnectListener); ok {
			l.connectListeners = append(l.connectListeners, connectListener)
		}

		if disconnectListener, ok := listener.(DisconnectListener); ok {
			l.disconnectListeners = append(l.disconnectListeners, disconnectListener)
		}

		if pongListener, ok := listener.(PongListener); ok {
			l.pongListeners = append(l.pongListeners, pongListener)
		}
	}
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

func (l *Listeners) OnPong(device Interface) {
	for _, listener := range l.pongListeners {
		listener.OnPong(device)
	}
}
