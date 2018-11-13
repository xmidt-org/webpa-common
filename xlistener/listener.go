package xlistener

import (
	"crypto/tls"
	"net"
	"strconv"
	"sync"
	"syscall"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/metrics/discard"
)

var (
	// netListen is the factory function for creating a net.Listener.  Defaults to net.Listen.  Only tests would change this variable.
	netListen = net.Listen

	// tlsListen is the factory function for creating a tls.Listener.  Defaults to tls.Listen.  Only tests would change this variable.
	tlsListen = tls.Listen
)

// Options defines the available options for configuring a listener
type Options struct {
	// Logger is the go-kit logger to use for output.  If unset, logging.DefaultLogger() is used.
	Logger log.Logger

	// MaxConnections is the maximum number of active connections the listener will permit.  If this
	// value is not positive, there is no limit to the number of connections.
	MaxConnections int

	// Rejected is is incremented each time the listener rejects a connection.  If unset, a go-kit discard Counter is used.
	Rejected xmetrics.Adder

	// Active is updated to reflect the current number of active connections.  If unset, a go-kit discard Gauge is used.
	Active xmetrics.Adder

	// Network is the network to listen on.  This value is only used if Next is unset.  Defaults to "tcp" if unset.
	Network string

	// Address is the address to listen on.  This value is only used if Next is unset.  Defaults to ":http" if unset.
	Address string

	// Next is the net.Listener to decorate.  If this field is set, Network and Address are ignored.
	Next net.Listener

	Config *tls.Config
}

// New constructs a new net.Listener using a set of options.
//
// If Next is set, that listener is decorated with connection limiting and other options specfied in Options.
// Otherwise, a new net.Listener is created, and that new listener is decorated.  Note that in the case
// where this function creates a new net.Listener, that listener will be occupying a port and should be cleaned
// up via Close() if higher level errors occur.
func New(o Options) (net.Listener, error) {
	if o.Logger == nil {
		o.Logger = logging.DefaultLogger()
	}

	var semaphore chan struct{}
	if o.MaxConnections > 0 {
		semaphore = make(chan struct{}, o.MaxConnections)
	}

	if o.Rejected == nil {
		o.Rejected = discard.NewCounter()
	}

	if o.Active == nil {
		o.Active = discard.NewGauge()
	}

	next := o.Next
	if next == nil {
		if len(o.Network) == 0 {
			o.Network = "tcp"
		}

		if len(o.Address) == 0 {
			o.Address = ":http"
		}

		var err error
		if o.Config != nil {
			next, err = tlsListen(o.Network, o.Address, o.Config)
		} else {
			next, err = netListen(o.Network, o.Address)
		}
		if err != nil {
			return nil, err
		}
	}

	return &listener{
		Listener:  next,
		logger:    log.With(o.Logger, "listenNetwork", next.Addr().Network(), "listenAddress", next.Addr().String()),
		semaphore: semaphore,
		rejected:  xmetrics.NewIncrementer(o.Rejected),
		active:    o.Active,
	}, nil
}

// listener decorates a net.Listener with metrics and optional maximum connection enforcement
type listener struct {
	net.Listener
	logger    log.Logger
	semaphore chan struct{}
	rejected  xmetrics.Incrementer
	active    xmetrics.Adder
}

// acquire attempts to obtain a semaphore resource.  If the semaphore has not been set (i.e. no maximum connections),
// this method immediately returns true.  Otherwise, the semaphore must be immediately acquired or this method returns false.
// In all cases, the active connections gauge is updated if appropriate.
func (l *listener) acquire() bool {
	if l.semaphore == nil {
		l.active.Add(1.0)
		return true
	}

	select {
	case l.semaphore <- struct{}{}:
		l.active.Add(1.0)
		return true
	default:
		return false
	}
}

// release returns a semaphore resource to the pool, if set.  This method also decrements the active connection gauge.
func (l *listener) release() {
	l.active.Add(-1.0)
	if l.semaphore != nil {
		<-l.semaphore
	}
}

// Accept invokes the delegate net.Listener's Accept method, then attempts to acquire the semaphore.
// If the semaphore was set and could not be acquired, the accepted connection is immediately closed.
func (l *listener) Accept() (net.Conn, error) {
	for {
		c, err := l.Listener.Accept()
		if err != nil {
			sysValue := ""
			if errno, ok := err.(syscall.Errno); ok {
				sysValue = "0x" + strconv.FormatInt(int64(errno), 16)
			}

			l.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "failed to accept connection", logging.ErrorKey(), err, "sysValue", sysValue)
			if err == syscall.ENFILE {
				l.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "ENFILE received.  translating to EMFILE")
				return nil, syscall.EMFILE
			}

			return nil, err
		}

		if !l.acquire() {
			l.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "rejected connection", "remoteAddress", c.RemoteAddr().String())
			l.rejected.Inc()
			c.Close()
			continue
		}

		l.logger.Log(level.Key(), level.DebugValue(), logging.MessageKey(), "accepted connection", "remoteAddress", c.RemoteAddr().String())
		return &conn{Conn: c, release: l.release}, nil
	}
}

// conn is a decorated net.Conn that supplies feedback to a listener when the connection is closed.
type conn struct {
	net.Conn
	releaseOnce sync.Once
	release     func()
}

// Close closes the decorated connection and invokes release on the listener that created it.  The release
// operation is idempotent.
func (c *conn) Close() error {
	err := c.Conn.Close()
	c.releaseOnce.Do(c.release)
	return err
}
