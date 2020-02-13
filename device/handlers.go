package device

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/xmidt-org/webpa-common/logging"
	"github.com/xmidt-org/webpa-common/xhttp"
	"github.com/xmidt-org/wrp-go/v2"
)

const (
	DefaultMessageTimeout time.Duration = 2 * time.Minute
	DefaultListRefresh    time.Duration = 10 * time.Second
)

// Timeout returns an Alice-style constructor which enforces a timeout for all device request contexts.
func Timeout(o *Options) func(http.Handler) http.Handler {
	timeout := o.requestTimeout()
	return func(delegate http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			ctx, cancel := context.WithTimeout(request.Context(), timeout)
			defer cancel()
			delegate.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

// IDFromRequest is a strategy type for extracting the device identifier from an HTTP request
type IDFromRequest func(*http.Request) (ID, error)

// UseID is a collection of Alice-style constructors that all insert the device ID
// into the delegate's request Context using various strategies.
var UseID = struct {
	// F is a configurable constructor that allows an arbitrary IDFromRequest strategy
	F func(IDFromRequest) func(http.Handler) http.Handler

	// FromHeader uses the device name header to extract the device identifier.
	// This constructor isn't configurable, and is used as-is: device.UseID.FromHeader.
	FromHeader func(http.Handler) http.Handler

	// FromPath is a configurable constructor that extracts the device identifier
	// from the URI path using the supplied variable name.  This constructor is
	// configurable: device.UseID.FromPath("deviceId").
	FromPath func(string) func(http.Handler) http.Handler
}{
	F: useID,

	FromHeader: useID(
		func(request *http.Request) (ID, error) {
			deviceName := request.Header.Get(DeviceNameHeader)
			if len(deviceName) == 0 {
				return invalidID, ErrorMissingDeviceNameHeader
			}

			return ParseID(deviceName)
		},
	),

	FromPath: func(variableName string) func(http.Handler) http.Handler {
		return useID(
			func(request *http.Request) (ID, error) {
				vars := mux.Vars(request)
				if vars == nil {
					return invalidID, ErrorMissingPathVars
				}

				deviceName := vars[variableName]
				if len(deviceName) == 0 {
					return invalidID, ErrorMissingDeviceNameVar
				}

				return ParseID(deviceName)
			},
		)
	},
}

// useID is the general purpose creator for an Alice-style constructor that passes the ID
// to the delegate via the request Context.  This internal function is exported via UseID.F.
func useID(f IDFromRequest) func(http.Handler) http.Handler {
	return func(delegate http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			id, err := f(request)
			if err != nil {
				xhttp.WriteErrorf(
					response,
					http.StatusBadRequest,
					"Could extract device id: %s",
					err,
				)

				return
			}

			ctx := WithID(id, request.Context())
			delegate.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

// MessageHandler is a configurable http.Handler which handles inbound WRP traffic
// to be sent to devices.
type MessageHandler struct {
	// Logger is the sink for logging output.  If not set, logging will be sent to a NOP logger
	Logger log.Logger

	// Router is the device message Router to use.  This field is required.
	Router Router
}

func (mh *MessageHandler) logger() log.Logger {
	if mh.Logger != nil {
		return mh.Logger
	}

	return logging.DefaultLogger()
}

// decodeRequest transforms an HTTP request into a device request.
func (mh *MessageHandler) decodeRequest(httpRequest *http.Request) (deviceRequest *Request, err error) {
	format, err := wrp.FormatFromContentType(httpRequest.Header.Get("Content-Type"), wrp.Msgpack)
	if err != nil {
		return nil, err
	}

	deviceRequest, err = DecodeRequest(httpRequest.Body, format)
	if err == nil {
		deviceRequest = deviceRequest.WithContext(httpRequest.Context())
	}

	return
}

func (mh *MessageHandler) ServeHTTP(httpResponse http.ResponseWriter, httpRequest *http.Request) {
	deviceRequest, err := mh.decodeRequest(httpRequest)
	if err != nil {
		mh.logger().Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Unable to decode request", logging.ErrorKey(), err)
		xhttp.WriteErrorf(
			httpResponse,
			http.StatusBadRequest,
			"Unable to decode request: %s",
			err,
		)

		return
	}

	responseFormat, err := wrp.FormatFromContentType(httpRequest.Header.Get("Accept"), deviceRequest.Format)
	if err != nil {
		mh.logger().Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Unable to determine response WRP format", logging.ErrorKey(), err)
		xhttp.WriteErrorf(
			httpResponse,
			http.StatusBadRequest,
			"Unable to determine response WRP format: %s",
			err,
		)

		return
	}

	// deviceRequest carries the context through the routing infrastructure
	if deviceResponse, err := mh.Router.Route(deviceRequest); err != nil {
		code := http.StatusGatewayTimeout
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

		mh.logger().Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not process device request", logging.ErrorKey(), err, "code", code)
		httpResponse.Header().Set("X-Xmidt-Message-Error", err.Error())
		xhttp.WriteErrorf(
			httpResponse,
			code,
			"Could not process device request: %s",
			err,
		)
	} else if deviceResponse != nil {
		if err := EncodeResponse(httpResponse, deviceResponse, responseFormat); err != nil {
			mh.logger().Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Error while writing transaction response", logging.ErrorKey(), err)
		}
	}

	// if deviceReponse == nil, that just means the request was not something that represented
	// the start of a transaction.  For example, events do not carry a transaction key because
	// they do not expect responses.
}

// ConnectHandler is used to initiate a concurrent connection between a Talaria and a device by upgrading a http connection to a websocket
type ConnectHandler struct {
	Logger         log.Logger
	Connector      Connector
	ResponseHeader http.Header
}

func (ch *ConnectHandler) logger() log.Logger {
	if ch.Logger != nil {
		return ch.Logger
	}

	return logging.DefaultLogger()
}

func (ch *ConnectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if device, err := ch.Connector.Connect(response, request, ch.ResponseHeader); err != nil {
		logging.Error(ch.logger()).Log(logging.MessageKey(), "Failed to connect device", logging.ErrorKey(), err)
	} else {
		logging.Debug(ch.logger()).Log(logging.MessageKey(), "Connected device", "id", device.ID())
	}
}

// ListHandler is an HTTP handler which can take updated JSON device lists.
type ListHandler struct {
	Logger   log.Logger
	Registry Registry
	Refresh  time.Duration

	lock        sync.RWMutex
	cacheExpiry time.Time
	cache       bytes.Buffer
	cacheBytes  []byte

	now func() time.Time
}

func (lh *ListHandler) refresh() time.Duration {
	if lh.Refresh < 1 {
		return DefaultListRefresh
	}

	return lh.Refresh
}

func (lh *ListHandler) _now() time.Time {
	if lh.now != nil {
		return lh.now()
	}

	return time.Now()
}

// tryCache returns the currently cache JSON bytes along with a flag indicating expiry.
// This method returns true if the cached JSON bytes have expired, false otherwise.
func (lh *ListHandler) tryCache() ([]byte, bool) {
	defer lh.lock.RUnlock()
	lh.lock.RLock()

	return lh.cacheBytes, lh.cacheExpiry.Before(lh._now())
}

func (lh *ListHandler) updateCache() []byte {
	defer lh.lock.Unlock()
	lh.lock.Lock()

	if lh.cacheExpiry.Before(lh._now()) {
		lh.cache.Reset()
		lh.cache.WriteString(`{"devices":[`)

		needsSeparator := false
		lh.Registry.VisitAll(func(d Interface) bool {
			if needsSeparator {
				lh.cache.WriteString(`,`)
			}

			if data, err := d.MarshalJSON(); err != nil {
				lh.cache.WriteString(
					fmt.Sprintf(`{"id": "%s", "error": "%s"}`, d.ID(), err),
				)
			} else {
				lh.cache.Write(data)
			}

			needsSeparator = true
			return true
		})

		lh.cache.WriteString(`]}`)
		lh.cacheBytes = lh.cache.Bytes()
		lh.cacheExpiry = lh._now().Add(lh.refresh())
	}

	return lh.cacheBytes
}

func (lh *ListHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	lh.Logger.Log(level.Key(), level.DebugValue(), "handler", "ListHandler", logging.MessageKey(), "ServeHTTP")
	response.Header().Set("Content-Type", "application/json")

	if cacheBytes, expired := lh.tryCache(); expired {
		response.Write(lh.updateCache())
	} else {
		response.Write(cacheBytes)
	}
}

// StatHandler is an http.Handler that returns device statistics.  The device name is specified
// as a gorilla path variable.
type StatHandler struct {
	Logger   log.Logger
	Registry Registry
	Variable string
}

func (sh *StatHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	sh.Logger.Log(level.Key(), level.DebugValue(), "handler", "StatHandler", logging.MessageKey(), "ServeHTTP")
	vars := mux.Vars(request)
	if len(vars) == 0 {
		sh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "no path variables present for request")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	name, ok := vars[sh.Variable]
	if !ok {
		sh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "missing path variable", "variable", sh.Variable)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := ParseID(name)
	if err != nil {
		sh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to parse identifier", "deviceName", name, logging.ErrorKey(), err)
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	d, ok := sh.Registry.Get(id)
	if !ok {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	data, err := d.MarshalJSON()
	if err != nil {
		sh.Logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "unable to marshal device as JSON", "deviceName", name, logging.ErrorKey(), err)
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(data)
}
