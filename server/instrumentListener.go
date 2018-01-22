package server

import (
	"net"
	"sync"

	"github.com/Comcast/webpa-common/logging"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics"
)

// InstrumentListener returns a net.Listener which tracks the number of current connections.  Any
// errors during Accept or Close are logged via the supplied logger.
func InstrumentListener(logger log.Logger, counter metrics.Counter, l net.Listener) net.Listener {
	return &instrumentedListener{l, logger, counter}
}

type instrumentedListener struct {
	net.Listener
	logger  log.Logger
	counter metrics.Counter
}

func (l *instrumentedListener) closeConn() {
	l.counter.Add(-1.0)
}

func (l *instrumentedListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		l.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to accept connection", logging.ErrorKey(), err)
		return nil, err
	}

	l.counter.Add(1.0)
	return &instrumentedConn{Conn: c, closeConn: l.closeConn}, nil
}

func (l *instrumentedListener) Close() error {
	err := l.Listener.Close()
	if err != nil {
		l.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "error while closing net.Listener", logging.ErrorKey(), err)
	}

	return err
}

type instrumentedConn struct {
	net.Conn
	closeOnce sync.Once
	closeConn func()
}

func (ic *instrumentedConn) Close() error {
	err := ic.Conn.Close()
	ic.closeOnce.Do(ic.closeConn)
	return err
}
