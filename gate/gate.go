package gate

import (
	"sync/atomic"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/discard"
)

const (
	gateOpen uint32 = iota
	gateClosed
)

// Interface represents a concurrent condition indicating whether HTTP traffic should be allowed.
type Interface interface {
	// Raise opens this gate.  Any handlers decorated via this gate will begin to receive traffic.
	// By default, gates are initially open.  Use WithInitiallyClosed to create a gate in the closed state.
	Raise()

	// Lower closes this gate.  Any HTTP traffic that would otherwise go to a decorated handler will instead
	// receive the configured status code, text, and headers.
	Lower()

	// IsOpen tests if this gate is open
	IsOpen() bool
}

// Option is a configuration option for a gate Interface
type Option func(*gate)

func WithInitiallyClosed() Option {
	return func(g *gate) {
		g.state = gateClosed
	}
}

func WithClosedGauge(gauge xmetrics.Setter) Option {
	return func(g *gate) {
		if gauge != nil {
			g.closedGauge = gauge
		} else {
			g.closedGauge = discard.NewGauge()
		}
	}
}

// New constructs a gate Interface with zero or more options.  By default, the returned
// gate is initially open and has a closed gauge that simply discards all metrics.
func New(options ...Option) Interface {
	g := &gate{
		state:       gateOpen,
		closedGauge: discard.NewGauge(),
	}

	for _, o := range options {
		o(g)
	}

	if g.state == gateOpen {
		g.closedGauge.Set(0.0)
	} else {
		g.closedGauge.Set(1.0)
	}

	return g
}

// gate is the internal Interface implementation
type gate struct {
	state       uint32
	closedGauge xmetrics.Setter
}

func (g *gate) Raise() {
	if atomic.CompareAndSwapUint32(&g.state, gateClosed, gateOpen) {
		g.closedGauge.Set(0.0)
	}
}

func (g *gate) Lower() {
	if atomic.CompareAndSwapUint32(&g.state, gateOpen, gateClosed) {
		g.closedGauge.Set(1.0)
	}
}

func (g *gate) IsOpen() bool {
	return atomic.LoadUint32(&g.state) == gateOpen
}
