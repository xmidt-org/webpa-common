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

// ListHandler is a handler which serves a JSON document containing connected devices.  This
// handler listens for connection and disconnection events and concurrently updates a cached
// JSON document.  This cached document is updated on the RefreshInterval.
type ListHandler struct {
	// RefreshInterval is the time interval at which the cached JSON device list is updated.
	// If this field is nonpositive, DefaultRefreshInterval is used.
	RefreshInterval time.Duration

	// Backlog is the number of connections and disconnections (each) allowed to queue up
	// internally.  If this field is not positive, DefaultListBacklog is used.
	Backlog uint32

	// Tick is a factory function that produces a ticker channel and a stop function.
	// If not set, time.Ticker is used and the stop function is ticker.Stop.
	Tick func(time.Duration) (<-chan time.Time, func())

	initializeOnce sync.Once

	lock sync.Mutex

	// cachedJSON holds the []byte of JSON holding information about connected devices
	cachedJSON atomic.Value

	// connections is the channel on which new devices are sent.  this
	// channel is never closed
	connections chan Interface

	// disconnections is the channel on which disconnected devices are sent.
	// this channel is never closed.
	disconnections chan Interface

	// shutdown is the signal channel indicating that this handler should stop listening.
	// it is closed and recreated as needed.
	shutdown chan struct{}
}

func (lh *ListHandler) refreshInterval() time.Duration {
	if lh.RefreshInterval > 0 {
		return lh.RefreshInterval
	}

	return DefaultRefreshInterval
}

func (lh *ListHandler) backlog() uint32 {
	if lh.Backlog > 0 {
		return lh.Backlog
	}

	return DefaultListBacklog
}

// newTick returns a ticker channel and a stop function for cleanup.  If tick is set,
// that function is used.  Otherwise, a time.Ticker is created and (ticker.C, ticker.Stop) is returned.
func (lh *ListHandler) newTick() (<-chan time.Time, func()) {
	refreshInterval := lh.refreshInterval()
	if lh.Tick != nil {
		return lh.Tick(refreshInterval)
	}

	ticker := time.NewTicker(refreshInterval)
	return ticker.C, ticker.Stop
}

// Stop stops listening for device connections and disconnections.  Be aware that
// OnDeviceEvent may block while this handler is not listening.
// Use this method with great care.
func (lh *ListHandler) Stop() {
	lh.lock.Lock()
	defer lh.lock.Unlock()

	if lh.shutdown == nil {
		return
	}

	close(lh.shutdown)
	lh.shutdown = nil
}

// Listen starts listening for device connections and disconnections.  This method
// is idempotent and should be called before content is served.
//
// Devices that are connected prior to Listen being called will not appear in the
// cached JSON.  For this reason, Listen must be called once, before the manager
// is created.
func (lh *ListHandler) Listen() {
	lh.lock.Lock()
	defer lh.lock.Unlock()

	if lh.shutdown != nil {
		return
	}

	lh.initializeOnce.Do(func() {
		lh.cachedJSON.Store([]byte(`{"devices":[]}`))
		lh.connections = make(chan Interface, lh.backlog())
		lh.disconnections = make(chan Interface, lh.backlog())
	})

	lh.shutdown = make(chan struct{})
	go lh.monitor(lh.shutdown)
}

// OnDeviceEvent receives events from a Manager.  This function must be
// inserted into Options.Listeners prior to creating a Manager.
func (lh *ListHandler) OnDeviceEvent(e *Event) {
	switch e.Type {
	case Connect:
		lh.connections <- e.Device
	case Disconnect:
		lh.disconnections <- e.Device
	}
}

// refresh takes a map of devices and builds a new cached JSON byte array
func (lh *ListHandler) refresh(devices map[Key][]byte) {
	var (
		output     = bytes.NewBufferString(`{"devices":[`)
		needsComma bool
	)

	for _, deviceJSON := range devices {
		if needsComma {
			output.WriteString(`,`)
		}

		output.Write(deviceJSON)
		needsComma = true
	}

	output.WriteString(`]}`)
	lh.cachedJSON.Store(output.Bytes())
}

// monitor is the goroutine that monitors the channels and collects changes.
// periodically, this function will invoke refresh.
func (lh *ListHandler) monitor(shutdown <-chan struct{}) {
	var (
		changeCount   uint32
		devices       = make(map[Key][]byte, 1000)
		refresh, stop = lh.newTick()
	)

	defer stop()

	for {
		select {
		case <-shutdown:
			return
		case newDevice := <-lh.connections:
			devices[newDevice.Key()] = []byte(newDevice.String())
			changeCount++
		case disconnectedDevice := <-lh.disconnections:
			delete(devices, disconnectedDevice.Key())
			changeCount++
		case <-refresh:
			if changeCount > 0 {
				lh.refresh(devices)
			}

			changeCount = 0
		}
	}
}

func (lh *ListHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	jsonResponse := lh.cachedJSON.Load().([]byte)

	response.Header().Set("Content-Type", "application/json")
	response.Header().Set("Content-Length", strconv.Itoa(len(jsonResponse)))
	response.Write(jsonResponse)
}
