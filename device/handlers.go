// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package device

import (
	"bytes"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/xhttp"
	"github.com/xmidt-org/wrp-go/v3"
	"go.uber.org/zap"
)

const (
	DefaultMessageTimeout time.Duration = 2 * time.Minute
	DefaultListRefresh    time.Duration = 10 * time.Second
)

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
					"failed to extract device ID: %s",
					err,
				)

				return
			}

			ctx := WithID(request.Context(), id)
			delegate.ServeHTTP(response, request.WithContext(ctx))
		})
	}
}

// MessageHandler is a configurable http.Handler which handles inbound WRP traffic
// to be sent to devices.
type MessageHandler struct {
	// Logger is the sink for logging output.  If not set, logging will be sent to a NOP logger
	Logger *zap.Logger

	// Router is the device message Router to use.  This field is required.
	Router Router
}

func (mh *MessageHandler) logger() *zap.Logger {
	if mh.Logger != nil {
		return mh.Logger
	}

	return sallust.Default()
}

// decodeRequest transforms an HTTP request into a device request.
func (mh *MessageHandler) decodeRequest(httpRequest *http.Request) (deviceRequest *Request, err error) {
	// nolint: typecheck
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
		mh.logger().Error("Unable to decode request", zap.Error(err))
		xhttp.WriteErrorf(
			httpResponse,
			http.StatusBadRequest,
			"Unable to decode request: %s",
			err,
		)

		return
	}

	// nolint: typecheck
	responseFormat, err := wrp.FormatFromContentType(httpRequest.Header.Get("Accept"), deviceRequest.Format)
	if err != nil {
		mh.logger().Error("Unable to determine response WRP format", zap.Error(err))
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

		mh.logger().Error("Could not process device request", zap.Error(err), zap.Int("code", code))
		httpResponse.Header().Set("X-Xmidt-Message-Error", err.Error())
		xhttp.WriteErrorf(
			httpResponse,
			code,
			"Could not process device request: %s",
			err,
		)
	} else if deviceResponse != nil {
		if err := EncodeResponse(httpResponse, deviceResponse, responseFormat); err != nil {
			mh.logger().Error("Error while writing transaction response", zap.Error(err))
		}
	}

	// if deviceReponse == nil, that just means the request was not something that represented
	// the start of a transaction.  For example, events do not carry a transaction key because
	// they do not expect responses.
}

// ConnectHandler is used to initiate a concurrent connection between a Talaria and a device by upgrading a http connection to a websocket
type ConnectHandler struct {
	Logger         *zap.Logger
	Connector      Connector
	ResponseHeader http.Header
}

func (ch *ConnectHandler) logger() *zap.Logger {
	if ch.Logger != nil {
		return ch.Logger
	}

	return sallust.Default()
}

func (ch *ConnectHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if device, err := ch.Connector.Connect(response, request, ch.ResponseHeader); err != nil {
		ch.logger().Error("Failed to connect device", zap.Error(err))
	} else {
		ch.logger().Debug("Connected device", zap.String("id", string(device.ID())))
	}
}

// ListHandler is an HTTP handler which can take updated JSON device lists.
type ListHandler struct {
	Logger   *zap.Logger
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

			// nolint: typecheck
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
	lh.Logger.Debug("ServeHTTP", zap.String("handler", "ListHandler"))
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
	Logger   *zap.Logger
	Registry Registry
	Variable string
}

func (sh *StatHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	sh.Logger.Debug("ServeHTTP", zap.String("handler", "StatHandler"))
	vars := mux.Vars(request)
	if len(vars) == 0 {
		sh.Logger.Error("no path variables present for request")
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	name, ok := vars[sh.Variable]
	if !ok {
		sh.Logger.Error("missing path variable", zap.String("variable", sh.Variable))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	id, err := ParseID(name)
	if err != nil {
		sh.Logger.Error("unable to parse identifier", zap.Error(err), zap.String("deviceName", name))
		response.WriteHeader(http.StatusBadRequest)
		return
	}

	d, ok := sh.Registry.Get(id)
	if !ok {
		response.WriteHeader(http.StatusNotFound)
		return
	}

	// nolint: typecheck
	data, err := d.MarshalJSON()
	if err != nil {
		sh.Logger.Error("unable to marshal device as JSON", zap.Error(err), zap.String("deviceName", name))
		response.WriteHeader(http.StatusInternalServerError)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.Write(data)
}
