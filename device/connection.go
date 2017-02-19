package device

import (
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

const (
	transferBufferSize = 64
)

// Connection represents a websocket connection to a WebPA-compatible device.
// Connection implementations abstract the semantics of serverside WRP message
// handling and enforce policies like idleness.
type Connection interface {
	io.WriteCloser

	// Read returns the next binary message frame.  If this method returns an error,
	// this connection should be abandoned and closed.  This method is not safe
	// for concurrent invocation and must not be invoked concurrently with Write().
	//
	// Read will skip frames if they are not supported by the WRP protocol.  For example,
	// text frames are not supported and are skipped.  Anytime a frame is skipped, this
	// method returns a false flag and a nil error.
	Read(io.ReaderFrom) (bool, error)

	// Ping sends a ping message to the device.  This method may be invoked concurrently
	// with any other method of this interface, including Ping() itself.
	Ping([]byte) error

	// SetPongCallback registers the given function to be invoked whenever this connection
	// notices a pong from the device.  Note that this is not the same as a handler.  This callback
	// cannot return an error, and is invoked as part of the internal pong handler that
	// enforces the idle policy.  The pong callback can be nil, which simply reverts back
	// to the internal default handler.
	//
	// This method cannot be called concurrently with Write().
	SetPongCallback(func(string))

	// SendClose transmits a close frame to the device.  After this method is invoked,
	// the only method that should be invoked is Close()
	SendClose() error
}

// connection is the internal implementation of Connection
type connection struct {
	webSocket    *websocket.Conn
	idlePeriod   time.Duration
	writeTimeout time.Duration
}

func (c *connection) updateReadDeadline() error {
	return c.webSocket.SetReadDeadline(
		time.Now().Add(c.idlePeriod),
	)
}

func (c *connection) nextWriteDeadline() time.Time {
	var deadline time.Time
	if c.writeTimeout > 0 {
		deadline = time.Now().Add(c.writeTimeout)
	}

	return deadline
}

func (c *connection) updateWriteDeadline() error {
	if c.writeTimeout > 0 {
		return c.webSocket.SetWriteDeadline(
			time.Now().Add(c.writeTimeout),
		)
	}

	return nil
}

func (c *connection) defaultPongHandler(data string) error {
	return c.updateReadDeadline()
}

func (c *connection) pongHandler(callback func(string)) func(string) error {
	return func(data string) (err error) {
		err = c.updateReadDeadline()
		callback(data)
		return
	}
}

func (c *connection) SetPongCallback(callback func(string)) {
	if callback != nil {
		c.webSocket.SetPongHandler(c.pongHandler(callback))
	} else {
		c.webSocket.SetPongHandler(c.defaultPongHandler)
	}
}

func (c *connection) Read(target io.ReaderFrom) (frameRead bool, err error) {
	if err = c.updateReadDeadline(); err != nil {
		return
	}

	var (
		messageType int
		frame       io.Reader
	)

	if messageType, frame, err = c.webSocket.NextReader(); err != nil {
		return
	}

	if messageType != websocket.BinaryMessage {
		// allow the caller to take some action for skipped frames
		return
	}

	frameRead = true
	_, err = target.ReadFrom(frame)
	return
}

func (c *connection) Write(message []byte) (count int, err error) {
	if err = c.updateWriteDeadline(); err != nil {
		return
	}

	var frame io.WriteCloser
	if frame, err = c.webSocket.NextWriter(websocket.BinaryMessage); err != nil {
		return
	}

	defer frame.Close()
	return frame.Write(message)
}

func (c *connection) Close() error {
	return c.webSocket.Close()
}

func (c *connection) SendClose() error {
	return c.webSocket.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, "close"),
		c.nextWriteDeadline(),
	)
}

func (c *connection) Ping(data []byte) error {
	return c.webSocket.WriteControl(websocket.PingMessage, data, c.nextWriteDeadline())
}

// ConnectionFactory provides the instantiation logic for Connections.  This interface
// is appropriate for server-side connections that enforce various WebPA policies,
// such as idleness and a write timeout.
type ConnectionFactory interface {
	NewConnection(http.ResponseWriter, *http.Request, http.Header) (Connection, error)
}

// NewConnectionFactory produces a ConnectionFactory instance from a set of Options.
func NewConnectionFactory(o *Options) ConnectionFactory {
	return &connectionFactory{
		upgrader: websocket.Upgrader{
			HandshakeTimeout: o.handshakeTimeout(),
			ReadBufferSize:   o.readBufferSize(),
			WriteBufferSize:  o.writeBufferSize(),
			Subprotocols:     o.subprotocols(),
		},
		idlePeriod:   o.idlePeriod(),
		writeTimeout: o.writeTimeout(),
	}
}

// connectionFactory is the default ConnectionFactory implementation
type connectionFactory struct {
	upgrader     websocket.Upgrader
	idlePeriod   time.Duration
	writeTimeout time.Duration
}

func (cf *connectionFactory) NewConnection(response http.ResponseWriter, request *http.Request, responseHeader http.Header) (Connection, error) {
	webSocket, err := cf.upgrader.Upgrade(response, request, responseHeader)
	if err != nil {
		return nil, err
	}

	c := &connection{
		webSocket:    webSocket,
		idlePeriod:   cf.idlePeriod,
		writeTimeout: cf.writeTimeout,
	}

	// initialize the pong callback to the default, which
	// also registers the handler that enforces the idle policy
	c.SetPongCallback(nil)

	return c, nil
}

// Dialer is a WebPA dialer for websocket Connections
type Dialer interface {
	Dial(URL string, id ID, convey Convey, extra http.Header) (Connection, *http.Response, error)
}

// NewDialer constructs a WebPA Dialer using a set of Options and a gorilla Dialer.  Both
// parameters are optional.  If the gorilla Dialer is supplied, it is copied for use internally.
// If an Options is supplied, the appropriate settings will override any gorilla Dialer, e.g. ReadBufferSize.
func NewDialer(o *Options, d *websocket.Dialer) Dialer {
	dialer := &dialer{
		idlePeriod:   o.idlePeriod(),
		writeTimeout: o.writeTimeout(),
	}

	if d != nil {
		dialer.webSocketDialer = *d
	}

	// Options only override the dialer if supplied, and if no
	// dialer is specified always use the Options to establish WebPA settings
	if (d != nil && o != nil) || d == nil {
		dialer.webSocketDialer.HandshakeTimeout = o.handshakeTimeout()
		dialer.webSocketDialer.ReadBufferSize = o.readBufferSize()
		dialer.webSocketDialer.WriteBufferSize = o.writeBufferSize()
		dialer.webSocketDialer.Subprotocols = o.subprotocols()
	}

	dialer.deviceNameHeader = o.deviceNameHeader()
	dialer.conveyHeader = o.conveyHeader()
	return dialer
}

// dialer is the internal implementation of Dialer.  This implemention wraps a gorilla Dialer
type dialer struct {
	webSocketDialer  websocket.Dialer
	deviceNameHeader string
	conveyHeader     string
	idlePeriod       time.Duration
	writeTimeout     time.Duration
}

func (d *dialer) Dial(URL string, id ID, convey Convey, extra http.Header) (Connection, *http.Response, error) {
	requestHeader := make(http.Header, len(extra)+2)
	for key, value := range extra {
		requestHeader[key] = value
	}

	requestHeader.Set(d.deviceNameHeader, string(id))
	if len(convey) > 0 {
		encoded, err := EncodeConvey(convey, nil)
		if err != nil {
			return nil, nil, err
		}

		requestHeader.Set(d.conveyHeader, encoded)
	}

	webSocket, response, err := d.webSocketDialer.Dial(URL, requestHeader)
	if err != nil {
		return nil, response, err
	}

	c := &connection{
		webSocket:    webSocket,
		idlePeriod:   d.idlePeriod,
		writeTimeout: d.writeTimeout,
	}

	// initialize the pong callback to the default, which
	// also registers the handler that enforces the idle policy
	c.SetPongCallback(nil)

	return c, response, nil
}
