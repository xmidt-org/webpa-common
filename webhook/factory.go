package webhook

import (
	"encoding/json"
	"github.com/gorilla/mux"
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
		Tick:               time.Tick,
		UndertakerInterval: 1 * time.Second,
		Undertaker:         f.RemoveExpiredHooks,
	}

	// allowing the viper instance to be nil allows a client to do
	// NewFactory(nil) to get a default Factory instance
	if v != nil {
		err = v.Unmarshal(f)
	}

	return
}

func (f *Factory) RemoveExpiredHooks() {

}

// NewListAndHandler returns a List instance for accessing webhooks and an HTTP handler
// which can receive updates from external systems.
func (f *Factory) NewListAndHandler() (List, http.Handler) {
	tick := f.Tick
	if tick == nil {
		tick = time.Tick
	}

	// Populate list from the static json
	hooks := LoadHooksList()

	monitor := &monitor{
		list:             NewList(hooks),
		undertaker:       f.Undertaker,
		changes:          make(chan []W, 10),
		undertakerTicker: tick(f.UndertakerInterval),
		updateTicker:     tick(5 * time.Minute),
	}

	go monitor.listen()
	return monitor.list, monitor.webHookPostHandler
}

func LoadHooksList() []W {
	// Load hoooks-list.json and populate in W[]
	file, err := os.Open("hooks-list.json")
	defer file.Close()
	if nil != err {
		fmt.Printf("Failed to open file: %v\n", err)
		return
	}
	var hooks []W
	err := json.UnMarshal(file, &hooks)
	if err != nil {
		fmt.Println("error:", err)
	}
	fmt.Println("%+v", hooks)
	return hooks
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
		case <-m.updateTicker:
			// update until
			var update []W
			len := m.list.Len()
			update = make([]W, len)
			i := 0
			for i < len {
				update[i] = *m.list.Get(i)
				update[i].Until = time.Now().Unix() + 300
			}
			m.list.Update(update)
		}
	}
}

func (m *monitor) webHookPostHandler(response http.ResponseWriter, request *http.Request) {
	// TODO: transform a request into a []W
	update := make([]W, 1)

	inHook := new(W)
	_, err := DecodeJsonPayload(request, inHook)
	if err != nil {
		log.Error("JSON decoding error %v", err.Error())
		TS.ResponseJsonErr(rw, "JSON decoding error", http.StatusBadRequest)
		return
	}

	p, err := ParseIP(requesr.RemoteAddr)
	if err != nil {
		log.Error("%v: %v", err)
	}
	inHook.Address = ip

	if inHook.Config.URL == "" {
		log.Error("invalid config.url")
		TS.ResponseJsonErr(rw, "invalid Config URL", http.StatusBadRequest)
		return http.StatusBadRequest
	}
	if inHook.Config.ContentType == "" || inHook.Config.ContentType != "json" {
		log.Error("invalid config.content_type %v.", inHook.Config.ContentType)
		TS.ResponseJsonErr(rw, "invalid config.content_type", http.StatusBadRequest)
		return http.StatusBadRequest
	}
	/*	if len(inHook.Matcher.DeviceId) == 0 {
			inHook.Matcher.DeviceId = []string{".*"} // match anything
		}
		if len(inHook.Events) == 0 {
			log.Error("invalid config.events %v.", inHook.Events)
			TS.ResponseJsonErr(rw, "invalid config.events", http.StatusBadRequest)
			return http.StatusBadRequest
		}
		groupNames := inHook.Groups
		if len(groupNames) == 0 {
			log.Debug("Empty group name")
		}

		found := false
		for _, hk := range wh.Hooks {
			if hk.Config.URL == inHook.Config.URL {
				found = true
				if inHook.Duration > 0 && inHook.Duration < wh.HookExpireSec {
					hk.Until = time.Now().Unix() + inHook.Duration
				} else {
					hk.Until = time.Now().Unix() + wh.HookExpireSec
				}
				hk.Matcher = inHook.Matcher
				hk.Events = inHook.Events
				hk.Groups = inHook.Groups
				hk.Config.ContentType = inHook.Config.ContentType
				hk.Config.Secret = inHook.Config.Secret
				log.Trace("hook already exists, until value updated %+v", hk)
				break
			}
		}
		if !found {
			inHook.Until = time.Now().Unix() + wh.HookExpireSec
			wh.Hooks = append(wh.Hooks, inHook)
			log.Trace("register hook %#v is added", inHook)
		}*/

	select {
	case m.changes <- update:
	default:
	}

}

func DecodeJsonPayload(req *http.Request, v interface{}) ([]byte, error) {
	ErrJsonEmpty := errors.New("JSON payload is empty")
	if req.Body == nil {
		return nil, ErrJsonEmpty
	}

	payload, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, ErrJsonEmpty
	}
	err = json.Unmarshal([]byte(payload), v)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
