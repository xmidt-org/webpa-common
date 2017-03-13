package device

import (
	"github.com/Comcast/webpa-common/wrp"
)

func defaultMessageReceivedListener(Interface, *wrp.Message, []byte)      {}
func defaultMessageFailedListener(Interface, *wrp.Message, []byte, error) {}
func defaultConnectListener(Interface)                                    {}
func defaultDisconnectListener(Interface)                                 {}
func defaultPongListener(Interface, string)                               {}

// MessageReceivedListener represents a sink for device messages.
//
// IMPORTANT: the supplied message and encoded bytes will be reused across multiple
// inbound messages from any given device.  Implementations should make copies of the message
// and the encoded bytes if any modifications are required.
type MessageReceivedListener func(Interface, *wrp.Message, []byte)

// MessageReceivedListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func MessageReceivedListeners(listeners ...MessageReceivedListener) MessageReceivedListener {
	if len(listeners) > 0 {
		return func(device Interface, message *wrp.Message, encoded []byte) {
			for _, l := range listeners {
				l(device, message, encoded)
			}
		}
	}

	return defaultMessageReceivedListener
}

// MessageFailedListener represents a sink for failed messages.
//
// IMPORTANT: the supplied message and encoded bytes will be reused across multiple
// inbound messages from any given device.  Implementations should make copies of the message
// and the encoded bytes if any modifications are required.
type MessageFailedListener func(Interface, *wrp.Message, []byte, error)

// MessageFailedListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func MessageFailedListeners(listeners ...MessageFailedListener) MessageFailedListener {
	if len(listeners) > 0 {
		return func(device Interface, message *wrp.Message, encoded []byte, err error) {
			for _, l := range listeners {
				l(device, message, encoded, err)
			}
		}
	}

	return defaultMessageFailedListener
}

// ConnectListener is a function which receives notifications when devices
// successfully connect to the system.
type ConnectListener func(Interface)

// ConnectListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func ConnectListeners(listeners ...ConnectListener) ConnectListener {
	if len(listeners) > 0 {
		return func(device Interface) {
			for _, l := range listeners {
				l(device)
			}
		}
	}

	return defaultConnectListener
}

// DisconnectListener is a function which receives notifications when devices
// disconnect (or, are disconnected) from the system
type DisconnectListener func(Interface)

// DisconnectListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func DisconnectListeners(listeners ...DisconnectListener) DisconnectListener {
	if len(listeners) > 0 {
		return func(device Interface) {
			for _, l := range listeners {
				l(device)
			}
		}
	}

	return defaultDisconnectListener
}

// PongListener is a function which receives notifications when devices
// disconnect (or, are disconnected) from the system
type PongListener func(Interface, string)

// PongListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func PongListeners(listeners ...PongListener) PongListener {
	if len(listeners) > 0 {
		return func(device Interface, data string) {
			for _, l := range listeners {
				l(device, data)
			}
		}
	}

	return defaultPongListener
}

// Listeners contains a set of device listeners
type Listeners struct {
	MessageReceived MessageReceivedListener
	MessageFailed   MessageFailedListener
	Connect         ConnectListener
	Disconnect      DisconnectListener
	Pong            PongListener
}

// EnsureDefaults sets any nil listener to its default.  This method ensures
// that no nil listeners are present, but preserves any custom listeners that are set.
func (l *Listeners) EnsureDefaults() {
	if l.MessageReceived == nil {
		l.MessageReceived = defaultMessageReceivedListener
	}

	if l.MessageFailed == nil {
		l.MessageFailed = defaultMessageFailedListener
	}

	if l.Connect == nil {
		l.Connect = defaultConnectListener
	}

	if l.Disconnect == nil {
		l.Disconnect = defaultDisconnectListener
	}

	if l.Pong == nil {
		l.Pong = defaultPongListener
	}
}
