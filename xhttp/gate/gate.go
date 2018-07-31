package gate

import (
	"fmt"
	"sync/atomic"

	"github.com/Comcast/webpa-common/xmetrics"
	"github.com/go-kit/kit/metrics/discard"
)

const (
	// Open indicates a gate that allows traffic
	Open uint32 = iota

	// Closed indicates a gate that disallows traffic
	Closed
)

const (
	// GaugeOpen is the value a gauge is set to that indicates the gate is open
	GaugeOpen float64 = 1.0

	// GaugeClosed is the value a gauge is set to that indicates the gate is closed
	GaugeClosed float64 = 0.0
)

// Interface represents a concurrent condition indicating whether HTTP traffic should be allowed.
// This type essentially represents an atomic boolean with some extra functionality, such as metrics gathering.
type Interface interface {
	fmt.Stringer

	// Raise opens this gate.  If the gate was raised as a result, this method returns true.  If the
	// gate was already raised, this method returns false.
	Raise() bool

	// Lower closes this gate.  If the gate was lowered as a result, this method returns true.  If the
	// gate was already lowered, this method returns false.
	Lower() bool

	// Open tests if this gate is open
	Open() bool
}

// GateOption is a configuration option for a gate Interface
type GateOption func(*gate)

// WithGauge configures a gate with a metrics Gauge that tracks the state of the gate.
func WithGauge(gauge xmetrics.Setter) GateOption {
	return func(g *gate) {
		if gauge != nil {
			g.gauge = gauge
		} else {
			g.gauge = discard.NewGauge()
		}
	}
}

// New constructs a gate Interface with zero or more options.  The returned gate takes on the given
// initial state, and any configured gauge is updated to reflect this initial state.
func New(initial uint32, options ...GateOption) Interface {
	if initial != Open && initial != Closed {
		panic("invalid initial state")
	}

	g := &gate{
		state: initial,
		gauge: discard.NewGauge(),
	}

	for _, o := range options {
		o(g)
	}

	if g.state == Open {
		g.gauge.Set(GaugeOpen)
	} else {
		g.gauge.Set(GaugeClosed)
	}

	return g
}

// gate is the internal Interface implementation
type gate struct {
	state uint32
	gauge xmetrics.Setter
}

func (g *gate) Raise() bool {
	if atomic.CompareAndSwapUint32(&g.state, Closed, Open) {
		g.gauge.Set(GaugeOpen)
		return true
	}

	return false
}

func (g *gate) Lower() bool {
	if atomic.CompareAndSwapUint32(&g.state, Open, Closed) {
		g.gauge.Set(GaugeClosed)
		return true
	}

	return false
}

func (g *gate) Open() bool {
	return atomic.LoadUint32(&g.state) == Open
}

func (g *gate) String() string {
	if g.Open() {
		return "open"
	} else {
		return "closed"
	}
}
