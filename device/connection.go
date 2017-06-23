package device

import (
	"io"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	transferBufferSize = 64
)

// Connection represents a websocket connection to a WebPA-compatible device.
// Connection implementations abstract the semantics of serverside WRP message
// handling and enforce policies like idleness.
//
// Connection implements both io.Writer and io.Closer, making it convenient for
// direct encoding via wrp.Encoder.  However, since each websocket frame is a separate
// byte sequence, this type does not implement io.Reader.  Rather, the Read method
// transfers the next frame to an io.ReaderFrom.
type Connection interface {
	io.WriteCloser

	// NextReader returns the next binary message frame.  If this method returns an error,
	// this connection should be abandoned and closed.  This method is not safe
	// for concurrent invocation and must not be invoked concurrently with Write.
	//
	// NextReader will skip frames if they are not supported by the WRP protocol.  For example,
	// text frames are not supported and are skipped.  Anytime a frame is skipped, this
	// method returns a nil reader and a nil error.
	NextReader() (io.Reader, error)

	// Read transfers the next binary frame to the given ReaderFrom instance.  If this method
	// returns an error, this connection should be abandoned and closed.  This method is not safe
	// for concurrent invocation and must not be invoked concurrently with Write.
	//
	// As with NextReader, this method skips frames that are not supported by the WRP protocol.
	// The first boolean return value indicates whether a frame was skipped.  If true, the frame's
	// contents were transferred to the target ReaderFrom.  If false, the error will always be nil
	// and no frame will have been read.
	Read(io.ReaderFrom) (bool, error)

	// NextWriter returns a WriteCloser which can be used to construct the next binary frame.
	// It's semantics are equivalent to the gorilla websocket's method of the same name.
	NextWriter() (io.WriteCloser, error)

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

func (c *connection) NextReader() (frame io.Reader, err error) {
	if err = c.updateReadDeadline(); err != nil {
		return
	}

	var messageType int
	if messageType, frame, err = c.webSocket.NextReader(); err != nil {
		return
	} else if messageType != websocket.BinaryMessage {
		// skip this frame, and allow the caller to take some action
		frame = nil
	}

	return
}

func (c *connection) Read(target io.ReaderFrom) (frameRead bool, err error) {
	var frame io.Reader
	frame, err = c.NextReader()
	frameRead = (frame != nil)
	if err == nil {
		_, err = target.ReadFrom(frame)
	}

	return
}

func (c *connection) NextWriter() (io.WriteCloser, error) {
	if err := c.updateWriteDeadline(); err != nil {
		return nil, err
	}

	return c.webSocket.NextWriter(websocket.BinaryMessage)
}

func (c *connection) Write(message []byte) (count int, err error) {
	var frame io.WriteCloser
	if frame, err = c.NextWriter(); err != nil {
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

	return dialer
}

// dialer is the internal implementation of Dialer.  This implemention wraps a gorilla Dialer
type dialer struct {
	webSocketDialer websocket.Dialer
	idlePeriod      time.Duration
	writeTimeout    time.Duration
}

func (d *dialer) Dial(URL string, id ID, convey Convey, extra http.Header) (Connection, *http.Response, error) {
	requestHeader := make(http.Header, len(extra)+2)
	for key, value := range extra {
		requestHeader[key] = value
	}

	requestHeader.Set(DeviceNameHeader, string(id))
	if len(convey) > 0 {
		encoded, err := EncodeConvey(convey, nil)
		if err != nil {
			return nil, nil, err
		}

		requestHeader.Set(ConveyHeader, encoded)
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
