package device

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"sync"
	"time"
)

// Interface is the core type for this package.  It provides
// access to public device metadata.
type Interface interface {
	// ID returns the canonicalized identifer for this device.  Note that
	// this is NOT globally unique.  It is possible for multiple devices
	// with the same ID to be connected.  This typically occurs due to fraud,
	// but we don't want to turn away duped devices.
	ID() ID

	// Convey returns the payload to convey with each web-bound request
	Convey() *Convey

	// ConnectedAt returns the time at which this device connected to the system
	ConnectedAt() time.Time
}

// connection is the low-level interface that websocket connections must implement.
// gorilla's *websocket.Conn implements this interface.
type connection interface {
	io.Closer

	NextReader() (int, io.Reader, error)
	NextWriter(int) (io.WriteCloser, error)

	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
	SetPongHandler(func(string) error)

	WriteControl(int, []byte, time.Time) error
}

// device is the internal Interface implementation
type device struct {
	id          ID
	convey      *Convey
	connectedAt time.Time
	logger      logging.Logger
	closeOnce   sync.Once

	connection   connection
	idlePeriod   time.Duration
	writeTimeout time.Duration
	messages     chan *wrp.Message
	shutdown     chan struct{}

	disconnectListener DisconnectListener
}

func (d *device) ID() ID {
	return d.id
}

func (d *device) Convey() *Convey {
	return d.convey
}

func (d *device) ConnectedAt() time.Time {
	return d.connectedAt
}

func (d *device) sendMessage(message *wrp.Message) {
	d.messages <- message
}

func (d *device) writeCloseFrame(closeCode int, text string) {
	closeMessage := websocket.FormatCloseMessage(closeCode, text)
	closeDeadline := time.Now().Add(d.writeTimeout)
	if err := d.connection.WriteControl(websocket.CloseMessage, closeMessage, closeDeadline); err != nil {
		d.logger.Error("[%s]: Error while writing close frame: %s", d.id, err)
	}
}

// ping sends a ping to the device
func (d *device) ping(message []byte) error {
	pingDeadline := time.Now().Add(d.writeTimeout)
	return d.connection.WriteControl(websocket.PingMessage, message, pingDeadline)
}

func (d *device) updateReadDeadline() error {
	return d.connection.SetReadDeadline(
		time.Now().Add(d.idlePeriod),
	)
}

// close handles sending a CloseMessage and shutting down the underlying socket.  If supplied,
// preClose is invoked prior to anything else.
//
// This method is idempotent.  It executes within a sync.Once, and is thus safe to call
// multiple times.  Only the first call to close will invoke preClose.  Subsequent invocations
// ignore the preClose function.
func (d *device) close(cause error, preClose func(*device)) {
	d.closeOnce.Do(func() {
		defer d.connection.Close()
		defer close(d.messages)
		defer close(d.shutdown)
		defer d.disconnectListener.OnDisconnect(d)

		if preClose != nil {
			preClose(d)
		}

		if cause == nil {
			// when there's no error, e.g. when a device is disconnected through the manager,
			// then send a close frame
			d.writeCloseFrame(websocket.CloseNormalClosure, string(d.id))
		} else {
			switch cause.(type) {
			case *websocket.CloseError:
				// the client sent a close frame to us in this case, so no need for us to send one
			case net.Error:
				// when an I/O error occurs, don't trust that the connection can transmit a message
			default:
				// any other error is assumed to be an internal server error
				d.writeCloseFrame(websocket.CloseInternalServerErr, cause.Error())
			}
		}
	})
}

// readPump is a goroutine that services messages on the device's connection.
// The MessageListener and the preClose function appropriate for a read-side termination
// of the connection are passed to this method, since they are specific to read operations.
func (d *device) readPump(messageListener MessageListener, readPreClose func(*device)) {
	var (
		err         error
		messageType int
		frame       io.Reader
	)

	defer d.close(err, readPreClose)
	decoder := wrp.NewDecoder(nil, wrp.Msgpack)

	for {
		if err = d.updateReadDeadline(); err != nil {
			return
		}

		if messageType, frame, err = d.connection.NextReader(); err != nil {
			return
		}

		if messageType != websocket.BinaryMessage {
			// Skip anything that's not a binary message
			// TODO: Log this
			continue
		}

		decoder.Reset(frame)
		message := new(wrp.Message)
		if err = decoder.Decode(message); err != nil {
			return
		}

		messageListener.OnMessage(d, message)
	}
}

// writePump is a goroutine that services a message queue for a device.  This goroutine
// also pings the device on the supplied period.  The pong listener and write-specifiec closure
// are also passed to this method, as they are specific to the write-side of the device connection.
func (d *device) writePump(pingPeriod time.Duration, pongListener PongListener, writePreClose func(*device)) {
	var (
		err   error
		frame io.WriteCloser
	)

	defer d.close(err, writePreClose)
	encoder := wrp.NewEncoder(nil, wrp.Msgpack)

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	d.connection.SetPongHandler(func(data string) error {
		defer pongListener.OnPong(d, data)
		return d.updateReadDeadline()
	})

	// identify who we're pinging on the wire, to make things easier to debug
	pingMessage := []byte(fmt.Sprintf("ping[%s]", d.id))

	for {
		select {
		case <-d.shutdown:
			return
		case message := <-d.messages:
			if frame, err = d.connection.NextWriter(websocket.BinaryMessage); err != nil {
				return
			}

			encoder.Reset(frame)
			if err = encoder.Encode(message); err != nil {
				// no need to close the frame if err != nil,
				// since the defer will handle cleanup
				return
			}

			if err = frame.Close(); err != nil {
				return
			}
		case <-pingTicker.C:
			if err = d.ping(pingMessage); err != nil {
				return
			}
		}
	}
}
