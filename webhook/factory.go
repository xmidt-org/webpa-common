package webhook

import (
	"bytes"
	"encoding/json"
	"github.com/Comcast/webpa-common/logging"
	AWS "github.com/Comcast/webpa-common/webhook/aws"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultUndertakerInterval time.Duration = time.Minute
)

// Factory is a classic Factory Object for various webhook things.
type Factory struct {
	// Other configuration stuff can go here
	cfg *AWS.AWSConfig

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

	f.cfg, err = AWS.NewAWSConfig(v)

	return
}

// monitor pointer instance shared between functions
var m *monitor

// log instance shared between functions
var log logging.Logger

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
	m = monitor

	go monitor.listen()
	return monitor.list, monitor
}

// Initialize i.e. prepare/set up required for processing webhooks
func (f *Factory) Initialize(logger logging.Logger, rtr *mux.Router,
	selfUrl url.URL) (err error) {

	if logger != nil {
		log = logger
	} else {
		log = logging.DefaultLogger()
	}
	m.server, err = AWS.NewSNSServer(f.cfg, logger, rtr, selfUrl, m)

	go m.server.Prepare()

	return
}

// monitor is an internal type that listens for webhook updates, invokes
// the undertaker at specified intervals, and responds to HTTP requests.
type monitor struct {
	list             UpdatableList
	undertaker       func([]W) []W
	changes          chan []W
	undertakerTicker <-chan time.Time
	server           *AWS.SNSServer
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

// ServeHTTP is used as POST handler for AWS SNS
// It transforms the message containing webhook to []W and updates the webhook list
func (m *monitor) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// TODO: transform a request into a []W
	message := m.server.NotificationHandle(response, request)

	// transform message to W
	bufStr := bytes.NewBufferString(message)
	msg := bufStr.Bytes()

	newHook := new(W)
	err := json.Unmarshal(msg, newHook)
	if err != nil {
		log.Error("JSON unmarshall of Notification Message to webhook failed - %v", err)
		return
	}

	var update []W
	update = make([]W, 1)
	update[0] = *newHook

	select {
	case m.changes <- update:
	default:
	}
}
