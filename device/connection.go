package device

import (
	"io"
	"time"

	"github.com/go-kit/kit/metrics"
	"github.com/gorilla/websocket"
)

// Connection describes the set of behaviors for device connections used by this package.
// Gorilla's websocket.Conn type implements this interface.
type Connection interface {
	io.Closer

	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	WritePreparedMessage(*websocket.PreparedMessage) error

	SetPongHandler(func(string) error)
	SetWriteDeadline(time.Time) error
	SetReadDeadline(time.Time) error
}

// Pinger creates a ping closure for the given connection.  Internally, a prepared message is created using the
// supplied data, and the given counter is incremented for each successful update of the write deadline.
func Pinger(c Connection, pc metrics.Counter, data []byte, nextTimeout func() time.Time) (func() error, error) {
	pm, err := websocket.NewPreparedMessage(websocket.PingMessage, data)
	if err != nil {
		return nil, err
	}

	return func() error {
		if err := c.SetWriteDeadline(nextTimeout()); err != nil {
			return err
		}

		pc.Add(1.0)
		return c.WritePreparedMessage(pm)
	}, nil
}

// SetPongHandler establishes an instrumented pong handler for the given connection that enforces
// the given read timeout.
func SetPongHandler(c Connection, pc metrics.Counter, nextTimeout func() time.Time) {
	c.SetPongHandler(func(_ string) error {
		pc.Add(1.0)
		return c.SetReadDeadline(nextTimeout())
	})
}

type instrumentedConnection struct {
	Connection
	measures   Measures
	statistics Statistics
}
