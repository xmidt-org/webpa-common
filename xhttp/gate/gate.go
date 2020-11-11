package gate

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-kit/kit/metrics/discard"
	"github.com/xmidt-org/webpa-common/xmetrics"
	"github.com/xmidt-org/wrp-go/v2"
)

const (
	// Open is the value a gauge is set to that indicates the gate is open
	Open float64 = 1.0

	// Closed is the value a gauge is set to that indicates the gate is closed
	Closed float64 = 0.0
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

	// State returns the current state (true for open, false for closed) along with the time
	// at which this gate entered that state.
	State() (bool, time.Time)

	//Filters returns a cpoy of the filters being applied in allowing requests
	Filters() map[string]map[string]bool

	//EditFilters allows for adding or removing filters from the filters map
	EditFilters(key string, value []string, add bool)

	//FilterRequest returns true if the message is allowed to pass and false if it isn't
	FilterRequest(msg wrp.Message) bool
}

// GateOption is a configuration option for a gate Interface
type GateOption func(*gate)

// WithGauge configures a gate with a metrics Gauge that tracks the state of the gate.
func WithGauge(gauge xmetrics.Setter) GateOption {
	return func(g *gate) {
		if gauge != nil {
			g.state = gauge
		} else {
			g.state = discard.NewGauge()
		}
	}
}

// New constructs a gate Interface with zero or more options.  The returned gate takes on the given
// initial state, and any configured gauge is updated to reflect this initial state.
func New(initial bool, options ...GateOption) Interface {
	g := &gate{
		open:    initial,
		now:     time.Now,
		state:   discard.NewGauge(),
		filters: make(map[string]map[string]bool),
	}

	for _, o := range options {
		o(g)
	}

	if g.open {
		g.state.Set(Open)
	} else {
		g.state.Set(Closed)
	}

	g.timestamp = g.now().UTC()
	return g
}

// gate is the internal Interface implementation
type gate struct {
	lock        sync.RWMutex
	filtersLock sync.RWMutex
	open        bool
	timestamp   time.Time
	now         func() time.Time
	filters     map[string]map[string]bool

	state xmetrics.Setter
}

func (g *gate) Raise() bool {
	defer g.lock.Unlock()
	g.lock.Lock()

	if g.open {
		return false
	}

	g.open = true
	g.state.Set(Open)
	g.timestamp = g.now().UTC()
	return true
}

func (g *gate) Lower() bool {
	defer g.lock.Unlock()
	g.lock.Lock()

	if !g.open {
		return false
	}

	g.open = false
	g.state.Set(Closed)
	g.timestamp = g.now().UTC()
	return true
}

func (g *gate) Open() bool {
	g.lock.RLock()
	open := g.open
	g.lock.RUnlock()

	return open
}

func (g *gate) State() (bool, time.Time) {
	g.lock.RLock()
	open := g.open
	timestamp := g.timestamp
	g.lock.RUnlock()

	return open, timestamp
}

func (g *gate) String() string {
	if g.Open() {
		return "open"
	} else {
		return "closed"
	}
}

func (g *gate) Filters() map[string]map[string]bool {
	filtersCopy := make(map[string]map[string]bool)

	g.filtersLock.RLock()
	for key, val := range g.filters {
		valCopy := make(map[string]bool)

		for k, v := range val {
			valCopy[k] = v
		}

		filtersCopy[key] = valCopy
	}
	g.filtersLock.RUnlock()

	return filtersCopy
}

//allows for adding and removing filters
//Will completely overwrite current filters in place
func (g *gate) EditFilters(key string, values []string, add bool) {
	g.filtersLock.Lock()
	if add {
		valuesMap := make(map[string]bool)

		for _, v := range values {
			valuesMap[v] = true
		}

		g.filters[key] = valuesMap
	} else {

		_, ok := g.filters[key]

		if ok {
			delete(g.filters, key)
		}
	}
	g.filtersLock.Unlock()
}

func (g *gate) FilterRequest(msg wrp.Message) bool {
	g.filtersLock.RLock()
	accept := true
filterLoop:
	for k, v := range g.filters {
		switch k {
		case partnerIDKey:
			for _, id := range msg.PartnerIDs {
				_, found := v[id]

				if found {
					accept = false
					break filterLoop
				}
			}
		}
	}
	g.filtersLock.RUnlock()

	return accept
}
