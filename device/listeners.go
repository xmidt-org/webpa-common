package device

import (
	"github.com/Comcast/webpa-common/wrp"
)

func defaultMessageListener(Interface, []byte, *wrp.Message) {}
func defaultConnectListener(Interface)                       {}
func defaultDisconnectListener(Interface)                    {}
func defaultPongListener(Interface, string)                  {}

// MessageListener represents a sink for device messages
type MessageListener func(Interface, []byte, *wrp.Message)

// MessageListeners aggregates multiple listeners into one.  If this
// method is passed zero (0) listeners, an internal default is used instead.
func MessageListeners(listeners ...MessageListener) MessageListener {
	if len(listeners) > 0 {
		return func(device Interface, raw []byte, message *wrp.Message) {
			for _, l := range listeners {
				l(device, raw, message)
			}
		}
	}

	return defaultMessageListener
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
