package device

import (
	"bytes"
	"context"
	"github.com/Comcast/webpa-common/httperror"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/wrp"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DefaultMessageTimeout  time.Duration = 2 * time.Minute
	DefaultRefreshInterval time.Duration = 10 * time.Second
	DefaultListBacklog     uint32        = 150
)

// MessageHandler is a configurable http.Handler which handles inbound WRP traffic
// to be sent to devices.
type MessageHandler struct {
	// Logger is the sink for logging output.  If not set, logging will be sent to logging.DefaultLogger().
	Logger logging.Logger

	// Decoders is the pool of wrp.Decoder objects used to decode http.Request bodies
	// sent to this handler.  This field is required.
	Decoders *wrp.DecoderPool

	// Encoders is the optional pool of wrp.Encoder objects used to encode wrp messages sent
	// as HTTP responses.  If not supplied, this handler assumes the format returned by the Router
	// is the format to be sent back in the HTTP response.
	Encoders *wrp.EncoderPool

	// Router is the device message Router to use.  This field is required.
	Router Router

	// Timeout is the optional timeout for all operations through this handler.
	// If this field is unset or is nonpositive, DefaultMessageTimeout is used instead.
	Timeout time.Duration
}

func (mh *MessageHandler) logger() logging.Logger {
	if mh.Logger != nil {
		return mh.Logger
	}

	return logging.DefaultLogger()
}

// createContext creates the Context object for routing operations.
// This method will never return nils.  There will always be a timeout on the
// returned context, which means there will always be a cancel function too.
func (mh *MessageHandler) createContext(httpRequest *http.Request) (context.Context, context.CancelFunc) {
	timeout := mh.Timeout
	if timeout < 1 {
		timeout = DefaultMessageTimeout
	}

	return context.WithTimeout(httpRequest.Context(), mh.Timeout)
}

// decodeRequest transforms an HTTP request into a device request.
func (mh *MessageHandler) decodeRequest(ctx context.Context, httpRequest *http.Request) (deviceRequest *Request, err error) {
	deviceRequest, err = DecodeRequest(httpRequest.Body, mh.Decoders)
	if err == nil {
		deviceRequest = deviceRequest.WithContext(ctx)
	}

	return
}

func (mh *MessageHandler) ServeHTTP(httpResponse http.ResponseWriter, httpRequest *http.Request) {
	ctx, cancel := mh.createContext(httpRequest)
	defer cancel()

	deviceRequest, err := mh.decodeRequest(ctx, httpRequest)
	if err != nil {
		httperror.Formatf(
			httpResponse,
			http.StatusBadRequest,
			"Could not decode WRP message: %s",
			err,
		)

		return
	}

	// deviceRequest carries the context through the routing infrastructure
	if deviceResponse, err := mh.Router.Route(deviceRequest); err != nil {
		code := http.StatusInternalServerError
		switch err {
		case ErrorInvalidDeviceName:
			code = http.StatusBadRequest
		case ErrorDeviceNotFound:
			code = http.StatusNotFound
		case ErrorNonUniqueID:
			code = http.StatusBadRequest
		case ErrorInvalidTransactionKey:
			code = http.StatusBadRequest
		case ErrorTransactionAlreadyRegistered:
			code = http.StatusBadRequest
		}

		httperror.Formatf(
			httpResponse,
			code,
			"Could not process device request: %s",
			err,
		)
	} else if deviceResponse != nil {
		if err := EncodeResponse(httpResponse, deviceResponse, mh.Encoders); err != nil {
			mh.logger().Error("Error while writing transaction response: %s", err)
		}
	}

	// if deviceReponse == nil, that just means the request was not something that represented
	// the start of a transaction.  For example, events do not carry a transaction key because
	// they do not expect responses.
}

type ConnectHandler struct {
	Logger         logging.Logger
	Connector      Connector
	ResponseHeader http.Header
}

func (ch *ConnectHandler) logger() logging.Logger {
	if ch.Logger != nil {
		return ch.Logger
	}

	return logging.DefaultLogger()
}

func (ch *ConnectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if device, err := ch.Connector.Connect(response, request, ch.ResponseHeader); err != nil {
		ch.logger().Error("Failed to connect device: %s", err)
	} else {
		ch.logger().Debug("Connected device: %s", device.ID())
	}
}

// ConnectedDeviceListener listens for connection and disconnection events and produces
// a JSON document containing information about connected devices.  It produces this document
// on a certain interval.
type ConnectedDeviceListener struct {
	// RefreshInterval is the time interval at which the cached JSON device list is updated.
	// If this field is nonpositive, DefaultRefreshInterval is used.
	RefreshInterval time.Duration

	// Tick is a factory function that produces a ticker channel and a stop function.
	// If not set, time.Ticker is used and the stop function is ticker.Stop.
	Tick func(time.Duration) (<-chan time.Time, func())

	lock           sync.Mutex
	initializeOnce sync.Once
	devices        map[Key][]byte
	changeCount    uint32
	updates        chan []byte
	shutdown       chan struct{}
}

func (l *ConnectedDeviceListener) refreshInterval() time.Duration {
	if l.RefreshInterval > 0 {
		return l.RefreshInterval
	}

	return DefaultRefreshInterval
}

// newTick returns a ticker channel and a stop function for cleanup.  If tick is set,
// that function is used.  Otherwise, a time.Ticker is created and (ticker.C, ticker.Stop) is returned.
func (l *ConnectedDeviceListener) newTick() (<-chan time.Time, func()) {
	refreshInterval := l.refreshInterval()
	if l.Tick != nil {
		return l.Tick(refreshInterval)
	}

	ticker := time.NewTicker(refreshInterval)
	return ticker.C, ticker.Stop
}

func (l *ConnectedDeviceListener) onDeviceEvent(e *Event) {
	switch e.Type {
	case Connect:
		l.lock.Lock()
		defer l.lock.Unlock()
		l.changeCount++
		l.devices[e.Device.Key()] = []byte(e.Device.String())
	case Disconnect:
		l.lock.Lock()
		defer l.lock.Unlock()
		l.changeCount++
		delete(l.devices, e.Device.Key())
	}
}

func (l *ConnectedDeviceListener) refresh() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.changeCount > 0 {
		l.changeCount = 0

		var (
			output     = bytes.NewBufferString(`{"devices":[`)
			needsComma bool
			comma      = []byte(`,`)
		)

		for _, deviceJSON := range l.devices {
			if needsComma {
				output.Write(comma)
			}

			output.Write(deviceJSON)
			needsComma = true
		}

		output.WriteString(`]}`)
		l.updates <- output.Bytes()
	}
}

// Stop stops updates coming from this listener.
func (l *ConnectedDeviceListener) Stop() {
	l.lock.Lock()
	defer l.lock.Unlock()

	if l.shutdown != nil {
		close(l.shutdown)
		close(l.updates)

		l.shutdown = nil
		l.updates = nil
	}
}

// Listen starts listening for changes to the set of connected devices.  The returned Listener may
// be placed into an Options.  This method is idempotent, and may be called to restart this handler
// after Stop is called.  If this method is called multiple times without calling Stop, it simply
// returns the same Listener and output channel.
//
// The returned channel will received updated JSON device list documents.  This channel can be
// used with ListHandler.Consume.
func (l *ConnectedDeviceListener) Listen() (Listener, <-chan []byte) {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.initializeOnce.Do(func() {
		l.devices = make(map[Key][]byte, 1000)
	})

	if l.shutdown == nil {
		l.shutdown = make(chan struct{})
		l.updates = make(chan []byte, 1)

		// spawn the monitor goroutine
		go func(shutdown <-chan struct{}) {
			refreshC, refreshStop := l.newTick()
			defer refreshStop()

			for {
				select {
				case <-shutdown:
					return
				case <-refreshC:
					l.refresh()
				}
			}
		}(l.shutdown)
	}

	return l.onDeviceEvent, l.updates
}

// ListHandler is an HTTP handler which can take updated JSON device lists.
type ListHandler struct {
	initializeOnce sync.Once
	cachedJSON     atomic.Value
}

// Consume spawns a goroutine that processes updated JSON from the given channel.
// This method can be called multiple times with different update sources.  Typically,
// this method is called once to consume updates from a ConnectedDeviceListener.
func (lh *ListHandler) Consume(updates <-chan []byte) {
	lh.initializeOnce.Do(func() {
		lh.cachedJSON.Store([]byte(`{"devices":[]}`))
	})

	go func() {
		for updatedJson := range updates {
			lh.cachedJSON.Store(updatedJson)
		}
	}()
}

// ServeHTTP emits the cached JSON into the response.  If Listen has not been called yet,
// or if for any reason there is no cached JSON, this handler returns http.StatusServiceUnavailable.
func (lh *ListHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if jsonResponse, _ := lh.cachedJSON.Load().([]byte); len(jsonResponse) > 0 {
		response.Header().Set("Content-Type", "application/json")
		response.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
		response.Write(jsonResponse)
	} else {
		response.WriteHeader(http.StatusServiceUnavailable)
	}
}
