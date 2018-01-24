package device

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

// websocketDialer defines the behavior expected of a low-level dialer for websockets.  Gorilla's
// websocket.Dialer implements this interface.
type websocketDialer interface {
	Dial(string, http.Header) (*websocket.Conn, *http.Response, error)
}

// defaultWebsocketDialer is the internal default websocket dialer for use
// when no websocket dialer is specified, such as in DialerOptions
var defaultWebsocketDialer websocketDialer = &websocket.Dialer{}

// defaultDialer is the default device Dialer
var defaultDialer Dialer = &dialer{
	deviceHeader: DeviceNameHeader,
	wd:           defaultWebsocketDialer,
}

// DefaultDialer returns a useful default device Dialer
func DefaultDialer() Dialer { return defaultDialer }

// Dialer is a device-specific dialer for device websocket connections.  This interface has a similar
// signature and usage pattern as gorilla's websocket.Dialer.
type Dialer interface {
	// DialDevice attempts to connect to the given device.  If supplied, the extra headers are passed
	// along in the Dial call.  However, the extra http.Header object is not modified by this method.
	DialDevice(deviceName, url string, extra http.Header) (*websocket.Conn, *http.Response, error)
}

// DialerOptions represents the set of options available when creating a custom device Dialer
type DialerOptions struct {
	// DeviceHeader is the HTTP header which carries the device name.
	// If unset, DeviceNameHeader is used
	DeviceHeader string

	// WSDialer is the low-level websocket dialer to use.  If unset, an internal default gorilla dialer is used.
	WSDialer websocketDialer
}

// NewDialer produces a device dialer using the supplied set of options
func NewDialer(o DialerOptions) Dialer {
	d := &dialer{
		deviceHeader: o.DeviceHeader,
		wd:           o.WSDialer,
	}

	if len(d.deviceHeader) == 0 {
		d.deviceHeader = DeviceNameHeader
	}

	if d.wd == nil {
		d.wd = defaultWebsocketDialer
	}

	return d
}

// dialer is the internal device Dialer implementation
type dialer struct {
	deviceHeader string
	wd           websocketDialer
}

func (d *dialer) DialDevice(deviceName, url string, extra http.Header) (*websocket.Conn, *http.Response, error) {
	requestHeader := make(http.Header, 1+len(extra))
	for name, values := range extra {
		for _, value := range values {
			requestHeader.Add(name, value)
		}
	}

	requestHeader.Set(d.deviceHeader, deviceName)
	return d.wd.Dial(url, requestHeader)
}

// MustDialDevice panics if the dial operation fails.  Mostly useful for test code.
func MustDialDevice(d Dialer, deviceName, url string, extra http.Header) (*websocket.Conn, *http.Response) {
	c, r, err := d.DialDevice(deviceName, url, extra)
	if err != nil {
		panic(fmt.Errorf("Dialing device %s at %s failed: %s", deviceName, url, err))
	}

	return c, r
}
