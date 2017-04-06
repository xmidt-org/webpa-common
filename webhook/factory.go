package webhook

import (
	"github.com/spf13/viper"
	"net/http"
	"time"
)

const (
	DefaultUndertakerInterval time.Duration = time.Minute
)

// Factory is a classic Factory Object for various webhook things.
type Factory struct {
	// Other configuration stuff can go here

	// Tick is an optional function that produces a channel for time ticks.
	// Test code can set this field to something that returns a channel under the control of the test.
	Tick func(time.Duration) <-chan time.Time `json:"-"`

	// UndertakerInterval is how often the Undertaker is invoked
	UndertakerInterval time.Duration `json:"undertakerInterval"`

	// Undertaker is set by clients after reading in a Factory from some external source.
	// The associated Undertaker is immutable after construction.
	Undertaker func([]W) []W `json:"-"`
}

// NewFactory creates a Factory from a Viper environment.  This function always returns
// a non-nil Factory instance.
//
// This example uses Viper, which I highly recommend.  You could just pass an io.Reader too, and use
// the encoding/json package.  In any case, allowing the configuration source to be nil makes a lot
// of things easier on clients, like creating a test Factory for tests in client code.
func NewFactory(v *viper.Viper) (f *Factory, err error) {
	f = &Factory{
	/* put in any system defaults here.  they won't be overridden by Viper unless they are present in external configuration */
	}

	// allowing the viper instance to be nil allows a client to do
	// NewFactory(nil) to get a default Factory instance
	if v != nil {
		err = v.Unmarshal(f)
	}

	return
}

// NewListAndHandler returns a List instance for accessing webhooks and an HTTP handler
// which can receive updates from external systems.
func (f *Factory) NewListAndHandler() (List, http.Handler) {
	tick := f.Tick
	if tick == nil {
		tick = time.Tick
	}

	monitor := &monitor{
		list:             NewList(nil),
		undertaker:       f.Undertaker,
		changes:          make(chan []W, 10),
		undertakerTicker: tick(f.UndertakerInterval),
	}

	go monitor.listen()
	return monitor.list, monitor
}

// monitor is an internal type that listens for webhook updates, invokes
// the undertaker at specified intervals, and responds to HTTP requests.
type monitor struct {
	list             UpdatableList
	undertaker       func([]W) []W
	changes          chan []W
	undertakerTicker <-chan time.Time
}

func (m *monitor) listen() {
	for {
		select {
		case update := <-m.changes:
			m.list.Update(update)
		case <-m.undertakerTicker:
			m.list.Filter(m.undertaker)
		}
	}
}

func (m *monitor) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// TODO: transform a request into a []W
	var update []W

	select {
	case m.changes <- update:
	default:
	}
}
