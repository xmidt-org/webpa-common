package device

import (
	"github.com/Comcast/webpa-common/wrp"
)

// nullListener implements all device listeners and provides "null" behavior.
// This type simply ignores all events.
type nullListener int

func (n nullListener) OnMessage(Interface, *wrp.Message) {}
func (n nullListener) OnConnect(Interface)               {}
func (n nullListener) OnDisconnect(Interface)            {}
func (n nullListener) OnPong(Interface, string)          {}

// defaultListener serves as a "null" value for any device listener.
var defaultListener = nullListener(0)

// MessageListener represents a sink for device messages
type MessageListener interface {
	OnMessage(Interface, *wrp.Message)
}

// MessageListeners is a slice type that implements MessageListener.
type MessageListeners []MessageListener

func (m MessageListeners) OnMessage(device Interface, message *wrp.Message) {
	for _, l := range m {
		l.OnMessage(device, message)
	}
}

// ConnectListener instances are notified whenever a device successfully
// connects to a manager
type ConnectListener interface {
	OnConnect(Interface)
}

// ConnectListeners is a slice type that implements ConnectListener.
type ConnectListeners []ConnectListener

func (m ConnectListeners) OnConnect(device Interface) {
	for _, l := range m {
		l.OnConnect(device)
	}
}

// DisconnectListener instances are notified whenever a device disconnects
// from a manager
type DisconnectListener interface {
	OnDisconnect(Interface)
}

// DisconnectListeners is a slice type that implements DisconnectListener.
type DisconnectListeners []DisconnectListener

func (m DisconnectListeners) OnDisconnect(device Interface) {
	for _, l := range m {
		l.OnDisconnect(device)
	}
}

// PongListener instances are notified each time a pong to a device is received
type PongListener interface {
	OnPong(Interface, string)
}

// PongListeners is a slice type that implements PongListener.
type PongListeners []PongListener

func (m PongListeners) OnPong(device Interface, data string) {
	for _, l := range m {
		l.OnPong(device, data)
	}
}
