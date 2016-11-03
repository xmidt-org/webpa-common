package device

import (
	"github.com/Comcast/webpa-common/wrp"
)

// nullListener implements all device listeners and provides "null" behavior.
// This type simply ignores all events.
type nullListener int

func (n nullListener) String() string                    { return "nullListener" }
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

// MessageListenerFunc is a function type that implements MessageListener
type MessageListenerFunc func(Interface, *wrp.Message)

func (f MessageListenerFunc) OnMessage(device Interface, message *wrp.Message) {
	f(device, message)
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

// ConnectListenerFunc is a function type that implements ConnectListener
type ConnectListenerFunc func(Interface)

func (f ConnectListenerFunc) OnConnect(device Interface) {
	f(device)
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

// DisconnectListenerFunc is a function type that implements DisconnectListener
type DisconnectListenerFunc func(Interface)

func (f DisconnectListenerFunc) OnDisconnect(device Interface) {
	f(device)
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

// PongListenerFunc is a function type that implements PongListener
type PongListenerFunc func(Interface, string)

func (f PongListenerFunc) OnPong(device Interface, data string) {
	f(device, data)
}

// PongListeners is a slice type that implements PongListener.
type PongListeners []PongListener

func (m PongListeners) OnPong(device Interface, data string) {
	for _, l := range m {
		l.OnPong(device, data)
	}
}
