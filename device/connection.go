package device

import (
	"github.com/Comcast/webpa-common/wrp"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"time"
)

// Connection represents a websocket connection to a WebPA-compatible device.
// Connection implementations abstract the semantics of serverside WRP message
// handling and enforce policies like idleness.
type Connection interface {
	io.Closer

	// Read returns the next WRP message frame.  If this method returns an error,
	// this connection should be abandoned and closed.  This method is not safe
	// for concurrent invocation and must not be invoked concurrently with Write().
	//
	// Read may skip frames if they are not supported by the WRP protocol.  For example,
	// text frames are not supported and are skipped.  Anytime a frame is skipped, this
	// method returns a nil message with a nil error.
	Read() (*wrp.Message, error)

	// Write sends a WRP frame to the device.  If this method returns an error,
	// this connection should be abandoned and closed.  This method is not safe
	// for concurrent invocation and must not be invoked concurrently with Read().
	Write(*wrp.Message) error

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
	decoder      wrp.Decoder
	encoder      wrp.Encoder
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

func (c *connection) Read() (*wrp.Message, error) {
	for {
		if err := c.updateReadDeadline(); err != nil {
			return nil, err
		}

		messageType, frame, err := c.webSocket.NextReader()
		if err != nil {
			return nil, err
		}

		if messageType != websocket.BinaryMessage {
			// allow the caller to take some action for skipped frames
			return nil, nil
		}

		c.decoder.Reset(frame)
		message := new(wrp.Message)
		if err := c.decoder.Decode(message); err != nil {
			return nil, err
		}

		return message, nil
	}
}

func (c *connection) Write(message *wrp.Message) error {
	if err := c.updateWriteDeadline(); err != nil {
		return err
	}

	frame, err := c.webSocket.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}

	c.encoder.Reset(frame)
	if err := c.encoder.Encode(message); err != nil {
		return err
	}

	if err := frame.Close(); err != nil {
		return err
	}

	return nil
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

// ConnectionFactory provides the instantiation logic for Connections
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
		decoder:      wrp.NewDecoder(nil, wrp.Msgpack),
		encoder:      wrp.NewEncoder(nil, wrp.Msgpack),
	}

	// initialize the pong callback to the default, which
	// also registers the handler that enforces the idle policy
	c.SetPongCallback(nil)

	return c, nil
}
